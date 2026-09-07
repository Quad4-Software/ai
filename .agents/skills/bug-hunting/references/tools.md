## Tools

- `list_methods` / `get_method {name}`: 22 named methods with when-to-use
  guidance, including injection/taint, secrets, crypto, concurrency,
  supply-chain, ci-pipeline, mcp-security, authz-matrix, ai-code
- `hotspots {since, limit}`: git churn ranked by recency
- `regression_mine {limit}`: fix/hotfix/revert commits and touched files
- `toctou_scan {path, limit}`: check-then-use pairs within a 25-line window
- `attack_surface {path, limit}`: mutating entry points such as exec, file
  writes, HTTP mutators, WS handlers, eval, deserializers and raw SQL
- `injection_scan {path, limit}`: shell, eval, unsafe deserialization,
  SQLi, SSTI, SSRF, archive traversal, pipe-to-shell (Go/Python/JS)
- `crypto_scan {path, limit}`: weak algorithms, insecure random, disabled
  TLS verification, JWT algorithm confusion, timing-unsafe comparisons
- `concurrency_scan {path, limit}`: Go loop-var capture, defer-in-loop,
  WaitGroup.Add in goroutines, unclosed response bodies, send-after-close
- `taint_scan {path, limit}`: intra-file source-to-sink pairs
- `secrets_scan {path, limit}`: credential-shaped literals, redacted
- `git_secrets {limit}`: secret-shaped added lines in recent history
- `supply_chain_scan {limit}`: manifests, workflows and installers for
  lifecycle scripts, unpinned deps and actions, fork-trigger secrets,
  pipe-to-shell, missing lockfiles
- `mcp_audit {path, limit}`: tool-poisoning descriptions, exec sinks in
  tool handlers
- `soft_fuzz_scan {path, limit}`: try/except pass and tests without
  assertions
- `complexity_scan {path, max_lines}`: functions over N lines
- `dead_code {path, limit}`: unreferenced functions, candidates only
- `markers_scan {path, limit}`: TODO/FIXME/HACK/SECURITY comment density
- `mutation_hints {file, name}`: mutants that expose weak oracles
- `charter {area}`: exploratory session template with suggested scans

Prompts: `hunt {area}` for a full methodology-plus-scanners briefing.

All scanner output is candidate findings. Confirm each with an oracle
test before reporting or changing code.
