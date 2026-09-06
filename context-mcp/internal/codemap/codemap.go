// SPDX-License-Identifier: 0BSD
// Package codemap extracts symbol skeletons and provides windowed,
// token-accounted code access. Designed for agents working large files:
// read the outline, then pull only the symbol bodies needed.
package codemap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Symbol is a declaration with a signature line and 1-based range.
type Symbol struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Sig   string `json:"sig"`
	Line  int    `json:"line"`
	End   int    `json:"end"`
	Lines int    `json:"lines"`
	Refs  int    `json:"refs,omitempty"` // reference count within scanned files
}

// EstimateTokens returns a rough token count (chars/4).
func EstimateTokens(s string) int { return len(s) / 4 }

var declRes = map[string][]struct {
	kind string
	re   *regexp.Regexp
}{
	".py": {
		{"class", regexp.MustCompile(`^class\s+(?P<name>[A-Za-z_][A-Za-z0-9_]*)`)},
		{"func", regexp.MustCompile(`^\s*(?:async\s+)?def\s+(?P<name>[A-Za-z_][A-Za-z0-9_]*)`)},
	},
	".go": {
		{"func", regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?(?P<name>[A-Za-z_][A-Za-z0-9_]*)`)},
		{"type", regexp.MustCompile(`^type\s+(?P<name>[A-Za-z_][A-Za-z0-9_]*)`)},
	},
	".ts": {
		{"decl", regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:function|class|interface|type|const|let)\s+(?P<name>[A-Za-z_$][A-Za-z0-9_$]*)`)},
	},
	".js": {
		{"decl", regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:function|class|const|let)\s+(?P<name>[A-Za-z_$][A-Za-z0-9_$]*)`)},
	},
	".vue": {
		{"decl", regexp.MustCompile(`^\s{2,4}(?:async\s+)?(?P<name>[A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)},
	},
	".svelte": {
		{"decl", regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:function|const|let)\s+(?P<name>[A-Za-z_$][A-Za-z0-9_$]*)`)},
	},
}

// ReadLines returns lines[start:end] (1-based inclusive) clamped.
func ReadLines(path string, start, end int) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return "", fmt.Errorf("range %d-%d outside %d lines", start, end, len(lines))
	}
	var b strings.Builder
	for i := start; i <= end; i++ {
		fmt.Fprintf(&b, "%d: %s\n", i, lines[i-1])
	}
	return b.String(), nil
}

// Outline extracts the symbol skeleton of a file.
func Outline(path string) ([]Symbol, int, error) {
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
		for _, d := range res {
			if m := d.re.FindStringSubmatch(line); m != nil {
				if last != nil {
					last.End = i
					last.Lines = i - last.Line + 1
				}
				name := ""
				for i, sn := range d.re.SubexpNames() {
					if sn == "name" {
						name = m[i]
					}
				}
				sig := strings.TrimSpace(line)
				if len(sig) > 120 {
					sig = sig[:120] + "…"
				}
				syms = append(syms, Symbol{Name: name, Kind: d.kind, Sig: sig, Line: i + 1})
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

var codeExts = map[string]bool{
	".py": true, ".go": true, ".js": true, ".ts": true, ".jsx": true,
	".tsx": true, ".vue": true, ".svelte": true, ".rs": true, ".java": true,
	".c": true, ".h": true, ".cpp": true, ".sh": true,
}

var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "vendor": true,
	"__pycache__": true, "storage": true, "build": true, "public": true,
	"assets": true, ".venv": true,
}

// RepoMap produces a compact skeleton of code files under dir:
// per-file path + top symbols by reference count (aider-style).
func RepoMap(root, sub string, maxFiles int) (string, error) {
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return "", fmt.Errorf("path escapes root")
	}
	// first pass: gather all symbol names and count cross-file references
	type fileSyms struct {
		rel   string
		syms  []Symbol
		lines int
	}
	var files []fileSyms
	var fileData = map[string]string{}
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !codeExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		fileData[rel] = string(data)
		syms, n, err := Outline(p)
		if err != nil || len(syms) == 0 {
			return nil
		}
		files = append(files, fileSyms{filepath.ToSlash(rel), syms, n})
		return nil
	})
	if err != nil {
		return "", err
	}
	// count references: how many files mention each name
	refCount := map[string]int{}
	for _, f := range files {
		for _, s := range f.syms {
			if s.Name == "" {
				continue
			}
			cnt := 0
			for rel, src := range fileData {
				if rel != f.rel && strings.Contains(src, s.Name) {
					cnt++
				}
			}
			refCount[f.rel+"\x00"+s.Name] = cnt
		}
	}
	// rank files by total refs of their symbols
	fileScore := map[string]int{}
	for i := range files {
		sc := 0
		for j := range files[i].syms {
			files[i].syms[j].Refs = refCount[files[i].rel+"\x00"+files[i].syms[j].Name]
			sc += files[i].syms[j].Refs
		}
		fileScore[files[i].rel] = sc
	}
	sort.Slice(files, func(i, j int) bool { return fileScore[files[i].rel] > fileScore[files[j].rel] })
	var b strings.Builder
	shown := 0
	for _, f := range files {
		if shown >= maxFiles {
			break
		}
		shown++
		fmt.Fprintf(&b, "%s (%d lines):\n", f.rel, f.lines)
		// top 12 symbols by refs
		sort.Slice(f.syms, func(a, c int) bool { return f.syms[a].Refs > f.syms[c].Refs })
		for i, s := range f.syms {
			if i >= 12 {
				fmt.Fprintf(&b, "  ... %d more symbols\n", len(f.syms)-12)
				break
			}
			fmt.Fprintf(&b, "  %4d %s\n", s.Line, s.Sig)
		}
	}
	return b.String(), nil
}

// ---- persistent symbol index ----

type fileEntry struct {
	mtime int64
	syms  []Symbol
	lines int
}

var (
	idxMu   sync.Mutex
	idxData = map[string]map[string]fileEntry{} // base dir -> rel -> entry
)

// indexDir walks sub once and keeps a per-file symbol cache keyed by
// mtime, so repeat queries only re-outline files that changed.
func indexDir(root, sub string) (map[string]fileEntry, error) {
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	idxMu.Lock()
	idx := idxData[base]
	if idx == nil {
		idx = map[string]fileEntry{}
		idxData[base] = idx
	}
	idxMu.Unlock()
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !codeExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		fi, err := d.Info()
		if err != nil || fi.Size() > 2<<20 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		key := filepath.ToSlash(rel)
		idxMu.Lock()
		e, ok := idx[key]
		idxMu.Unlock()
		if ok && e.mtime == fi.ModTime().UnixNano() {
			return nil // fresh
		}
		syms, n, err := Outline(p)
		if err != nil {
			return nil
		}
		idxMu.Lock()
		idx[key] = fileEntry{fi.ModTime().UnixNano(), syms, n}
		idxMu.Unlock()
		return nil
	})
	return idx, err
}

// SymHit is a ranked symbol search result.
type SymHit struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Sig   string `json:"sig"`
	Score int    `json:"score"`
}

// SearchSymbols scores symbol names against a query: exact > prefix >
// substring > token overlap. Index cached across calls.
func SearchSymbols(root, sub, query, kind string, limit int) ([]SymHit, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, fmt.Errorf("empty query")
	}
	idx, err := indexDir(root, sub)
	if err != nil {
		return nil, err
	}
	var out []SymHit
	idxMu.Lock()
	for file, e := range idx {
		for _, s := range e.syms {
			if kind != "" && s.Kind != kind {
				continue
			}
			n := strings.ToLower(s.Name)
			var score int
			switch {
			case n == q:
				score = 100
			case strings.HasPrefix(n, q):
				score = 50
			case strings.Contains(n, q):
				score = 25
			default:
				// token overlap
				for _, tok := range strings.FieldsFunc(q, func(r rune) bool { return r == '_' || r == ' ' || r == '-' }) {
					if tok != "" && strings.Contains(n, tok) {
						score += 8
					}
				}
			}
			if score > 0 {
				out = append(out, SymHit{file, s.Line, s.Name, s.Kind, s.Sig, score})
			}
		}
	}
	idxMu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].File < out[j].File
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// DiffSymbols reports which symbols a git diff touches: for each changed
// file, the hunks are mapped to enclosing declarations via Outline.
func DiffSymbols(ctx context.Context, root, ref string, staged bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	args := []string{"-C", root, "diff", "--unified=0"}
	if staged {
		args = append(args, "--cached")
	} else if ref != "" {
		args = append(args, ref)
	}
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- fixed argv, no shell, command allowlisted
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	var (
		b       strings.Builder
		curFile string
		syms    []Symbol
		pending []string // buffered per file; emitted only if a symbol maps
	)
	fileRe := regexp.MustCompile(`^\+\+\+ b/(.+)$`)
	hunkRe := regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
	flush := func() {
		var hits []string
		for _, p := range pending {
			if !strings.Contains(p, "outside any") {
				hits = append(hits, p)
			}
		}
		if len(hits) > 0 && curFile != "" {
			fmt.Fprintf(&b, "%s:\n", curFile)
			for _, p := range hits {
				b.WriteString("  " + p + "\n")
			}
		}
		pending = pending[:0]
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if m := fileRe.FindStringSubmatch(line); m != nil {
			flush()
			curFile = m[1]
			syms = nil
			if codeExts[strings.ToLower(filepath.Ext(curFile))] {
				p := filepath.Join(root, filepath.FromSlash(curFile))
				syms, _, _ = Outline(p) // best effort
			}
			continue
		}
		m := hunkRe.FindStringSubmatch(line)
		if m == nil || curFile == "" || len(syms) == 0 {
			continue
		}
		var start, count int
		fmt.Sscanf(m[1], "%d", &start) // #nosec G104 -- error tolerated; empty/default is handled downstream
		if m[2] == "" {
			count = 1
		} else {
			fmt.Sscanf(m[2], "%d", &count) // #nosec G104 -- error tolerated; empty/default is handled downstream
		}
		end := max(start+count-1, start)
		var hit []string
		for _, s := range syms {
			if s.End >= start && s.Line <= end+1 { // hunk touches or abuts the decl
				hit = append(hit, fmt.Sprintf("%s@%d", s.Name, s.Line))
			}
		}
		if len(hit) == 0 {
			pending = append(pending, fmt.Sprintf("lines %d-%d: outside any declaration", start, end))
		} else {
			pending = append(pending, fmt.Sprintf("lines %d-%d: %s", start, end, strings.Join(hit, ", ")))
		}
	}
	flush()
	if b.Len() == 0 {
		return "no diff or no mapped symbols", nil
	}
	return b.String(), nil
}

// EnclosingSymbol returns the innermost declaration containing line.
func EnclosingSymbol(path string, line int) (*Symbol, error) {
	syms, _, err := Outline(path)
	if err != nil {
		return nil, err
	}
	var best *Symbol
	for i := range syms {
		s := &syms[i]
		if s.Line <= line && s.End >= line {
			if best == nil || s.Line > best.Line {
				best = s
			}
		}
	}
	return best, nil
}

var callRe = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

// Callers finds symbols whose bodies reference `name(` — heuristic
// callers. Each hit is resolved to the enclosing declaration.
func Callers(root, sub, name string, limit int) ([]SymHit, error) {
	if !regexp.MustCompile(`^[\w$.-]+$`).MatchString(name) {
		return nil, fmt.Errorf("invalid name")
	}
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	callPat := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\(`)
	var out []SymHit
	seen := map[string]bool{}
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || len(out) >= limit {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !codeExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		if !callPat.Match(data) {
			return nil
		}
		syms, _, _ := Outline(p)
		rel, _ := filepath.Rel(root, p)
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !callPat.MatchString(line) {
				continue
			}
			// skip the declaration itself
			if defRe := regexp.MustCompile(`(def|func|function|class)\s+` + regexp.QuoteMeta(name)); defRe.MatchString(line) {
				continue
			}
			caller := "(module level)"
			var best *Symbol
			for j := range syms {
				s := &syms[j]
				if s.Line <= i+1 && s.End >= i+1 && (best == nil || s.Line > best.Line) {
					best = s
				}
			}
			if best != nil {
				caller = best.Name
			}
			key := filepath.ToSlash(rel) + "/" + caller
			if seen[key] {
				continue
			}
			seen[key] = true
			sig := caller
			if best != nil {
				sig = best.Sig
			}
			out = append(out, SymHit{filepath.ToSlash(rel), i + 1, caller, "caller", sig, 0})
			if len(out) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	return out, err
}

// Callees lists `name(`-shaped calls inside a symbol's body and tries to
// resolve each to its definition via the symbol index.
func Callees(root, path, name string, number int) (string, error) {
	p := filepath.Join(root, filepath.FromSlash(path))
	if !strings.HasPrefix(p, root) {
		return "", fmt.Errorf("path escapes root")
	}
	syms, _, err := Outline(p)
	if err != nil {
		return "", err
	}
	var hit []Symbol
	for _, s := range syms {
		if s.Name == name {
			hit = append(hit, s)
		}
	}
	if len(hit) == 0 {
		return "", fmt.Errorf("symbol %q not found in %s", name, path)
	}
	if number <= 0 {
		number = 1
	}
	if number > len(hit) {
		return "", fmt.Errorf("only %d occurrence(s) of %q", len(hit), name)
	}
	s := hit[number-1]
	body, err := ReadLines(p, s.Line, s.End)
	if err != nil {
		return "", err
	}
	idx, err := indexDir(root, "")
	if err != nil {
		idx = nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s calls (lines %d-%d):\n", name, s.Line, s.End)
	seen := map[string]bool{}
	for _, m := range callRe.FindAllStringSubmatch(body, -1) {
		callee := m[1]
		if seen[callee] || callee == name {
			continue
		}
		seen[callee] = true
		loc := ""
		for f, e := range idx {
			for _, ds := range e.syms {
				if ds.Name == callee {
					loc = fmt.Sprintf(" -> %s:%d", f, ds.Line)
					break
				}
			}
			if loc != "" {
				break
			}
		}
		fmt.Fprintf(&b, "  %s()%s\n", callee, loc)
	}
	if len(seen) == 0 {
		b.WriteString("  (no calls found)")
	}
	return b.String(), nil
}

// FindReferences locates identifier references across code files.
type Ref struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func FindReferences(root, sub, name string, limit int) ([]Ref, error) {
	if !regexp.MustCompile(`^[\w$.-]+$`).MatchString(name) {
		return nil, fmt.Errorf("invalid name")
	}
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	re, err := regexp.Compile(`\b` + regexp.QuoteMeta(name) + `\b`)
	if err != nil {
		return nil, err
	}
	var out []Ref
	err = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || len(out) >= limit {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !codeExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		for i, line := range strings.Split(string(data), "\n") {
			if re.MatchString(line) {
				t := strings.TrimSpace(line)
				if len(t) > 160 {
					t = t[:160] + "…"
				}
				out = append(out, Ref{File: filepath.ToSlash(rel), Line: i + 1, Text: t})
				if len(out) >= limit {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	return out, err
}

// Search runs a regex over code files returning context windows.
func Search(root, sub, pattern string, ctx, limit int) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("bad regex: %w", err)
	}
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return "", fmt.Errorf("path escapes root")
	}
	var b strings.Builder
	hits := 0
	err = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || hits >= limit {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !codeExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		rel, _ := filepath.Rel(root, p)
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			lo := max(i-ctx, 0)
			hi := i + ctx
			if hi >= len(lines) {
				hi = len(lines) - 1
			}
			fmt.Fprintf(&b, "-- %s:%d --\n", filepath.ToSlash(rel), i+1)
			for j := lo; j <= hi; j++ {
				fmt.Fprintf(&b, "%d: %s\n", j+1, lines[j])
			}
			if hits++; hits >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if hits == 0 {
		return "no matches", nil
	}
	return b.String(), err
}
