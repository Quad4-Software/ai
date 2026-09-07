// SPDX-License-Identifier: 0BSD
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func serveTools(t *testing.T, input string, tools []Tool) []map[string]any {
	t.Helper()
	srv := NewServer("t", "0.1", tools, nil)
	var out strings.Builder
	if err := srv.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	for l := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("bad output line %q: %v", l, err)
		}
		res = append(res, m)
	}
	return res
}

func serve(t *testing.T, input string) []map[string]any {
	t.Helper()
	srv := NewServer("t", "0.1", []Tool{
		{Name: "ok", Description: "d", InputSchema: map[string]any{},
			Handle: func(context.Context, json.RawMessage) (string, error) { return "fine", nil }},
		{Name: "panic", Description: "d", InputSchema: map[string]any{},
			Handle: func(context.Context, json.RawMessage) (string, error) { panic("boom") }},
	}, nil)
	var out strings.Builder
	if err := srv.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	for l := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("bad output line %q: %v", l, err)
		}
		res = append(res, m)
	}
	return res
}

func TestMalformedLine(t *testing.T) {
	res := serve(t, "not json\n")
	if res[0]["error"] == nil {
		t.Fatalf("expected parse error: %v", res)
	}
}

func TestPanickingTool(t *testing.T) {
	res := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"panic","arguments":{}}}`+"\n")
	r := res[0]["result"].(map[string]any)
	if r["isError"] != true {
		t.Fatalf("panic should be isError: %v", r)
	}
	// server must still answer afterwards
	res2 := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"panic","arguments":{}}}`+"\n"+
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ok","arguments":{}}}`+"\n")
	r2 := res2[1]["result"].(map[string]any)
	if r2["isError"] == true {
		t.Fatalf("server should survive panic: %v", r2)
	}
}

func TestUnknownToolListsAvailable(t *testing.T) {
	res := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope","arguments":{}}}`+"\n")
	msg := res[0]["error"].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "ok") || !strings.Contains(msg, "panic") {
		t.Fatalf("should list tools: %s", msg)
	}
}

func TestInitializeInstructions(t *testing.T) {
	res := serve(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	r := res[0]["result"].(map[string]any)
	if r["instructions"] == "" {
		t.Fatalf("initialize should include instructions: %v", r)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	res := serve(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	if len(res) != 0 {
		t.Fatalf("notification must not get a response: %v", res)
	}
}

func TestProtocolNegotiation(t *testing.T) {
	tools := []Tool{{Name: "x", Description: "x",
		Handle: func(context.Context, json.RawMessage) (string, error) { return "ok", nil }}}
	res := serveTools(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n", tools)
	if res[0]["result"].(map[string]any)["protocolVersion"] != "2025-06-18" {
		t.Fatal("should echo known client version")
	}
	res = serveTools(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2999-01-01"}}`+"\n", tools)
	if res[0]["result"].(map[string]any)["protocolVersion"] != ProtocolVersion {
		t.Fatal("unknown version should get ours")
	}
}

func TestTasksLifecycle(t *testing.T) {
	srv := NewServer("t", "0", []Tool{{Name: "slow", Description: "x", Handle: func(context.Context, json.RawMessage) (string, error) {
		time.Sleep(20 * time.Millisecond)
		return "done", nil
	}}}, nil)
	var out strings.Builder
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"slow","arguments":{},"task":{}}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tasks/list","params":{}}` + "\n"
	srv.Serve(context.Background(), strings.NewReader(input), &out)
	var res []map[string]any
	for l := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		json.Unmarshal([]byte(l), &m)
		res = append(res, m)
	}
	taskID := res[0]["result"].(map[string]any)["task"].(map[string]any)["taskId"].(string)
	if taskID == "" {
		t.Fatal("no taskId returned")
	}
	time.Sleep(80 * time.Millisecond)
	var out2 strings.Builder
	srv.Serve(context.Background(), strings.NewReader(
		`{"jsonrpc":"2.0","id":3,"method":"tasks/result","params":{"taskId":"`+taskID+`"}}`+"\n"), &out2)
	var res2 map[string]any
	json.Unmarshal([]byte(strings.TrimSpace(out2.String())), &res2)
	got := res2["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"]
	if got != "done" {
		t.Fatalf("task result: %v", got)
	}
}

func TestTaskCancel(t *testing.T) {
	tools := []Tool{{Name: "slow", Description: "x", Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			return "done", nil
		}
	}}}
	srv := NewServer("t", "0", tools, nil)
	var out strings.Builder
	srv.Serve(context.Background(), strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"slow","arguments":{},"task":{}}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"tasks/cancel","params":{"taskId":"task-1"}}`+"\n"), &out)
	if !strings.Contains(out.String(), `"cancelled"`) {
		t.Fatalf("cancel failed: %s", out.String())
	}
}

func TestToolsListDeterministic(t *testing.T) {
	res := serveTools(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`+"\n", []Tool{
		{Name: "zeta", Description: "z", Handle: func(context.Context, json.RawMessage) (string, error) { return "", nil }},
		{Name: "alpha", Description: "a", Handle: func(context.Context, json.RawMessage) (string, error) { return "", nil }},
	})
	tools := res[0]["result"].(map[string]any)["tools"].([]any)
	if tools[0].(map[string]any)["name"] != "alpha" {
		t.Fatal("tools/list must be sorted for cacheability")
	}
}

func TestOversizedToolOutputRejected(t *testing.T) {
	srv := NewServer("t", "0", []Tool{{Name: "big", Description: "x", Handle: func(context.Context, json.RawMessage) (string, error) {
		return strings.Repeat("x", DefaultMaxToolOutputBytes+1), nil
	}}}, nil)
	res := serveTools2(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"big","arguments":{}}}`+"\n")
	r := res[0]["result"].(map[string]any)
	if r["isError"] != true {
		t.Fatalf("oversized output should be isError: %v", r)
	}
	if !strings.Contains(r["content"].([]any)[0].(map[string]any)["text"].(string), "exceeded") {
		t.Fatalf("expected output-too-large error: %v", r)
	}
}

func TestToolCallTimeout(t *testing.T) {
	tools := []Tool{
		{Name: "hang", Description: "x", Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
			// ignores ctx on purpose: the server must bound it anyway
			time.Sleep(5 * time.Second)
			return "never", nil
		}},
		{Name: "ok", Description: "x", Handle: func(context.Context, json.RawMessage) (string, error) {
			return "fine", nil
		}},
	}
	srv := NewServer("t", "0", tools, nil)
	srv.ToolCallTimeout = 50 * time.Millisecond
	var out strings.Builder
	start := time.Now()
	srv.Serve(context.Background(), strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hang","arguments":{}}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ok","arguments":{}}}`+"\n"), &out)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("hanging tool stalled the server for %s", elapsed)
	}
	if !strings.Contains(out.String(), "timed out") {
		t.Fatalf("expected timeout error: %s", out.String())
	}
	if !strings.Contains(out.String(), "fine") {
		t.Fatalf("server should answer the next call: %s", out.String())
	}
}

func TestTimeoutContextPropagates(t *testing.T) {
	sawDeadline := make(chan bool, 1)
	tools := []Tool{{Name: "ctx", Description: "x", Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
		_, ok := ctx.Deadline()
		sawDeadline <- ok
		return "ok", nil
	}}}
	srv := NewServer("t", "0", tools, nil)
	serveTools2(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ctx","arguments":{}}}`+"\n")
	if !<-sawDeadline {
		t.Fatal("handler context should carry the call deadline")
	}
}

func serveTools2(t *testing.T, srv *Server, input string) []map[string]any {
	t.Helper()
	var out strings.Builder
	if err := srv.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	for l := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("bad output line %q: %v", l, err)
		}
		res = append(res, m)
	}
	return res
}

func TestConcurrentServeCalls(t *testing.T) {
	// one Server, many concurrent Serve calls on independent streams:
	// outputs must not race or interleave (run under -race)
	srv := NewServer("t", "0", []Tool{
		{Name: "echo", Description: "x", Handle: func(_ context.Context, a json.RawMessage) (string, error) {
			return string(a), nil
		}},
	}, nil)
	var wg sync.WaitGroup
	errs := make(chan string, 16)
	for g := range 16 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			var in strings.Builder
			for i := range 10 {
				fmt.Fprintf(&in, `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"echo","arguments":{"g":%d}}}`+"\n", i, g)
			}
			var out strings.Builder
			if err := srv.Serve(context.Background(), strings.NewReader(in.String()), &out); err != nil {
				errs <- err.Error()
				return
			}
			for l := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
				var m map[string]any
				if err := json.Unmarshal([]byte(l), &m); err != nil {
					errs <- fmt.Sprintf("bad line %q: %v", l, err)
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func TestOversizedInputLine(t *testing.T) {
	// a single JSON-RPC line above the 8 MiB scanner cap ends the stream
	// with an error rather than a crash
	srv := NewServer("t", "0", nil, nil)
	var out strings.Builder
	big := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"` + strings.Repeat("x", 9<<20) + `"}}` + "\n"
	err := srv.Serve(context.Background(), strings.NewReader(big), &out)
	if err == nil {
		t.Fatal("expected scanner error for >8 MiB line")
	}
}

func TestNoGoroutineLeakAcrossServes(t *testing.T) {
	srv := NewServer("t", "0", []Tool{
		{Name: "echo", Description: "x", Handle: func(_ context.Context, a json.RawMessage) (string, error) {
			return string(a), nil
		}},
	}, nil)
	runtime.GC()
	base := runtime.NumGoroutine()
	for range 500 {
		var out strings.Builder
		if err := srv.Serve(context.Background(), strings.NewReader(
			`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"x":1}}}`+"\n"), &out); err != nil {
			t.Fatal(err)
		}
	}
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > base+2 {
		t.Fatalf("possible goroutine leak: base=%d after=%d", base, after)
	}
}

func TestNoGoroutineLeakWithTimeout(t *testing.T) {
	srv := NewServer("t", "0", []Tool{
		{Name: "fast", Description: "x", Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Millisecond):
				return "ok", nil
			}
		}},
	}, nil)
	srv.ToolCallTimeout = 100 * time.Millisecond
	runtime.GC()
	base := runtime.NumGoroutine()
	for range 200 {
		var out strings.Builder
		_ = srv.Serve(context.Background(), strings.NewReader(
			`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fast","arguments":{}}}`+"\n"), &out)
	}
	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > base+2 {
		t.Fatalf("possible goroutine leak after repeated timeouts: base=%d after=%d", base, after)
	}
}

func TestServeListenerShared(t *testing.T) {
	srv := NewServer("t", "0.1", []Tool{
		{Name: "ok", Description: "d", InputSchema: map[string]any{},
			Handle: func(context.Context, json.RawMessage) (string, error) { return "fine", nil }},
	}, nil)
	l, err := net.Listen("unix", filepath.Join(t.TempDir(), "s.sock"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.ServeListener(ctx, l) // #nosec -- test
	roundTrip := func() error {
		c, err := net.Dial("unix", l.Addr().String())
		if err != nil {
			return err
		}
		defer c.Close()
		fmt.Fprintln(c, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ok","arguments":{}}}`)
		var res map[string]any
		if err := json.NewDecoder(c).Decode(&res); err != nil {
			return err
		}
		result, _ := res["result"].(map[string]any)
		content, _ := result["content"].([]any)
		first, _ := content[0].(map[string]any)
		if first["text"] != "fine" {
			return fmt.Errorf("bad result: %v", res)
		}
		return nil
	}
	// many concurrent clients share one server without racing
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Go(func() {
			for range 3 {
				if err := roundTrip(); err != nil {
					errs <- err
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	cancel()
}
