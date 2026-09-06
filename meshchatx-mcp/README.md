# meshchatx-mcp

Stdio MCP server that exposes the [MeshChatX documentation](https://meshchatx.com/docs/overview)
to MCP clients. Go stdlib only, no dependencies, single static binary.

## Tools

- `list_topics` - topic index (id, title, URL, summary) for the 19 docs pages
- `get_topic` `{id}` - fetch one page, uses the markdown export endpoint when available
- `list_sections` `{id}` - heading index of a page with anchors
- `get_section` `{id, section}` - fetch one section by anchor or heading name
- `search_docs` `{query, limit}` - section-aware full-text search across all pages
- `fetch_page` `{url}` - fetch any page on meshchatx.com (host allowlisted)

## Prompts

- `docs_answer` `{question}` - answer from the docs only, citing page URLs.

## Build and test

```
go test ./...
go build -ldflags="-s -w" -o meshchatx-mcp .
```

## MCP config

```json
{
  "mcpServers": {
    "meshchatx": { "command": "/path/to/meshchatx-mcp" }
  }
}
```

Pages are fetched on demand over HTTPS with a 10 min LRU cache (32 pages, 1 MiB each).

## Codebase tools (need MCP_REPO_ROOT or a .git cwd)

- `god_files {path, limit}` - rank oversized files by lines/symbols/mixed concerns
- `split_plan {file}` - symbol inventory with line ranges and suggested extraction groups
- `surface_snapshot {path}` / `surface_diff {path, before}` - observable surface
  (routes, WS types, i18n keys, emits, symbols) so a split can prove nothing was lost
- `scaffold {kind, name}` - convention-correct file sets
- `checks_for {surface}` - the right verification commands per surface
- `impact {symbol}` - blast radius: refs, definition site, owner hint, probable tests
- `route_map {path}` - registered HTTP routes + test coverage signal
