---
name: meshchatx
description: >
  This skill exposes the MeshChatX documentation and optional GitHub issue
  tools through meshchatx-mcp. Use when you need MeshChatX docs, codebase
  snapshots, or issue handling.
metadata:
  server: meshchatx-mcp
---

## When to use this skill

- You need to look up a MeshChatX doc page or section.
- You want to scaffold, split, or snapshot MeshChatX code.
- You need to create, view, or update a MeshChatX GitHub issue.

## How to use

1. Build meshchatx-mcp: `cd meshchatx-mcp && go test ./... && go build`.
2. Add the binary to your MCP client config as `meshchatx`.
3. Call `list_topics` or `search_docs {query}` for docs, `god_files` or `surface_snapshot` for code.
4. Issue tools need a `MESHCHATX_GITHUB_TOKEN`, `GITHUB_TOKEN`, or `GH_TOKEN` and are optional.

## Examples

- "Search MeshChatX docs for `propagation node setup`."
- "Show the `god_files` ranking for `meshchatx-mcp/internal/`."
- "Create a bug issue with the `bug_report` template."
