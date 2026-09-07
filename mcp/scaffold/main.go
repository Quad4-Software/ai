// SPDX-License-Identifier: 0BSD
// Command your is a template for new stdio MCP servers in this
// toolkit. Copy this directory, rename, replace the example tools,
// keep the error-handling contract.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/Quad4-Software/ai/mcp/scaffold/internal/mcp"
)

// Conventions to keep:
//   - tools are read-only by default; mutators need an explicit policy
//   - validate every argument; errors name the missing arg and valid values
//   - bound every result (counts, bytes, lines); say when output is capped
//   - path arguments are jailed to an approved root, never joined raw
//   - external commands use argv form only, never a shell string
//   - no secrets, tokens, or identity material in output or errors

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// strArg, intArg, obj build input schemas compactly.
func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intArg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "echo",
			Description: "Echo a short string back. Template tool, replace it.",
			InputSchema: obj(map[string]any{
				"text": strArg("text to echo, max 4 KiB"),
			}, "text"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"text": "hello"}`)},
			},
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Text == "" {
					return "", fmt.Errorf("missing required argument: text")
				}
				if len(a.Text) > 4<<10 {
					return "", fmt.Errorf("text capped at 4 KiB; got %d bytes", len(a.Text))
				}
				return a.Text, nil
			},
		},
		{
			Name:        "validate_name",
			Description: "Check whether a string is a valid identifier shape.",
			InputSchema: obj(map[string]any{
				"name": strArg("identifier to validate"),
			}, "name"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"name": "my_symbol"}`)},
			},
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil {
					return "", fmt.Errorf("invalid arguments: %v", err)
				}
				if a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				if identRe.MatchString(a.Name) {
					return "valid identifier", nil
				}
				return "", fmt.Errorf("not an identifier: %q; want [A-Za-z_][A-Za-z0-9_]*", a.Name)
			},
		},
	}
}

func main() {
	srv := mcp.NewServer("your", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "your:", err)
		os.Exit(1)
	}
}
