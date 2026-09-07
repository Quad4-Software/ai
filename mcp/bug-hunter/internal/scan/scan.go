// SPDX-License-Identifier: 0BSD
// Package scan implements mechanical bug-hunting scanners: git churn
// hotspots, TOCTOU check-then-use pairs, mutating attack surface, and
// soft-fuzz test anti-patterns. All read-only.
package scan

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Hotspot ranks files by commits in a window weighted by recency.
type Hotspot struct {
	Path    string  `json:"path"`
	Commits int     `json:"commits"`
	Score   float64 `json:"score"` // commits decayed by recency
	Bytes   int64   `json:"bytes"`
	Gone    bool    `json:"gone,omitempty"` // file deleted or moved since
}

// Hotspots runs git log over the window and ranks files.
// Research basis: defect density concentrates in high-churn, large files.
func Hotspots(ctx context.Context, root, since string, limit int) ([]Hotspot, error) {
	if _, err := time.ParseDuration(since); err == nil {
		since = time.Now().Add(-mustDur(since)).Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", since); err != nil {
		return nil, fmt.Errorf("since must be YYYY-MM-DD or a duration like 720h: %q", since)
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", // #nosec G204 -- fixed argv, no shell, command allowlisted
		"--since="+since, "--name-only", "--format=commit %at")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	now := float64(time.Now().Unix())
	acc := map[string]*Hotspot{}
	var ts float64
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if after, ok := strings.CutPrefix(line, "commit "); ok {
			v, _ := strconv.ParseInt(after, 10, 64)
			ts = float64(v)
			continue
		}
		if line == "" || strings.Contains(line, " ") {
			continue
		}
		if !codeExts[strings.ToLower(filepath.Ext(line))] {
			continue // data/doc churn is not defect signal
		}
		h := acc[line]
		if h == nil {
			h = &Hotspot{Path: line}
			acc[line] = h
		}
		h.Commits++
		days := (now - ts) / 86400
		h.Score += 1 / (1 + days/30) // monthly decay
	}
	var list []Hotspot
	for _, h := range acc {
		if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(h.Path))); err == nil {
			h.Bytes = fi.Size()
		} else {
			h.Gone = true
		}
		list = append(list, *h)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })
	if len(list) > limit {
		list = list[:limit]
	}
	return list, sc.Err()
}

func mustDur(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

// FixCommit is a past fix commit and the files it touched.
type FixCommit struct {
	Hash  string   `json:"hash"`
	Title string   `json:"title"`
	Files []string `json:"files"`
}

var fixMsgRe = regexp.MustCompile(`(?i)\b(fix|hotfix|revert|regression|bug)\b`)

// RegressionMine lists recent fix/revert commits with the files they
// touched. Past bugs predict future ones; siblings of a fixed bug are
// often unreported.
func RegressionMine(ctx context.Context, root string, n int) ([]FixCommit, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", // #nosec G204 -- fixed argv, no shell, command allowlisted
		"--name-only", "--format=@@@%h %s", "-n", strconv.Itoa(n*10))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	var out2 []FixCommit
	var cur *FixCommit
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "@@@") {
			title := line[3:]
			if fixMsgRe.MatchString(title) {
				if len(out2) < n {
					parts := strings.SplitN(title, " ", 2)
					fc := FixCommit{Hash: parts[0]}
					if len(parts) > 1 {
						fc.Title = parts[1]
					}
					out2 = append(out2, fc)
					cur = &out2[len(out2)-1]
				} else {
					cur = nil
				}
			} else {
				cur = nil
			}
			continue
		}
		if cur != nil && strings.TrimSpace(line) != "" {
			cur.Files = append(cur.Files, line)
		}
	}
	return out2, sc.Err()
}

// Pair is a check-then-use candidate.
type Pair struct {
	File       string `json:"file"`
	CheckLine  int    `json:"check_line"`
	UseLine    int    `json:"use_line"`
	Check      string `json:"check"`
	Use        string `json:"use"`
	Confidence string `json:"confidence"` // high when check and use share an identifier
}

var callArgRe = regexp.MustCompile(`\(\s*([A-Za-z_][\w.\[\]]*)`)

// sharedIdent returns the argument identifier of the check call, used to
// score confidence when it also appears in the use line.
func sharedIdent(checkLine string) string {
	m := callArgRe.FindStringSubmatch(checkLine)
	if m == nil {
		return ""
	}
	return m[1]
}

var toctouChecks = []*regexp.Regexp{
	regexp.MustCompile(`\b(os\.(Stat|Lstat|PathExists|IsExist)|fs\.(existsSync|statSync|accessSync)|os\.access|Path\(.+\)\.(exists|is_file)|pathlib.*\.exists|os\.path\.(exists|isfile|isdir|lexists)|access\()\b`),
}
var toctouUses = []*regexp.Regexp{
	regexp.MustCompile(`\b(os\.(Open|Create|Remove|Rename|Mkdir(All)?)|fs\.(readFile|writeFile|unlink|rename|rm)Sync?|open\(|os\.remove|os\.rename|shutil\.|ioutil\.(ReadFile|WriteFile))\b`),
}

const toctouWindow = 25 // max lines between check and use

// TOCTOU finds check-then-use pairs within toctouWindow lines.
func TOCTOU(root, sub string, limit int) ([]Pair, error) {
	var out []Pair
	err := walkCode(root, sub, func(p string, lines []string) {
		var checks []int
		for i, line := range lines {
			for _, re := range toctouChecks {
				if re.MatchString(line) {
					checks = append(checks, i)
					break
				}
			}
		}
		for _, ci := range checks {
			for j := ci + 1; j < len(lines) && j <= ci+toctouWindow; j++ {
				for _, re := range toctouUses {
					if re.MatchString(lines[j]) {
						rel, _ := filepath.Rel(root, p)
						conf := "medium"
						if id := sharedIdent(lines[ci]); id != "" {
							if re, err := regexp.Compile(`\b` + regexp.QuoteMeta(id) + `\b`); err == nil && re.MatchString(lines[j]) {
								conf = "high"
							}
						}
						out = append(out, Pair{
							File: filepath.ToSlash(rel), CheckLine: ci + 1, UseLine: j + 1,
							Check: strings.TrimSpace(lines[ci]), Use: strings.TrimSpace(lines[j]),
							Confidence: conf,
						})
						goto next
					}
				}
			}
		next:
		}
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].File+fmt.Sprint(out[i].CheckLine) < out[j].File+fmt.Sprint(out[j].CheckLine)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// Sink is a mutating entry point.
type Sink struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Kind  string `json:"kind"`
	Match string `json:"match"`
}

var sinks = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"exec", regexp.MustCompile(`\b(subprocess\.|os\.system|exec\.Command|child_process|shell_exec|Popen)\b`)},
	{"file-write", regexp.MustCompile(`\b(os\.(Create|WriteFile|Remove|Rename)|fs\.(writeFile|rm|unlink)|open\([^)]*['"]w|open\([^)]*['"]a)\b`)},
	{"http-mutator", regexp.MustCompile(`\.(post|put|delete|patch)\(|@(app|router|bp)\.(post|put|delete|patch)|HandleFunc\([^)]*POST`)},
	{"ws-handler", regexp.MustCompile(`registry\.register\s*\(|@app\.websocket|@websocket|ws\.(on|handle)|case\s+['"][a-z_]+\.[a-z_]+['"]`)},
	{"eval", regexp.MustCompile(`\b(eval\(|exec\(|Function\(|setTimeout\(['"]|innerHTML\s*=)`)},
	{"deserialize", regexp.MustCompile(`\b(pickle\.loads|yaml\.load\(|unserialize|msgpack\.(loads|unpack)|cbor2?\.loads)\b`)},
	{"sql", regexp.MustCompile(`(execute\(|executemany\(|raw\s*SQL|SELECT .*\+|f"[^"]*(SELECT|INSERT|UPDATE|DELETE))`)},
}

// Surface scans for mutating sinks under sub.
func Surface(root, sub string, limit int) ([]Sink, error) {
	var out []Sink
	err := walkCode(root, sub, func(p string, lines []string) {
		rel, _ := filepath.Rel(root, p)
		for i, line := range lines {
			for _, s := range sinks {
				if s.re.MatchString(line) {
					out = append(out, Sink{
						File: filepath.ToSlash(rel), Line: i + 1,
						Kind: s.kind, Match: strings.TrimSpace(line),
					})
					break
				}
			}
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].File+fmt.Sprint(out[i].Line) < out[j].File+fmt.Sprint(out[j].Line) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// SoftFuzz flags test functions that assert nothing or swallow errors.
type SoftFuzz struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Why  string `json:"why"`
}

var (
	testFileRe = regexp.MustCompile(`(test_.*\.py|.*_test\.(go|py)|.*\.(test|spec)\.(js|ts))$`)
	tryPass    = regexp.MustCompile(`except\b[^:]*:\s*pass|try\s*{[^}]*}\s*catch\s*\([^)]*\)\s*{\s*}`)
	assertNone = regexp.MustCompile(`def test_|func Test|it\(|test\(`)
	hasAssert  = regexp.MustCompile(`\b(assert|expect|require\.|t\.(Error|Fatal|Equal)|pytest\.raises|assertRaises|must)\b`)
)

// SoftFuzzScan finds test files and flags weak patterns: bare try/except
// pass, and test functions with no assertion call.
func SoftFuzzScan(root, sub string, limit int) ([]SoftFuzz, error) {
	var out []SoftFuzz
	err := walkCode(root, sub, func(p string, lines []string) {
		if !testFileRe.MatchString(filepath.Base(p)) {
			return
		}
		rel, _ := filepath.Rel(root, p)
		joined := strings.Join(lines, "\n")
		for _, m := range tryPass.FindAllStringIndex(joined, -1) {
			out = append(out, SoftFuzz{File: filepath.ToSlash(rel),
				Line: strings.Count(joined[:m[0]], "\n") + 1,
				Why:  "try/except pass swallows failures; soft fuzz without oracle"})
		}
		// test function bodies with no assert
		for i, line := range lines {
			if !assertNone.MatchString(line) {
				continue
			}
			// scan next 40 lines for an assertion
			found := false
			for j := i + 1; j < len(lines) && j < i+40; j++ {
				if assertNone.MatchString(lines[j]) {
					break
				}
				if hasAssert.MatchString(lines[j]) {
					found = true
					break
				}
			}
			if !found {
				out = append(out, SoftFuzz{File: filepath.ToSlash(rel), Line: i + 1,
					Why: "test with no assertion in body; cannot confirm or reject"})
			}
		}
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// LongFunc flags functions over a line threshold (deep nesting proxy via
// indentation width is folded into lines for simplicity).
type LongFunc struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Name  string `json:"name"`
	Lines int    `json:"lines"`
}

var funcStart = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:func(?:\s+\([^)]*\))?\s+|def\s+|function\s+)([A-Za-z_$][A-Za-z0-9_$]*)`)

// Complexity finds functions longer than maxLines (default 60).
// Length is measured to the next decl at the same or lower indent.
func Complexity(root, sub string, maxLines, limit int) ([]LongFunc, error) {
	if maxLines <= 0 {
		maxLines = 60
	}
	var out []LongFunc
	err := walkCode(root, sub, func(p string, lines []string) {
		rel, _ := filepath.Rel(root, p)
		for i, line := range lines {
			m := funcStart.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				l := lines[j]
				t := strings.TrimSpace(l)
				if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") {
					continue
				}
				li := len(l) - len(strings.TrimLeft(l, " \t"))
				if li <= indent {
					end = j
					break
				}
			}
			if n := end - i; n > maxLines {
				out = append(out, LongFunc{File: filepath.ToSlash(rel), Line: i + 1, Name: m[1], Lines: n})
			}
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Lines > out[j].Lines })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

var codeExts = map[string]bool{
	".py": true, ".go": true, ".js": true, ".ts": true, ".jsx": true,
	".tsx": true, ".vue": true, ".svelte": true, ".rs": true, ".java": true,
	".c": true, ".h": true, ".cpp": true, ".sh": true,
}

var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true,
	"vendor": true, ".venv": true, "__pycache__": true, "storage": true,
}

func walkCode(root, sub string, fn func(path string, lines []string)) error {
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return fmt.Errorf("path escapes root")
	}
	return filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
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
		fn(p, strings.Split(string(data), "\n"))
		return nil
	})
}

// deadDeclRes finds function/method declarations for dead-code
// candidates: a name with zero references anywhere else is suspect.
var deadDeclRes = map[string]*regexp.Regexp{
	".py":     regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
	".go":     regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?([a-z_][A-Za-z0-9_]*)\s*\(`),
	".ts":     regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([a-z_][A-Za-z0-9_]*)\s*\(`),
	".js":     regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([a-z_][A-Za-z0-9_]*)\s*\(`),
	".svelte": regexp.MustCompile(`^\s*function\s+([a-z_][A-Za-z0-9_]*)\s*\(`),
	".vue":    regexp.MustCompile(`^\s*function\s+([a-z_][A-Za-z0-9_]*)\s*\(`),
}

var deadSkip = regexp.MustCompile(`^(main|init|setup|teardown|setUp|tearDown|test_|__|on_|handle_)`)

// DeadCode lists declared functions that no scanned file references.
// Candidates only: dynamic dispatch, string-based invocation, and
// plugin hooks defeat static counting.
// DeadSym is a dead-code candidate.
type DeadSym struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Name string `json:"name"`
}

func DeadCode(root, sub string, limit int) ([]DeadSym, error) {
	base := filepath.Join(root, filepath.FromSlash(sub))
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	// decls under sub; references counted repo-wide so cross-module
	// callers keep a symbol alive
	srcs := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
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
		srcs[p] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// one pass: token -> set of files containing it. Candidate checks
	// become map lookups instead of per-name regex scans.
	var tokRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	live := map[string]map[string]bool{}
	for p, src := range srcs {
		for _, tok := range tokRe.FindAllString(src, -1) {
			m := live[tok]
			if m == nil {
				m = map[string]bool{}
				live[tok] = m
			}
			m[p] = true
		}
	}
	var out []DeadSym
	for p, src := range srcs {
		if !strings.HasPrefix(p, base) { // decls only under sub
			continue
		}
		re := deadDeclRes[strings.ToLower(filepath.Ext(p))]
		if re == nil || strings.Contains(filepath.Base(p), "_test") ||
			strings.Contains(filepath.Base(p), ".test.") {
			continue
		}
		for i, line := range strings.Split(src, "\n") {
			m := re.FindStringSubmatch(line)
			if m == nil || deadSkip.MatchString(m[1]) {
				continue
			}
			files := live[m[1]]
			// dead = appears only in its own file, and only once
			// (the declaration itself - no local callers either)
			if len(files) == 1 && files[p] &&
				len(regexp.MustCompile(`\b`+regexp.QuoteMeta(m[1])+`\b`).FindAllString(src, -1)) <= 1 {
				rel, _ := filepath.Rel(root, p)
				out = append(out, DeadSym{filepath.ToSlash(rel), i + 1, m[1]})
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

// Mutation is a suggested code mutant for testing a function's oracle.
type Mutation struct {
	Line     int    `json:"line"`
	Original string `json:"original"`
	Mutant   string `json:"mutant"`
	Rule     string `json:"rule"`
}

var mutantRules = []struct {
	rule string
	re   *regexp.Regexp
	repl string
}{
	{"boundary-off-by-one", regexp.MustCompile(`>=`), ">"},
	{"boundary-off-by-one", regexp.MustCompile(`<=`), "<"},
	{"negate-comparison", regexp.MustCompile(`==`), "!="},
	{"negate-comparison", regexp.MustCompile(`!=`), "=="},
	{"swap-relation", regexp.MustCompile(`\s<\s`), " > "},
	{"swap-relation", regexp.MustCompile(`\s>\s`), " < "},
	{"drop-negation", regexp.MustCompile(`\bnot\s+`), ""},
	{"swap-logic", regexp.MustCompile(`\band\b`), "or"},
	{"swap-logic", regexp.MustCompile(`\bor\b`), "and"},
	{"off-by-one", regexp.MustCompile(`\+\s*1\b`), "-1"},
	{"off-by-one", regexp.MustCompile(`-\s*1\b`), "+1"},
	{"empty-vs-none", regexp.MustCompile(`\bis not None\b`), "is None"},
	{"empty-vs-none", regexp.MustCompile(`\bis None\b`), "is not None"},
	{"empty-vs-nil", regexp.MustCompile(`==\s*nil\b`), "!= nil"},
	{"empty-vs-nil", regexp.MustCompile(`!=\s*nil\b`), "== nil"},
	{"empty-vs-zero", regexp.MustCompile(`==\s*0\b`), "== 1"},
	{"empty-vs-empty", regexp.MustCompile(`==\s*""`), `== "x"`},
	{"empty-vs-empty", regexp.MustCompile(`==\s*''`), `== 'x'`},
}

// MutationHints extracts a function's body and produces the mutants
// most likely to expose a weak oracle. Heuristic: one mutant per
// operator occurrence, capped.
func MutationHints(root, file, name string, max int) ([]Mutation, error) {
	p := filepath.Join(root, filepath.FromSlash(file))
	if !strings.HasPrefix(p, root) {
		return nil, fmt.Errorf("path escapes root")
	}
	data, err := os.ReadFile(p) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	// find the decl
	declRe := regexp.MustCompile(`^(\s*)(?:async\s+)?(?:def|func|function)\s+` + regexp.QuoteMeta(name) + `\s*[\(:]`)
	start := -1
	indent := 0
	for i, l := range lines {
		if m := declRe.FindStringSubmatch(l); m != nil {
			start = i
			indent = len(m[1])
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("function %q not found in %s", name, file)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		cur := len(l) - len(strings.TrimLeft(l, " \t"))
		if cur <= indent && strings.TrimSpace(l) != "" {
			end = i
			break
		}
	}
	var out []Mutation
	for i := start; i < end && len(out) < max; i++ {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		for _, r := range mutantRules {
			if r.re.MatchString(line) {
				out = append(out, Mutation{
					Line: i + 1, Original: strings.TrimSpace(line),
					Mutant: strings.TrimSpace(r.re.ReplaceAllString(line, r.repl)),
					Rule:   r.rule,
				})
				break // one mutant per line keeps hints readable
			}
		}
	}
	return out, nil
}
