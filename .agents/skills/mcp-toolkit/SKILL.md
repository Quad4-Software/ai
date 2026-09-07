---
name: mcp-toolkit
description: >
  This skill gives the big picture for the Quad4-Software/ai MCP toolkit
  repo. Use when you are adding a server, building or testing the repo, or
  need to understand the shared internal packages and security rules.
metadata:
  repo: Quad4-Software/ai
---

## When to use this skill

- You are adding a new `mcp/<name>` server or editing the repo layout.
- You need the build, test, lint, or security rules.
- You want the shared `internal/mcp` patterns or module path convention.

## How to use

1. Read this skill for the multi-module layout and security rules.
2. Run `make all`, `make gosec`, and per-server `make test`.
3. Copy `mcp/scaffold/` to start a new server and follow the conventions.
4. Cross-reference per-server skills such as `rns`, `meshchatx`, or `workspace`.

## Examples

- "Add a new `mcp/metrics` server from the scaffold."
- "Explain the path-jailing and secret-redaction rules for this repo."
- "Run `make all` after changing `internal/mcp/server.go`."

# MCP Toolkit Project Overview

## Repo layout

The repo root is a multi-module Go workspace of stdio MCP servers. Each
server lives in its own mcp/<name>/ directory with its own go.mod, main.go,
Makefile, README.md, and internal/ packages. mcp/scaffold/ is the
template for new servers.

Current servers and purposes:

- agents - exposes a repo's .agents knowledge graph and docs (skills,
  conventions, module ownership, doc search). Read-only, path-jailed.
- bug-hunter - bug-hunting methodology plus scanners: git churn
  hotspots, TOCTOU pairs, attack-surface sinks, soft-fuzz detection.
- ci-security - CI security guidance and scanning: GitHub Actions
  secure-use docs, workflow/Dockerfile linters, action/image pinning, ref
  resolution via the GitHub API.
- context - token-efficient code access for agents: outlines, windowed
  reads, symbol bodies, repo maps, ranked references. Reports estimated token
  cost per response.
- gateway - multiplexes many stdio MCP servers behind five tools
  (servers, tools, tool_schema, invoke, gateway_stats): lazy child spawn and respawn.
- i18n - locale coverage checks: missing keys, per-key lookups,
  hardcoded UI string candidates.
- lxmf - reads local Reticulum state (~/.reticulum): sanitized config,
  storage inventory, identity names, decoded destinations. Never returns key
  material, secrets are redacted.
- lxmfy - LXMFy docs, bot scaffolding, static diagnostics, test guidance.
- meshchatx - MeshChatX documentation as searchable, section-aware tools,
  plus opt-in GitHub issue tools (issue_templates, issue_create, issue_view,
  issue_update for edit/close/reopen, issue_comment, issue_search,
  issue_references). Bodies auto-link known terms (BCP 47, LXMF, ...)
  to canonical URLs. Writes
  need MESHCHATX_GITHUB_TOKEN, GITHUB_TOKEN, or GH_TOKEN; repo defaults to
  Quad4-Software/MeshChatX, override with MESHCHATX_ISSUES_REPO.
- micron - Micron markup: parse, lint, render HTML/ANSI, extract
  links/headings, search, templates, syntax reference.
- no-slop - lints prose against no-AI-slop and MeshChatX style rules.
- rns - Reticulum manual and API reference as searchable, section-aware tools. Any manual page can be fetched with `get_topic` or `search_docs`. Also exposes the `zen_review` and `nomadnet_context` prompts.
- workspace - wraps repo dev commands: Taskfile targets, git health,
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

Per server: cd mcp/<name> && make test (or go test ./...).

## Module path convention

Modules are github.com/Quad4-Software/ai/mcp/<name>. Remote is
git@github.com:Quad4-Software/ai.git, default branch master. Go 1.27 with
custom toolchain go1.27.1-X:nodwarf5 (see AGENTS.md for caveats).

## Adding a new server

1. Copy mcp/scaffold/ to mcp/<name>/.
2. Update go.mod module name to github.com/Quad4-Software/ai/mcp/<name>.
3. Keep it stdlib-only, read-only, offline-capable.
4. Register tools in main.go via mcp.NewServer.
5. The root Makefile picks it up automatically via $(wildcard mcp/*), excluding scaffold.

## Security rules

- Stdlib-first: no third-party deps unless vendored or justified.
- Read-only by default: never mutate host state, jails and redaction required.
  Opt-in write tools are allowed when env-gated behind a token that is never
  returned (meshchatx issue tools are the precedent).
- Path jailing: resolve all file access to an allowed root, reject traversal.
- Secrets: never return key material, tokens, or config secrets, redact.
- #nosec annotations require an inline justification comment.
- Offline builds: no network access at build/test time except explicit
  features (for example ci-security GitHub ref resolution).
- Vendored deps live under <server>/third_party/ with a replace directive
  in that server's go.mod (see micron for the pattern).
