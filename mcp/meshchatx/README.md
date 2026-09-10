# meshchatx

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
go build -ldflags="-s -w" -o meshchatx .
```

## MCP config

```json
{
  "mcpServers": {
    "meshchatx": { "command": "/path/to/meshchatx" }
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

## Issue tools (GitHub, opt-in write)

These talk to the GitHub REST API for the MeshChatX issue tracker. Reads and
writes need a token in the environment: `MESHCHATX_GITHUB_TOKEN`, `GITHUB_TOKEN`,
or `GH_TOKEN`. The repo defaults to `Quad4-Software/MeshChatX` and can be
overridden with `MESHCHATX_ISSUES_REPO` (owner/name). The token is only sent to
api.github.com and is never returned in tool output.

- `issue_templates {}` - the bug and feature templates with every field id,
  required flag, and hint. Call before issue_create.
- `issue_create {kind, title, fields, labels?}` - file an issue. The body is
  rendered like the issue forms (### section headings, human prose). Field text
  is normalized: no em dashes, no semicolons, no inline backticks. Fenced code
  blocks are kept.
- `issue_view {number}` - one issue: title, state, labels, author, body.
- `issue_update {number, title?, body?, labels?, state?, state_reason?}` - edit
  fields, close with `state: "closed"` plus `state_reason`, reopen with
  `state: "open"`. Labels replace the whole set.
- `issue_comment {number, body}` - add a comment.
- `issue_search {query, state?, limit?}` - duplicate check before filing.
- `issue_references {}` - the terms that get auto-linked to canonical sources.

Bodies and comments pass through reference linking: the first plain-text
occurrence of each known term (BCP 47, LXMF, RNode, KISS, Landlock, zipapp,
MeshChat, MeshChatX, RRC, ...) becomes a markdown link to its canonical URL,
once per document. Terms inside fenced code blocks, existing links, or URLs are
left alone.
