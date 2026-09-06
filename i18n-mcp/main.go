// SPDX-License-Identifier: 0BSD
// Command i18n-mcp checks locale coverage: missing keys across locale
// files, per-key lookups, and hardcoded UI string candidates.
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

	"github.com/Quad4-Software/ai/i18n-mcp/internal/mcp"
)

var (
	root string
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
		for _, cand := range []string{
			"meshchatx/src/frontend/locales",
			"src/frontend/locales",
			"locales",
		} {
			if st, err := os.Stat(filepath.Join(dir, cand)); err == nil && st.IsDir() {
				return dir, nil
			}
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", fmt.Errorf("no locales dir found above cwd; set MCP_REPO_ROOT")
		}
		dir = up
	}
}

func localesPath() string {
	if v := os.Getenv("MCP_LOCALES_DIR"); v != "" {
		if filepath.IsAbs(v) {
			return v
		}
		return filepath.Join(root, v)
	}
	for _, cand := range []string{
		"meshchatx/src/frontend/locales",
		"src/frontend/locales",
		"locales",
	} {
		p := filepath.Join(root, cand)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return filepath.Join(root, "locales")
}

// flatten expands nested JSON objects to dotted keys.
func flatten(m map[string]any, prefix string, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flatten(t, key, out)
		case string:
			out[key] = t
		default:
			b, _ := json.Marshal(v)
			out[key] = string(b)
		}
	}
}

func loadLocales() (map[string]map[string]string, error) {
	ents, err := os.ReadDir(localesPath())
	if err != nil {
		return nil, err
	}
	res := map[string]map[string]string{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(localesPath(), e.Name()))
		if err != nil {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(b, &raw) != nil {
			continue
		}
		flat := map[string]string{}
		flatten(raw, "", flat)
		res[strings.TrimSuffix(e.Name(), ".json")] = flat
	}
	return res, nil
}

// hardcoded candidates: >text< in templates and literal strings in
// user-facing attributes, excluding t()/i18n calls.
var (
	tmplText = regexp.MustCompile(`>\s*([A-Z][A-Za-z][^<>{}\n]{2,80}?)\s*<`)
	attrLit  = regexp.MustCompile(`\b(placeholder|title|label|aria-label|alt)="([^"{}$]{4,80})"`)
	i18nCall = regexp.MustCompile(`\b(t|\$t|i18n)\s*\(`)
)

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

var usedKeyRe = regexp.MustCompile(`["'\x60]([a-zA-Z0-9_]+(\.[a-zA-Z0-9_]+){1,})["'\x60]`)

var codeExtsForScan = map[string]bool{
	".py": true, ".js": true, ".ts": true, ".vue": true, ".svelte": true, ".jsx": true, ".tsx": true,
}

// unusedKeys returns flattened keys in the reference locale that appear
// quoted nowhere in source files under the repo. Candidates, not proof:
// dynamic key construction can defeat it.
func unusedKeys(root, src string, limit int) ([]string, error) {
	locs, err := loadLocales()
	if err != nil {
		return nil, err
	}
	ref := locs["en"]
	if ref == nil {
		return nil, fmt.Errorf("no 'en' locale found")
	}
	// collect all dotted quoted strings from source
	used := map[string]bool{}
	base := filepath.Join(root, filepath.FromSlash(src))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes repo root")
	}
	if err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "dist", "vendor", "__pycache__", "locales":
				return filepath.SkipDir
			}
			return nil
		}
		if !codeExtsForScan[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		for _, m := range usedKeyRe.FindAllSubmatch(data, -1) {
			used[string(m[1])] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}
	var out []string
	for k := range ref {
		if !used[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

var placeholderRe = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}|%[sdif]`)

// keyStats reports: total keys, longest values, duplicate values, and
// placeholder mismatches across locales (a missing {var} breaks the UI).
func keyStats(limit int) (string, error) {
	locs, err := loadLocales()
	if err != nil {
		return "", err
	}
	ref := locs["en"]
	if ref == nil {
		return "", fmt.Errorf("no 'en' locale")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d keys in en, %d locales\n\n", len(ref), len(locs))

	// longest en values
	type kv struct {
		k, v string
	}
	var vals []kv
	for k, v := range ref {
		vals = append(vals, kv{k, v})
	}
	sort.Slice(vals, func(i, j int) bool { return len(vals[i].v) > len(vals[j].v) })
	b.WriteString("longest values:\n")
	for i := 0; i < limit && i < len(vals); i++ {
		fmt.Fprintf(&b, "  %5d  %s\n", len(vals[i].v), vals[i].k)
	}

	// duplicate values in en
	byVal := map[string][]string{}
	for _, p := range vals {
		if len(p.v) > 15 {
			byVal[p.v] = append(byVal[p.v], p.k)
		}
	}
	var dups []string
	for v, ks := range byVal {
		if len(ks) > 1 {
			dups = append(dups, fmt.Sprintf("%q x%d: %s", trunc(v, 60), len(ks), strings.Join(ks[:min(4, len(ks))], ", ")))
		}
	}
	sort.Strings(dups)
	fmt.Fprintf(&b, "\nduplicate values (>15 chars): %d groups\n", len(dups))
	for i := 0; i < limit && i < len(dups); i++ {
		b.WriteString("  " + dups[i] + "\n")
	}

	// placeholder mismatches across locales
	var mism []string
	for loc, m := range locs {
		if loc == "en" {
			continue
		}
		for k, refV := range ref {
			lv, ok := m[k]
			if !ok {
				continue
			}
			rp := sortedSet(placeholderRe.FindAllString(refV, -1))
			lp := sortedSet(placeholderRe.FindAllString(lv, -1))
			if strings.Join(rp, ",") != strings.Join(lp, ",") {
				mism = append(mism, fmt.Sprintf("%s %s: en=%v %s=%v", k, loc, rp, loc, lp))
			}
		}
	}
	sort.Strings(mism)
	fmt.Fprintf(&b, "\nplaceholder mismatches: %d\n", len(mism))
	for i := 0; i < limit && i < len(mism); i++ {
		b.WriteString("  " + mism[i] + "\n")
	}
	return b.String(), nil
}

func sortedSet(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_locales",
			Description: "List locale files with flattened key counts and the locales directory in use.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				locs, err := loadLocales()
				if err != nil {
					return "", err
				}
				var b strings.Builder
				fmt.Fprintf(&b, "locales dir: %s\n", localesPath())
				names := make([]string, 0, len(locs))
				for n := range locs {
					names = append(names, n)
				}
				sort.Strings(names)
				for _, n := range names {
					fmt.Fprintf(&b, "%s: %d keys\n", n, len(locs[n]))
				}
				return b.String(), nil
			},
		},
		{
			Name:          "missing_keys",
			Description:   "Compare flattened keys of the reference locale (default en) against all others; report missing per locale.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"locale": "de"}`)}},
			InputSchema: obj(map[string]any{
				"reference": strArg("reference locale, default en"),
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Reference string `json:"reference"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				ref := a.Reference
				if ref == "" {
					ref = "en"
				}
				locs, err := loadLocales()
				if err != nil {
					return "", err
				}
				base, ok := locs[ref]
				if !ok {
					return "", fmt.Errorf("reference locale %q not found", ref)
				}
				var b strings.Builder
				total := 0
				for name, m := range locs {
					if name == ref {
						continue
					}
					var missing []string
					for k := range base {
						if _, ok := m[k]; !ok {
							missing = append(missing, k)
						}
					}
					if len(missing) > 0 {
						sort.Strings(missing)
						fmt.Fprintf(&b, "%s: %d missing\n", name, len(missing))
						for _, k := range missing {
							fmt.Fprintf(&b, "  %s\n", k)
						}
						total += len(missing)
					}
				}
				if total == 0 {
					return "all locales complete against " + ref, nil
				}
				return b.String(), nil
			},
		},
		{
			Name:          "key_lookup",
			Description:   "Look up a flattened key across all locales.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"key": "banishment.title"}`)}},
			InputSchema:   obj(map[string]any{"key": strArg("dotted key, e.g. settings.title")}, "key"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Key string `json:"key"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Key == "" {
					return "", fmt.Errorf("missing required argument: key")
				}
				locs, err := loadLocales()
				if err != nil {
					return "", err
				}
				var b strings.Builder
				found := false
				for name, m := range locs {
					if v, ok := m[a.Key]; ok {
						fmt.Fprintf(&b, "%s: %s\n", name, v)
						found = true
					}
				}
				if !found {
					return "", fmt.Errorf("key %q not in any locale file", a.Key)
				}
				return b.String(), nil
			},
		},
		{
			Name:        "hardcoded_strings",
			Description: "Scan .vue/.svelte files under a path for template text and attribute literals not routed through i18n. Heuristic, expect some noise.",
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir to scan, default meshchatx/src/frontend/src"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 40"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Path == "" {
					a.Path = "meshchatx/src/frontend/src"
				}
				if a.Limit <= 0 || a.Limit > 300 {
					a.Limit = 40
				}
				base := filepath.Join(root, filepath.FromSlash(a.Path))
				if !strings.HasPrefix(base, root) {
					return "", fmt.Errorf("path escapes repo root")
				}
				var b strings.Builder
				n := 0
				err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
					if err != nil || d.IsDir() || n >= a.Limit {
						return err
					}
					ext := strings.ToLower(filepath.Ext(p))
					if ext != ".vue" && ext != ".svelte" && ext != ".ts" && ext != ".js" && ext != ".tsx" && ext != ".jsx" {
						return nil
					}
					data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
					if err != nil || len(data) > 1<<20 {
						return nil
					}
					rel, _ := filepath.Rel(root, p)
					for i, line := range strings.Split(string(data), "\n") {
						if i18nCall.MatchString(line) {
							continue
						}
						var match string
						if m := tmplText.FindStringSubmatch(line); m != nil {
							match = "template text: " + m[1]
						} else if m := attrLit.FindStringSubmatch(line); m != nil {
							match = m[1] + " attr: " + m[2]
						}
						if match != "" {
							fmt.Fprintf(&b, "%s:%d: %s\n", filepath.ToSlash(rel), i+1, match)
							if n++; n >= a.Limit {
								return filepath.SkipAll
							}
						}
					}
					return nil
				})
				if err != nil {
					return "", err
				}
				if n == 0 {
					return "no hardcoded string candidates found", nil
				}
				return b.String(), nil
			},
		},
		{
			Name:        "key_usage",
			Description: "Find where a key is referenced in repo source files (first 30 hits).",
			InputSchema: obj(map[string]any{
				"key":  strArg("dotted key or substring"),
				"path": strArg("repo-relative dir, default repo root"),
			}, "key"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Key  string `json:"key"`
					Path string `json:"path"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Key == "" {
					return "", fmt.Errorf("missing required argument: key")
				}
				base := root
				if a.Path != "" {
					base = filepath.Join(root, filepath.FromSlash(a.Path))
				}
				if !strings.HasPrefix(base, root) {
					return "", fmt.Errorf("path escapes repo root")
				}
				var b strings.Builder
				n := 0
				err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
					if err != nil || d.IsDir() || n >= 30 {
						return err
					}
					name := d.Name()
					if name == "node_modules" || name == ".git" || name == "dist" {
						return filepath.SkipDir
					}
					if d.IsDir() {
						return nil
					}
					data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
					if err != nil || len(data) > 1<<20 {
						return nil
					}
					if !strings.Contains(string(data), a.Key) {
						return nil
					}
					rel, _ := filepath.Rel(root, p)
					for i, line := range strings.Split(string(data), "\n") {
						if strings.Contains(line, a.Key) {
							fmt.Fprintf(&b, "%s:%d: %s\n", filepath.ToSlash(rel), i+1, strings.TrimSpace(line))
							if n++; n >= 30 {
								return filepath.SkipAll
							}
						}
					}
					return nil
				})
				if err != nil {
					return "", err
				}
				if n == 0 {
					return "key not referenced", nil
				}
				return b.String(), nil
			},
		},
		{
			Name:          "key_stats",
			Description:   "Locale stats: key counts, longest values, duplicate values, and placeholder ({var} / %s) mismatches across locales.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"limit": 10}`)}},
			InputSchema: obj(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "max rows per section, default 10"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Limit int `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 100 {
					a.Limit = 10
				}
				return keyStats(a.Limit)
			},
		},
		{
			Name:          "unused_keys",
			Description:   "Report flattened reference-locale keys that appear quoted nowhere in source. Candidates only: dynamic key construction can produce false positives.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/src", "limit": 30}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative source dir, default root"),
				"limit": map[string]any{"type": "integer", "description": "max keys, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 500 {
					a.Limit = 50
				}
				keys, err := unusedKeys(root, a.Path, a.Limit)
				if err != nil {
					return "", err
				}
				if len(keys) == 0 {
					return "no unused keys detected", nil
				}
				return fmt.Sprintf("%d unused key candidates (first %d):\n%s",
					len(keys), min(a.Limit, len(keys)), strings.Join(keys, "\n")), nil
			},
		},
	}
}

func main() {
	var err error
	root, err = findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "i18n-mcp:", err)
		os.Exit(1)
	}
	srv := mcp.NewServer("i18n-mcp", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "i18n-mcp:", err)
		os.Exit(1)
	}
}
