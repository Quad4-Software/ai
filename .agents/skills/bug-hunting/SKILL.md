---
name: bug-hunting
description: >
  This skill guides structured bug hunting with bug-hunter. Use when you
  need to choose a methodology, run mechanical security scans, mine git
  history for hotspots, or audit a repo for bugs.
compatibility: stdio-mcp
metadata:
  server: bug-hunter
---

## When to use this skill

- You are planning a bug hunt or chartering an area.
- You need to run a scanner such as `hotspots`, `toctou_scan`, or `injection_scan`.
- You want to audit an MCP tool surface for poisoning.
- You are confirming a hypothesis with an oracle before reporting.

## How to use

1. Build and test bug-hunter: `cd mcp/bug-hunter && go test ./... && go build`.
2. Add the binary to your MCP client config as `bug-hunter`.
3. Call `list_methods` to see the 22 methodologies, or run a scanner like `hotspots {since, limit}`.
4. Treat every scanner hit as a candidate. Confirm with an oracle before calling it a bug.

## Examples

- "Run `churn` hotspots and `toctou_scan` on the last 90 days of `internal/`."
- "Audit `mcp_audit` on `.agents/skills/` for tool-poisoning patterns."
- "Generate a `hunt` prompt for the authz area and propose the right scanners."

# Bug hunting

This repo includes bug-hunter, a read-only, stdlib-only stdio MCP server for
structured bug hunting: methodology guidance plus mechanical scanners over any
repo rooted at `MCP_REPO_ROOT` (or the nearest .git ancestor).

## Methodology tools

- `list_methods` / `get_method {name}`: 22 methods with when-to-use guidance
- `charter {area}`: exploratory session template with suggested scans
- `hunt {area}` prompt: full briefing combining methodology and scanners

## Mechanical scanners

Coverage and git mining:

- `hotspots {since, limit}`: git churn ranked by recency, defects cluster there
- `regression_mine {limit}`: fix/hotfix/revert commits and touched files
- `complexity_scan {path, max_lines}`: oversized functions hide untested paths
- `dead_code {path, limit}`: unreferenced functions, candidates only
- `markers_scan {path, limit}`: TODO/FIXME/HACK/SECURITY comments
- `mutation_hints {file, name}`: mutants that expose weak test oracles
- `soft_fuzz_scan {path, limit}`: try/except pass, tests without assertions

Code-level security:

- `attack_surface {path, limit}`: mutating entry points and sinks
- `toctou_scan {path, limit}`: check-then-use pairs within 25 lines
- `injection_scan {path, limit}`: shell, eval, deserialization, SQLi, SSTI,
  SSRF, archive extraction, pipe-to-shell across Go/Python/JS
- `crypto_scan {path, limit}`: weak algorithms, insecure random, disabled
  verification, JWT alg confusion, timing-unsafe comparisons
- `concurrency_scan {path, limit}`: Go loop-var capture, defer in loops,
  WaitGroup misuse, unclosed bodies, send-after-close, unlocked shared state
- `taint_scan {path, limit}`: naive source-to-sink pairs within a file

Repo and history security:

- `secrets_scan {path, limit}`: credential-shaped literals, redacted output
- `git_secrets {limit}`: secrets added in recent history, deleted still leaks
- `supply_chain_scan {limit}`: lifecycle scripts, unpinned deps and actions,
  pull_request_target with secrets, pipe-to-shell, missing lockfiles
- `mcp_audit {path, limit}`: tool-poisoning descriptions, exec sinks in
  agent tools

All findings are heuristic candidates, not verdicts. Confirm before reporting.

## Discipline

State a hypothesis before touching code. Confirm with an oracle that accepts or
rejects independently of the buggy path. Record intentional behaviour so it is
not "fixed" by accident. Every scanner hit is a hypothesis, not a bug.

## Reference checklists

- `references/go-checklist.md` - Go bug and security patterns (races, goroutine
  leaks, defer-in-loop, SQLi, SSRF, TLS, crypto, unsafe, gosec mapping)
- `references/python-checklist.md` - Python patterns (pickle/yaml, shell=True,
  f-string SQL, tarfile/zipfile traversal, SSTI, XXE, assert authz, bandit
  mapping)
- `references/supply-chain.md` - 2025-2026 supply chain: typosquats,
  slopsquats, tj-actions, XZ, npm worms, dependency confusion, CI/CD PPE,
  verification tooling
- `references/mcp-agent-security.md` - MCP tool poisoning, rug pulls, exec
  sinks, prompt injection, AI-generated code review
- `references/web-checklist.md` - JWT confusion, request smuggling, prototype
  pollution, open redirects, secrets-in-git workflows
