package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestEcho(t *testing.T) {
	h := tools()[0].Handle
	out, err := h(context.Background(), json.RawMessage(`{"text":"hi"}`))
	if err != nil || out != "hi" {
		t.Fatalf("%q %v", out, err)
	}
	if _, err := h(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("missing arg should error")
	}
	if _, err := h(context.Background(), json.RawMessage(`{"text":"`+strings.Repeat("x", 5000)+`"}`)); err == nil {
		t.Fatal("oversize should error")
	}
}

func TestValidateName(t *testing.T) {
	h := tools()[1].Handle
	out, err := h(context.Background(), json.RawMessage(`{"name":"ok_name"}`))
	if err != nil || !strings.Contains(out, "valid") {
		t.Fatalf("%q %v", out, err)
	}
	if _, err := h(context.Background(), json.RawMessage(`{"name":"9bad"}`)); err == nil {
		t.Fatal("bad identifier should error with the valid shape")
	}
}
