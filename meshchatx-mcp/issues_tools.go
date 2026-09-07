// SPDX-License-Identifier: 0BSD
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Quad4-Software/ai/meshchatx-mcp/internal/issues"
	"github.com/Quad4-Software/ai/meshchatx-mcp/internal/mcp"
)

// issueTools exposes GitHub issue workflows for the MeshChatX tracker:
// template-aware creation plus view, edit, close/reopen, comment and
// duplicate search. Write tools need a token in the environment
// (MESHCHATX_GITHUB_TOKEN, GITHUB_TOKEN, or GH_TOKEN). The target repo
// defaults to Quad4-Software/MeshChatX, override with
// MESHCHATX_ISSUES_REPO.
func issueTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "issue_templates",
			Description: "List the MeshChatX issue templates (bug, feature) with every field id, label, required flag, and hint. Call this before issue_create so the body matches the issue forms.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, err := json.MarshalIndent(issues.Templates, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "issue_create",
			Write:         true,
			Description:   "File a MeshChatX issue from a template. The body is rendered in the same section style as the issue forms (### headings, human prose). Known spec and ecosystem terms (BCP 47, LXMF, RNode, ...) are auto-linked to canonical references, see issue_references. Write op: needs a GitHub token in the environment.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"kind": "feature", "title": "Language picker should use native names", "fields": {"os": "All platforms", "problem": "...", "proposal": "..."}}`)}},
			InputSchema: obj(map[string]any{
				"kind":   map[string]any{"type": "string", "enum": issues.Kinds(), "description": "issue template"},
				"title":  strArg("short title, the [Bug]: or [Feature]: prefix is added for you"),
				"fields": map[string]any{"type": "object", "description": "field id to text, see issue_templates", "additionalProperties": map[string]any{"type": "string"}},
				"labels": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "extra labels on top of the template labels"},
			}, "kind", "title", "fields"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Kind   string            `json:"kind"`
					Title  string            `json:"title"`
					Fields map[string]string `json:"fields"`
					Labels []string          `json:"labels"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Kind == "" || a.Title == "" {
					return "", fmt.Errorf("missing required arguments: kind, title, fields")
				}
				body, err := issues.BuildBody(a.Kind, a.Fields)
				if err != nil {
					return "", err
				}
				body = issues.LinkReferences(body)
				title, err := issues.Title(a.Kind, a.Title)
				if err != nil {
					return "", err
				}
				for _, l := range a.Labels {
					if !regexp.MustCompile(`^[\w .:/-]{1,50}$`).MatchString(l) {
						return "", fmt.Errorf("invalid label %q", l)
					}
				}
				cli, err := issues.NewClientFromEnv()
				if err != nil {
					return "", err
				}
				labels := append([]string{}, issues.Templates[a.Kind].Labels...)
				labels = append(labels, a.Labels...)
				it, err := cli.Create(ctx, title, body, labels)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("created #%d %s\n%s", it.Number, it.Title, it.HTMLURL), nil
			},
		},
		{
			Name:        "issue_view",
			Description: "Fetch one MeshChatX issue by number: title, state, labels, author, body.",
			InputSchema: obj(map[string]any{"number": map[string]any{"type": "integer", "description": "issue number"}}, "number"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Number int `json:"number"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Number <= 0 {
					return "", fmt.Errorf("missing required argument: number")
				}
				cli, err := issues.NewClientFromEnv()
				if err != nil {
					return "", err
				}
				it, err := cli.Get(ctx, a.Number)
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(it, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "issue_update",
			Write:         true,
			Description:   "Edit or close a MeshChatX issue. Pass only what changes: title, body, labels (replaces the set), or state open/closed with optional state_reason completed/not_planned. Write op: needs a GitHub token in the environment.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"number": 88, "state": "closed", "state_reason": "completed"}`)}},
			InputSchema: obj(map[string]any{
				"number":       map[string]any{"type": "integer", "description": "issue number"},
				"title":        strArg("new title"),
				"body":         strArg("new body markdown"),
				"labels":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "full replacement label set"},
				"state":        map[string]any{"type": "string", "enum": []string{"open", "closed"}, "description": "closed closes, open reopens"},
				"state_reason": map[string]any{"type": "string", "enum": []string{"completed", "not_planned", "reopened"}},
			}, "number"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Number      int      `json:"number"`
					Title       string   `json:"title"`
					Body        string   `json:"body"`
					Labels      []string `json:"labels"`
					State       string   `json:"state"`
					StateReason string   `json:"state_reason"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Number <= 0 {
					return "", fmt.Errorf("missing required argument: number")
				}
				var p issues.Patch
				if a.Title != "" {
					t := issues.CleanProse(a.Title)
					p.Title = &t
				}
				if a.Body != "" {
					b := issues.LinkReferences(a.Body)
					p.Body = &b
				}
				if a.Labels != nil {
					p.Labels = &a.Labels
				}
				if a.State != "" {
					p.State = &a.State
				}
				if a.StateReason != "" {
					p.StateReason = &a.StateReason
				}
				cli, err := issues.NewClientFromEnv()
				if err != nil {
					return "", err
				}
				it, err := cli.Update(ctx, a.Number, p)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("updated #%d (%s)\n%s", it.Number, it.State, it.HTMLURL), nil
			},
		},
		{
			Name:        "issue_comment",
			Write:       true,
			Description: "Add a comment to a MeshChatX issue. Write op: needs a GitHub token in the environment.",
			InputSchema: obj(map[string]any{
				"number": map[string]any{"type": "integer", "description": "issue number"},
				"body":   strArg("comment markdown"),
			}, "number", "body"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Number int    `json:"number"`
					Body   string `json:"body"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Number <= 0 || strings.TrimSpace(a.Body) == "" {
					return "", fmt.Errorf("missing required arguments: number, body")
				}
				cli, err := issues.NewClientFromEnv()
				if err != nil {
					return "", err
				}
				u, err := cli.Comment(ctx, a.Number, issues.LinkReferences(a.Body))
				if err != nil {
					return "", err
				}
				return "commented\n" + u, nil
			},
		},
		{
			Name:        "issue_search",
			Description: "Search existing MeshChatX issues to avoid duplicates before filing. Returns number, title, state, URL.",
			InputSchema: obj(map[string]any{
				"query": strArg("search terms"),
				"state": map[string]any{"type": "string", "enum": []string{"open", "closed"}, "description": "omit for any state"},
				"limit": map[string]any{"type": "integer", "description": "max results, default 10, max 50"},
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					State string `json:"state"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				cli, err := issues.NewClientFromEnv()
				if err != nil {
					return "", err
				}
				hits, err := cli.Search(ctx, a.Query, a.State, a.Limit)
				if err != nil {
					return "", err
				}
				if len(hits) == 0 {
					return "no matching issues", nil
				}
				b, err := json.MarshalIndent(hits, "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "issue_references",
			Description: "List the terms issue_create auto-links in issue bodies (BCP 47, LXMF, RNode, ...) with their canonical URLs. Link these manually when writing issue_comment or custom bodies.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, err := json.MarshalIndent(issues.References, "", "  ")
				return string(b), err
			},
		},
	}
}
