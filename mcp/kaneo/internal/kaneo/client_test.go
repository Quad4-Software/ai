package kaneo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewClientURLRules(t *testing.T) {
	if _, err := NewClient(Config{APIURL: "http://evil.example.com"}); err == nil {
		t.Fatal("plain http must be rejected off loopback")
	}
	for _, u := range []string{"http://localhost:3000/api", "http://127.0.0.1:3000/api"} {
		if _, err := NewClient(Config{APIURL: u}); err != nil {
			t.Fatalf("loopback http should pass: %v", err)
		}
	}
	if _, err := NewClient(Config{APIURL: "https://cloud.kaneo.app/api"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewClient(Config{APIURL: "notaurl"}); err == nil {
		t.Fatal("garbage URL must error")
	}
}

func TestLoadConfigEnvWins(t *testing.T) {
	t.Setenv("KANEO_API_KEY", "env-key")
	t.Setenv("KANEO_API_URL", "https://env.example.com/api")
	t.Setenv("KANEO_PROJECT_ID", "pe")
	t.Setenv("KANEO_WORKSPACE_ID", "we")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "env-key" || cfg.Source != "env" {
		t.Fatalf("env should win: %+v", cfg)
	}
	if cfg.APIURL != "https://env.example.com/api" || cfg.DefaultProject != "pe" {
		t.Fatalf("env fields not applied: %+v", cfg)
	}
}

// fakeKeyring stubs the secret backend so tests never touch the real
// OS keyring.
func fakeKeyring(t *testing.T) (stored map[string]string) {
	t.Helper()
	stored = map[string]string{}
	oldLook, oldST, oldPass := lookPath, runSecretTool, runPass
	lookPath = func(name string) (string, error) {
		if name == "secret-tool" {
			return "/fake/secret-tool", nil
		}
		return "", os.ErrNotExist
	}
	runSecretTool = func(args []string, stdin string) (string, error) {
		if args[0] == "store" {
			stored["key"] = stdin
			return "", nil
		}
		return stored["key"], nil
	}
	runPass = func(args []string, stdin string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	t.Cleanup(func() { lookPath, runSecretTool, runPass = oldLook, oldST, oldPass })
	return stored
}

func TestLoadConfigMigratesPlaintextKey(t *testing.T) {
	stored := fakeKeyring(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	for _, k := range []string{"KANEO_API_KEY", "KANEO_API_URL", "KANEO_PROJECT_ID", "KANEO_WORKSPACE_ID"} {
		t.Setenv(k, "")
	}
	p := filepath.Join(dir, "kaneo", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"apiKey":"k","apiUrl":"https://x/api","repos":{"a":"b"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "k" {
		t.Fatalf("key not loaded: %+v", cfg)
	}
	if stored["key"] != "k" {
		t.Fatalf("key not migrated to keyring: %v", stored)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "apiKey") || strings.Contains(string(data), `"k"`) {
		t.Fatalf("key still in config file: %s", data)
	}
	var rewritten map[string]any
	if err := json.Unmarshal(data, &rewritten); err != nil {
		t.Fatal(err)
	}
	if rewritten["keyBackend"] != BackendKeyring || rewritten["repos"] == nil {
		t.Fatalf("rewrite lost fields: %s", data)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
		t.Fatalf("config perms %o, want 600", st.Mode().Perm())
	}
}

func TestKeyringBeatsPlaintext(t *testing.T) {
	stored := fakeKeyring(t)
	stored["key"] = "ring-key"
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	for _, k := range []string{"KANEO_API_KEY", "KANEO_API_URL", "KANEO_PROJECT_ID", "KANEO_WORKSPACE_ID"} {
		t.Setenv(k, "")
	}
	p := filepath.Join(dir, "kaneo", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"apiKey":"old","apiUrl":"https://x/api"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "ring-key" || cfg.Source != BackendKeyring {
		t.Fatalf("keyring should win: %+v", cfg)
	}
}

func TestResolveProject(t *testing.T) {
	cfg := &Config{Projects: []ProjectRef{
		{Name: "Melovian", Slug: "melovian", WorkspaceID: "w1", ProjectID: "p1"},
	}}
	for ref, want := range map[string]string{
		"melovian": "p1", "Melovian": "p1", "p1": "p1", "unknown": "unknown",
	} {
		if got := cfg.ResolveProject(ref); got != want {
			t.Fatalf("ResolveProject(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestDoRedactsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL, APIKey: "super-secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	err = c.do(context.Background(), http.MethodGet, "/task/x", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func TestMoveAndDelete(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MoveTask(context.Background(), "t1", "done", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.MoveTask(context.Background(), "t1", "", "p2"); err != nil {
		t.Fatal(err)
	}
	if err := c.MoveTask(context.Background(), "t1", "", ""); err == nil {
		t.Fatal("statusless same-project move must error")
	}
	if err := c.MoveTask(context.Background(), "t1", "bogus", ""); err == nil {
		t.Fatal("bad status must error")
	}
	if err := c.DeleteTask(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	want := "PUT /task/status/t1,PUT /task/move/t1,DELETE /task/t1"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls %v, want %v", calls, want)
	}
}

func TestGetTaskTopLevelFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"t9","title":"x","status":"to-do"}`))
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	task, err := c.GetTask(context.Background(), "t9")
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != "t9" {
		t.Fatalf("got %+v", task)
	}
}
