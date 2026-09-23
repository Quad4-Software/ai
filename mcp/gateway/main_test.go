package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"mcpServers": {
		"a": {"command": "/bin/a", "env": {"X": "1"}},
		"b": {"command": "/bin/b", "disabled": true},
		"c": {"url": "https://x/mcp"},
		"gateway": {"command": "/bin/gateway"},
		"d": {"command": "/bin/d", "args": ["--flag"]}
	}}`
	p := filepath.Join(dir, "mcp_config.json")
	os.WriteFile(p, []byte(cfg), 0o644)
	defs, err := loadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected a+d, got %v", defs)
	}
	if defs[0].Name != "a" || defs[0].Env["X"] != "1" || defs[1].Name != "d" || defs[1].Args[0] != "--flag" {
		t.Fatalf("defs: %+v", defs)
	}
}

func TestHealthCheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	if healthCheck(ok.URL) != 0 {
		t.Fatal("expected 0 for healthy endpoint")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()
	if healthCheck(bad.URL) == 0 {
		t.Fatal("expected nonzero for 503 endpoint")
	}
	ok.Close()
	if healthCheck(ok.URL) == 0 {
		t.Fatal("expected nonzero for unreachable endpoint")
	}
}

func TestSessionStoreCap(t *testing.T) {
	st := newSessionStore(2)
	a := &httpSession{id: "a"}
	b := &httpSession{id: "b"}
	c := &httpSession{id: "c"}
	if !st.add(a) || !st.add(b) {
		t.Fatal("first two sessions should fit")
	}
	if st.add(c) {
		t.Fatal("third session should be rejected at cap")
	}
	if st.get("a") != a || st.get("nope") != nil {
		t.Fatal("get mismatch")
	}
	if st.count() != 2 {
		t.Fatalf("count=%d", st.count())
	}
	st.remove("a")
	if !st.add(c) {
		t.Fatal("slot should free after remove")
	}
}

func TestFindToolNamespacing(t *testing.T) {
	children := []*child{{def: serverDef{Name: "s1"}}}
	_, tool, err := findTool(children, "s1.do_thing")
	if err != nil || tool != "do_thing" {
		t.Fatalf("%v %v", tool, err)
	}
	if _, _, err := findTool(children, "no-dot"); err == nil {
		t.Fatal("should reject unnamespaced name")
	}
	if _, _, err := findTool(children, "other.x"); err == nil {
		t.Fatal("should reject unknown server")
	}
}
