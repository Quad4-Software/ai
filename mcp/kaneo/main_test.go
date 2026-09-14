package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Quad4-Software/ai/mcp/kaneo/internal/kaneo"
)

// stub returns an in-process Kaneo API for tests. It records the
// Authorization header so tests can assert the key never leaks.
func stub(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *string) {
	t.Helper()
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &auth
}

func testClient(t *testing.T, srv *httptest.Server) {
	t.Helper()
	c, err := kaneo.NewClient(kaneo.Config{
		APIURL: srv.URL, APIKey: "secret-key",
		ProjectID: "p1", WorkspaceID: "w1",
	})
	if err != nil {
		t.Fatal(err)
	}
	client = c
	cfg = &kaneo.Config{APIURL: srv.URL, APIKey: "secret-key", Source: "env",
		ProjectID: "p1", WorkspaceID: "w1"}
}

func TestAuthStatusNeverExposesKey(t *testing.T) {
	srv, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {})
	testClient(t, srv)
	var h func(context.Context, json.RawMessage) (string, error)
	for _, tl := range tools() {
		if tl.Name == "auth_status" {
			h = tl.Handle
		}
	}
	out, err := h(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "secret-key") {
		t.Fatalf("key leaked in auth_status output: %s", out)
	}
	if !strings.Contains(out, `"configured": true`) {
		t.Fatalf("expected configured true: %s", out)
	}
}

func TestWriteToolsRequireKey(t *testing.T) {
	client = &kaneo.Client{}
	cfg = &kaneo.Config{}
	for _, tl := range tools() {
		if !tl.Write {
			continue
		}
		_, err := tl.Handle(context.Background(), json.RawMessage(`{"id":"x","title":"t","taskId":"x","label":"l","content":"c"}`))
		if err == nil || !strings.Contains(err.Error(), "no Kaneo API key") {
			t.Fatalf("%s should require key, got %v", tl.Name, err)
		}
	}
}

func TestCreateTask(t *testing.T) {
	srv, auth := stub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/task/p1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "hello" || body["status"] != "to-do" {
			t.Errorf("bad payload %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"t1","title":"hello","status":"to-do","priority":"high","number":9}`))
	})
	testClient(t, srv)
	var h func(context.Context, json.RawMessage) (string, error)
	for _, tl := range tools() {
		if tl.Name == "create_task" {
			h = tl.Handle
		}
	}
	out, err := h(context.Background(), json.RawMessage(`{"title":"hello","priority":"high"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"number": 9`) {
		t.Fatalf("unexpected output %s", out)
	}
	if *auth != "Bearer secret-key" {
		t.Fatalf("auth header %q", *auth)
	}
}

func TestCreateTaskValidation(t *testing.T) {
	testClient(t, httptest.NewServer(http.NotFoundHandler()))
	var h func(context.Context, json.RawMessage) (string, error)
	for _, tl := range tools() {
		if tl.Name == "create_task" {
			h = tl.Handle
		}
	}
	if _, err := h(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("missing title should error")
	}
	_, err := h(context.Background(), json.RawMessage(`{"title":"x","priority":"bogus"}`))
	if err == nil || !strings.Contains(err.Error(), "invalid priority") {
		t.Fatalf("bad priority should name valid values, got %v", err)
	}
}

func TestGetBoardCompact(t *testing.T) {
	srv, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"p1","name":"Melovian","slug":"MEL",
			"columns":[{"id":"to-do","slug":"to-do","name":"To Do",
			"tasks":[{"id":"t1","number":1,"title":"a","status":"to-do","priority":"high","description":"long body"}]}]}}`))
	})
	testClient(t, srv)
	var h func(context.Context, json.RawMessage) (string, error)
	for _, tl := range tools() {
		if tl.Name == "get_board" {
			h = tl.Handle
		}
	}
	out, err := h(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "long body") {
		t.Fatal("get_board must omit descriptions")
	}
	if !strings.Contains(out, `"title": "a"`) {
		t.Fatalf("missing task %s", out)
	}
}

func TestUpdateTaskGranular(t *testing.T) {
	var calls []string
	srv, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"t1","title":"x","status":"done"}`))
	})
	testClient(t, srv)
	var h func(context.Context, json.RawMessage) (string, error)
	for _, tl := range tools() {
		if tl.Name == "update_task" {
			h = tl.Handle
		}
	}
	if _, err := h(context.Background(), json.RawMessage(`{"id":"t1","status":"done","priority":"urgent"}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"PUT /task/priority/t1", "PUT /task/status/t1", "GET /task/t1"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls %v, want %v", calls, want)
	}
	if _, err := h(context.Background(), json.RawMessage(`{"id":"t1"}`)); err == nil {
		t.Fatal("empty patch should error")
	}
}
