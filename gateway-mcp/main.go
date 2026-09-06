// SPDX-License-Identifier: 0BSD
// Command gateway-mcp multiplexes many stdio MCP servers behind four
// tools (servers, tools, tool_schema, invoke) so agents load one small
// tool surface instead of every server's full schema set. Children are
// spawned lazily on first use and respawned if they die.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Quad4-Software/ai/gateway-mcp/internal/mcp"
)

// serverDef is one child entry from an MCP client config file.
type serverDef struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// loadConfig reads an mcp_config.json-style file. Skips url-only,
// disabled, and self-referential entries.
func loadConfig(path string) ([]serverDef, error) {
	data, err := os.ReadFile(path) // #nosec G703 G304 -- path constructed under jailed root
	if err != nil {
		return nil, err
	}
	var cfg struct {
		MCPServers map[string]struct {
			Command  string            `json:"command"`
			Args     []string          `json:"args"`
			Env      map[string]string `json:"env"`
			URL      string            `json:"url"`
			Disabled bool              `json:"disabled"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	self, _ := os.Executable()
	var out []serverDef
	for name, s := range cfg.MCPServers {
		if s.Disabled || s.Command == "" || s.URL != "" {
			continue
		}
		if strings.Contains(s.Command, "gateway-mcp") || s.Command == self || name == "gateway" {
			continue // never proxy ourselves
		}
		out = append(out, serverDef{name, s.Command, s.Args, s.Env})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// toolMeta is a child's advertised tool.
type toolMeta struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	server      string
}

// child wraps one spawned stdio MCP server. All calls are serialized
// per child; a dead process is respawned transparently.
type child struct {
	def     serverDef
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	nextID  int
	tools   []toolMeta
	started bool
	dead    error // permanent start failure

	calls    int
	errs     int
	spawns   int
	latency  time.Duration // total across calls
	lastCall time.Time
}

func (c *child) ensure() error {
	if c.dead != nil {
		return c.dead
	}
	if c.started {
		return nil
	}
	cmd := exec.Command(c.def.Command, c.def.Args...) // #nosec G204 -- fixed argv, no shell, command allowlisted
	env := os.Environ()
	for k, v := range c.def.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.dead = err
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.dead = err
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		c.dead = fmt.Errorf("start %s: %w", c.def.Name, err)
		return c.dead
	}
	c.cmd, c.stdin = cmd, stdin
	c.stdout = bufio.NewReaderSize(stdout, 1<<20)
	c.started = true
	c.spawns++
	// initialize handshake; tolerate failure, some servers are lenient
	c.callLocked("initialize", map[string]any{ // #nosec G104 -- error tolerated; empty/default is handled downstream
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "gateway-mcp", "version": "0.1.0"},
	})
	io.WriteString(c.stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n") // #nosec G104 -- error tolerated; empty/default is handled downstream
	return nil
}

// callLocked sends one request and reads lines until the matching id
// arrives. Caller must hold c.mu.
func (c *child) callLocked(method string, params any) (json.RawMessage, error) {
	c.nextID++
	req, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": c.nextID, "method": method, "params": params,
	})
	want := fmt.Sprintf(`"id":%d`, c.nextID)
	if _, err := c.stdin.Write(append(req, '\n')); err != nil {
		return nil, err
	}
	for range 10000 { // skip notifications/other traffic
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if !strings.Contains(string(line), want) {
			continue
		}
		var res struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(line, &res); err != nil {
			continue
		}
		if res.Error != nil {
			return nil, fmt.Errorf("%s: %s", c.def.Name, res.Error.Message)
		}
		return res.Result, nil
	}
	return nil, fmt.Errorf("%s: no response to %s", c.def.Name, method)
}

// call runs a method with transparent respawn on transport failure.
func (c *child) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t0 := time.Now()
	c.calls++
	c.lastCall = t0
	defer func() { c.latency += time.Since(t0) }()
	if err := c.ensure(); err != nil {
		c.errs++
		return nil, err
	}
	res, err := c.callLocked(method, params)
	if err == nil {
		return res, nil
	}
	c.errs++
	// transport broke: kill and retry once
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill() // #nosec G104 -- error tolerated; empty/default is handled downstream
	}
	c.started, c.tools = false, nil
	if c.dead != nil {
		return nil, c.dead
	}
	if e2 := c.ensure(); e2 != nil {
		return nil, fmt.Errorf("respawn %s: %w (first error: %v)", c.def.Name, e2, err)
	}
	return c.callLocked(method, params)
}

func (c *child) toolList(ctx context.Context) ([]toolMeta, error) {
	if c.tools != nil {
		return c.tools, nil
	}
	res, err := c.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var r struct {
		Tools []toolMeta `json:"tools"`
	}
	if err := json.Unmarshal(res, &r); err != nil {
		return nil, err
	}
	for i := range r.Tools {
		r.Tools[i].server = c.def.Name
	}
	c.tools = r.Tools
	return c.tools, nil
}

var children []*child

func findTool(name string) (*child, string, error) {
	srv, tool, ok := strings.Cut(name, ".")
	if !ok {
		return nil, "", fmt.Errorf("tool names are namespaced: <server>.<tool>, e.g. reticulum.list_topics")
	}
	for _, c := range children {
		if c.def.Name == srv {
			return c, tool, nil
		}
	}
	var names []string
	for _, c := range children {
		names = append(names, c.def.Name)
	}
	return nil, "", fmt.Errorf("no server %q; available: %s", srv, strings.Join(names, ", "))
}

func main() {
	cfgPath := os.Getenv("GATEWAY_CONFIG")
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".config", "mcp", "mcp.json")
	}
	defs, err := loadConfig(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gateway-mcp:", err)
		os.Exit(1)
	}
	for _, d := range defs {
		children = append(children, &child{def: d})
	}

	tools := []mcp.Tool{
		{
			Name:        "servers",
			Description: "List connected MCP servers and whether they are reachable. Call first to orient.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				type st struct {
					Name   string `json:"name"`
					Tools  int    `json:"tools,omitempty"`
					Status string `json:"status"`
				}
				var out []st
				for _, c := range children {
					tl, err := c.toolList(ctx)
					if err != nil {
						out = append(out, st{c.def.Name, 0, "error: " + err.Error()})
					} else {
						out = append(out, st{c.def.Name, len(tl), "ok"})
					}
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:          "tools",
			Description:   "Compact index of every tool across all servers: <server>.<name> - one line description. Schemas on demand via tool_schema.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"server": "meshchatx"}`)}},
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"server": map[string]any{"type": "string", "description": "only this server, optional"},
			}},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Server string `json:"server"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				var b strings.Builder
				for _, c := range children {
					if a.Server != "" && c.def.Name != a.Server {
						continue
					}
					tl, err := c.toolList(ctx)
					if err != nil {
						fmt.Fprintf(&b, "%s: error: %v\n", c.def.Name, err)
						continue
					}
					for _, t := range tl {
						desc := t.Description
						if i := strings.Index(desc, ". "); i > 0 && i < 90 {
							desc = desc[:i]
						}
						if len(desc) > 90 {
							desc = desc[:90] + "..."
						}
						fmt.Fprintf(&b, "%s.%s - %s\n", c.def.Name, t.Name, desc)
					}
				}
				if b.Len() == 0 {
					return "no tools", nil
				}
				return b.String(), nil
			},
		},
		{
			Name:        "gateway_stats",
			Description: "Per-server call counts, errors, respawns, and average latency for this gateway session.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				type st struct {
					Server    string `json:"server"`
					Calls     int    `json:"calls"`
					Errors    int    `json:"errors"`
					Spawns    int    `json:"spawns"`
					AvgMs     int64  `json:"avg_ms"`
					LastCall  string `json:"last_call,omitempty"`
					ToolCount int    `json:"tools,omitempty"`
				}
				var out []st
				for _, c := range children {
					s := st{Server: c.def.Name}
					c.mu.Lock()
					s.Calls, s.Errors, s.Spawns = c.calls, c.errs, c.spawns
					if c.calls > 0 {
						s.AvgMs = c.latency.Milliseconds() / int64(c.calls)
					}
					if !c.lastCall.IsZero() {
						s.LastCall = c.lastCall.Format("15:04:05")
					}
					s.ToolCount = len(c.tools)
					c.mu.Unlock()
					out = append(out, s)
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:          "tool_schema",
			Description:   "Full JSON schema and description for one tool, e.g. reticulum.list_topics. Fetch before first invoke if unsure of arguments.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"name": "context.file_outline"}`)}},
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"name": map[string]any{"type": "string", "description": "<server>.<tool>"},
			}, "required": []string{"name"}},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				c, tool, err := findTool(a.Name)
				if err != nil {
					return "", err
				}
				tl, err := c.toolList(ctx)
				if err != nil {
					return "", err
				}
				for _, t := range tl {
					if t.Name == tool {
						b, _ := json.MarshalIndent(map[string]any{
							"name":        c.def.Name + "." + t.Name,
							"description": t.Description,
							"inputSchema": t.InputSchema,
						}, "", "  ")
						return string(b), nil
					}
				}
				var names []string
				for _, t := range tl {
					names = append(names, t.Name)
				}
				return "", fmt.Errorf("no tool %q on %s; available: %s", tool, c.def.Name, strings.Join(names, ", "))
			},
		},
		{
			Name:          "invoke",
			Description:   "Call a tool on any connected server. name is <server>.<tool>, arguments is its schema-shaped object.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"name": "rns.list_topics", "arguments": {}}`)}},
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"name":      map[string]any{"type": "string", "description": "<server>.<tool>"},
				"arguments": map[string]any{"type": "object", "description": "tool arguments"},
			}, "required": []string{"name"}},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				c, tool, err := findTool(a.Name)
				if err != nil {
					return "", err
				}
				if len(a.Arguments) == 0 {
					a.Arguments = json.RawMessage("{}")
				}
				res, err := c.call(ctx, "tools/call", map[string]any{
					"name": tool, "arguments": a.Arguments,
				})
				if err != nil {
					return "", err
				}
				var r struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				}
				if err := json.Unmarshal(res, &r); err != nil {
					return string(res), nil // nonstandard child response: pass through
				}
				var b strings.Builder
				for _, ct := range r.Content {
					b.WriteString(ct.Text)
					b.WriteByte('\n')
				}
				out := strings.TrimRight(b.String(), "\n")
				if r.IsError {
					return "", fmt.Errorf("%s", out)
				}
				return out, nil
			},
		},
	}

	srv := mcp.NewServer("gateway-mcp", "0.1.0", tools, nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "gateway-mcp:", err)
		os.Exit(1)
	}
}
