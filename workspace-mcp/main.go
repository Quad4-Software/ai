// SPDX-License-Identifier: 0BSD
// Command workspace-mcp wraps repo dev commands: Taskfile targets, git
// health, and test-file mapping. Commands are validated and bounded.
// Stdlib only.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Quad4-Software/ai/workspace-mcp/internal/mcp"
)

const (
	maxOutput = 256 << 10
	timeout   = 180 * time.Second
)

var (
	root   string
	taskRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,60}$`)
	numRe  = regexp.MustCompile(`^[0-9]{1,4}$`)
	refRe  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9./_@^-]{0,120}$`)
)

func findRoot() (string, error) {
	if r := os.Getenv("MCP_REPO_ROOT"); r != "" {
		return filepath.Abs(r)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && st.IsDir() {
			return dir, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", fmt.Errorf("no git root above cwd; set MCP_REPO_ROOT")
		}
		dir = up
	}
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- fixed argv, no shell, command allowlisted
	cmd.Dir = root
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := buf.Bytes()
	if len(out) > maxOutput {
		out = append(out[:maxOutput], []byte("\n[output truncated]")...)
	}
	s := string(bytes.TrimSpace(out))
	if err != nil && s == "" {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	if s == "" {
		s = "(no output)"
	}
	if err != nil {
		s += fmt.Sprintf("\n[exit: %v]", err)
	}
	return s, nil
}

// testCandidates maps a source file to probable test file paths.
var genericStems = map[string]bool{
	"manager": true, "index": true, "core": true, "main": true,
	"mod": true, "app": true, "utils": true, "helpers": true,
}

func testCandidates(rel string) []string {
	base := filepath.Base(rel)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	// generic stems like manager.py or index.ts are ambiguous; also try
	// the parent directory name
	parent := filepath.Base(filepath.Dir(rel))
	stems := []string{stem}
	if genericStems[stem] && parent != "" && parent != "." && parent != "src" {
		stems = append(stems, parent, parent+"_"+stem)
	}
	var out []string
	for _, s := range stems {
		switch strings.ToLower(filepath.Ext(rel)) {
		case ".py":
			out = append(out, "tests/backend/test_"+s+".py", "tests/backend/"+s+"_test.py")
		case ".ts", ".tsx", ".js", ".jsx", ".vue", ".svelte":
			out = append(out,
				"tests/frontend/"+s+".test.js",
				"tests/frontend/"+s+".test.ts",
				"tests/frontend/"+s+".spec.js")
		case ".go":
			out = append(out, filepath.Join(filepath.Dir(rel), s+"_test.go"))
		}
	}
	return out
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "task_list",
			Description: "List Taskfile targets (task --list.)",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				return run(ctx, "task", "--list")
			},
		},
		{
			Name:        "task_run",
			Write:       true,
			Description: "Run a Taskfile target by name (e.g. test, lint, fmt). Name is validated; output capped.",
			InputSchema: obj(map[string]any{"name": strArg("task name from task_list")}, "name"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				if !taskRe.MatchString(a.Name) {
					return "", fmt.Errorf("invalid task name %q", a.Name)
				}
				return run(ctx, "task", a.Name)
			},
		},
		{
			Name:        "git_status",
			Description: "git status --porcelain=v1 plus current branch.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				br, _ := run(ctx, "git", "branch", "--show-current")
				st, err := run(ctx, "git", "status", "--porcelain=v1")
				if err != nil {
					return "", err
				}
				return "branch: " + br + "\n" + st, nil
			},
		},
		{
			Name:        "git_log",
			Description: "Recent commit log (oneline).",
			InputSchema: obj(map[string]any{"n": strArg("count, default 15, digits only")}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					N string `json:"n"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.N == "" {
					a.N = "15"
				}
				if !numRe.MatchString(a.N) {
					return "", fmt.Errorf("invalid count %q", a.N)
				}
				return run(ctx, "git", "log", "--oneline", "-n", a.N)
			},
		},
		{
			Name:        "git_diff_stat",
			Description: "git diff --stat for working tree, or diffstat vs a ref.",
			InputSchema: obj(map[string]any{"ref": strArg("optional ref, e.g. main...HEAD")}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Ref string `json:"ref"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				gitArgs := []string{"diff", "--stat"}
				if a.Ref != "" {
					if !refRe.MatchString(a.Ref) {
						return "", fmt.Errorf("invalid ref %q", a.Ref)
					}
					gitArgs = append(gitArgs, a.Ref)
				}
				return run(ctx, "git", gitArgs...)
			},
		},
		{
			Name:          "run_tests",
			Write:         true,
			Description:   "Find the mapped test file for a source file and run it (pytest, vitest, or go test).",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/src/backend/plugin_permissions.py"}`)}},
			InputSchema:   obj(map[string]any{"file": strArg("repo-relative source path")}, "file"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					File string `json:"file"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.File == "" {
					return "", fmt.Errorf("missing required argument: file")
				}
				if strings.Contains(a.File, "..") || filepath.IsAbs(a.File) {
					return "", fmt.Errorf("path must be repo-relative")
				}
				var existing []string
				for _, c := range testCandidates(a.File) {
					if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err == nil && !st.IsDir() {
						existing = append(existing, c)
					}
				}
				if len(existing) == 0 {
					return "", fmt.Errorf("no test file found for %s; candidates: %s", a.File, strings.Join(testCandidates(a.File), ", "))
				}
				t := existing[0]
				_, out, err := runTestsFor(ctx, t)
				if err != nil {
					return "", err
				}
				return out, nil
			},
		},
		{
			Name:          "git_blame_context",
			Description:   "git blame summary for a line range: which commits/authors last touched it, line counts per commit. Answer 'who changed this and why' without reading history.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/meshchat.py", "start": 2150, "end": 2200}`)}},
			InputSchema: obj(map[string]any{
				"file":  strArg("repo-relative file"),
				"start": map[string]any{"type": "integer"},
				"end":   map[string]any{"type": "integer"},
			}, "file", "start", "end"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					File  string `json:"file"`
					Start int    `json:"start"`
					End   int    `json:"end"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.File == "" || a.Start <= 0 {
					return "", fmt.Errorf("missing required arguments: file, start, end")
				}
				return blameContext(ctx, a.File, a.Start, a.End)
			},
		},
		{
			Name:          "test_triage",
			Write:         true,
			Description:   "Run the mapped test and return a compact failure report: file:line, assertion, one-line cause. Cheaper than reading raw test output.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/src/backend/plugin_permissions.py"}`)}},
			InputSchema:   obj(map[string]any{"file": strArg("repo-relative source path")}, "file"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					File string `json:"file"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.File == "" {
					return "", fmt.Errorf("missing required argument: file")
				}
				if strings.Contains(a.File, "..") || filepath.IsAbs(a.File) {
					return "", fmt.Errorf("path must be repo-relative")
				}
				var existing []string
				for _, c := range testCandidates(a.File) {
					if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err == nil && !st.IsDir() {
						existing = append(existing, c)
					}
				}
				if len(existing) == 0 {
					return "", fmt.Errorf("no test file found for %s", a.File)
				}
				cmd, out, err := runTestsFor(ctx, existing[0])
				if err != nil {
					return "", err
				}
				return triageTestOutput(existing[0], cmd, out), nil
			},
		},
		{
			Name:          "find_test",
			Description:   "Map a source file to probable test file paths and report which exist.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/meshchat.py"}`)}},
			InputSchema:   obj(map[string]any{"file": strArg("repo-relative source path")}, "file"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					File string `json:"file"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.File == "" {
					return "", fmt.Errorf("missing required argument: file")
				}
				if strings.Contains(a.File, "..") || filepath.IsAbs(a.File) {
					return "", fmt.Errorf("path must be repo-relative")
				}
				var b strings.Builder
				found := false
				for _, c := range testCandidates(a.File) {
					exists := ""
					if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err == nil && !st.IsDir() {
						exists = " [exists]"
						found = true
					}
					fmt.Fprintf(&b, "%s%s\n", c, exists)
				}
				if !found {
					b.WriteString("(none exist yet)")
				}
				return b.String(), nil
			},
		},
	}
}

// runTestsFor executes the mapped test file and returns the command
// label plus combined output.
func runTestsFor(ctx context.Context, t string) (string, string, error) {
	switch {
	case strings.HasSuffix(t, ".py"):
		out, err := run(ctx, "uv", "run", "pytest", t, "-q", "--tb=short")
		return "uv run pytest " + t, out, err
	case strings.HasSuffix(t, ".js"), strings.HasSuffix(t, ".ts"):
		out, err := run(ctx, "pnpm", "exec", "vitest", "run", t)
		return "pnpm exec vitest run " + t, out, err
	case strings.HasSuffix(t, "_test.go"):
		out, err := run(ctx, "go", "test", "./"+filepath.Dir(t))
		return "go test ./" + filepath.Dir(t), out, err
	}
	return "", "", fmt.Errorf("no runner for %s", t)
}

var (
	pytestFailRe = regexp.MustCompile(`^FAILED\s+(\S+)`)
	pytestLocRe  = regexp.MustCompile(`^(\S+\.py):(\d+):`)
	pytestErrRe  = regexp.MustCompile(`^E\s+(.+)`)
	vitestFailRe = regexp.MustCompile(`(?:FAIL|✗|×)\s+(\S+)`)
	vitestLocRe  = regexp.MustCompile(`(\S+\.(?:js|ts|vue|svelte)):(\d+)`)
	goFailRe     = regexp.MustCompile(`^\s*--- FAIL:\s+(\S+)`)
	goLocRe      = regexp.MustCompile(`^(\S+_test\.go):(\d+):`)
)

// triageTestOutput condenses runner output into a compact report.
func triageTestOutput(testFile, cmd, out string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "test: %s\ncmd: %s\n", testFile, cmd)
	lines := strings.Split(out, "\n")
	var failures, locs, asserts []string
	for _, l := range lines {
		if m := pytestFailRe.FindStringSubmatch(l); m != nil {
			failures = append(failures, m[1])
		}
		if m := goFailRe.FindStringSubmatch(l); m != nil {
			failures = append(failures, m[1])
		}
		if m := vitestFailRe.FindStringSubmatch(l); m != nil && strings.Contains(m[1], ".") {
			failures = append(failures, m[1])
		}
		for _, re := range []*regexp.Regexp{pytestLocRe, vitestLocRe, goLocRe} {
			if m := re.FindStringSubmatch(l); m != nil {
				loc := m[1] + ":" + m[2]
				if !slices.Contains(locs, loc) {
					locs = append(locs, loc)
				}
				break
			}
		}
		if m := pytestErrRe.FindStringSubmatch(l); m != nil && len(asserts) < 10 {
			asserts = append(asserts, "E "+m[1])
		}
	}
	if len(failures) == 0 {
		b.WriteString("result: " + lastNonEmpty(lines) + "\n")
		return b.String()
	}
	fmt.Fprintf(&b, "failures (%d):\n", len(failures))
	for _, f := range failures {
		b.WriteString("  " + f + "\n")
	}
	if len(locs) > 0 {
		b.WriteString("locations: " + strings.Join(locs[:min(10, len(locs))], ", ") + "\n")
	}
	if len(asserts) > 0 {
		b.WriteString("assertions:\n")
		for _, a := range asserts {
			b.WriteString("  " + a + "\n")
		}
	}
	return b.String()
}

func lastNonEmpty(lines []string) string {
	for _, line := range slices.Backward(lines) {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return "(no output)"
}

// blameContext runs git blame -L on a line range and summarizes who
// last touched each region plus the commits involved.
func blameContext(ctx context.Context, file string, start, end int) (string, error) {
	if strings.Contains(file, "..") || filepath.IsAbs(file) {
		return "", fmt.Errorf("path must be repo-relative")
	}
	if end < start || end-start > 500 {
		return "", fmt.Errorf("range capped at 500 lines")
	}
	out, err := run(ctx, "git", "blame", "-L", fmt.Sprintf("%d,%d", start, end),
		"--line-porcelain", "--", file)
	if err != nil {
		return "", err
	}
	type commit struct {
		author, summary string
		lines           int
	}
	commits := map[string]*commit{}
	var order []string
	var cur string
	for line := range strings.SplitSeq(out, "\n") {
		if m := regexp.MustCompile(`^([0-9a-f]{40})`).FindStringSubmatch(line); m != nil {
			cur = m[1]
			if _, ok := commits[cur]; !ok {
				commits[cur] = &commit{}
				order = append(order, cur)
			}
			commits[cur].lines++
		}
		if strings.HasPrefix(line, "author ") && cur != "" {
			commits[cur].author = strings.TrimPrefix(line, "author ")
		}
		if strings.HasPrefix(line, "summary ") && cur != "" {
			commits[cur].summary = strings.TrimPrefix(line, "summary ")
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s lines %d-%d blame:\n", file, start, end)
	for _, h := range order {
		c := commits[h]
		fmt.Fprintf(&b, "  %s %s (%d lines): %s\n", h[:8], c.author, c.lines, c.summary)
	}
	return b.String(), nil
}

func main() {
	var err error
	root, err = findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "workspace-mcp:", err)
		os.Exit(1)
	}
	srv := mcp.NewServer("workspace-mcp", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "workspace-mcp:", err)
		os.Exit(1)
	}
}
