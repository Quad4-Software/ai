// SPDX-License-Identifier: 0BSD
// Package refactor provides god-file detection, split planning, and
// surface snapshots for safe mega-page breakdowns. The goal: nothing
// gets dropped silently during a split.
package refactor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Symbol is a top-level declaration with a line range.
type Symbol struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"` // func, class, method, const, route, ws-type, i18n-key, emit
	Line  int    `json:"line"`
	End   int    `json:"end"`
	Lines int    `json:"lines"`
}

// GodFile reports a file that does too much.
type GodFile struct {
	Path    string   `json:"path"`
	Lines   int      `json:"lines"`
	Symbols int      `json:"symbols"`
	Kinds   int      `json:"kinds"`   // distinct symbol kinds
	Signals []string `json:"signals"` // why it was flagged
	Score   float64  `json:"score"`
}

var declRes = map[string][]*regexp.Regexp{
	".py": {
		regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)`),
		regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)`),
	},
	".go": {
		regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)`),
		regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)`),
	},
	".ts": {
		regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
		regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`),
		regexp.MustCompile(`^(?:export\s+)?(?:class|interface|type)\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
	},
	".js": {
		regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
		regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`),
		regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)?`),
	},
	".vue": {
		regexp.MustCompile(`^\s{2,4}(?:async\s+)?([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`), // methods
		regexp.MustCompile(`^\s*(?:const|let|var|function)\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
	},
	".svelte": {
		regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
		regexp.MustCompile(`^\s*(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`),
	},
}

func declKind(line, ext string) string {
	switch ext {
	case ".py":
		if strings.HasPrefix(strings.TrimSpace(line), "class") {
			return "class"
		}
		return "func"
	case ".go":
		if strings.HasPrefix(line, "type") {
			return "type"
		}
		return "func"
	}
	t := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(t, "class"), strings.Contains(t, " class "):
		return "class"
	case strings.HasPrefix(t, "interface"), strings.HasPrefix(t, "type"):
		return "type"
	default:
		return "func"
	}
}

// Symbols extracts top-level declarations with line ranges.
func Symbols(path string) ([]Symbol, int, error) {
	ext := strings.ToLower(filepath.Ext(path))
	res := declRes[ext]
	if res == nil {
		return nil, 0, fmt.Errorf("unsupported file type %s", ext)
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		return nil, 0, err
	}
	lines := strings.Split(string(data), "\n")
	var syms []Symbol
	var last *Symbol
	for i, line := range lines {
		for _, re := range res {
			if m := re.FindStringSubmatch(line); m != nil {
				if last != nil {
					last.End = i
					last.Lines = i - last.Line + 1
				}
				name := ""
				if len(m) > 1 {
					name = m[1]
				}
				syms = append(syms, Symbol{Name: name, Kind: declKind(line, ext), Line: i + 1})
				last = &syms[len(syms)-1]
				break
			}
		}
	}
	if last != nil {
		last.End = len(lines)
		last.Lines = len(lines) - last.Line + 1
	}
	return syms, len(lines), nil
}

// GodFiles ranks files under dir by size and symbol count.
func GodFiles(root, sub string, limit int) ([]GodFile, error) {
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	var out []GodFile
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "dist", "vendor", "__pycache__", "storage", "public", "assets", "build":
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if declRes[ext] == nil {
			return nil
		}
		syms, nlines, err := Symbols(p)
		if err != nil || nlines < 400 {
			return nil
		}
		var signals []string
		kinds := map[string]bool{}
		for _, s := range syms {
			kinds[s.Kind] = true
		}
		if nlines > 1000 {
			signals = append(signals, "over 1000 lines")
		}
		if len(syms) > 25 {
			signals = append(signals, fmt.Sprintf("%d top-level symbols", len(syms)))
		}
		if len(kinds) >= 3 {
			signals = append(signals, "mixed kinds (class+func+type)")
		}
		if ext == ".vue" && nlines > 600 {
			signals = append(signals, "Vue mega-page")
		}
		if len(signals) == 0 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, GodFile{
			Path: filepath.ToSlash(rel), Lines: nlines, Symbols: len(syms),
			Kinds: len(kinds), Signals: signals,
			Score: float64(nlines)/400 + float64(len(syms))/8,
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// SplitGroup is a suggested extraction target.
type SplitGroup struct {
	Target  string   `json:"target"`
	Symbols []string `json:"symbols"`
	Lines   int      `json:"lines"`
	Reason  string   `json:"reason"`
}

var commonPrefixes = []string{"get", "set", "load", "save", "on", "handle", "render", "fetch", "update", "compute", "watch"}

// domainGroup clusters symbols by shared underscore/domain tokens so
// split groups reflect features, not just prefixes.
func domainToken(name string, freq map[string]int) string {
	best := ""
	for _, tok := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return r == '_' || r == '-'
	}) {
		if len(tok) >= 4 && freq[tok] >= 3 && len(tok) > len(best) {
			best = tok
		}
	}
	return best
}

// SplitPlan inventories a file and groups its symbols into suggested
// extraction targets. Grouping: shared name prefix clusters, then
// leftover misc.
func SplitPlan(root, rel string) ([]SplitGroup, []Symbol, error) {
	p := filepath.Join(root, filepath.FromSlash(rel))
	if !strings.HasPrefix(p, root) {
		return nil, nil, fmt.Errorf("path escapes root")
	}
	syms, _, err := Symbols(p)
	if err != nil {
		return nil, nil, err
	}
	// count token frequency for domain clustering
	freq := map[string]int{}
	for _, s := range syms {
		for _, tok := range strings.FieldsFunc(strings.ToLower(s.Name), func(r rune) bool {
			return r == '_' || r == '-'
		}) {
			freq[tok]++
		}
	}
	groups := map[string]*SplitGroup{}
	for _, s := range syms {
		if s.Name == "" {
			continue
		}
		lower := strings.ToLower(strings.TrimLeft(s.Name, "_"))
		prefix := "misc"
		if tok := domainToken(s.Name, freq); tok != "" {
			prefix = tok
		} else {
			for _, cp := range commonPrefixes {
				if strings.HasPrefix(lower, cp) && len(lower) > len(cp) {
					prefix = cp
					break
				}
			}
		}
		g := groups[prefix]
		if g == nil {
			g = &SplitGroup{Target: "internal/" + prefix}
			groups[prefix] = g
		}
		g.Symbols = append(g.Symbols, s.Name)
		g.Lines += s.Lines
	}
	var out []SplitGroup
	for _, g := range groups {
		g.Reason = fmt.Sprintf("%d symbols sharing prefix", len(g.Symbols))
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Lines > out[j].Lines })
	return out, syms, nil
}

// Surface patterns capture the observable contract of a file: things a
// refactor must not lose.
var surfaceRes = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"registry", regexp.MustCompile(`registerFeature|registerTool|registerCommand|registerNav`)},
	{"ws", regexp.MustCompile(`registry\.register\s*\(\s*['"]([a-zA-Z0-9_.-]+)['"]`)},
	{"event", regexp.MustCompile(`(?:\.on|emit|dispatch)\s*\(\s*['"]([a-zA-Z0-9_.-]+)['"]`)},
	{"i18n", regexp.MustCompile(`(?:\$t|\bt|i18n\.t)\s*\(\s*['"]([a-zA-Z0-9_.-]+)['"]`)},
	{"symbol", regexp.MustCompile(`(?:def|function|func)\s+([A-Za-z_][A-Za-z0-9_]*)`)},
	{"route", regexp.MustCompile(`@\w+\.(?:route|get|post|put|delete|patch)\s*\(\s*['"]([^'"]+)['"]`)},
	{"config-key", regexp.MustCompile(`(?:name|path|id):\s*['"]([a-zA-Z0-9_/.-]+)['"]`)},
	{"framework", regexp.MustCompile(`@app\.(?:route|websocket)|HandleFunc|add_url_rule`)},
}

// SurfaceSnapshot extracts the observable contract of a path (file or dir).
func SurfaceSnapshot(root, sub string) ([]string, error) {
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	seen := map[string]bool{}
	var files []string
	st, err := os.Stat(base)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error { // #nosec G104 -- error tolerated; empty/default is handled downstream
			if err != nil || d.IsDir() {
				return err
			}
			if declRes[strings.ToLower(filepath.Ext(p))] != nil || strings.HasSuffix(p, ".json") {
				files = append(files, p)
			}
			return nil
		})
	} else {
		files = []string{base}
	}
	for _, p := range files {
		data, err := os.ReadFile(p) // #nosec G304 -- path jailed to repo root or local config
		if err != nil || len(data) > 2<<20 {
			continue
		}
		rel, _ := filepath.Rel(root, p)
		for _, s := range surfaceRes {
			for _, m := range s.re.FindAllStringSubmatch(string(data), -1) {
				val := m[0]
				for _, g := range m[1:] {
					if g != "" {
						val = g
						break
					}
				}
				seen[s.kind+":"+val] = true
			}
		}
		seen["file:"+filepath.ToSlash(rel)] = true
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// SurfaceDiff compares a before snapshot to the current surface.
func SurfaceDiff(root, sub string, before []string) (missing, added []string, err error) {
	now, err := SurfaceSnapshot(root, sub)
	if err != nil {
		return nil, nil, err
	}
	nowSet := map[string]bool{}
	for _, k := range now {
		nowSet[k] = true
	}
	beforeSet := map[string]bool{}
	for _, k := range before {
		beforeSet[k] = true
	}
	for _, k := range before {
		if !nowSet[k] {
			missing = append(missing, k)
		}
	}
	for _, k := range now {
		if !beforeSet[k] {
			added = append(added, k)
		}
	}
	return missing, added, nil
}
