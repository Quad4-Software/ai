# rns-mcp

Stdio MCP server that exposes the [Reticulum manual](https://reticulum.network/manual/index.html)
to MCP clients. Go stdlib only, no dependencies, single static binary.

## Tools

- `list_topics` - topic index (id, title, URL, summary) for the 15 manual chapters
- `get_topic` `{id}` - fetch one page as plain text (10 min cache, 32 pages, 1 MiB cap each)
- `list_sections` `{id}` - heading index of a page with anchors
- `get_section` `{id, section}` - fetch one section by anchor or heading name, cheap on large pages
- `search_docs` `{query, limit}` - section-aware full-text search across all pages
- `fetch_page` `{url}` - fetch any page on reticulum.network (host allowlisted)

## Local utility tools

Wrap the locally installed RNS utilities (pipx `rns` package). Arguments are
validated before exec, only read-only subcommands are exposed.

- `rns_status` `{all}` - `rnstatus` shared instance and interface info
- `rns_path_table` `{max_hops}` - `rnpath -t` known path table
- `rns_path_lookup` `{destination}` - `rnpath` path request for a hex destination hash
- `rns_destination_hash` `{identity, aspects}` - `rnid -i <id> -H <aspects>` destination hash derivation
- `rns_probe` `{app_name, destination}` - `rnprobe` reachability probe
- `rns_util_help` `{utility}` - `--help` output for rnsd, rnstatus, rnpath, rnprobe,
  rnid, rncp, rnx, rnsh, rngit, rnodeconf, or nomadnet

rngit is a repository node daemon, so it is not wrapped for exec, its usage is
available via `rns_util_help` and its workflows via the `git` manual topic.
rncp, rnx, rnsh, and rnodeconf are mutating tools and likewise stay help-only.

## Prompts

- `nomadnet_context` - NomadNet context including the image rendering prototype and screenshot from rns.recipes.
- `zen_review` `{design}` - review a design against the Zen of Reticulum finish gates.

## Build and test

```
go test ./...
go build -ldflags="-s -w" -o rns-mcp .
```

## MCP config

```json
{
  "mcpServers": {
    "reticulum": { "command": "/path/to/rns-mcp" }
  }
}
```

Pages are fetched on demand over HTTPS, so the binary stays small and docs stay current.

## Community tools

- `unsigned_posts` - unsigned.io article index
- `forum_categories` / `forum_threads {category}` / `forum_search {query}` / `forum_latest` - rns.recipes forum
- `github_discussions {limit}` - markqvist/Reticulum discussions (parsed HTML)
- `github_search {query}` - issues/PRs via GitHub REST (repo mostly uses discussions)
- `community_read {url}` - read a forum thread, unsigned post, or discussion page. rns.recipes forum threads are fetched from their markdown export so images and links stay readable.
