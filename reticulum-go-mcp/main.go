// SPDX-License-Identifier: 0BSD
// Command reticulum-go-mcp is a stdio MCP server exposing the Reticulum-Go
// documentation as searchable, section-aware tools. Stdlib only, low memory.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Quad4-Software/ai/reticulum-go-mcp/internal/docs"
	"github.com/Quad4-Software/ai/reticulum-go-mcp/internal/mcp"
)

const base = "https://reticulum-go.quad4.io"

var topics = []docs.Topic{
	{ID: "overview", Title: "Overview", URL: base + "/docs/overview", Summary: "What Reticulum-Go is, design goals, feature status, and how it fits with Python Reticulum"},
	{ID: "getting-started", Title: "Getting started", URL: base + "/docs/getting-started", Summary: "Install, build, first daemon, and quick start"},
	{ID: "examples", Title: "Examples", URL: base + "/docs/examples", Summary: "Minimal usage examples and common patterns"},
	{ID: "architecture", Title: "Architecture", URL: base + "/docs/architecture", Summary: "High-level architecture and component overview"},
	{ID: "package-map", Title: "Package map", URL: base + "/docs/package-map", Summary: "Layout of pkg/ and cmd/ packages"},
	{ID: "api-reference", Title: "API reference", URL: base + "/docs/api-reference", Summary: "Go API reference and exported types"},
	{ID: "microvm", Title: "Firecracker microvm", URL: base + "/docs/microvm", Summary: "Running reticulum-go inside Firecracker microVMs"},
	{ID: "configuration", Title: "Configuration", URL: base + "/docs/configuration", Summary: "Daemon configuration, config files, and environment"},
	{ID: "interfaces", Title: "Interfaces", URL: base + "/docs/interfaces", Summary: "Supported network interfaces and configuration"},
	{ID: "transport", Title: "Transport", URL: base + "/docs/transport", Summary: "Transport layer, paths, announces, and relay"},
	{ID: "utilities", Title: "CLI utilities", URL: base + "/docs/utilities", Summary: "Command-line utilities bundled with reticulum-go"},
	{ID: "packet-debug", Title: "Packet debug", URL: base + "/docs/packet-debug", Summary: "Tools and techniques for debugging packets"},
	{ID: "identity-and-destinations", Title: "Identity and destinations", URL: base + "/docs/identity-and-destinations", Summary: "Identity creation, destination hashes, and aspects"},
	{ID: "links-channels-and-resources", Title: "Links, channels, and resources", URL: base + "/docs/links-channels-and-resources", Summary: "Links, channels, buffers, and resources"},
	{ID: "cryptography", Title: "Cryptography", URL: base + "/docs/cryptography", Summary: "Wire-compatible crypto primitives with Python RNS"},
	{ID: "embedding-and-wasm", Title: "Embedding and WebAssembly", URL: base + "/docs/embedding-and-wasm", Summary: "Embedding the library and WebAssembly builds"},
	{ID: "control-api", Title: "Control API", URL: base + "/docs/control-api", Summary: "Localhost control API for external applications"},
	{ID: "librns", Title: "librns", URL: base + "/docs/librns", Summary: "C-compatible librns bindings"},
	{ID: "compatibility", Title: "Compatibility", URL: base + "/docs/compatibility", Summary: "Parity matrix with Python Reticulum"},
	{ID: "security", Title: "Security", URL: base + "/docs/security", Summary: "Security model, sandboxing, and reporting"},
	{ID: "development-and-testing", Title: "Development and testing", URL: base + "/docs/development-and-testing", Summary: "Hacking on Reticulum-Go and test suite"},
	{ID: "interop-timeline", Title: "Interop timeline", URL: base + "/docs/interop-timeline", Summary: "Roadmap and interoperability timeline"},
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func intArg(desc string, def int) map[string]any {
	return map[string]any{"type": "integer", "description": desc, "default": def}
}

func tools(src *docs.Source) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_topics",
			Description: "List Reticulum-Go doc topics (id, title, URL, summary).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, err := json.MarshalIndent(src.List(), "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_topic",
			Description: "Fetch the full plain text of a Reticulum-Go doc page by topic id.",
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
			Description: "List the section headings of a Reticulum-Go doc page.",
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
				b, _ := json.MarshalIndent(secs, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "get_section",
			Description: "Fetch one section of a Reticulum-Go doc page by id and section anchor or heading name.",
			InputSchema: obj(map[string]any{
				"id":      strArg("topic id from list_topics"),
				"section": strArg("section anchor or heading name"),
			}, "id", "section"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string `json:"id"`
					Section string `json:"section"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" || a.Section == "" {
					return "", fmt.Errorf("missing required arguments: id, section")
				}
				return src.GetSection(ctx, strings.TrimSpace(a.ID), strings.TrimSpace(a.Section))
			},
		},
		{
			Name:        "search_docs",
			Description: "Search Reticulum-Go docs by regex across all topic pages.",
			InputSchema: obj(map[string]any{
				"query": strArg("regex or text"),
				"limit": intArg("max results, default 15", 15),
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				if a.Limit <= 0 || a.Limit > 50 {
					a.Limit = 15
				}
				return src.Search(ctx, a.Query, a.Limit)
			},
		},
		{
			Name:        "fetch_page",
			Description: "Fetch any Reticulum-Go docs page by URL and return plain text.",
			InputSchema: obj(map[string]any{"url": strArg("full URL on reticulum-go.quad4.io")}, "url"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.URL == "" {
					return "", fmt.Errorf("missing required argument: url")
				}
				u := strings.TrimSpace(a.URL)
				if !strings.HasPrefix(u, "https://reticulum-go.quad4.io/") && !strings.HasPrefix(u, "https://reticulum-go.quad4.io") {
					return "", fmt.Errorf("url must be on reticulum-go.quad4.io")
				}
				return src.FetchURL(ctx, u)
			},
		},
	}
}

func main() {
	src := docs.NewSource("Reticulum-Go Docs", topics)
	srv := mcp.NewServer("reticulum-go-mcp", "0.1.0", tools(src), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "reticulum-go-mcp:", err)
		os.Exit(1)
	}
}
