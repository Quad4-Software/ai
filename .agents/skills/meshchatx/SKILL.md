---
name: meshchatx
description: >
  This skill exposes MeshChatX docs, codebase scaffolding, and GitHub
  issue tools through meshchatx. Use when you need MeshChatX manual
  topics, feature/scaffold checks, or issue management for the repo.
metadata:
  server: meshchatx
---

## When to use this skill

- You need MeshChatX docs (architecture, messaging, plugins, Android).
- You are scaffolding a Svelte feature, backend manager, or plugin.
- You are creating or triaging MeshChatX GitHub issues.

## How to use

1. Build meshchatx: `cd mcp/meshchatx && go test ./... && go build`.
2. Add the binary to your MCP client config as `meshchatx`.
3. `list_topics`/`get_topic` for docs, `scaffold`/`checks_for` for
   code, and the issue tools need a token.

## Examples

- "Fetch the messaging section of the MeshChatX docs."
- "Scaffold a plugin named `relay-stats`."
- "File a bug issue with version and reproduction fields."

# meshchatx

meshchatx is the largest server: 22 tools in three families plus one
prompt. Root is `MCP_REPO_ROOT` or the nearest `.git`, and codebase
tools error without a root.

## Docs tools (6)

`list_topics`, `get_topic {id}`, `list_sections {id}`,
`get_section {id, section}`, `search_docs {query, limit 5-20}`,
`fetch_page {url}` (host allowlisted to the topic index, i.e.
meshchatx.com). Topics cover overview through linux-sandbox (19
total). Pages fetch over HTTPS on demand, so they are not offline.

## Codebase tools (9, need repo root)

- `scaffold {kind, name}`: kinds `svelte-feature`,
  `svelte-component`, `backend-manager`, `ws-handler`, `plugin`,
  `metamorphic-test`. Names match `^[a-z][a-z0-9_-]{0,48}$`.
- `checks_for {surface}`: surfaces `svelte`, `backend`, `plugin`,
  `ws`, `docs`, `i18n`, `electron`, `landlock`.
- `god_files`, `split_plan`, `surface_snapshot`, `surface_diff`,
  `feature_check`, `route_map`, `impact` for refactor analysis.

## Issue tools (7)

`issue_templates`, `issue_create`, `issue_view`, `issue_update`,
`issue_comment`, `issue_search`, `issue_references`. Create, update,
and comment are Write-marked.

Environment: token lookup order `MESHCHATX_GITHUB_TOKEN`,
`GITHUB_TOKEN`, `GH_TOKEN`. Repo override `MESHCHATX_ISSUES_REPO`
(default `Quad4-Software/MeshChatX`, validated `owner/name`). The
token goes only to api.github.com. All issue tools, including
reads, require a token. Bodies are normalized to house style and
about 30 known terms get auto-linked.

## Prompts

`docs_answer {question}` wraps the docs index for a Q&A pass.

## Notes and quirks

- `MCP_READ_ONLY=1`/`--read-only` hides the Write-marked tools.
- Doc pages cache in a 32-entry LRU that never expires within a
  session (`cacheTTL` is declared but unused). Restart for fresh docs.
- README drift: `feature_check` is implemented but undocumented.
- Errors arrive as `error: ` tool content, not JSON-RPC errors.
