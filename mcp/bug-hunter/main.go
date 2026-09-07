// SPDX-License-Identifier: 0BSD
// Command bug-hunter combines bug-hunting methodology guidance with
// mechanical scanners: git churn hotspots, TOCTOU pairs, attack surface
// sinks, and soft-fuzz test detection. Read-only. Stdlib only.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quad4-Software/ai/mcp/bug-hunter/internal/mcp"
	"github.com/Quad4-Software/ai/mcp/bug-hunter/internal/scan"
)

//go:embed methods/methods.md
var methodsText string

var methodNames = []string{
	"exploratory", "oracle", "property", "metamorphic", "differential",
	"toctou", "churn", "error-injection", "boundary", "combinatorial",
	"state-machine", "attack-surface", "regression", "injection",
	"secrets", "crypto", "concurrency", "supply-chain", "ci-pipeline",
	"mcp-security", "authz-matrix", "ai-code",
}

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

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func methodSection(name string) (string, error) {
	// map tool names to heading substrings
	want := map[string]string{
		"exploratory": "Exploratory testing", "oracle": "Oracle testing",
		"property": "Property-based", "metamorphic": "Metamorphic",
		"differential": "Differential", "toctou": "TOCTOU",
		"churn": "Churn and hotspot", "error-injection": "Error-path",
		"boundary": "Boundary", "combinatorial": "Combinatorial",
		"state-machine": "State machine", "attack-surface": "Attack surface",
		"regression": "Regression mining", "injection": "Injection and taint",
		"secrets": "Secrets and credential", "crypto": "Crypto misuse",
		"concurrency": "Concurrency review", "supply-chain": "Supply chain audit",
		"ci-pipeline": "CI/CD pipeline", "mcp-security": "MCP and agent-tool",
		"authz-matrix": "Authorization matrix", "ai-code": "AI-generated code",
	}
	h, ok := want[name]
	if !ok {
		return "", fmt.Errorf("unknown method %q; valid: %s", name, strings.Join(methodNames, ", "))
	}
	idx := strings.Index(methodsText, "## "+h)
	if idx < 0 {
		return "", fmt.Errorf("section not found for %q", name)
	}
	rest := methodsText[idx:]
	if j := strings.Index(rest[len("## "):], "\n## "); j >= 0 {
		rest = rest[:len("## ")+j]
	}
	return strings.TrimSpace(rest), nil
}

func limitArg(a json.RawMessage, def, max int) (int, string) {
	var p struct {
		Limit int    `json:"limit"`
		Path  string `json:"path"`
	}
	json.Unmarshal(a, &p) // #nosec G104 -- error tolerated; empty/default is handled downstream
	if p.Limit <= 0 || p.Limit > max {
		p.Limit = def
	}
	return p.Limit, p.Path
}

func marshal(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return string(b), err
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_methods",
			Description: "List bug-hunting methods (exploratory, oracle, property, metamorphic, differential, toctou, churn, and more).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return strings.Join(methodNames, "\n"), nil
			},
		},
		{
			Name:        "get_method",
			Description: "Return the guidance section for one method.",
			InputSchema: obj(map[string]any{"name": strArg("method name from list_methods")}, "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				return methodSection(strings.ToLower(strings.TrimSpace(a.Name)))
			},
		},
		{
			Name:          "hotspots",
			Description:   "Rank files by git churn weighted by recency. Defects cluster in high-churn files; hunt there first.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"limit": 15}`)}},
			InputSchema: obj(map[string]any{
				"since": strArg("YYYY-MM-DD or duration like 2160h (90d); default 2160h"),
				"limit": map[string]any{"type": "integer", "description": "max files, default 25"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Since string `json:"since"`
					Limit int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Since == "" {
					a.Since = "2160h"
				}
				if a.Limit <= 0 || a.Limit > 200 {
					a.Limit = 25
				}
				hs, err := scan.Hotspots(ctx, root, a.Since, a.Limit)
				if err != nil {
					return "", err
				}
				return marshal(hs)
			},
		},
		{
			Name:          "toctou_scan",
			Description:   "Find check-then-use pairs (exists/stat/access followed by open/write/remove) within 25 lines. TOCTOU candidates.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max pairs, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 50, 300)
				pairs, err := scan.TOCTOU(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(pairs) == 0 {
					return "no check-then-use pairs found", nil
				}
				return marshal(pairs)
			},
		},
		{
			Name:          "attack_surface",
			Description:   "List mutating sinks: exec, file writes, HTTP mutators, WS handlers, eval, deserializers, raw SQL.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/src/backend/http"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max sinks, default 100"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 100, 500)
				sinks, err := scan.Surface(root, sub, lim)
				if err != nil {
					return "", err
				}
				return marshal(sinks)
			},
		},
		{
			Name:        "soft_fuzz_scan",
			Description: "Flag test anti-patterns: try/except pass, tests with no assertion. Soft fuzz finds nothing.",
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default tests"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Path == "" {
					a.Path = "tests"
				}
				if a.Limit <= 0 || a.Limit > 300 {
					a.Limit = 50
				}
				out, err := scan.SoftFuzzScan(root, a.Path, a.Limit)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no soft-fuzz anti-patterns found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:        "regression_mine",
			Description: "List recent fix/hotfix/revert commits and the files they touched. Past bugs predict where bugs live.",
			InputSchema: obj(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "max commits, default 15"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Limit int `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 50 {
					a.Limit = 15
				}
				commits, err := scan.RegressionMine(ctx, root, a.Limit)
				if err != nil {
					return "", err
				}
				if len(commits) == 0 {
					return "no fix commits found", nil
				}
				return marshal(commits)
			},
		},
		{
			Name:          "mutation_hints",
			Description:   "Suggest mutants for a function: the operator/constant changes most likely to expose a weak test oracle. Pair with run_tests to see if mutants are caught.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/src/backend/plugin_permissions.py", "name": "permission_id_for_hook"}`)}},
			InputSchema: obj(map[string]any{
				"file":  strArg("repo-relative file"),
				"name":  strArg("function name"),
				"limit": map[string]any{"type": "integer", "description": "max mutants, default 20"},
			}, "file", "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					File  string `json:"file"`
					Name  string `json:"name"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.File == "" || a.Name == "" {
					return "", fmt.Errorf("missing required arguments: file, name")
				}
				if a.Limit <= 0 || a.Limit > 100 {
					a.Limit = 20
				}
				out, err := scan.MutationHints(root, a.File, a.Name, a.Limit)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no mutatable operators found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "dead_code",
			Description:   "Functions declared under path that no repo file references. Cleanup candidates; decorator-registered handlers, dynamic dispatch, and plugin hooks can false-positive. Repo-wide scan, may take seconds.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx/src/backend", "limit": 20}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 30"},
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
				out, err := scan.DeadCode(root, a.Path, a.Limit)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no dead-code candidates", nil
				}
				return marshal(out)
			},
		},
		{
			Name:        "complexity_scan",
			Description: "List functions longer than N lines. Long functions hide branches and untested paths.",
			InputSchema: obj(map[string]any{
				"path":      strArg("repo-relative dir, default repo root"),
				"max_lines": map[string]any{"type": "integer", "description": "flag functions longer than this, default 60"},
				"limit":     map[string]any{"type": "integer", "description": "max findings, default 30"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path     string `json:"path"`
					MaxLines int    `json:"max_lines"`
					Limit    int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 300 {
					a.Limit = 30
				}
				out, err := scan.Complexity(root, a.Path, a.MaxLines, a.Limit)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no oversized functions found", nil
				}
				return marshal(out)
			},
		},

		{
			Name:          "secrets_scan",
			Description:   "Find credential-shaped literals (API keys, tokens, private key blocks, password= assignments). Values are redacted in output; every hit is a live secret until disproven.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "config"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 50, 300)
				out, err := scan.Secrets(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no secret-shaped literals found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "injection_scan",
			Description:   "Dangerous sinks across Go, Python and JS: shell=True, os.system, exec.Command sh -c, eval, pickle/yaml.load, f-string SQL, SSTI, SSRF, archive extractall, unsafe, text/template for HTML, pipe-to-shell.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "src"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 100"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 100, 500)
				out, err := scan.Injection(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no injection sinks found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "crypto_scan",
			Description:   "Weak crypto and verification bypasses: md5/sha1/des/rc4/ecb, math/rand or Math.random for secrets, InsecureSkipVerify, verify=False, old TLS, JWT alg confusion, == on secrets (timing).",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": ""}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 100"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 100, 500)
				out, err := scan.Crypto(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no crypto misuse found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "concurrency_scan",
			Description:   "Go race and lifecycle heuristics: loop-var capture in goroutines, defer inside loops, WaitGroup.Add inside goroutines, unclosed response bodies, send-after-close, package-level shared maps without sync.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "internal"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 50, 300)
				out, err := scan.Concurrency(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no concurrency heuristics hit", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "taint_scan",
			Description:   "Naive intra-file taint: identifiers assigned from request/argv/env sources reaching exec, SQL, template, file-path or HTTP sinks within 30 lines. Candidates only; confirm flow by hand.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "src/backend"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 50, 300)
				out, err := scan.Taint(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no tainted source-to-sink pairs found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "supply_chain_scan",
			Description:   "Audit manifests, CI workflows and install scripts: npm lifecycle scripts, unpinned deps, go.mod replace, unpinned GitHub Actions, pull_request_target plus secrets, pipe-to-shell installers, missing lockfiles.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{}`)}},
			InputSchema: obj(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "max findings, default 100"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, _ := limitArg(args, 100, 500)
				out, err := scan.SupplyChain(root, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no supply-chain indicators found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:        "git_secrets",
			Description: "Scan added lines in recent git history for credential-shaped literals. Bounded to the last N commits; output is redacted.",
			InputSchema: obj(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "commits to scan, default 200, max 1000"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Limit int `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 1000 {
					a.Limit = 200
				}
				out, err := scan.GitSecrets(ctx, root, a.Limit)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no secrets in scanned history", nil
				}
				return marshal(out)
			},
		},
		{
			Name:        "markers_scan",
			Description: "Find TODO/FIXME/HACK/XXX/BUG/SECURITY comments. Markers cluster where developers already suspected problems.",
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 100"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 100, 500)
				out, err := scan.Markers(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no annotated markers found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:          "mcp_audit",
			Description:   "Audit MCP/agent-tool code: tool descriptions carrying hidden directives (tool poisoning), imperative pressure on the model, and exec sinks reachable from tool arguments.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": "meshchatx"}`)}},
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default repo root"),
				"limit": map[string]any{"type": "integer", "description": "max findings, default 50"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				lim, sub := limitArg(args, 50, 300)
				out, err := scan.MCPAudit(root, sub, lim)
				if err != nil {
					return "", err
				}
				if len(out) == 0 {
					return "no agent-tool issues found", nil
				}
				return marshal(out)
			},
		},
		{
			Name:        "charter",
			Description: "Generate an exploratory test charter template for a target area.",
			InputSchema: obj(map[string]any{"area": strArg("subsystem or file area to hunt in")}, "area"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Area string `json:"area"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Area == "" {
					return "", fmt.Errorf("missing required argument: area")
				}
				return fmt.Sprintf(`Charter: %s

Target: %s
Window: one focused session

1. Map the state machine or ACL matrix from the code, not memory.
2. Write 5-15 hypotheses (H1..Hn): file reference + predicted wrong behaviour.
   H1:
   H2:
   H3:
3. For each high-priority hypothesis, write a failing oracle test first.
4. Confirm with focused test runs. Report confirmed bugs only.
5. Record intentional behaviours so they are not changed by accident.

Suggested scans first: attack_surface path=%q, toctou_scan path=%q, injection_scan, taint_scan, hotspots.`,
					a.Area, a.Area, a.Area, a.Area), nil
			},
		},
	}
}

func prompts() []mcp.Prompt {
	return []mcp.Prompt{
		{
			Name:        "hunt",
			Description: "Full bug-hunt briefing: methodology plus scanner results for an area.",
			Arguments:   []mcp.PromptArg{{Name: "area", Description: "subsystem to hunt in", Required: true}},
			Handle: func(args map[string]string) (string, error) {
				return methodsText + "\n\nRun the scans (hotspots, attack_surface, toctou_scan, soft_fuzz_scan) on " + args["area"] +
					", then write explicit hypotheses and confirm each with an oracle test before changing code.", nil
			},
		},
	}
}

func main() {
	var err error
	root, err = findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bug-hunter:", err)
		os.Exit(1)
	}
	srv := mcp.NewServer("bug-hunter", "0.1.0", tools(), prompts())
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "bug-hunter:", err)
		os.Exit(1)
	}
}
