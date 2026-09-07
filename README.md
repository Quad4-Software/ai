# Quad4 MCP toolkit

<a href="https://github.com/Quad4-Software/ai/actions/workflows/ci.yml"><img src="https://raw.githubusercontent.com/Quad4-Software/ai/master/badges/ci.svg" alt="ci"></a>
<a href="https://github.com/Quad4-Software/ai/actions/workflows/gosec.yml"><img src="https://raw.githubusercontent.com/Quad4-Software/ai/master/badges/gosec.svg" alt="gosec"></a>
<a href="https://github.com/Quad4-Software/ai/actions/workflows/race.yml"><img src="https://raw.githubusercontent.com/Quad4-Software/ai/master/badges/race.svg" alt="race"></a>
<a href="https://github.com/Quad4-Software/ai/releases/latest"><img src="https://raw.githubusercontent.com/Quad4-Software/ai/master/badges/release.svg" alt="release"></a>
<a href="https://github.com/Quad4-Software/ai/blob/master/agents-mcp/go.mod"><img src="https://raw.githubusercontent.com/Quad4-Software/ai/master/badges/go.svg" alt="go version"></a>
<a href="https://github.com/Quad4-Software/ai/blob/master/LICENSE"><img src="https://raw.githubusercontent.com/Quad4-Software/ai/master/badges/license.svg" alt="license"></a>

Small standalone MCP servers and Skills for agents working on Reticulum and
MeshChatX. Go, stdlib-first, stdio transport, safe read-only defaults.

| Server | Purpose |
| --- | --- |
| rns-mcp | RNS manual, zen, forum/GitHub community, rngit, read-only rn* probes |
| reticulum-go-mcp | Reticulum-Go docs at reticulum-go.quad4.io: searchable, section-aware, fetch |
| memory-mcp | Persistent agent memory: notes, people, RNS destinations, tasks, fuzzy recall |
| meshchatx-mcp | MeshChatX docs, scaffolding, god-file detection, split plans, surface snapshots, opt-in GitHub issue tools (env-token gated) |
| context-mcp | Token-efficient code access: outlines, symbol reads, repo maps |
| agents-mcp | Repo skills, conventions, module ownership, file tree |
| no-slop-mcp | Anti-slop linter for text, diffs, files, directories |
| bug-hunter-mcp | Hotspots, TOCTOU, attack surface, complexity, regression mining |
| ci-security-mcp | GitHub Actions + Dockerfile scanning, action pinning, YAML lint |
| i18n-mcp | Locale keys, coverage, usage, hardcoded strings |
| workspace-mcp | Taskfile list/run, git status/log/diff, test mapping and runs |
| lxmf-mcp | Local Reticulum state: config, storage, known destinations |
| lxmfy-mcp | LXMFy docs, bot scaffolding, diagnostics, and test guidance |
| micron-mcp | Micron parsing, linting, rendering, extraction, search, templates, and syntax reference |
| gateway-mcp | Multiplexes all servers behind tools/tool_schema/invoke. One config entry, tiny tool surface. |

## Install

    git clone git@github.com:Quad4-Software/ai.git
    cd ai
    make all
    make gosec
    make install

make install prints an mcpServers entry for every server. Copy the
entries you need into ~/.config/mcp/mcp.json and adjust the paths.

## the gateway-mcp

Point your client at gateway-mcp instead of the individual servers.
It spawns each configured command server lazily, exposes a compact
tool index, and fetches full schemas only on demand. External servers
in the same config (docker-based, etc.) are proxied too.

To share one gateway across every client window, use shared daemon
mode: register the gateway with args ["--attach"], which bridges each
window's stdio to a single long-lived daemon on an owner-only unix
socket (auto-started on first connect). See gateway-mcp/README.md.

## Design

- Newline-delimited JSON-RPC 2.0 over stdio. MCP protocol 2025-11-25.
- Malformed input returns a parse error, never a crash.
- A panicking tool handler returns isError content. The server lives.
- Tool failures return error: ... text naming valid arguments.
- Filesystem access is path-jailed to the repo root. Secrets redacted.
- No arbitrary command execution anywhere.

## Config

Each server is one binary. Add to ~/.config/mcp/mcp.json under
mcpServers, for example:

    "rns": {"command": "/path/ai/rns-mcp/rns-mcp"},
    "context":  {"command": "/path/ai/context-mcp/context-mcp",
                 "env": {"MCP_REPO_ROOT": "/path/to/repo"}}

Use `memory-mcp` to persist destinations, learned facts, and todo items.

License: [0BSD](LICENSE)
