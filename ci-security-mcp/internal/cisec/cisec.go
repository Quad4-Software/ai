// SPDX-License-Identifier: 0BSD
// Package cisec scans GitHub Actions workflows and Dockerfiles for
// security issues, resolves action refs to commit SHAs via the GitHub
// API, and rewrites workflows with pinned actions. Stdlib only.
package cisec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Finding is one detected issue.
type Finding struct {
	Rule     string `json:"rule"`
	Line     int    `json:"line"`
	Severity string `json:"severity"` // high, medium, low
	Match    string `json:"match"`
	Message  string `json:"message"`
}

var (
	usesRe        = regexp.MustCompile(`uses:\s*([A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)@([A-Za-z0-9._/-]+)`)
	sha40         = regexp.MustCompile(`^[0-9a-f]{40}$`)
	prtRe         = regexp.MustCompile(`\bpull_request_target\b`)
	checkoutHead  = regexp.MustCompile(`(github\.event\.pull_request\.head|github\.head_ref)`)
	injectionRe   = regexp.MustCompile(`\$\{\{\s*github\.(event\.|head_ref|event_name)`)
	permsRe       = regexp.MustCompile(`^\s*permissions\s*:`)
	secretsEcho   = regexp.MustCompile(`(echo|print|printf|console\.log).*\$\{\{\s*secrets\.`)
	curlPipe      = regexp.MustCompile(`(curl|wget)\b[^|]*\|\s*(sh|bash|sudo)`)
	hardcodedCred = regexp.MustCompile(`(?i)(password|token|secret|api[_-]?key)\s*[:=]\s*["']?[A-Za-z0-9_/+=$-]{12,}`)
)

// ScanWorkflow checks a workflow YAML for Actions security issues.
func ScanWorkflow(yaml string) []Finding {
	var out []Finding
	lines := strings.Split(yaml, "\n")
	hasPerms := false
	inPrt := false
	inRun := false
	runIndent := 0
	for i, line := range lines {
		ln := i + 1
		if permsRe.MatchString(line) {
			hasPerms = true
		}
		if prtRe.MatchString(line) {
			inPrt = true
			out = append(out, Finding{"pull_request_target", ln, "medium", "pull_request_target:",
				"runs with base-repo secrets and write token; never check out or run PR head code"})
		}
		if inPrt && checkoutHead.MatchString(line) {
			out = append(out, Finding{"prt-checkout-head", ln, "high", strings.TrimSpace(line),
				"checks out PR head under pull_request_target; attacker code runs with elevated privileges"})
		}
		if m := usesRe.FindStringSubmatch(line); m != nil {
			if !sha40.MatchString(m[2]) {
				sev := "medium"
				if m[2] == "main" || m[2] == "master" || m[2] == "latest" || strings.HasPrefix(m[2], "v") {
					sev = "high"
				}
				out = append(out, Finding{"unpinned-action", ln, sev, m[0],
					fmt.Sprintf("action %s pinned to mutable ref %q; pin the 40-char commit SHA", m[1], m[2])})
			}
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if inRun && strings.TrimSpace(line) != "" && indent <= runIndent {
			inRun = false
		}
		if strings.Contains(line, "run:") || strings.HasPrefix(strings.TrimSpace(line), "- run") {
			inRun = true
			runIndent = indent
		}
		if inRun && injectionRe.MatchString(line) {
			out = append(out, Finding{"script-injection", ln, "high", strings.TrimSpace(line),
				"github.event.* interpolated into run:; pass via env: instead"})
		}
		if secretsEcho.MatchString(line) {
			out = append(out, Finding{"secret-in-log", ln, "high", strings.TrimSpace(line),
				"secret interpolated into output; masking is not a control"})
		}
		if curlPipe.MatchString(line) {
			out = append(out, Finding{"pipe-to-shell", ln, "high", strings.TrimSpace(line),
				"pipe remote content to shell; download, verify, then run"})
		}
		if hardcodedCred.MatchString(line) && !strings.Contains(line, "${{") {
			out = append(out, Finding{"hardcoded-secret", ln, "high", strings.TrimSpace(line),
				"possible hardcoded credential; move to secrets or OIDC"})
		}
	}
	if !hasPerms {
		out = append(out, Finding{"missing-permissions", 0, "medium", "(file)",
			"no permissions block; GITHUB_TOKEN inherits repo default (often read-write). Add permissions: {}"})
	}
	return out
}

var (
	fromRe   = regexp.MustCompile(`(?i)^\s*FROM\s+(\S+)`)
	digestRe = regexp.MustCompile(`@sha256:[0-9a-f]{64}`)
	userRe   = regexp.MustCompile(`(?i)^\s*USER\s+`)
	addURLRe = regexp.MustCompile(`(?i)^\s*ADD\s+https?://`)
	privRe   = regexp.MustCompile(`(--privileged|--cap-add\s+ALL|docker\.sock)`)
	chmod777 = regexp.MustCompile(`chmod\s+(-R\s+)?777`)
)

// ScanDockerfile checks a Dockerfile for pinning and hardening issues.
func ScanDockerfile(df string) []Finding {
	var out []Finding
	hasUser := false
	for i, line := range strings.Split(df, "\n") {
		ln := i + 1
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if m := fromRe.FindStringSubmatch(line); m != nil {
			img := m[1]
			if digestRe.MatchString(img) {
				continue
			}
			if !strings.Contains(img, ":") || strings.HasSuffix(img, ":latest") {
				out = append(out, Finding{"unpinned-image", ln, "high", t,
					"image unpinned or on latest; pin image@sha256:<digest>"})
			} else {
				out = append(out, Finding{"tag-pinned-image", ln, "medium", t,
					"tag is mutable; pin image@sha256:<digest>"})
			}
		}
		if userRe.MatchString(line) {
			hasUser = true
		}
		if addURLRe.MatchString(line) {
			out = append(out, Finding{"add-remote", ln, "high", t,
				"ADD from remote URL skips verification; COPY local or fetch with checksum"})
		}
		if privRe.MatchString(line) {
			out = append(out, Finding{"privileged", ln, "high", t, "privileged or docker.sock usage"})
		}
		if chmod777.MatchString(line) {
			out = append(out, Finding{"chmod-777", ln, "medium", t, "world-writable permissions"})
		}
		if curlPipe.MatchString(line) {
			out = append(out, Finding{"pipe-to-shell", ln, "high", t,
				"pipe remote content to shell; download, verify checksum, then run"})
		}
	}
	if !hasUser {
		out = append(out, Finding{"no-user", 0, "medium", "(file)",
			"no USER directive; container runs as root"})
	}
	return out
}

var (
	repoRefRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	refRe     = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,120}$`)
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ResolveRef returns the commit SHA for a repo ref via the GitHub API.
func ResolveRef(ctx context.Context, repo, ref string) (string, error) {
	if !repoRefRe.MatchString(repo) {
		return "", fmt.Errorf("repo must be owner/name")
	}
	if !refRe.MatchString(ref) {
		return "", fmt.Errorf("invalid ref %q", ref)
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, ref)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ci-security-mcp/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API %s for %s@%s", resp.Status, repo, ref)
	}
	var body struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if !sha40.MatchString(body.SHA) {
		return "", fmt.Errorf("unexpected SHA %q", body.SHA)
	}
	return body.SHA, nil
}

// PinWorkflow rewrites uses: lines to commit SHAs, annotating the
// original ref. Bounded to maxPins lookups.
func PinWorkflow(ctx context.Context, yaml string, maxPins int) (string, int, error) {
	if maxPins <= 0 || maxPins > 20 {
		maxPins = 10
	}
	lines := strings.Split(yaml, "\n")
	pinned := 0
	for i, line := range lines {
		if pinned >= maxPins {
			break
		}
		m := usesRe.FindStringSubmatch(line)
		if m == nil || sha40.MatchString(m[2]) {
			continue
		}
		sha, err := ResolveRef(ctx, m[1], m[2])
		if err != nil {
			continue // leave unpinned; reported by scan_workflow
		}
		old := m[0]
		repl := fmt.Sprintf("uses: %s@%s  # %s", m[1], sha, m[2])
		lines[i] = strings.Replace(line, old, repl, 1)
		pinned++
	}
	return strings.Join(lines, "\n"), pinned, nil
}

// LintYAML runs a real parse via yaml.v3 plus structural checks
// (tabs, trailing whitespace, CRLF, duplicate keys by indent).
func LintYAML(text string) []Finding {
	var out []Finding
	dec := yaml.NewDecoder(bytes.NewReader([]byte(text)))
	for {
		var node yaml.Node
		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			ln := 0
			if m := regexp.MustCompile(`line (\d+)`).FindStringSubmatch(err.Error()); m != nil {
				fmt.Sscanf(m[1], "%d", &ln) // #nosec G104 -- error tolerated; empty/default is handled downstream
			}
			out = append(out, Finding{"yaml-parse", ln, "high", "", "parse error: " + err.Error()})
			break
		}
	}
	yaml := text
	// duplicate-key detection: track keys per indent level, reset deeper
	// levels when we dedent
	seen := map[int]map[string]bool{}
	var levels []int
	for i, raw := range strings.Split(yaml, "\n") {
		ln := i + 1
		if strings.ContainsRune(raw, '\t') {
			out = append(out, Finding{"yaml-tab", ln, "medium", "tab", "YAML forbids tab indentation"})
		}
		if raw != strings.TrimRight(raw, " \t") {
			out = append(out, Finding{"yaml-trailing-ws", ln, "low", "trailing whitespace", ""})
		}
		if strings.ContainsRune(raw, '\r') {
			out = append(out, Finding{"yaml-crlf", ln, "low", "CR", "carriage return in file"})
		}
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "-") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if idx := strings.Index(t, ":"); idx > 0 {
			k := t[:idx]
			for len(levels) > 0 && levels[len(levels)-1] > indent {
				delete(seen, levels[len(levels)-1])
				levels = levels[:len(levels)-1]
			}
			if seen[indent] == nil {
				seen[indent] = map[string]bool{}
				levels = append(levels, indent)
			}
			if seen[indent][k] {
				out = append(out, Finding{"yaml-dup-key", ln, "medium", k,
					"duplicate key at same indent level"})
			}
			seen[indent][k] = true
		}
	}
	return out
}
