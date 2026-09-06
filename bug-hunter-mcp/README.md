# bug-hunter-mcp

Stdio MCP server for structured bug hunting: methodology guidance plus
mechanical scanners. Read-only. Stdlib only.

Root: `MCP_REPO_ROOT` or nearest `.git` ancestor.

## Tools

- `list_methods` / `get_method {name}` - 13 methods: exploratory charters,
  oracle testing, property-based, metamorphic, differential, TOCTOU/race,
  churn-hotspot, error injection, boundary, combinatorial, state machine,
  attack surface, regression mining
- `hotspots {since, limit}` - git churn weighted by recency, defects cluster
  in high-churn files
- `toctou_scan {path, limit}` - check-then-use pairs within 25 lines
- `attack_surface {path, limit}` - exec, file writes, HTTP mutators, WS
  handlers, eval, deserializers, raw SQL
- `soft_fuzz_scan {path, limit}` - try/except pass and assert-free tests
- `charter {area}` - exploratory session template with suggested scans

## Prompts

- `hunt {area}` - full methodology plus scanner-driven hunt briefing
- `dead_code {path, limit}` - functions declared under path that no repo file references; refs counted repo-wide
- `mutation_hints {file, name}` - operator/constant mutants for a function
