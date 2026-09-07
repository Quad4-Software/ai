# bug-hunter

Stdio MCP server for structured bug hunting: methodology guidance plus
mechanical scanners. Read-only. Stdlib only.

Root: `MCP_REPO_ROOT` or nearest `.git` ancestor.

## Methods (22)

`list_methods` / `get_method {name}` return guidance for:

exploratory, oracle, property, metamorphic, differential, toctou,
churn, error-injection, boundary, combinatorial, state-machine,
attack-surface, regression, injection, secrets, crypto, concurrency,
supply-chain, ci-pipeline, mcp-security, authz-matrix, ai-code.

## Scanners

- `hotspots {since, limit}` - git churn weighted by recency; defects
  cluster in high-churn files
- `regression_mine {limit}` - recent fix/hotfix/revert commits and the
  files they touched; past bugs predict future ones
- `toctou_scan {path, limit}` - check-then-use pairs within 25 lines
- `attack_surface {path, limit}` - exec, file writes, HTTP mutators, WS
  handlers, eval, deserializers, raw SQL
- `injection_scan {path, limit}` - shell=True, sh -c, eval, pickle and
  yaml.load, f-string SQL, SSTI, SSRF, extractall, unsafe, text/template
  for HTML, pipe-to-shell
- `crypto_scan {path, limit}` - md5/sha1/des/rc4/ecb, insecure random,
  InsecureSkipVerify / verify=False, old TLS, JWT alg confusion, == on
  secrets
- `secrets_scan {path, limit}` - token-shaped literals and key
  assignments across code and config files; values redacted
- `git_secrets {limit}` - secret-shaped lines added in recent history;
  a deleted commit still leaks
- `concurrency_scan {path, limit}` - Go loop-var capture, defer in
  loops, WaitGroup.Add inside goroutines, unclosed response bodies,
  send-after-close, shared state without sync
- `taint_scan {path, limit}` - naive intra-file taint: request/argv/env
  sources reaching sinks within 30 lines
- `supply_chain_scan {limit}` - manifests and workflows: npm lifecycle
  scripts, unpinned deps, go.mod replace, unpinned actions,
  pull_request_target plus secrets, pipe-to-shell, missing lockfiles
- `mcp_audit {path, limit}` - tool-poisoning patterns in MCP tool
  descriptions, exec sinks reachable from tool arguments
- `soft_fuzz_scan {path, limit}` - try/except pass and assert-free tests
- `complexity_scan {path, max_lines, limit}` - functions over N lines
- `dead_code {path, limit}` - functions declared under path that no repo
  file references; refs counted repo-wide
- `markers_scan {path, limit}` - TODO/FIXME/HACK/XXX/BUG/SECURITY
  comments
- `mutation_hints {file, name}` - operator/constant mutants for a
  function
- `charter {area}` - exploratory session template with suggested scans

All findings are heuristic candidates. State a hypothesis, confirm with
an oracle test, then report.

## Prompts

- `hunt {area}` - full methodology plus scanner-driven hunt briefing

## Discipline

State a hypothesis before touching code. Confirm with an oracle that
accepts or rejects independently of the buggy path. Record intentional
behaviour so it is not fixed by accident.
