// SPDX-License-Identifier: 0BSD
// Command ci-security-mcp provides CI security guidance and scanning:
// GitHub Actions secure-use docs, workflow and Dockerfile linters,
// action/image pinning, and ref resolution via the GitHub API.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Quad4-Software/ai/ci-security-mcp/internal/cisec"
	"github.com/Quad4-Software/ai/ci-security-mcp/internal/docs"
	"github.com/Quad4-Software/ai/ci-security-mcp/internal/mcp"
)

//go:embed reference/reference.md
var referenceText string

var topics = []docs.Topic{
	{ID: "secure-use", Title: "Secure use reference", URL: "https://docs.github.com/en/actions/reference/security/secure-use", Summary: "GitHub Actions secure use: permissions, tokens, pinning, injection"},
	{ID: "pull-request-target", Title: "pull_request_target security", URL: "https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target", Summary: "Safe and unsafe uses of pull_request_target"},
	{ID: "secrets", Title: "Secrets in Actions", URL: "https://docs.github.com/en/actions/reference/security/secrets", Summary: "Secret storage, masking, fork rules, OIDC"},
	{ID: "workflow-syntax", Title: "Workflow syntax", URL: "https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax", Summary: "Full workflow YAML syntax reference"},
	{ID: "security-hardening", Title: "Security hardening for Actions", URL: "https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions", Summary: "Script injection, third-party actions, OpenID Connect, compromised actions"},
	{ID: "docker-owasp", Title: "OWASP Docker Security Cheat Sheet", URL: "https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html", Summary: "Container hardening: user, capabilities, resources, images, secrets"},
	{ID: "immutable-releases", Title: "Immutable releases", URL: "https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/immutable-releases", Summary: "Prevent tag re-pointing and release asset tampering"},
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func marshalFindings(fs []cisec.Finding, emptyMsg string) (string, error) {
	if len(fs) == 0 {
		return emptyMsg, nil
	}
	b, err := json.MarshalIndent(fs, "", "  ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d findings:\n%s", len(fs), b), nil
}

func tools(src *docs.Source) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_guides",
			Description: "List CI security guides (GitHub Actions secure use, workflow syntax, secrets, pull_request_target, OWASP Docker, immutable releases).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, _ := json.MarshalIndent(src.List(), "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "get_guide",
			Description: "Fetch a security guide page as plain text by id.",
			InputSchema: obj(map[string]any{"id": strArg("guide id from list_guides")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				return src.Get(ctx, strings.TrimSpace(a.ID))
			},
		},
		{
			Name:        "reference",
			Description: "Embedded CI security cheat sheet: pinning, pull_request_target, secrets, permissions, injection, Docker hardening. Works offline.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return referenceText, nil
			},
		},
		{
			Name:          "scan_workflow",
			Description:   "Scan GitHub Actions workflow YAML: unpinned actions, pull_request_target traps, script injection, missing permissions, secret leaks, pipe-to-shell.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"yaml": "on: push\njobs:\n  a:\n    steps:\n    - uses: actions/checkout@v4"}`)}},
			InputSchema:   obj(map[string]any{"yaml": strArg("workflow file contents")}, "yaml"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					YAML string `json:"yaml"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.YAML == "" {
					return "", fmt.Errorf("missing required argument: yaml")
				}
				return marshalFindings(cisec.ScanWorkflow(a.YAML), "no issues found")
			},
		},
		{
			Name:        "scan_dockerfile",
			Description: "Scan a Dockerfile: image pinning, USER, ADD remote URLs, privileged flags, pipe-to-shell, chmod 777.",
			InputSchema: obj(map[string]any{"dockerfile": strArg("Dockerfile contents")}, "dockerfile"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Dockerfile string `json:"dockerfile"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Dockerfile == "" {
					return "", fmt.Errorf("missing required argument: dockerfile")
				}
				return marshalFindings(cisec.ScanDockerfile(a.Dockerfile), "no issues found")
			},
		},
		{
			Name:          "lint_yaml",
			Description:   "Real YAML parse (yaml.v3) plus structural checks: tabs, trailing whitespace, CRLF, duplicate keys at the same indent.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"path": ".github/workflows/ci.yml"}`)}},
			InputSchema:   obj(map[string]any{"yaml": strArg("YAML text")}, "yaml"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					YAML string `json:"yaml"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.YAML == "" {
					return "", fmt.Errorf("missing required argument: yaml")
				}
				return marshalFindings(cisec.LintYAML(a.YAML), "no lint issues")
			},
		},
		{
			Name:        "resolve_ref",
			Description: "Resolve a GitHub repo ref (tag/branch) to its commit SHA via api.github.com. Use before pinning.",
			InputSchema: obj(map[string]any{
				"repo": strArg("owner/name, e.g. actions/checkout"),
				"ref":  strArg("tag, branch, or ref, e.g. v4.2.2"),
			}, "repo", "ref"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Repo string `json:"repo"`
					Ref  string `json:"ref"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Repo == "" || a.Ref == "" {
					return "", fmt.Errorf("missing required arguments: repo, ref")
				}
				return cisec.ResolveRef(ctx, a.Repo, a.Ref)
			},
		},
		{
			Name:          "pin_workflow",
			Description:   "Rewrite workflow YAML pinning every mutable uses: ref to its commit SHA (annotated with the original tag). Max 10 lookups.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"yaml": "on: push\njobs:\n  a:\n    steps:\n    - uses: actions/checkout@v4"}`)}},
			InputSchema: obj(map[string]any{
				"yaml": strArg("workflow file contents"),
			}, "yaml"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					YAML string `json:"yaml"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.YAML == "" {
					return "", fmt.Errorf("missing required argument: yaml")
				}
				out, n, err := cisec.PinWorkflow(ctx, a.YAML, 10)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("pinned %d action(s):\n\n%s", n, out), nil
			},
		},
	}
}

func prompts() []mcp.Prompt {
	return []mcp.Prompt{
		{
			Name:        "audit_workflow",
			Description: "Audit a workflow file against the embedded reference.",
			Arguments:   []mcp.PromptArg{{Name: "yaml", Description: "workflow YAML", Required: true}},
			Handle: func(args map[string]string) (string, error) {
				return referenceText + "\n\nAudit this workflow against the reference. Report each violation with line and fix.\n\n" + args["yaml"], nil
			},
		},
	}
}

func main() {
	src := docs.NewSource("CI Security Guides", topics)
	srv := mcp.NewServer("ci-security-mcp", "0.1.0", tools(src), prompts())
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ci-security-mcp:", err)
		os.Exit(1)
	}
}
