// SPDX-License-Identifier: 0BSD
// Command agents-mcp exposes a repo's .agents knowledge graph and docs
// as MCP tools: skills, conventions, module ownership, doc search.
// Read-only, path-jailed to the repo root. Stdlib only.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Quad4-Software/ai/agents-mcp/internal/mcp"
)

const maxFile = 1 << 20

var root string

// findRoot returns MCP_REPO_ROOT or walks up from cwd for .agents.
func findRoot() (string, error) {
	if r := os.Getenv("MCP_REPO_ROOT"); r != "" {
		return filepath.Abs(r)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, ".agents")); err == nil && st.IsDir() {
			return dir, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", fmt.Errorf("no .agents dir found above %s; set MCP_REPO_ROOT", dir)
		}
		dir = up
	}
}

// jailed resolves rel under root and refuses escapes.
func jailed(rel string) (string, error) {
	p := filepath.Clean(filepath.Join(root, rel))
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

func readJailed(rel string) (string, error) {
	p, err := jailed(rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if info.Size() > maxFile {
		return "", fmt.Errorf("file too large (>1 MiB)")
	}
	b, err := os.ReadFile(p) // #nosec G304 -- path jailed to repo root or local config
	return string(b), err
}

type docEntry struct {
	Name string `json:"name"`
	Path string `json:"path"` // repo-relative
}

func listDir(sub string) ([]docEntry, error) {
	base, err := jailed(sub)
	if err != nil {
		return nil, err
	}
	var out []docEntry
	err = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, docEntry{Name: d.Name(), Path: filepath.ToSlash(rel)})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

var skillHead = regexp.MustCompile(`(?m)^description:\s*"?([^"\n]+)"?|^#\s+(.+)$`)

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "repo_root",
			Description: "Return the detected repo root (MCP_REPO_ROOT or nearest .agents parent).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return root, nil
			},
		},
		{
			Name:        "list_skills",
			Description: "List skills under .agents/skills with their SKILL.md paths.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				ents, err := listDir(".agents/skills")
				if err != nil {
					return "", err
				}
				var tops []docEntry
				for _, e := range ents {
					if strings.HasSuffix(e.Path, "/SKILL.md") {
						tops = append(tops, e)
					}
				}
				b, _ := json.MarshalIndent(tops, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:          "get_skill",
			Description:   "Return a skill's SKILL.md text by directory name (e.g. test-oracles).",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"name": "test-oracles"}`)}},
			InputSchema:   obj(map[string]any{"name": strArg("skill directory name")}, "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				if strings.ContainsAny(a.Name, "/\\") {
					return "", fmt.Errorf("name must be a directory name, not a path")
				}
				return readJailed(filepath.Join(".agents/skills", a.Name, "SKILL.md"))
			},
		},
		{
			Name:        "list_conventions",
			Description: "List files under .agents/conventions plus top-level .agents docs.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				var out []docEntry
				for _, sub := range []string{".agents/conventions"} {
					ents, err := listDir(sub)
					if err == nil {
						out = append(out, ents...)
					}
				}
				for _, f := range []string{".agents/README.md", ".agents/overview.md", ".agents/module-ownership.md", "AGENTS.md"} {
					if p, err := jailed(f); err == nil {
						if st, err := os.Stat(p); err == nil && !st.IsDir() {
							out = append(out, docEntry{Name: filepath.Base(f), Path: f})
						}
					}
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "read_file",
			Description: "Read any file in the repo (path-jailed to root, 1 MiB cap). Use repo-relative paths.",
			InputSchema: obj(map[string]any{"path": strArg("repo-relative path")}, "path"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" {
					return "", fmt.Errorf("missing required argument: path")
				}
				return readJailed(a.Path)
			},
		},
		{
			Name:        "search_docs",
			Description: "Regex search across .agents/ and docs/ returning file:line matches.",
			InputSchema: obj(map[string]any{
				"query": strArg("regex or plain text to search"),
				"limit": map[string]any{"type": "integer", "description": "max matches, default 30"},
			}, "query"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				if a.Limit <= 0 || a.Limit > 200 {
					a.Limit = 30
				}
				re, err := regexp.Compile("(?i)" + a.Query)
				if err != nil {
					return "", fmt.Errorf("bad regex: %w", err)
				}
				var b strings.Builder
				n := 0
				for _, sub := range []string{".agents", "docs", "AGENTS.md"} {
					base, err := jailed(sub)
					if err != nil {
						continue
					}
					st, err := os.Stat(base)
					if err != nil {
						continue
					}
					walk := func(p string, d os.DirEntry, err error) error {
						if err != nil || d.IsDir() || n >= a.Limit {
							return err
						}
						if strings.HasSuffix(d.Name(), ".md") || strings.HasSuffix(d.Name(), ".mdc") || strings.HasSuffix(d.Name(), ".txt") {
							data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
							if err != nil || len(data) > maxFile {
								return nil
							}
							for i, line := range strings.Split(string(data), "\n") {
								if re.MatchString(line) {
									rel, _ := filepath.Rel(root, p)
									fmt.Fprintf(&b, "%s:%d: %s\n", filepath.ToSlash(rel), i+1, strings.TrimSpace(line))
									if n++; n >= a.Limit {
										return filepath.SkipAll
									}
								}
							}
						}
						return nil
					}
					if st.IsDir() {
						filepath.WalkDir(base, walk) // #nosec G104 -- error tolerated; empty/default is handled downstream
					} else {
						data, _ := os.ReadFile(base) // #nosec G304 -- path jailed to repo root or local config
						for i, line := range strings.Split(string(data), "\n") {
							if re.MatchString(line) {
								fmt.Fprintf(&b, "%s:%d: %s\n", sub, i+1, strings.TrimSpace(line))
								n++
							}
						}
					}
				}
				if n == 0 {
					return "no matches", nil
				}
				return b.String(), nil
			},
		},
		{
			Name:          "tree",
			Description:   "Directory listing of a repo-relative path (depth-limited, skips node_modules/.git/dist).",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": ".agents", "depth": 2}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default root"),
				"depth": map[string]any{"type": "integer", "description": "max depth, default 2"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path  string `json:"path"`
					Depth int    `json:"depth"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Depth <= 0 || a.Depth > 6 {
					a.Depth = 2
				}
				base, err := jailed(a.Path)
				if err != nil {
					return "", err
				}
				skip := map[string]bool{"node_modules": true, ".git": true, "dist": true, "vendor": true, "__pycache__": true, "storage": true, "build": true}
				var b strings.Builder
				baseDepth := strings.Count(base, string(filepath.Separator))
				err = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
					if err != nil {
						return nil
					}
					depth := strings.Count(p, string(filepath.Separator)) - baseDepth
					if d.IsDir() && (depth > a.Depth || skip[d.Name()]) {
						return filepath.SkipDir
					}
					if depth > a.Depth {
						return nil
					}
					rel, _ := filepath.Rel(root, p)
					suffix := ""
					if d.IsDir() {
						suffix = "/"
					}
					fmt.Fprintf(&b, "%s%s%s\n", strings.Repeat("  ", depth), filepath.Base(p), suffix)
					_ = rel
					return nil
				})
				if err != nil {
					return "", err
				}
				return b.String(), nil
			},
		},
		{
			Name:          "skill_for",
			Description:   "Match a task description to the right .agents skill: ranked skill names + paths. Use before starting work so you load the right conventions.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"task": "add a websocket mutator with CSRF"}`)}},
			InputSchema: obj(map[string]any{
				"task":  strArg("what you are about to do, e.g. 'add a websocket handler that mutates state'"),
				"limit": map[string]any{"type": "integer", "description": "max, default 5"},
			}, "task"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Task  string `json:"task"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Task == "" {
					return "", fmt.Errorf("missing required argument: task")
				}
				if a.Limit <= 0 || a.Limit > 15 {
					a.Limit = 5
				}
				return skillFor(a.Task, a.Limit)
			},
		},
		{
			Name:        "module_owners",
			Description: "Return .agents/module-ownership.md (where code lives and who owns what).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return readJailed(".agents/module-ownership.md")
			},
		},
	}
}

// skillMatch scores skills against a free-text task description.
type SkillMatch struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Desc  string `json:"desc"`
	Score int    `json:"score"`
}

var frontRe = regexp.MustCompile(`(?m)^name:\s*(\S+)\s*$|^description:\s*(.+)$`)

func skillFor(task string, limit int) (string, error) {
	dir := filepath.Join(root, ".agents", "skills")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	words := map[string]bool{}
	for _, w := range regexp.MustCompile(`[a-zA-Z]{3,}`).FindAllString(strings.ToLower(task), -1) {
		words[w] = true
	}
	var out []SkillMatch
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md")) // #nosec G304 -- path jailed to repo root or local config
		if err != nil {
			continue
		}
		name, desc := e.Name(), ""
		for _, m := range frontRe.FindAllStringSubmatch(string(data[:min(len(data), 2048)]), -1) {
			if m[1] != "" {
				name = m[1]
			}
			if m[2] != "" {
				desc = strings.TrimSpace(m[2])
			}
		}
		score := 0
		low := strings.ToLower(name + " " + e.Name() + " " + desc)
		for w := range words {
			if strings.Contains(strings.ToLower(name), w) || strings.Contains(strings.ToLower(e.Name()), w) {
				score += 3
			} else if strings.Contains(low, w) {
				score++
			}
		}
		if score > 0 {
			out = append(out, SkillMatch{e.Name(), filepath.ToSlash(filepath.Join(".agents/skills", e.Name(), "SKILL.md")), desc, score})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	if len(out) == 0 {
		return "no skill matched; try list_skills", nil
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return string(b), nil
}

func main() {
	var err error
	root, err = findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "agents-mcp:", err)
		os.Exit(1)
	}
	srv := mcp.NewServer("agents-mcp", "0.1.0", append(tools(), askTool, fuzzySearchTool), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "agents-mcp:", err)
		os.Exit(1)
	}
}
