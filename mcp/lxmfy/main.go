// SPDX-License-Identifier: 0BSD
// Command lxmfy is a stdio MCP server for LXMFy: searchable docs, bot
// scaffolding, static diagnostics, and test guidance. Stdlib only.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Quad4-Software/ai/mcp/lxmfy/internal/docs"
	"github.com/Quad4-Software/ai/mcp/lxmfy/internal/mcp"
	"github.com/Quad4-Software/ai/mcp/lxmfy/internal/scaffold"
)

const (
	base                = "https://lxmfy.quad4.io/"
	defaultDocLimit     = 5
	maxDocLimit         = 20
	maxDiagnoseBotBytes = 64 << 10
)

var topics = []docs.Topic{
	{ID: "quick-start", Title: "Quick Start", URL: base + "quick-start.html", Summary: "Prerequisites, first bot via the CLI, advanced features, next steps"},
	{ID: "creating-bots", Title: "Creating Bots", URL: base + "creating-bots.html", Summary: "Basic structure, templates, cogs, external cogs, NLP, RRC, storage, permissions, signatures, delivery"},
	{ID: "api-reference", Title: "Core Components / API Reference", URL: base + "api-reference.html", Summary: "LXMFBot, storage, commands, events, help, permissions, middleware, attachments, scheduler, signatures, delivery, RRC, templates, CLI tools, error handling"},
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intArg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools(src *docs.Source) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_topics",
			Description: "List LXMFy documentation topics (id, title, URL, summary).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, err := json.MarshalIndent(src.List(), "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_topic",
			Description: "Fetch the full plain text of an LXMFy docs page by topic id.",
			InputSchema: obj(map[string]any{"id": strArg("topic id from list_topics")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				return src.Get(ctx, strings.TrimSpace(a.ID))
			},
		},
		{
			Name:        "list_sections",
			Description: "List the section headings of an LXMFy docs page.",
			InputSchema: obj(map[string]any{"id": strArg("topic id from list_topics")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				secs, err := src.Sections(ctx, strings.TrimSpace(a.ID))
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(secs, "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_section",
			Description: "Fetch one section of a docs page by anchor or heading name. Cheaper than get_topic on large pages.",
			InputSchema: obj(map[string]any{
				"id":      strArg("topic id from list_topics"),
				"section": strArg("section anchor or heading substring from list_sections"),
			}, "id", "section"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string `json:"id"`
					Section string `json:"section"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" || a.Section == "" {
					return "", fmt.Errorf("missing required arguments: id, section")
				}
				return src.GetSection(ctx, strings.TrimSpace(a.ID), a.Section)
			},
		},
		{
			Name:          "search_docs",
			Description:   "Full-text search across the LXMFy docs. Returns ranked excerpts with topic and section anchors.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"query": "cog"}`)}},
			InputSchema: obj(map[string]any{
				"query": strArg("search terms"),
				"limit": intArg(fmt.Sprintf("max results, default %d, max %d", defaultDocLimit, maxDocLimit)),
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				return src.Search(ctx, a.Query, a.Limit)
			},
		},
		{
			Name:        "fetch_page",
			Description: "Fetch any page on lxmfy.quad4.io as plain text (host allowlisted). Use for deep links not in the topic index.",
			InputSchema: obj(map[string]any{"url": strArg("full URL on lxmfy.quad4.io")}, "url"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.URL == "" {
					return "", fmt.Errorf("missing required argument: url")
				}
				return src.FetchURL(ctx, strings.TrimSpace(a.URL))
			},
		},
		{
			Name:        "list_templates",
			Description: "List built-in LXMFy bot templates: minimal, echo, note, reminder, rrc, cogtest.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return strings.Join(scaffold.Templates, "\n"), nil
			},
		},
		{
			Name:          "scaffold_bot",
			Description:   "Generate LXMFy bot file(s) from a template: minimal, echo, note, reminder, rrc, cogtest.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"template": "minimal", "name": "mybot"}`)}},
			InputSchema: obj(map[string]any{
				"template": map[string]any{"type": "string", "enum": scaffold.Templates, "description": "bot template"},
				"name":     strArg("lowercase bot name, e.g. mybot"),
			}, "template", "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Template string `json:"template"`
					Name     string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Template == "" || a.Name == "" {
					return "", fmt.Errorf("missing required arguments: template, name")
				}
				files, err := scaffold.GenerateBot(strings.ToLower(a.Template), strings.ToLower(a.Name))
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(files, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "scaffold_cog",
			Description:   "Generate a cogs/<name>.py extension for a Python LXMFy bot.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"name": "utility"}`)}},
			InputSchema: obj(map[string]any{
				"name": strArg("lowercase cog name, e.g. utility"),
			}, "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				files, err := scaffold.GenerateCog(strings.ToLower(a.Name))
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(files, "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "list_test_scenarios",
			Description: "List LXMFy reliability and stress test categories and how to run them.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return scaffold.Tests(), nil
			},
		},
		{
			Name:        "diagnose_bot",
			Description: "Statically analyze a bot Python file string for common mistakes: missing admins, landlock disabled, unsafe threading, missing run guard, etc.",
			InputSchema: obj(map[string]any{
				"code": strArg(fmt.Sprintf("bot source code, max %d bytes", maxDiagnoseBotBytes)),
			}, "code"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Code == "" {
					return "", fmt.Errorf("missing required argument: code")
				}
				if len(a.Code) > maxDiagnoseBotBytes {
					return "", fmt.Errorf("code capped at 64 KiB; got %d bytes", len(a.Code))
				}
				return scaffold.DiagnoseBot(a.Code)
			},
		},
	}
}

func main() {
	src := docs.NewSource("LXMFy docs", topics)
	srv := mcp.NewServer("lxmfy", "0.1.0", tools(src), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lxmfy:", err)
		os.Exit(1)
	}
}
