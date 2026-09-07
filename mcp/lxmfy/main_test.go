// SPDX-License-Identifier: 0BSD
package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quad4-Software/ai/mcp/lxmfy/internal/docs"
	"github.com/Quad4-Software/ai/mcp/lxmfy/internal/mcp"
)

func TestListTopics(t *testing.T) {
	src := docs.NewSource("t", topics)
	for _, tool := range tools(src) {
		if tool.Name == "list_topics" {
			_, err := tool.Handle(context.Background(), json.RawMessage(`{}`))
			if err != nil {
				t.Fatalf("list_topics: %v", err)
			}
			return
		}
	}
	t.Fatal("list_topics not found")
}

func TestScaffoldBot(t *testing.T) {
	var bot mcp.Tool
	for _, tool := range tools(nil) {
		if tool.Name == "scaffold_bot" {
			bot = tool
			break
		}
	}
	if bot.Name == "" {
		t.Fatal("scaffold_bot not found")
	}
	_, err := bot.Handle(context.Background(), json.RawMessage(`{"template":"nope","name":"x"}`))
	if err == nil {
		t.Fatal("invalid template should error")
	}
	out, err := bot.Handle(context.Background(), json.RawMessage(`{"template":"minimal","name":"mybot"}`))
	if err != nil {
		t.Fatalf("minimal scaffold: %v", err)
	}
	if !strings.Contains(out, "mybot.py") {
		t.Fatalf("expected mybot.py in output: %s", out)
	}
}

func TestDiagnoseBot(t *testing.T) {
	for _, tool := range tools(nil) {
		if tool.Name == "diagnose_bot" {
			out, err := tool.Handle(context.Background(), json.RawMessage(`{"code":"from lxmfy import LXMFBot\nbot = LXMFBot(name='x')\nbot.run()\n"}`))
			if err != nil {
				t.Fatalf("diagnose: %v", err)
			}
			if !strings.Contains(out, "admins") {
				t.Fatalf("expected admin warning: %s", out)
			}
			return
		}
	}
	t.Fatal("diagnose_bot not found")
}
