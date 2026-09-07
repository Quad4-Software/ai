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

### MCP servers

    git clone git@github.com:Quad4-Software/ai.git
    cd ai
    make all
    make gosec
    make install

make install prints a single client mcp.json entry for gateway-mcp
plus a generated quad4-mcp.json for the gateway to consume. Copy the
gateway entry into your MCP client config, then place the gateway
config at ~/.config/mcp/quad4-mcp.json. The gateway lazily spawns the
other servers behind one compact tool surface.

    make mcp-config   # regenerate dist/mcp.json and dist/quad4-mcp.json
    make server-json  # generate registry server.json files per server
    make inspector    # smoke test every built server for tools/list

Prebuilt binaries are also attached to each GitHub release.

### Agent skills

Skills live in .agents/skills/ and follow the Agent Skills
specification. Install them into your agent with the skills.sh CLI:

    npx skills add Quad4-Software/ai

Useful flags:

    npx skills add Quad4-Software/ai --list            # preview available skills
    npx skills add Quad4-Software/ai --skill micron    # install only one skill
    npx skills add Quad4-Software/ai -g                # global install

Skill groups are defined in skills.sh.json: MCP toolkit (build, test,
security, release, style) and Reticulum (mesh networking, LXMF,
tooling, interface operation).

## the gateway-mcp

Point your client at gateway-mcp instead of the individual servers.
It spawns each configured command server lazily, exposes a compact
tool index, and fetches full schemas only on demand. External servers
in the same config (docker-based, etc.) are proxied too.

To share one gateway across every client window, use shared daemon
mode: register the gateway with args ["--attach"], which bridges each
window's stdio to a single long-lived daemon on an owner-only unix
socket (auto-started on first connect). See gateway-mcp/README.md.

## One manifest source

mcp-servers.json is the single source of truth for every server. The
scripts/gen-configs.py tool reads it to emit the client mcp.json, the
gateway quad4-mcp.json, and per-server registry server.json files. The
mcp-servers.json list also drives the skills.sh descriptions.

## Design

- Newline-delimited JSON-RPC 2.0 over stdio. MCP protocol 2025-11-25.
- Malformed input returns a parse error, never a crash.
- A panicking tool handler returns isError content. The server lives.
- Tool failures return error text naming the required arguments.
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

<a href="https://skills.sh/Quad4-Software/ai"><img src="https://skills.sh/b/Quad4-Software/ai" alt="skills.sh"></a>
