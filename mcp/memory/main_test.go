package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func setupMem(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MEMORY_DIR", dir)
	memoryDir = dir
	memoryFile = filepath.Join(dir, "memories.jsonl")
}

func TestRememberAndRecall(t *testing.T) {
	setupMem(t)
	h := toolByName("remember")
	if h == nil {
		t.Fatal("remember tool not found")
	}
	out, err := h(context.Background(), json.RawMessage(`{"content":"rns://06a54b/public/LXMFy","type":"destination","tags":["lxmfy","public"],"source":"user prompt"}`))
	if err != nil || !strings.Contains(out, "remembered destination as") {
		t.Fatalf("remember failed: %q %v", out, err)
	}

	recall := toolByName("recall")
	if recall == nil {
		t.Fatal("recall tool not found")
	}
	out, err = recall(context.Background(), json.RawMessage(`{"query":"LXMFy","limit":5}`))
	if err != nil || !strings.Contains(out, "rns://") {
		t.Fatalf("recall failed: %q %v", out, err)
	}
}

func TestRecallFiltersAndFuzzy(t *testing.T) {
	setupMem(t)
	remember := toolByName("remember")
	recall := toolByName("recall")
	for _, c := range []struct{ content, typ, tag string }{
		{"release tag must be v prefix", "note", "release"},
		{"run go test -race before push", "task", "workflow"},
		{"do not expose private keys", "not-do", "security"},
	} {
		if _, err := remember(context.Background(), json.RawMessage(`{"content":"`+c.content+`","type":"`+c.typ+`","tags":["`+c.tag+`"]}`)); err != nil {
			t.Fatalf("remember %s: %v", c.typ, err)
		}
	}
	out, err := recall(context.Background(), json.RawMessage(`{"query":"race before","tags":["workflow"],"limit":2}`))
	if err != nil || !strings.Contains(out, "go test -race") {
		t.Fatalf("filtered recall failed: %q %v", out, err)
	}
	if strings.Contains(out, "private keys") {
		t.Fatal("filter should exclude not-do security memory")
	}
}

func TestForgetAndStatus(t *testing.T) {
	setupMem(t)
	remember := toolByName("remember")
	out, err := remember(context.Background(), json.RawMessage(`{"content":"to delete","type":"note","tags":["temp"]}`))
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Fields(out)[len(strings.Fields(out))-1]

	forget := toolByName("forget")
	if _, err := forget(context.Background(), json.RawMessage(`{"ids":["`+id+`"]}`)); err != nil {
		t.Fatalf("forget failed: %v", err)
	}
	recall := toolByName("recall")
	out2, _ := recall(context.Background(), json.RawMessage(`{"query":"to delete"}`))
	if strings.Contains(out2, "to delete") {
		t.Fatal("deleted memory still recalled")
	}
	status := toolByName("memory_status")
	out, err = status(context.Background(), nil)
	if err != nil || !strings.Contains(out, "0 memory(s)") {
		t.Fatalf("status wrong: %q %v", out, err)
	}
}

func TestBounds(t *testing.T) {
	setupMem(t)
	h := toolByName("remember")
	_, err := h(context.Background(), json.RawMessage(`{"content":"`+strings.Repeat("x", maxContentBytes+1)+`"}`))
	if err == nil || !strings.Contains(err.Error(), "capped") {
		t.Fatalf("expected cap error, got %v", err)
	}
}

func toolByName(name string) func(context.Context, json.RawMessage) (string, error) {
	for _, tt := range tools() {
		if tt.Name == name {
			return tt.Handle
		}
	}
	return nil
}
