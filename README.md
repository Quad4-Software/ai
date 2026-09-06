# Quad4 MCP toolkit

[![ci](https://img.shields.io/github/actions/workflow/status/Quad4-Software/ai/ci.yml?branch=master&logo=github&label=ci)](https://github.com/Quad4-Software/ai/actions/workflows/ci.yml)
[![gosec](https://img.shields.io/github/actions/workflow/status/Quad4-Software/ai/gosec.yml?branch=master&logo=github&label=gosec)](https://github.com/Quad4-Software/ai/actions/workflows/gosec.yml)
[![race](https://img.shields.io/github/actions/workflow/status/Quad4-Software/ai/race.yml?branch=master&logo=github&label=race)](https://github.com/Quad4-Software/ai/actions/workflows/race.yml)
[![release](https://img.shields.io/github/v/release/Quad4-Software/ai?logo=github&label=release)](https://github.com/Quad4-Software/ai/releases/latest)
[![go version](https://img.shields.io/github/go-mod/go-version/Quad4-Software/ai?filename=agents-mcp/go.mod&logo=go&label=go%20version)](https://github.com/Quad4-Software/ai/blob/master/agents-mcp/go.mod)
[![license](https://img.shields.io/github/license/Quad4-Software/ai?logo=opensourceinitiative&label=license)](https://github.com/Quad4-Software/ai/blob/master/LICENSE)

Small standalone MCP servers for agents working on Reticulum and
MeshChatX. Go, stdlib-first, stdio transport, safe read-only defaults.

| Server | Purpose |
| --- | --- |
| `rns-mcp` | RNS manual, zen, forum/GitHub community, read-only rn* probes |
| `meshchatx-mcp` | MeshChatX docs, scaffolding, god-file detection, split plans, surface snapshots |
| `context-mcp` | Token-efficient code access: outlines, symbol reads, repo maps |
| `agents-mcp` | Repo skills, conventions, module ownership, file tree |
| `no-slop-mcp` | Anti-slop linter for text, diffs, files, directories |
| `bug-hunter-mcp` | Hotspots, TOCTOU, attack surface, complexity, regression mining |
| `ci-security-mcp` | GitHub Actions + Dockerfile scanning, action pinning, YAML lint |
| `i18n-mcp` | Locale keys, coverage, usage, hardcoded strings |
| `workspace-mcp` | Taskfile list/run, git status/log/diff, test mapping and runs |
| `lxmf-mcp` | Local Reticulum state: config, storage, known destinations |
| `lxmfy-mcp` | LXMFy docs, bot scaffolding, diagnostics, and test guidance |
| `micron-mcp` | Micron parsing, linting, rendering, extraction, search, templates, and syntax reference |
| `gateway-mcp` | Multiplexes all servers behind `tools`/`tool_schema`/`invoke` - one config entry, tiny tool surface |

## Install

```bash
git clone git@github.com:Quad4-Software/ai.git
cd ai
make all
make gosec
make install
```

`make install` prints an `mcpServers` entry for every server. Copy the
entries you need into `~/.config/devin/mcp_config.json` or
`~/.cursor/mcp.json` and adjust the paths.

## gateway-mcp: the token saver

Point your client at `gateway-mcp` instead of the individual servers.
It spawns each configured `command` server lazily, exposes a compact
tool index, and fetches full schemas only on demand. External servers
in the same config (docker-based, etc.) are proxied too.

## Design contract

- Newline-delimited JSON-RPC 2.0 over stdio, MCP protocol 2025-11-25.
- Malformed input returns a parse error, never a crash.
- A panicking tool handler returns isError content, the server lives.
- Tool failures return `error: ...` text naming valid arguments.
- Filesystem access is path-jailed to the repo root; secrets redacted.
- No arbitrary command execution anywhere.

## Config

Each server is one binary; add to `~/.config/devin/mcp_config.json` or
`~/.cursor/mcp.json` under `mcpServers`, e.g.

```json
"rns": {"command": "/path/mcp/rns-mcp/rns-mcp"},
"context":  {"command": "/path/mcp/context-mcp/context-mcp",
             "env": {"MCP_REPO_ROOT": "/path/to/repo"}}
```

## Development

Every dir has a Makefile: `make all` runs fmt, vet, test, build;
`make gosec` runs the security scan. The top-level Makefile fans the
same targets across every server. New servers start from
`mcp-scaffold/` (see its README). CI: `.github/workflows/ci.yml`.

Repo-aware servers (`context`, `agents`, `meshchatx`, `i18n`, `workspace`,
`bug-hunter`) use `MCP_REPO_ROOT` or walk up to the nearest `.git`.

## Release

Create an annotated tag and push it to start the release workflow:

```bash
git tag -a v0.1.0 -m "release v0.1.0"
git push origin v0.1.0
```

The `release` workflow builds cross-platform archives, generates
[notes.md](notes.md) from `dist/checksums.txt`, and publishes the
SHA-256 table to the GitHub release notes.

License: 0BSD, see LICENSE.
