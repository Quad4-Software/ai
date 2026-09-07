# reticulum-go

Stdio MCP server that exposes the [Reticulum-Go](https://reticulum-go.quad4.io/)
docs to MCP clients. Go stdlib only, no dependencies, single static binary.

## Tools

- `list_topics` - topic index (id, title, URL, summary) for the Reticulum-Go docs
- `get_topic` `{id}` - fetch one page as plain text (10 min cache, 1 MiB cap each)
- `list_sections` `{id}` - heading index of a page with anchors
- `get_section` `{id, section}` - fetch one section by anchor or heading name
- `search_docs` `{query, limit}` - section-aware full-text search across all pages
- `fetch_page` `{url}` - fetch any page on reticulum-go.quad4.io (host allowlisted)
- `github_repo` - list Reticulum-Go repository info and open issues/PRs

## Build and test

```
go test ./...
go build -ldflags="-s -w" -o reticulum-go .
```

## MCP config

```json
{
  "mcpServers": {
    "reticulum-go": { "command": "/path/to/reticulum-go" }
  }
}
```

Pages are fetched on demand over HTTPS, so the binary stays small and docs stay current.
