package kaneo

import (
	"context"
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
	if cfg.APIURL != "https://env.example.com/api" || cfg.ProjectID != "pe" {
		t.Fatalf("env fields not applied: %+v", cfg)
	}
}

func TestLoadConfigFilePerms(t *testing.T) {
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		t.Skip("perm bits are posix-only")
	}
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
	if err := os.WriteFile(p, []byte(`{"apiKey":"k","apiUrl":"https://x/api"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("world-readable key file must error, got %v", err)
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "k" || cfg.Source != "config" || cfg.APIURL != "https://x/api" {
		t.Fatalf("config load wrong: %+v", cfg)
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
