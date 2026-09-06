---
name: mcp-toolkit
description: >
  Use when working in the Quad4-Software/ai MCP toolkit repo: adding or
  modifying stdio MCP servers, running builds/tests/lints, understanding
  the shared internal/mcp package, module layout, and security rules.
---

# MCP Toolkit Project Overview

## Repo layout

The repo root is a multi-module Go workspace of stdio MCP servers. Each
server lives in its own *-mcp/ directory with its own go.mod, main.go,
Makefile, README.md, and internal/ packages. mcp-scaffold/ is the
template for new servers.

Current servers and purposes:

- agents-mcp - exposes a repo's .agents knowledge graph and docs (skills,
  conventions, module ownership, doc search). Read-only, path-jailed.
- bug-hunter-mcp - bug-hunting methodology plus scanners: git churn
  hotspots, TOCTOU pairs, attack-surface sinks, soft-fuzz detection.
- ci-security-mcp - CI security guidance and scanning: GitHub Actions
  secure-use docs, workflow/Dockerfile linters, action/image pinning, ref
  resolution via the GitHub API.
- context-mcp - token-efficient code access for agents: outlines, windowed
  reads, symbol bodies, repo maps, ranked references. Reports estimated token
  cost per response.
- gateway-mcp - multiplexes many stdio MCP servers behind five tools
  (servers, tools, tool_schema, invoke, gateway_stats): lazy child spawn and respawn.
- i18n-mcp - locale coverage checks: missing keys, per-key lookups,
  hardcoded UI string candidates.
- lxmf-mcp - reads local Reticulum state (~/.reticulum): sanitized config,
  storage inventory, identity names, decoded destinations. Never returns key
  material, secrets are redacted.
- lxmfy-mcp - LXMFy docs, bot scaffolding, static diagnostics, test guidance.
- meshchatx-mcp - MeshChatX documentation as searchable, section-aware tools.
- micron-mcp - Micron markup: parse, lint, render HTML/ANSI, extract
  links/headings, search, templates, syntax reference.
- no-slop-mcp - lints prose against no-AI-slop and MeshChatX style rules.
- rns-mcp - Reticulum manual as searchable, section-aware tools. Low memory.
- workspace-mcp - wraps repo dev commands: Taskfile targets, git health,
  test-file mapping. Commands are validated and bounded.

## Shared packages

Each server has its own internal/ tree. Common pattern: internal/mcp
provides the stdio JSON-RPC server scaffolding (mcp.NewServer(name, version,
tools, resources)), plus domain packages (internal/docs, internal/codemap,
internal/cisec, internal/lint, internal/community, internal/templates).

## Build and test

From repo root:

```
make all       # fmt + go-fix + vet + test + build across all servers
make gosec     # gosec security scan per server
make golangci  # golangci-lint run ./... per server
make build / test / vet / fmt / clean   # delegated to each server Makefile
```

Per server: cd <name>-mcp && make test (or go test ./...).

## Module path convention

Modules are github.com/Quad4-Software/ai/<name>-mcp. Remote is
git@github.com:Quad4-Software/ai.git, default branch master. Go 1.27 with
custom toolchain go1.27.1-X:nodwarf5 (see AGENTS.md for caveats).

## Adding a new server

1. Copy mcp-scaffold/ to <name>-mcp/.
2. Update go.mod module name to github.com/Quad4-Software/ai/<name>-mcp.
3. Keep it stdlib-only, read-only, offline-capable.
4. Register tools in main.go via mcp.NewServer.
5. The root Makefile picks it up automatically via $(wildcard *-mcp).

## Security rules

- Stdlib-first: no third-party deps unless vendored or justified.
- Read-only by default: never mutate host state, jails and redaction required.
- Path jailing: resolve all file access to an allowed root, reject traversal.
- Secrets: never return key material, tokens, or config secrets, redact.
- #nosec annotations require an inline justification comment.
- Offline builds: no network access at build/test time except explicit
  features (for example ci-security GitHub ref resolution).
- Vendored deps live under <server>/third_party/ with a replace directive
  in that server's go.mod (see micron-mcp for the pattern).
