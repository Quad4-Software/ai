// SPDX-License-Identifier: 0BSD
// Command no-slop is a stdio MCP server that lints prose against
// the no-AI-slop rules and the MeshChatX style rules. Fully offline.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quad4-Software/ai/mcp/no-slop/internal/lint"
	"github.com/Quad4-Software/ai/mcp/no-slop/internal/mcp"
)

//go:embed rules/rules.md
var rulesText string

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:          "check_text",
			Description:   "Lint text for AI slop and style violations. Returns JSON findings with rule, line, match, and fix.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"text": "Simply use this robust solution!"}`)}},
			InputSchema: obj(map[string]any{
				"text": strArg("the text to check"),
				"kind": map[string]any{
					"type":        "string",
					"description": "prose (default), doc, or comment. doc and comment ban semicolons; comment also bans backticks",
					"enum":        []string{"prose", "doc", "comment"},
				},
			}, "text"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Text string `json:"text"`
					Kind string `json:"kind"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Text == "" {
					return "", fmt.Errorf("missing required argument: text")
				}
				kind := lint.Kind(a.Kind)
				if kind == "" {
					kind = lint.KindProse
				}
				findings := lint.Check(a.Text, kind)
				if len(findings) == 0 {
					return "clean: no violations found", nil
				}
				b, err := json.MarshalIndent(findings, "", "  ")
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%d findings:\n%s", len(findings), b), nil
			},
		},
		{
			Name:        "list_rules",
			Description: "Return the full no-AI-slop and MeshChatX style ruleset.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return rulesText, nil
			},
		},
		{
			Name:          "check_file",
			Description:   "Lint a file on disk. kind auto-detects from extension (.md/.txt -> doc, source files -> comment) unless overridden.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "docs/en/readme.md"}`)}},
			InputSchema: obj(map[string]any{
				"path": strArg("absolute file path to lint"),
				"kind": map[string]any{"type": "string", "enum": []string{"prose", "doc", "comment"}},
			}, "path"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
					Kind string `json:"kind"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" {
					return "", fmt.Errorf("missing required argument: path")
				}
				b, err := os.ReadFile(a.Path)
				if err != nil {
					return "", err
				}
				if len(b) > 1<<20 {
					return "", fmt.Errorf("file too large (>1 MiB)")
				}
				kind := lint.Kind(a.Kind)
				if kind == "" {
					kind = lint.KindForPath(a.Path)
				}
				findings := lint.Check(string(b), kind)
				if len(findings) == 0 {
					return fmt.Sprintf("clean: %s (kind=%s)", a.Path, kind), nil
				}
				out, err := json.MarshalIndent(findings, "", "  ")
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%s (kind=%s): %d findings:\n%s", a.Path, kind, len(findings), out), nil
			},
		},
		{
			Name:        "check_dir",
			Description: "Lint every file matching a glob under a directory (e.g. *.md, *.go). Returns per-file finding counts.",
			InputSchema: obj(map[string]any{
				"path": strArg("directory to scan"),
				"glob": strArg("file glob, e.g. *.md; default all text files"),
				"kind": map[string]any{"type": "string", "enum": []string{"prose", "doc", "comment"}},
			}, "path"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
					Glob string `json:"glob"`
					Kind string `json:"kind"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" {
					return "", fmt.Errorf("missing required argument: path")
				}
				kind := lint.Kind(a.Kind)
				var b strings.Builder
				files := 0
				err := filepath.WalkDir(a.Path, func(p string, d os.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return err
					}
					if a.Glob != "" {
						ok, _ := filepath.Match(a.Glob, d.Name())
						if !ok {
							return nil
						}
					}
					data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
					if err != nil || len(data) > 1<<20 {
						return nil
					}
					k := kind
					if k == "" {
						k = lint.KindForPath(p)
					}
					fs := lint.Check(string(data), k)
					if len(fs) > 0 {
						fmt.Fprintf(&b, "%s: %d findings\n", p, len(fs))
						for _, f := range fs[:min(len(fs), 5)] {
							fmt.Fprintf(&b, "  line %d [%s] %s\n", f.Line, f.Rule, f.Message)
						}
					}
					files++
					return nil
				})
				if err != nil {
					return "", err
				}
				if b.Len() == 0 {
					return fmt.Sprintf("clean: %d files checked", files), nil
				}
				return b.String(), nil
			},
		},
		{
			Name:          "check_commit",
			Description:   "Lint the added lines of a git commit or ref range (git show <ref> or git diff <ref>). Same rules as check_diff.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"ref": "HEAD"}`)}},
			InputSchema: obj(map[string]any{
				"ref":  strArg("commit sha, tag, or range; default HEAD"),
				"kind": strArg("override kind, else auto"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Ref  string `json:"ref"`
					Kind string `json:"kind"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Ref != "" && !regexp.MustCompile(`^[\w.^~:\/-]{1,80}$`).MatchString(a.Ref) {
					return "", fmt.Errorf("invalid ref")
				}
				dir := os.Getenv("MCP_REPO_ROOT")
				if dir == "" {
					dir, _ = os.Getwd()
				}
				gitArgs := []string{"-C", dir}
				if a.Ref == "" {
					gitArgs = append(gitArgs, "show", "--format=", "HEAD")
				} else if strings.Contains(a.Ref, "..") {
					gitArgs = append(gitArgs, "diff", a.Ref)
				} else {
					gitArgs = append(gitArgs, "show", "--format=", a.Ref)
				}
				cmd := exec.CommandContext(ctx, "git", gitArgs...) // #nosec G702 G204 -- validated argv construction
				out, err := cmd.Output()
				if err != nil {
					return "", fmt.Errorf("git: %w", err)
				}
				kind := lint.Kind(a.Kind)
				if kind == "" {
					kind = lint.KindProse
				}
				findings := lint.CheckDiff(string(out), kind)
				if len(findings) == 0 {
					return "clean: no findings in added lines", nil
				}
				b, _ := json.MarshalIndent(findings, "", "  ")
				return fmt.Sprintf("%d findings in added lines:\n%s", len(findings), b), nil
			},
		},
		{
			Name:        "check_diff",
			Description: "Lint added lines of a unified diff (git diff / patch). Findings report new-file line numbers.",
			InputSchema: obj(map[string]any{
				"diff": strArg("unified diff text"),
				"kind": map[string]any{"type": "string", "enum": []string{"prose", "doc", "comment"}},
			}, "diff"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Diff string `json:"diff"`
					Kind string `json:"kind"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Diff == "" {
					return "", fmt.Errorf("missing required argument: diff")
				}
				kind := lint.Kind(a.Kind)
				if kind == "" {
					kind = lint.KindProse
				}
				findings := lint.CheckDiff(a.Diff, kind)
				if len(findings) == 0 {
					return "clean: no violations in added lines", nil
				}
				out, err := json.MarshalIndent(findings, "", "  ")
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%d findings in added lines:\n%s", len(findings), out), nil
			},
		},
		{
			Name:        "fix_text",
			Description: "Lint text and return the findings plus rewrite instructions for each violation.",
			InputSchema: obj(map[string]any{
				"text": strArg("the text to check and fix"),
				"kind": map[string]any{"type": "string", "enum": []string{"prose", "doc", "comment"}},
			}, "text"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Text string `json:"text"`
					Kind string `json:"kind"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Text == "" {
					return "", fmt.Errorf("missing required argument: text")
				}
				kind := lint.Kind(a.Kind)
				if kind == "" {
					kind = lint.KindProse
				}
				findings := lint.Check(a.Text, kind)
				if len(findings) == 0 {
					return "clean: no violations found", nil
				}
				var b strings.Builder
				fmt.Fprintf(&b, "%d violations. Rewrite the text fixing each, keeping every real fact. Replace vague claims with specific checkable details only if they are already present or verifiable; never invent numbers.\n\n", len(findings))
				for _, f := range findings {
					fmt.Fprintf(&b, "line %d [%s] %q: %s\n", f.Line, f.Rule, f.Match, f.Message)
				}
				b.WriteString("\nOriginal text:\n" + a.Text)
				return b.String(), nil
			},
		},
	}
}

func prompts() []mcp.Prompt {
	return []mcp.Prompt{
		{
			Name:        "review_prose",
			Description: "Review a piece of prose against the no-AI-slop rules and rewrite violations.",
			Arguments:   []mcp.PromptArg{{Name: "text", Description: "the prose to review", Required: true}},
			Handle: func(args map[string]string) (string, error) {
				return rulesText + "\n\nApply every rule above to this text. List each violation with its rule number, then produce a clean rewrite that keeps all verifiable facts and invents none.\n\n" + args["text"], nil
			},
		},
	}
}

func main() {
	srv := mcp.NewServer("no-slop", "0.1.0", tools(), prompts())
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "no-slop:", err)
		os.Exit(1)
	}
}
