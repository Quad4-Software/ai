// SPDX-License-Identifier: 0BSD
// Command context-mcp provides token-efficient code access for agents:
// outlines, windowed reads, symbol bodies, repo maps, ranked references.
// Every response reports an estimated token cost. Read-only, path-jailed.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quad4-Software/ai/context-mcp/internal/codemap"
	"github.com/Quad4-Software/ai/context-mcp/internal/mcp"
)

var root string

func findRoot() (string, error) {
	if r := os.Getenv("MCP_REPO_ROOT"); r != "" {
		return filepath.Abs(r)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", fmt.Errorf("no git root above cwd; set MCP_REPO_ROOT")
		}
		dir = up
	}
}

func jailed(rel string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("no repo root; set MCP_REPO_ROOT")
	}
	p := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if p != root && !strings.HasPrefix(p, root+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repo root")
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return "", fmt.Errorf("symlink escapes repo root")
	}
	return real, nil
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

func withTokens(body string) string {
	return fmt.Sprintf("%s\n\n[est. tokens: %d]", body, codemap.EstimateTokens(body))
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:          "file_outline",
			Description:   "Signature skeleton of a file: decls with line ranges, not bodies. Start here for big files instead of reading them.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/meshchat.py"}`)}},
			InputSchema:   obj(map[string]any{"path": strArg("repo-relative file")}, "path"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" {
					return "", fmt.Errorf("missing required argument: path")
				}
				p, err := jailed(a.Path)
				if err != nil {
					return "", err
				}
				syms, total, err := codemap.Outline(p)
				if err != nil {
					return "", err
				}
				var b strings.Builder
				fmt.Fprintf(&b, "%s: %d lines, %d symbols\n", a.Path, total, len(syms))
				for _, s := range syms {
					fmt.Fprintf(&b, "%5d-%5d %-6s %s\n", s.Line, s.End, s.Kind, s.Sig)
				}
				return withTokens(b.String()), nil
			},
		},
		{
			Name:          "read_symbol",
			Description:   "Return only the body of one named symbol (function/class). Cheapest way to read one thing from a large file.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/meshchat.py", "name": "hotswap_identity"}`)}},
			InputSchema: obj(map[string]any{
				"path":   strArg("repo-relative file"),
				"name":   strArg("symbol name from file_outline"),
				"number": intArg("which occurrence, 1-based, default 1"),
			}, "path", "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path   string `json:"path"`
					Name   string `json:"name"`
					Number int    `json:"number"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" || a.Name == "" {
					return "", fmt.Errorf("missing required arguments: path, name")
				}
				if a.Number <= 0 {
					a.Number = 1
				}
				p, err := jailed(a.Path)
				if err != nil {
					return "", err
				}
				syms, _, err := codemap.Outline(p)
				if err != nil {
					return "", err
				}
				var hits []codemap.Symbol
				for _, s := range syms {
					if s.Name == a.Name {
						hits = append(hits, s)
					}
				}
				if len(hits) == 0 {
					var names []string
					for _, s := range syms {
						names = append(names, s.Name)
					}
					return "", fmt.Errorf("symbol %q not found; available: %s", a.Name, strings.Join(names, ", "))
				}
				if a.Number > len(hits) {
					return "", fmt.Errorf("only %d occurrence(s) of %q", len(hits), a.Name)
				}
				s := hits[a.Number-1]
				body, err := codemap.ReadLines(p, s.Line, s.End)
				if err != nil {
					return "", err
				}
				return withTokens(fmt.Sprintf("%s lines %d-%d (%d lines):\n%s", a.Path, s.Line, s.End, s.Lines, body)), nil
			},
		},
		{
			Name:        "read_window",
			Description: "Read a line range from a file with line numbers. Use after file_outline instead of whole-file reads.",
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative file"),
				"start": intArg("first line, 1-based"),
				"end":   intArg("last line"),
			}, "path", "start", "end"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path  string `json:"path"`
					Start int    `json:"start"`
					End   int    `json:"end"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" {
					return "", fmt.Errorf("missing required arguments: path, start, end")
				}
				if a.End-a.Start > 2000 {
					return "", fmt.Errorf("window capped at 2000 lines; narrow it")
				}
				p, err := jailed(a.Path)
				if err != nil {
					return "", err
				}
				body, err := codemap.ReadLines(p, a.Start, a.End)
				if err != nil {
					return "", err
				}
				return withTokens(body), nil
			},
		},
		{
			Name:          "repo_map",
			Description:   "Compact skeleton of code under a dir: files ranked by cross-file reference count, top symbols each. Orients before any deep read.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/src/backend"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default root"),
				"limit": intArg("max files, default 30"),
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 200 {
					a.Limit = 30
				}
				out, err := codemap.RepoMap(root, a.Path, a.Limit)
				if err != nil {
					return "", err
				}
				return withTokens(out), nil
			},
		},
		{
			Name:          "find_references",
			Description:   "All uses of an identifier across the repo (word-boundary), for impact analysis before changing it.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"name": "LXMessage"}`)}},
			InputSchema: obj(map[string]any{
				"name":  strArg("identifier"),
				"path":  strArg("repo-relative dir, default root"),
				"limit": intArg("max refs, default 40"),
			}, "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name  string `json:"name"`
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				if a.Limit <= 0 || a.Limit > 300 {
					a.Limit = 40
				}
				refs, err := codemap.FindReferences(root, a.Path, a.Name, a.Limit)
				if err != nil {
					return "", err
				}
				var b strings.Builder
				for _, r := range refs {
					fmt.Fprintf(&b, "%s:%d: %s\n", r.File, r.Line, r.Text)
				}
				if b.Len() == 0 {
					return "no references", nil
				}
				return withTokens(b.String()), nil
			},
		},
		{
			Name:          "search_symbols",
			Description:   "Search symbol names (functions, classes, types) across the repo, ranked. Cached index: repeat calls only re-scan changed files. Cheaper than reading files to find where something lives.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"query": "identity"}`)}},
			InputSchema: obj(map[string]any{
				"query": strArg("name or partial name, e.g. hotswap_identity"),
				"kind":  strArg("optional: func, class, type, decl"),
				"path":  strArg("repo-relative dir, default root"),
				"limit": intArg("max results, default 20"),
			}, "query"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Kind  string `json:"kind"`
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				if a.Limit <= 0 || a.Limit > 100 {
					a.Limit = 20
				}
				hits, err := codemap.SearchSymbols(root, a.Path, a.Query, a.Kind, a.Limit)
				if err != nil {
					return "", err
				}
				if len(hits) == 0 {
					return "no matching symbols", nil
				}
				b, _ := json.MarshalIndent(hits, "", "  ")
				return withTokens(string(b)), nil
			},
		},
		{
			Name:          "diff_symbols",
			Description:   "Map a git diff to the symbols it touches: per file, each hunk resolves to enclosing declarations. Read this instead of the raw diff when you only need the blast surface.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"ref": "HEAD~3"}`)}},
			InputSchema: obj(map[string]any{
				"ref":    strArg("commit/range to diff against; empty = unstaged working tree"),
				"staged": map[string]any{"type": "boolean", "description": "diff --cached"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Ref    string `json:"ref"`
					Staged bool   `json:"staged"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				out, err := codemap.DiffSymbols(ctx, root, a.Ref, a.Staged)
				if err != nil {
					return "", err
				}
				return withTokens(out), nil
			},
		},
		{
			Name:          "read_callsite",
			Description:   "Which function contains this line: returns the enclosing symbol's signature and range. Cheaper than reading the file to orient.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/meshchat.py", "line": 2200}`)}},
			InputSchema: obj(map[string]any{
				"path": strArg("repo-relative file"),
				"line": intArg("1-based line number"),
			}, "path", "line"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
					Line int    `json:"line"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" || a.Line <= 0 {
					return "", fmt.Errorf("missing required arguments: path, line")
				}
				p, err := jailed(a.Path)
				if err != nil {
					return "", err
				}
				s, err := codemap.EnclosingSymbol(p, a.Line)
				if err != nil {
					return "", err
				}
				if s == nil {
					return fmt.Sprintf("%s:%d is outside any declaration (module level)", a.Path, a.Line), nil
				}
				return fmt.Sprintf("%s:%d is inside %s (%s, lines %d-%d, %d lines)\nsignature: %s",
					a.Path, a.Line, s.Name, s.Kind, s.Line, s.End, s.Lines, s.Sig), nil
			},
		},
		{
			Name:          "callers",
			Description:   "Heuristic callers of a function: files+enclosing symbol where `name(` appears outside its definition. For impact analysis.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"name": "teardown_identity"}`)}},
			InputSchema: obj(map[string]any{
				"name":  strArg("function name"),
				"path":  strArg("repo-relative dir, default root"),
				"limit": intArg("max callers, default 30"),
			}, "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name  string `json:"name"`
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				if a.Limit <= 0 || a.Limit > 200 {
					a.Limit = 30
				}
				hits, err := codemap.Callers(root, a.Path, a.Name, a.Limit)
				if err != nil {
					return "", err
				}
				if len(hits) == 0 {
					return "no callers found", nil
				}
				var b strings.Builder
				for _, h := range hits {
					fmt.Fprintf(&b, "%s:%d %s\n", h.File, h.Line, h.Sig)
				}
				return withTokens(b.String()), nil
			},
		},
		{
			Name:          "callees",
			Description:   "What a function calls: each `x(` inside its body, resolved to its definition via the symbol index when possible.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/meshchat.py", "name": "hotswap_identity"}`)}},
			InputSchema: obj(map[string]any{
				"path":   strArg("repo-relative file"),
				"name":   strArg("symbol name"),
				"number": intArg("occurrence, default 1"),
			}, "path", "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path   string `json:"path"`
					Name   string `json:"name"`
					Number int    `json:"number"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" || a.Name == "" {
					return "", fmt.Errorf("missing required arguments: path, name")
				}
				out, err := codemap.Callees(root, a.Path, a.Name, a.Number)
				if err != nil {
					return "", err
				}
				return withTokens(out), nil
			},
		},
		{
			Name:        "search_code",
			Description: "Regex search over code files returning ±N lines of context per hit. Bounded and line-numbered.",
			InputSchema: obj(map[string]any{
				"pattern": strArg("regex"),
				"path":    strArg("repo-relative dir, default root"),
				"context": intArg("context lines each side, default 3"),
				"limit":   intArg("max hits, default 20"),
			}, "pattern"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Pattern string `json:"pattern"`
					Path    string `json:"path"`
					Context int    `json:"context"`
					Limit   int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Pattern == "" {
					return "", fmt.Errorf("missing required argument: pattern")
				}
				if a.Context < 0 || a.Context > 20 {
					a.Context = 3
				}
				if a.Limit <= 0 || a.Limit > 100 {
					a.Limit = 20
				}
				out, err := codemap.Search(root, a.Path, a.Pattern, a.Context, a.Limit)
				if err != nil {
					return "", err
				}
				return withTokens(out), nil
			},
		},
	}
}

func main() {
	var err error
	root, err = findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "context-mcp:", err)
		os.Exit(1)
	}
	srv := mcp.NewServer("context-mcp", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "context-mcp:", err)
		os.Exit(1)
	}
}
