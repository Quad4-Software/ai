// SPDX-License-Identifier: 0BSD
// Package mcp implements a minimal MCP server over stdio using
// newline-delimited JSON-RPC 2.0, stdlib only. It supports the tools,
// prompts, and resources capabilities.
//
// Reliability contract:
//   - malformed input produces a JSON-RPC parse error, never a crash
//   - a panicking tool handler returns isError content, the server lives
//   - EOF on stdin exits cleanly
//   - initialize advertises instructions so agents know the happy path
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

const ProtocolVersion = "2025-11-25"

const (
	// DefaultMaxToolOutputBytes caps how many bytes a single tool call
	// may return. Larger results are rejected with an error instead of
	// being truncated, so agents never see half a JSON document.
	DefaultMaxToolOutputBytes = 8 << 20 // 8 MiB
	// DefaultToolCallTimeout bounds one tool invocation. A handler that
	// ignores its context still cannot stall the read loop forever;
	// the call is abandoned and reported as an error.
	DefaultToolCallTimeout = 30 * time.Second
)

// knownVersions are protocol versions this server is compatible with.
// Newer is backward-compatible for the methods we implement.
var knownVersions = map[string]bool{
	"2024-11-05": true, "2025-03-26": true, "2025-06-18": true, "2025-11-25": true,
}

// negotiateVersion returns the client's version when known, else ours.
func negotiateVersion(client string) string {
	if knownVersions[client] {
		return client
	}
	return ProtocolVersion
}

// Tool is a callable MCP tool. Handle returns plain text content.
type Tool struct {
	Name        string
	Description string
	// InputSchema is a JSON Schema object for the tool arguments.
	InputSchema map[string]any
	// InputExamples shows agents concrete valid calls (2025-11-25+).
	InputExamples []map[string]any `json:"inputExamples,omitempty"`
	// Write marks a mutating tool; it is hidden/blocked in read-only mode.
	Write         bool
	Handle        func(ctx context.Context, args json.RawMessage) (string, error)
}

// PromptArg describes one argument of a prompt template.
type PromptArg struct {
	Name        string
	Description string
	Required    bool
}

// Prompt is an MCP prompt template. Handle returns the user message text.
type Prompt struct {
	Name        string
	Description string
	Arguments   []PromptArg
	Handle      func(args map[string]string) (string, error)
}

// Resource is a readable MCP resource. Handle returns the resource
// content as text. Read handlers must stay inside their allowed root
// and never return secret material, same rules as tools.
type Resource struct {
	// URI is the resource identifier, for example docs://readme.
	URI         string
	Name        string
	Description string
	// MIMEType is the content type, for example text/plain or
	// application/json. Empty defaults to text/plain on read.
	MIMEType string
	Handle   func(ctx context.Context) (string, error)
}

type Server struct {
	name      string
	version   string
	tools     map[string]Tool
	torder    []string
	prompts   map[string]Prompt
	porder    []string
	resources map[string]Resource
	rorder    []string

	// MaxToolOutputBytes caps tool result size in bytes; <= 0 uses
	// DefaultMaxToolOutputBytes. Oversized output is an error, never
	// silently truncated.
	MaxToolOutputBytes int
	// ToolCallTimeout bounds a single tool call; <= 0 uses
	// DefaultToolCallTimeout.
	ToolCallTimeout time.Duration

	// ReadOnly disables all tools with Write: true when set.
	ReadOnly bool

	tasksMu sync.Mutex
	tasks   map[string]*task
	taskSeq int
}

// task is a durable handle for a task-augmented tools/call
// (2025-11-25 experimental tasks support).
type task struct {
	ID     string
	Status string // working, completed, failed, cancelled
	Result any
	Err    string
	Cancel context.CancelFunc
}

func NewServer(name, version string, tools []Tool, prompts []Prompt) *Server {
	s := &Server{
		name:               name,
		version:            version,
		tools:              make(map[string]Tool, len(tools)),
		prompts:            make(map[string]Prompt, len(prompts)),
		tasks:              map[string]*task{},
		MaxToolOutputBytes: DefaultMaxToolOutputBytes,
		ToolCallTimeout:    DefaultToolCallTimeout,
	}
	if os.Getenv("MCP_READ_ONLY") == "1" || os.Getenv("READ_ONLY") == "1" {
		s.ReadOnly = true
	}
	for _, a := range os.Args[1:] {
		if a == "--read-only" || a == "-read-only" {
			s.ReadOnly = true
		}
	}
	for _, t := range tools {
		s.tools[t.Name] = t
		s.torder = append(s.torder, t.Name)
	}
	// deterministic order: tools/list is cacheable per 2025-11-25
	sort.Strings(s.torder)
	for _, p := range prompts {
		s.prompts[p.Name] = p
		s.porder = append(s.porder, p.Name)
	}
	s.resources = make(map[string]Resource)
	return s
}

// RegisterResources adds resources to the server after construction so
// the NewServer signature stays stable for existing callers. Duplicate
// URIs overwrite earlier registrations. The list order is sorted so
// resources/list output is deterministic.
func (s *Server) RegisterResources(resources []Resource) {
	for _, r := range resources {
		if _, exists := s.resources[r.URI]; !exists {
			s.rorder = append(s.rorder, r.URI)
		}
		s.resources[r.URI] = r
	}
	sort.Strings(s.rorder)
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return e.Message }

// Serve reads newline-delimited JSON-RPC messages from r and writes
// responses to w until EOF or a fatal read error. The output stream is
// scoped to this call, so concurrent Serve calls on one Server (each with
// its own r/w) do not race or interleave bytes.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	var outMu sync.Mutex
	out := bufio.NewWriterSize(w, 64*1024)
	send := func(res response) {
		b, err := json.Marshal(res)
		if err != nil {
			return
		}
		outMu.Lock()
		defer outMu.Unlock()
		_, _ = out.Write(b)     // #nosec G104 -- broken pipe is unrecoverable
		_ = out.WriteByte('\n') // #nosec G104
		_ = out.Flush()         // #nosec G104
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			send(response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if len(req.ID) == 0 { // notification: no response
			continue
		}
		send(s.dispatch(ctx, &req))
	}
	return sc.Err()
}

// ServeListener accepts connections from l and serves each on its own
// goroutine via Serve, so many clients share one Server and its state.
// It returns when l is closed or ctx is cancelled.
func (s *Server) ServeListener(ctx context.Context, l net.Listener) error {
	go func() {
		<-ctx.Done()
		l.Close() // #nosec G104 -- shutdown path, error irrelevant
	}()
	for {
		c, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go func() {
			defer c.Close()    // #nosec G104 -- per-conn cleanup
			s.Serve(ctx, c, c) // #nosec G104 -- per-conn errors end that conn only
		}()
	}
}

func (s *Server) dispatch(ctx context.Context, req *request) (res response) {
	res = response{JSONRPC: "2.0", ID: req.ID}
	defer func() {
		if r := recover(); r != nil {
			res = response{JSONRPC: "2.0", ID: req.ID,
				Error: &rpcError{Code: -32603, Message: fmt.Sprintf("internal error: %v", r)}}
		}
	}()
	result, err := s.handle(ctx, req.Method, req.Params)
	if err != nil {
		var re *rpcError
		if !errors.As(err, &re) {
			re = &rpcError{Code: -32603, Message: err.Error()}
		}
		res.Error = re
		return res
	}
	res.Result = result
	return res
}

func (s *Server) capabilities() map[string]any {
	caps := map[string]any{}
	if len(s.torder) > 0 {
		caps["tools"] = map[string]any{"listChanged": false}
	}
	if len(s.porder) > 0 {
		caps["prompts"] = map[string]any{"listChanged": false}
	}
	if len(s.rorder) > 0 {
		caps["resources"] = map[string]any{"listChanged": false, "subscribe": false}
	}
	// experimental tasks support (2025-11-25): task-augmented tools/call
	caps["tasks"] = map[string]any{
		"list":   map[string]any{},
		"cancel": map[string]any{},
		"requests": map[string]any{
			"tools": map[string]any{"call": true},
		},
	}
	return caps
}

func (s *Server) handle(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(params, &p) // #nosec G104 -- optional field; empty is fine
		return map[string]any{
			"protocolVersion": negotiateVersion(p.ProtocolVersion),
			"capabilities":    s.capabilities(),
			"serverInfo":      map[string]any{"name": s.name, "version": s.version},
			"instructions": fmt.Sprintf(
				"%s MCP server. Tools: %s. Read tool descriptions before calling; "+
					"validation failures return isError content that names the required arguments and valid values.",
				s.name, s.toolNames()),
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		type toolDef struct {
			Name          string           `json:"name"`
			Description   string           `json:"description"`
			InputSchema   map[string]any   `json:"inputSchema"`
			InputExamples []map[string]any `json:"inputExamples,omitempty"`
		}
		list := make([]toolDef, 0, len(s.torder))
		for _, name := range s.torder {
			t := s.tools[name]
			if s.ReadOnly && t.Write {
				continue
			}
			schema := t.InputSchema
			if len(t.InputExamples) > 0 {
				// copy so the stored schema is never mutated
				schema = make(map[string]any, len(t.InputSchema)+1)
				for k, v := range t.InputSchema {
					schema[k] = v
				}
				schema["examples"] = t.InputExamples
			}
			list = append(list, toolDef{t.Name, t.Description, schema, t.InputExamples})
		}
		return map[string]any{"tools": list}, nil
	case "resources/list":
		type resourceDef struct {
			URI         string `json:"uri"`
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			MIMEType    string `json:"mimeType,omitempty"`
		}
		list := make([]resourceDef, 0, len(s.rorder))
		for _, uri := range s.rorder {
			r := s.resources[uri]
			list = append(list, resourceDef{r.URI, r.Name, r.Description, r.MIMEType})
		}
		return map[string]any{"resources": list}, nil
	case "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		r, ok := s.resources[p.URI]
		if !ok {
			return nil, &rpcError{Code: -32602,
				Message: fmt.Sprintf("unknown resource %q", p.URI)}
		}
		mime := r.MIMEType
		if mime == "" {
			mime = "text/plain"
		}
		text, err := r.Handle(ctx)
		if err != nil {
			return nil, &rpcError{Code: -32603, Message: err.Error()}
		}
		return map[string]any{
			"contents": []map[string]any{{
				"uri":      r.URI,
				"mimeType": mime,
				"text":     text,
			}},
		}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
			Task      json.RawMessage `json:"task"` // 2025-11-25 task augmentation
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		t, ok := s.tools[p.Name]
		if !ok {
			return nil, &rpcError{Code: -32602,
				Message: fmt.Sprintf("unknown tool %q; available: %s", p.Name, s.toolNames())}
		}
		if s.ReadOnly && t.Write {
			return nil, &rpcError{Code: -32603, Message: fmt.Sprintf("tool %q is disabled in read-only mode", t.Name)}
		}
		if len(p.Task) > 0 {
			return s.startTask(ctx, t, p.Arguments), nil
		}
		text, err := s.callTool(ctx, t, p.Arguments)
		if err != nil {
			return map[string]any{
				"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}},
				"isError": true,
			}, nil
		}
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
		}, nil
	case "tasks/list":
		s.tasksMu.Lock()
		defer s.tasksMu.Unlock()
		var out []map[string]any
		for _, tk := range s.tasks {
			out = append(out, map[string]any{"taskId": tk.ID, "status": tk.Status})
		}
		sort.Slice(out, func(i, j int) bool { return out[i]["taskId"].(string) < out[j]["taskId"].(string) })
		return map[string]any{"tasks": out}, nil
	case "tasks/get":
		var p struct {
			TaskID string `json:"taskId"`
		}
		_ = json.Unmarshal(params, &p) // #nosec G104 -- empty taskId errors below
		s.tasksMu.Lock()
		defer s.tasksMu.Unlock()
		tk, ok := s.tasks[p.TaskID]
		if !ok {
			return nil, &rpcError{Code: -32602, Message: "unknown task " + p.TaskID}
		}
		return map[string]any{"taskId": tk.ID, "status": tk.Status, "error": tk.Err}, nil
	case "tasks/result":
		var p struct {
			TaskID string `json:"taskId"`
		}
		_ = json.Unmarshal(params, &p) // #nosec G104 -- empty taskId errors below
		s.tasksMu.Lock()
		tk, ok := s.tasks[p.TaskID]
		s.tasksMu.Unlock()
		if !ok {
			return nil, &rpcError{Code: -32602, Message: "unknown task " + p.TaskID}
		}
		if tk.Status == "working" || tk.Status == "cancelled" {
			return nil, &rpcError{Code: -32602, Message: "task " + p.TaskID + " is " + tk.Status}
		}
		return tk.Result, nil
	case "tasks/cancel":
		var p struct {
			TaskID string `json:"taskId"`
		}
		_ = json.Unmarshal(params, &p) // #nosec G104 -- empty taskId errors below
		s.tasksMu.Lock()
		defer s.tasksMu.Unlock()
		tk, ok := s.tasks[p.TaskID]
		if !ok {
			return nil, &rpcError{Code: -32602, Message: "unknown task " + p.TaskID}
		}
		if tk.Cancel != nil {
			tk.Cancel()
		}
		tk.Status = "cancelled"
		return map[string]any{"taskId": tk.ID, "status": "cancelled"}, nil
	case "prompts/list":
		type argDef struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			Required    bool   `json:"required,omitempty"`
		}
		type promptDef struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Arguments   []argDef `json:"arguments,omitempty"`
		}
		list := make([]promptDef, 0, len(s.porder))
		for _, name := range s.porder {
			p := s.prompts[name]
			args := make([]argDef, 0, len(p.Arguments))
			for _, a := range p.Arguments {
				args = append(args, argDef{a.Name, a.Description, a.Required})
			}
			list = append(list, promptDef{p.Name, p.Description, args})
		}
		return map[string]any{"prompts": list}, nil
	case "prompts/get":
		var p struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		pr, ok := s.prompts[p.Name]
		if !ok {
			return nil, &rpcError{Code: -32602, Message: fmt.Sprintf("unknown prompt %q", p.Name)}
		}
		for _, a := range pr.Arguments {
			if a.Required && p.Arguments[a.Name] == "" {
				return nil, &rpcError{Code: -32602, Message: fmt.Sprintf("missing required argument %q", a.Name)}
			}
		}
		text, err := pr.Handle(p.Arguments)
		if err != nil {
			return nil, &rpcError{Code: -32603, Message: err.Error()}
		}
		return map[string]any{
			"description": pr.Description,
			"messages": []map[string]any{{
				"role":    "user",
				"content": map[string]any{"type": "text", "text": text},
			}},
		}, nil
	}
	return nil, &rpcError{Code: -32601, Message: "method not found"}
}

// startTask runs a tool in the background and returns a task handle.
// Results stay available until the process exits.
func (s *Server) startTask(ctx context.Context, t Tool, args json.RawMessage) any {
	s.tasksMu.Lock()
	s.taskSeq++
	tk := &task{ID: fmt.Sprintf("task-%d", s.taskSeq), Status: "working"}
	s.tasks[tk.ID] = tk
	s.tasksMu.Unlock()
	tctx, cancel := context.WithCancel(ctx)
	tk.Cancel = cancel
	go func() {
		text, err := s.callTool(tctx, t, args)
		s.tasksMu.Lock()
		defer s.tasksMu.Unlock()
		if tk.Status == "cancelled" {
			return
		}
		if err != nil {
			tk.Status = "failed"
			tk.Err = err.Error()
			tk.Result = map[string]any{
				"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}},
				"isError": true,
			}
			return
		}
		tk.Status = "completed"
		tk.Result = map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
		}
	}()
	return map[string]any{"task": map[string]any{"taskId": tk.ID, "status": "working"}}
}

// callTool runs one handler inside a bounded envelope:
//   - a panic becomes an error so the server keeps serving
//   - the call times out after ToolCallTimeout even if the handler
//     ignores its context (the handler runs in a goroutine and its
//     result is abandoned; the buffered channel lets a late-finishing
//     handler exit without leaking a blocked sender)
//   - results larger than MaxToolOutputBytes are rejected outright
func (s *Server) callTool(ctx context.Context, t Tool, args json.RawMessage) (text string, err error) {
	timeout := s.ToolCallTimeout
	if timeout <= 0 {
		timeout = DefaultToolCallTimeout
	}
	limit := s.MaxToolOutputBytes
	if limit <= 0 {
		limit = DefaultMaxToolOutputBytes
	}
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("tool %q panicked: %v", t.Name, r)}
			}
		}()
		txt, e := t.Handle(tctx, args)
		done <- result{text: txt, err: e}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			return "", r.err
		}
		if len(r.text) > limit {
			return "", fmt.Errorf("tool %q output exceeded %d bytes", t.Name, limit)
		}
		return r.text, nil
	case <-tctx.Done():
		return "", fmt.Errorf("tool %q timed out after %s", t.Name, timeout)
	}
}

// toolNames lists registered tool names for error messages.
func (s *Server) toolNames() string {
	return strings.Join(s.torder, ", ")
}
