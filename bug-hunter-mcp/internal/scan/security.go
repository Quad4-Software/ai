// SPDX-License-Identifier: 0BSD
// Security-oriented scanners: secrets, injection sinks, crypto misuse,
// concurrency heuristics, naive taint pairs, supply-chain indicators,
// git-history leaks, TODO density, and MCP tool-poisoning hints.
// All read-only, regex-based heuristics. Findings are candidates, not
// verdicts; confirm each with the method docs.
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

// Finding is a generic pattern hit with severity and rule provenance.
type Finding struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Rule       string `json:"rule"`
	Severity   string `json:"severity"` // crit, high, med, low
	Match      string `json:"match"`
	Confidence string `json:"confidence,omitempty"`
}

type rule struct {
	name string
	sev  string
	re   *regexp.Regexp
}

// ruleScan runs a flat rule list over code lines under sub.
func ruleScan(root, sub string, rules []rule, limit int, redact bool) ([]Finding, error) {
	return ruleScanWalk(root, sub, rules, limit, redact, walkCode)
}

func ruleScanWalk(root, sub string, rules []rule, limit int, redact bool,
	walk func(string, string, func(string, []string)) error) ([]Finding, error) {
	var out []Finding
	err := walk(root, sub, func(p string, lines []string) {
		rel, _ := filepath.Rel(root, p)
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") && !strings.Contains(trim, "PRIVATE KEY") {
				continue
			}
			for _, r := range rules {
				if r.re.MatchString(line) {
					m := trim
					if redact {
						m = redactLine(m)
					}
					out = append(out, Finding{
						File: filepath.ToSlash(rel), Line: i + 1,
						Rule: r.name, Severity: r.sev, Match: m,
					})
					break
				}
			}
		}
	})
	sortFindings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

func sortFindings(out []Finding) {
	rank := map[string]int{"crit": 0, "high": 1, "med": 2, "low": 3}
	sort.SliceStable(out, func(i, j int) bool {
		if rank[out[i].Severity] != rank[out[j].Severity] {
			return rank[out[i].Severity] < rank[out[j].Severity]
		}
		return out[i].File+fmt.Sprint(out[i].Line) < out[j].File+fmt.Sprint(out[j].Line)
	})
}

var secretValueRe = regexp.MustCompile(`(?i)(password|passwd|api[_-]?key|secret|token|private[_-]?key|credential)\s*[:=]\s*["']([^"']{8,})["']`)
var placeholderRe = regexp.MustCompile(`(?i)(example|sample|dummy|test|changeme|change-me|placeholder|<[^>]+>|\$\{|\{\{|xxx|your[_-])`)

// redactLine shortens obvious secret literals so tool output never
// carries key material back to the caller.
func redactLine(line string) string {
	if m := secretValueRe.FindStringSubmatch(line); m != nil {
		v := m[2]
		show := v
		if len(v) > 6 {
			show = v[:4] + "...REDACTED"
		}
		line = strings.Replace(line, v, show, 1)
	}
	for _, re := range secretTokenRes {
		line = re.ReplaceAllStringFunc(line, func(s string) string {
			if len(s) > 10 {
				return s[:6] + "...REDACTED"
			}
			return s
		})
	}
	if len(line) > 160 {
		line = line[:160] + "..."
	}
	return line
}

var secretTokenRes = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`ghp_[A-Za-z0-9]{30,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{30,}`),
	regexp.MustCompile(`sk-(?:ant-)?[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`),
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

var secretRules = []rule{
	{"token-shaped-literal", "high", secretTokenRes[0]},
	{"token-shaped-literal", "high", secretTokenRes[1]},
	{"token-shaped-literal", "high", secretTokenRes[2]},
	{"token-shaped-literal", "high", secretTokenRes[3]},
	{"token-shaped-literal", "high", secretTokenRes[4]},
	{"token-shaped-literal", "high", secretTokenRes[5]},
	{"jwt-literal", "med", secretTokenRes[6]},
	{"private-key-block", "crit", secretTokenRes[7]},
	{"hardcoded-secret", "high", secretValueRe},
}

var textExts = map[string]bool{
	".env": true, ".json": true, ".yaml": true, ".yml": true,
	".toml": true, ".ini": true, ".cfg": true, ".conf": true,
	".pem": true, ".key": true, ".tf": true, ".txt": true,
	".sh": true, ".py": true, ".go": true, ".js": true, ".ts": true,
}

// walkText visits files likely to hold secrets, wider than codeExts.
func walkText(root, sub string, fn func(path string, lines []string)) error {
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
		name := d.Name()
		if !textExts[strings.ToLower(filepath.Ext(name))] && !strings.HasPrefix(name, ".env") {
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

// Secrets finds credential-shaped literals. Values are redacted in
// output. Placeholder-looking assignments get low confidence.
func Secrets(root, sub string, limit int) ([]Finding, error) {
	out, err := ruleScanWalk(root, sub, secretRules, limit*4, true, walkText)
	if err != nil {
		return nil, err
	}
	var filtered []Finding
	for _, f := range out {
		if f.Rule == "hardcoded-secret" {
			if m := secretValueRe.FindStringSubmatch(f.Match); m == nil {
				// value already redacted; recheck original is impossible,
				// so keep unless the visible text screams placeholder
			}
			if placeholderRe.MatchString(f.Match) {
				f.Confidence = "low"
			} else {
				f.Confidence = "high"
			}
		}
		// .env and config files: raise confidence
		base := filepath.Base(f.File)
		if strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") {
			f.Confidence = "high"
		}
		filtered = append(filtered, f)
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

var injectionRules = []rule{
	// Python
	{"py-shell-true", "high", regexp.MustCompile(`subprocess\.(call|run|Popen|check_output|check_call)\s*\([^)]*shell\s*=\s*True`)},
	{"py-os-system", "high", regexp.MustCompile(`\bos\.(system|popen[0-9]?)\s*\(`)},
	{"py-eval-exec", "med", regexp.MustCompile(`(^|[^.\w])(eval|exec)\s*\(`)},
	{"py-unsafe-deserialize", "crit", regexp.MustCompile(`\b(pickle|cPickle|dill|_pickle|shelve)\.(load|loads|open|Unpickler)\b|shelve\.open\s*\(`)},
	{"py-yaml-load", "high", regexp.MustCompile(`\byaml\.load\s*\(`)},
	{"py-fstring-sql", "high", regexp.MustCompile(`\.(execute|executemany|executescript|raw)\s*\(\s*(f["']|["'][^"']*(\{|\%s)[^"']*["']\s*%)`)},
	{"py-sql-concat", "high", regexp.MustCompile(`\.(execute|executemany)\s*\([^)]*(\+\s*\w|\.\s*format\s*\()`)},
	{"py-ssti", "high", regexp.MustCompile(`(render_template_string|jinja2\.Template|Template)\s*\([^)]*(\+|%|\.format|f["'])`)},
	{"py-archive-extract", "high", regexp.MustCompile(`\.extractall\s*\(`)},
	{"py-xml-untrusted", "med", regexp.MustCompile(`\b(lxml\.etree\.(parse|fromstring|XML)|xml\.(etree|sax|dom)\.|xml\.parsers\.)\s*\(`)},
	{"py-ssrf", "med", regexp.MustCompile(`(requests|httpx)\.\w+\s*\(\s*(f["']|[^)]*\+\s*\w)|urllib\.request\.urlopen\s*\([^)]*(\+|f["']|format)`)},
	{"py-markup-unsafe", "med", regexp.MustCompile(`\bMarkup\s*\(`)},
	{"py-assert-authz", "med", regexp.MustCompile(`\bassert\s+.*(auth|perm|role|admin|token|session|owner|login)`)},
	{"py-late-closure", "low", regexp.MustCompile(`for\s+\w+\s+in\s+.*:\s*$`)},
	// Go
	{"go-shell-c", "crit", regexp.MustCompile(`exec\.Command\s*\(\s*"(sh|bash|cmd|powershell)"\s*,\s*"-[cC]"`)},
	{"go-exec-concat", "high", regexp.MustCompile(`exec\.Command\s*\([^)]*\+|exec\.Command\s*\(\s*fmt\.Sprintf`)},
	{"go-sqli", "high", regexp.MustCompile(`\.(Query|QueryRow|QueryContext|Exec|ExecContext|Prepare)\s*\(\s*(fmt\.Sprintf|"[^"]*"\s*\+|\w+\s*\+|ctx\s*,\s*fmt\.Sprintf)`)},
	{"go-ssrf", "med", regexp.MustCompile(`http\.(Get|Post|PostForm|Head)\s*\([^)]*(\+|fmt\.Sprintf)|\.Do\s*\(.*Sprintf`)},
	{"go-text-template-html", "high", regexp.MustCompile(`"text/template"|template\.(HTML|JS|URL)\s*\(`)},
	{"go-yaml-unmarshal", "low", regexp.MustCompile(`\byaml\.Unmarshal\s*\(`)},
	{"go-open-redirect", "med", regexp.MustCompile(`Header\(\)\.Set\s*\(\s*"Location"\s*,|http\.Redirect\s*\([^)]*(\+|Sprintf|r\.URL)`)},
	{"go-unsafe-pkg", "high", regexp.MustCompile(`"unsafe"|unsafe\.(Pointer|Slice|Sizeof)`)},
	{"go-fmt-user-input", "med", regexp.MustCompile(`fmt\.Sprintf\s*\([^)]*(SELECT|INSERT|UPDATE|DELETE|DROP)`)},
	// JS/TS
	{"js-child-exec", "high", regexp.MustCompile(`child_process\.(exec|execSync|spawn)\s*\(`)},
	{"js-eval", "med", regexp.MustCompile(`\beval\s*\(|new Function\s*\(`)},
	{"js-dom-sink", "med", regexp.MustCompile(`innerHTML\s*=|outerHTML\s*=|document\.write|dangerouslySetInnerHTML|\.insertAdjacentHTML`)},
	{"js-exec-concat", "high", regexp.MustCompile(`(execSync|exec|spawnSync)\s*\([^)]*\+`)},
	{"js-proto-key", "high", regexp.MustCompile(`__proto__|\.constructor\s*\[|prototype\]\s*=`)},
	// Generic
	{"curl-pipe-sh", "high", regexp.MustCompile(`(curl|wget)\s+[^|]*\|\s*(sudo\s+)?(bash|sh|python|python3)\b`)},
	{"ldap-inject", "med", regexp.MustCompile(`ldap.*(\+|Sprintf|format).*(\w+)\s*\)`)},
}

// Injection finds dangerous sinks that commonly receive user input.
func Injection(root, sub string, limit int) ([]Finding, error) {
	return ruleScan(root, sub, injectionRules, limit, false)
}

var cryptoRules = []rule{
	{"weak-hash-import", "med", regexp.MustCompile(`"crypto/(md5|sha1|des|rc4)"|hashlib\.(md5|sha1)\s*\(|hashlib\.new\s*\(\s*["'](md5|sha1)|createHash\(["'](md5|sha1)`)},
	{"weak-cipher", "high", regexp.MustCompile(`"crypto/des"|"crypto/rc4"|des\.New|rc4\.New|Crypto\.Cipher\.(DES|ARC2|ARC4|Blowfish)`)},
	{"insecure-rand-go", "high", regexp.MustCompile(`"math/rand"`)},
	{"insecure-rand-py", "high", regexp.MustCompile(`\brandom\.(choice|choices|randint|randrange|getrandbits|randbytes|sample|shuffle)\s*\(`)},
	{"insecure-rand-js", "high", regexp.MustCompile(`Math\.random\s*\(`)},
	{"tls-skip-verify", "high", regexp.MustCompile(`InsecureSkipVerify\s*:\s*true|verify\s*=\s*False|rejectUnauthorized\s*:\s*false|check_hostname\s*=\s*False|CERT_NONE`)},
	{"tls-old-version", "high", regexp.MustCompile(`MinVersion\s*:\s*tls\.Version(SSL30|TLS10|TLS11)|PROTOCOL_TLSv1(_1)?\b|ssl\.PROTOCOL_SSLv`)},
	{"jwt-alg-confusion", "crit", regexp.MustCompile(`algorithms\s*[=:]\s*\[[^\]]*none|jwt\.decode\s*\([^)]*verify\s*=\s*False`)},
	{"jwt-no-alg-pin", "med", regexp.MustCompile(`jwt\.decode\s*\(`)},
	{"timing-compare-secret", "high", regexp.MustCompile(`(?i)\b(password|secret|token|signature|hmac|digest|apikey|api_key)\w*\s*==\s*[^=]`)},
	{"ecb-mode", "high", regexp.MustCompile(`MODE_ECB|NewECBEncrypter|NewECBDecrypter`)},
	{"hardcoded-iv-salt", "med", regexp.MustCompile(`(?i)(iv|salt|nonce)\s*[:=]\s*["'][0-9a-fA-F]{8,}["']`)},
}

// Crypto finds weak algorithms, disabled verification, insecure random
// for secrets, and secret comparisons that leak via timing.
func Crypto(root, sub string, limit int) ([]Finding, error) {
	return ruleScan(root, sub, cryptoRules, limit, true)
}

// --- concurrency heuristics (Go-focused, windowed like TOCTOU) ---

var (
	goForLoop    = regexp.MustCompile(`^\s*for\s+`)
	goRangeDecl  = regexp.MustCompile(`for\s+(?:\w+\s*,\s*)?(\w+)\s*:?=\s*range\s+`)
	goForIdx     = regexp.MustCompile(`for\s+(\w+)\s*:=`)
	goFuncRe     = regexp.MustCompile(`\bgo\s+(func\b|\w+\s*\()`)
	goDeferRe    = regexp.MustCompile(`\bdefer\s+`)
	wgAddRe      = regexp.MustCompile(`\b\w+\.Add\s*\(`)
	respAssignRe = regexp.MustCompile(`(\w+)\s*,\s*\w*\s*:?=\s*(http\.(Get|Post|Head|PostForm)|\w+\.Do|client\.Do)\s*\(`)
	closeChRe    = regexp.MustCompile(`\bclose\s*\(\s*(\w+)\s*\)`)
	sendChRe     = regexp.MustCompile(`(\w+)\s*<-\s*`)
	pkgVarMapRe  = regexp.MustCompile(`^var\s+(\w+)\s+(?:=\s*(?:make\()?)?(map\[|\[\])`)
	timeSleepRe  = regexp.MustCompile(`time\.Sleep\s*\(`)
)

// Concurrency flags race and lifecycle heuristics: loop-var capture,
// defer in loops, WaitGroup.Add inside goroutines, unclosed response
// bodies, send-after-close risk, package-level shared maps.
func Concurrency(root, sub string, limit int) ([]Finding, error) {
	var out []Finding
	err := walkCode(root, sub, func(p string, lines []string) {
		if filepath.Ext(p) != ".go" {
			return
		}
		rel, _ := filepath.Rel(root, p)
		joined := strings.Join(lines, "\n")
		hasGo := strings.Contains(joined, "go func") || strings.Contains(joined, "go ")
		add := func(line int, r, sev, match string) {
			out = append(out, Finding{File: filepath.ToSlash(rel), Line: line,
				Rule: r, Severity: sev, Match: strings.TrimSpace(match)})
		}
		// windowed scans
		const win = 20
		for i, line := range lines {
			// loop variable capture: for v := range ... go func() { use v }
			vars := []string{}
			if m := goRangeDecl.FindStringSubmatch(line); m != nil {
				vars = append(vars, m[1])
			}
			if m := goForIdx.FindStringSubmatch(line); m != nil {
				vars = append(vars, m[1])
			}
			if len(vars) > 0 {
				for j := i + 1; j < len(lines) && j <= i+win; j++ {
					l := lines[j]
					if goForLoop.MatchString(l) && len(l)-len(strings.TrimLeft(l, " \t")) <= len(line)-len(strings.TrimLeft(line, " \t")) {
						break // left the loop body
					}
					if goFuncRe.MatchString(l) {
						for _, v := range vars {
							// skip when the loop var is passed in as a param
							sig := regexp.MustCompile(`func\s*\([^)]*\b` + regexp.QuoteMeta(v) + `\b`)
							if sig.MatchString(l) {
								continue
							}
							vre := regexp.MustCompile(`\b` + regexp.QuoteMeta(v) + `\b`)
							for k := j; k < len(lines) && k <= j+10; k++ {
								if vre.MatchString(lines[k]) {
									add(k+1, "loop-var-capture", "high", lines[k])
									break
								}
							}
						}
						break
					}
				}
			}
			// defer inside for body
			if goForLoop.MatchString(line) {
				for j := i + 1; j < len(lines) && j <= i+30; j++ {
					l := lines[j]
					if goDeferRe.MatchString(l) {
						add(j+1, "defer-in-loop", "med", l)
						break
					}
					if goForLoop.MatchString(l) || (strings.TrimSpace(l) == "}" && len(l)-len(strings.TrimLeft(l, " \t")) <= len(line)-len(strings.TrimLeft(line, " \t"))) {
						break
					}
				}
			}
			// wg.Add inside goroutine (Add must precede go)
			if goFuncRe.MatchString(line) {
				for j := i; j < len(lines) && j <= i+8; j++ {
					if wgAddRe.MatchString(lines[j]) {
						add(j+1, "waitgroup-add-in-goroutine", "med", lines[j])
						break
					}
				}
			}
			// response body not closed
			if m := respAssignRe.FindStringSubmatch(line); m != nil {
				name := m[1]
				closed := false
				bodyRe := regexp.MustCompile(regexp.QuoteMeta(name) + `\.Body\.Close|defer\s+` + regexp.QuoteMeta(name) + `\.Body`)
				for j := i + 1; j < len(lines) && j <= i+25; j++ {
					if bodyRe.MatchString(lines[j]) {
						closed = true
						break
					}
				}
				if !closed {
					add(i+1, "resp-body-not-closed", "med", line)
				}
			}
			// close(ch) then later send on ch
			if m := closeChRe.FindStringSubmatch(line); m != nil {
				sre := regexp.MustCompile(`\b` + regexp.QuoteMeta(m[1]) + `\s*<-`)
				for j := i + 1; j < len(lines) && j <= i+win; j++ {
					if sre.MatchString(lines[j]) {
						add(j+1, "send-after-close", "high", lines[j])
						break
					}
				}
			}
		}
		// package-level shared mutable state in files that spawn goroutines
		if hasGo {
			for i, line := range lines {
				if pkgVarMapRe.MatchString(line) && !strings.Contains(joined, "sync.") {
					add(i+1, "shared-state-no-sync", "med", line)
				}
			}
		}
		_ = sendChRe // reserved
		_ = timeSleepRe
	})
	sortFindings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// --- naive taint: source-assigned identifiers reaching sinks ---

var (
	taintSourceRe = regexp.MustCompile(`(\w+)\s*(?::=|=)\s*.*` +
		`(r\.FormValue|r\.PostFormValue|r\.URL\.Query|r\.Form\.|r\.Body|r\.Header\.Get|` +
		`os\.Args|os\.Getenv|flag\.(String|Int|Arg)|c\.(Param|Query|DefaultQuery|PostForm|Bind)|` +
		`request\.(args|form|values|GET|POST|data|json|files|headers|cookies)|` +
		`req\.(query|body|params|headers)|input\(|sys\.argv|os\.environ|process\.argv|` +
		`ctx\.(Query|Param|Body)|json\.NewDecoder|ioutil\.ReadAll|bufio\.NewReader)`)
	taintSinkRe = regexp.MustCompile(`(exec\.Command|os\.system|subprocess\.|\.execute|\.raw|os\.Open|os\.Create|open\(|` +
		`http\.Get|requests\.|urlopen|render_template|Template|InnerHTML|innerHTML|fmt\.Sprintf|filepath\.Join|os\.path\.join)`)
)

// Taint finds lines where an identifier assigned from a request/argv/
// env source reaches a dangerous sink in the same file within 30 lines.
// Intra-file heuristic only; cross-function flow needs real analysis.
func Taint(root, sub string, limit int) ([]Finding, error) {
	var out []Finding
	err := walkCode(root, sub, func(p string, lines []string) {
		rel, _ := filepath.Rel(root, p)
		tainted := map[string]int{} // ident -> assignment line
		for i, line := range lines {
			if m := taintSourceRe.FindStringSubmatch(line); m != nil {
				tainted[m[1]] = i
			}
		}
		if len(tainted) == 0 {
			return
		}
		for i, line := range lines {
			if !taintSinkRe.MatchString(line) {
				continue
			}
			for name, src := range tainted {
				if i < src || i-src > 30 {
					continue
				}
				re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
				if re.MatchString(line) {
					out = append(out, Finding{
						File: filepath.ToSlash(rel), Line: i + 1,
						Rule: "tainted-input", Severity: "high",
						Match: fmt.Sprintf("%s (tainted %q from line %d)",
							strings.TrimSpace(line), name, src+1),
					})
					break
				}
			}
		}
	})
	sortFindings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// --- supply chain ---

// SCFinding is a supply-chain indicator in a manifest, workflow or script.
type SCFinding struct {
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

var (
	scLifecycle  = regexp.MustCompile(`"(preinstall|postinstall|install|prepare)"\s*:`)
	scLooseSpec  = regexp.MustCompile(`^\s*["'][\w@/.-]+["']\s*:\s*["'](\*|latest|\*|git\+|https?://|file:)`)
	scSetupRisk  = regexp.MustCompile(`base64|b64decode|exec\s*\(|eval\s*\(|compile\s*\(|__import__|urllib|requests\.(get|post)`)
	scReqLoose   = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*$|^\s*([A-Za-z0-9_.-]+)\s*[<>]=?|git\+|https?://`)
	scGoReplace  = regexp.MustCompile(`^\s*replace\s|=>`)
	scGHUses     = regexp.MustCompile(`uses:\s*([\w./-]+)@([\w.-]+)`)
	scGHFullSHA  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	scPRT        = regexp.MustCompile(`pull_request_target|workflow_run|repository_dispatch`)
	scSecrets    = regexp.MustCompile(`secrets\.`)
	scPipeSh     = regexp.MustCompile(`(curl|wget)\s+[^|]*\|\s*(sudo\s+)?(bash|sh)\b`)
	scPipInstall = regexp.MustCompile(`pip\s+install\s+[^-r]`)
	scCheckoutPR = regexp.MustCompile(`github\.event\.pull_request\.head\.(sha|ref)`)
)

var manifestNames = map[string]bool{
	"package.json": true, "setup.py": true, "setup.cfg": true,
	"pyproject.toml": true, "go.mod": true, "go.sum": false,
	"requirements.txt": true, "Makefile": true, "Dockerfile": true,
}

// SupplyChain scans dependency manifests, CI workflows and install
// scripts for supply-chain risk indicators: lifecycle scripts, loose
// version specs, unpinned actions, fork-triggered secret access,
// pipe-to-shell installs, replace directives.
func SupplyChain(root string, limit int) ([]SCFinding, error) {
	var out []SCFinding
	add := func(f string, line int, r, sev, d string) {
		out = append(out, SCFinding{File: f, Line: line, Rule: r, Severity: sev, Detail: d})
	}
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
		rel, _ := filepath.Rel(root, p)
		rels := filepath.ToSlash(rel)
		base := d.Name()
		isWorkflow := strings.HasPrefix(rels, ".github/workflows/") &&
			(strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml"))
		isShell := strings.HasSuffix(base, ".sh")
		isReq := strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt")
		if !manifestNames[base] && !isWorkflow && !isShell && !isReq {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		fileHasPRT := false
		fileHasSecrets := false
		for i, line := range lines {
			t := strings.TrimSpace(line)
			switch {
			case base == "package.json":
				if scLifecycle.MatchString(line) {
					add(rels, i+1, "npm-lifecycle-script", "high", t)
				}
				if scLooseSpec.MatchString(line) {
					add(rels, i+1, "loose-dep-spec", "med", t)
				}
			case base == "setup.py" || base == "setup.cfg":
				if scSetupRisk.MatchString(line) {
					add(rels, i+1, "setup-code-exec", "high", t)
				}
			case isReq:
				if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "-") {
					continue
				}
				if !strings.Contains(t, "==") {
					sev := "med"
					if strings.Contains(t, "git+") || strings.Contains(t, "http") {
						sev = "high"
					}
					add(rels, i+1, "unpinned-pip-dep", sev, t)
				}
			case base == "go.mod":
				if scGoReplace.MatchString(line) {
					add(rels, i+1, "go-mod-replace", "med", t)
				}
			case isWorkflow || isShell:
				if scPipeSh.MatchString(line) {
					add(rels, i+1, "pipe-to-shell", "high", t)
				}
				if isWorkflow {
					if m := scGHUses.FindStringSubmatch(line); m != nil {
						if !scGHFullSHA.MatchString(m[2]) {
							add(rels, i+1, "unpinned-action", "high", t)
						}
					}
					if scPRT.MatchString(line) {
						fileHasPRT = true
						add(rels, i+1, "fork-trigger", "med", t)
					}
					if scSecrets.MatchString(line) {
						fileHasSecrets = true
					}
					if scCheckoutPR.MatchString(line) {
						add(rels, i+1, "checkout-pr-head", "high", t)
					}
				}
			case base == "Makefile":
				if scPipeSh.MatchString(line) {
					add(rels, i+1, "pipe-to-shell", "high", t)
				}
			case base == "Dockerfile":
				if scPipeSh.MatchString(line) || scPipInstall.MatchString(line) {
					add(rels, i+1, "docker-install-risk", "med", t)
				}
			}
		}
		if fileHasPRT && fileHasSecrets {
			add(rels, 0, "fork-pr-secrets", "crit",
				"workflow runs on fork triggers and reads secrets: poisoned pipeline risk")
		}
		return nil
	})
	// missing lockfiles
	for _, pair := range [][2]string{
		{"go.mod", "go.sum"},
		{"package.json", "package-lock.json"},
	} {
		if _, err := os.Stat(filepath.Join(root, pair[0])); err == nil {
			if _, err := os.Stat(filepath.Join(root, pair[1])); err != nil {
				add(pair[0], 0, "missing-lockfile", "med",
					pair[1]+" absent: dependencies are not pinned")
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		rank := map[string]int{"crit": 0, "high": 1, "med": 2, "low": 3}
		if rank[out[i].Severity] != rank[out[j].Severity] {
			return rank[out[i].Severity] < rank[out[j].Severity]
		}
		return out[i].File < out[j].File
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// GitLeak is a secret-shaped line added in git history.
type GitLeak struct {
	Commit string `json:"commit"`
	File   string `json:"file"`
	Match  string `json:"match"` // redacted
	Rule   string `json:"rule"`
}

// GitSecrets scans added lines in recent history for credential-shaped
// literals. Bounded to the last n commits. Output is redacted.
func GitSecrets(ctx context.Context, root string, n int) ([]GitLeak, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", // #nosec G204 -- fixed argv
		"-p", "--format=@@@%h", "-n", strconv.Itoa(n))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log -p: %w", err)
	}
	var leaks []GitLeak
	var commit, file string
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "@@@") {
			commit = line[3:]
			continue
		}
		if strings.HasPrefix(line, "+++ b/") {
			file = line[6:]
			continue
		}
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		body := line[1:]
		for _, re := range secretTokenRes {
			if re.MatchString(body) {
				leaks = append(leaks, GitLeak{Commit: commit, File: file,
					Rule: "token-shaped-literal", Match: redactLine(strings.TrimSpace(body))})
				break
			}
		}
		if m := secretValueRe.FindStringSubmatch(body); m != nil && !placeholderRe.MatchString(m[2]) {
			leaks = append(leaks, GitLeak{Commit: commit, File: file,
				Rule: "hardcoded-secret", Match: redactLine(strings.TrimSpace(body))})
		}
	}
	return leaks, sc.Err()
}

// --- TODO/FIXME density ---

var markerRe = regexp.MustCompile(`\b(TODO|FIXME|HACK|XXX|BUG|SAFETY|SECURITY|TEMP|WORKAROUND)\b`)

// MarkerHit is one annotated-marker occurrence.
type MarkerHit struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Markers finds TODO/FIXME/HACK/SECURITY comments, which cluster where
// developers already suspected problems. Sorted by kind then file.
func Markers(root, sub string, limit int) ([]MarkerHit, error) {
	var out []MarkerHit
	err := walkCode(root, sub, func(p string, lines []string) {
		rel, _ := filepath.Rel(root, p)
		for i, line := range lines {
			m := markerRe.FindString(line)
			if m == "" {
				continue
			}
			t := strings.TrimSpace(line)
			if len(t) > 200 {
				t = t[:200]
			}
			out = append(out, MarkerHit{File: filepath.ToSlash(rel), Line: i + 1, Kind: m, Text: t})
		}
	})
	rank := map[string]int{"SECURITY": 0, "SAFETY": 1, "FIXME": 2, "BUG": 3, "HACK": 4, "XXX": 5}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := rank[out[i].Kind], rank[out[j].Kind]
		if ri != rj {
			if ri == 0 || rj == 0 || ri == 1 || rj == 1 {
				return ri < rj
			}
			return out[i].File < out[j].File
		}
		return out[i].File < out[j].File
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// --- MCP / agent-tool audit ---

var (
	mcpDescRe    = regexp.MustCompile(`(?i)"?(description|instructions|_meta)"?\s*[:=]`)
	poisonStrong = regexp.MustCompile(`(?i)(ignore (all |any |previous )?instructions|forget (your|previous)|do not (tell|inform|reveal)|system prompt|exfiltrat|secretly|covertly|<IMPORTANT>|IMPORTANT:.*(before|first|also|then))`)
	poisonWeak   = regexp.MustCompile(`(?i)(always|never|must|required to).{0,80}(send|read|include|attach|fetch|upload|forward|exfiltrat|token|key|secret|password|ssh|env)`)
	mcpExecRe    = regexp.MustCompile(`exec\.Command|subprocess\.|child_process|os\.system|spawn`)
)

// MCPAudit hunts agent-tool security issues: tool descriptions that
// carry injected instructions (tool poisoning), descriptions that
// pressure the model into data movement, and exec sinks inside tool
// handlers that take caller-controlled arguments.
func MCPAudit(root, sub string, limit int) ([]Finding, error) {
	var out []Finding
	err := walkCode(root, sub, func(p string, lines []string) {
		rel, _ := filepath.Rel(root, p)
		for i, line := range lines {
			if mcpDescRe.MatchString(line) {
				if poisonStrong.MatchString(line) {
					out = append(out, Finding{File: filepath.ToSlash(rel), Line: i + 1,
						Rule: "tool-poisoning", Severity: "crit",
						Match: strings.TrimSpace(line)})
					continue
				}
				if poisonWeak.MatchString(line) && len(line) > 200 {
					out = append(out, Finding{File: filepath.ToSlash(rel), Line: i + 1,
						Rule: "suspicious-tool-desc", Severity: "med",
						Match: strings.TrimSpace(line)})
				}
			}
			if mcpExecRe.MatchString(line) && (strings.Contains(line, "args") ||
				strings.Contains(line, "arg") || strings.Contains(line, "+") ||
				strings.Contains(line, "Sprintf") || strings.Contains(line, "f\"")) {
				out = append(out, Finding{File: filepath.ToSlash(rel), Line: i + 1,
					Rule: "mcp-exec-sink", Severity: "high",
					Match: strings.TrimSpace(line)})
			}
		}
	})
	sortFindings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}
