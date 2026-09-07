---
name: rns
description: >
  This skill exposes the Reticulum manual and local utilities through
  rns-mcp. Use when you need searchable manual topics, local RNS state,
  or utility help.
metadata:
  server: rns-mcp
---

## When to use this skill

- You need a Reticulum manual chapter, section, or search result.
- You want to read `~/.reticulum` state safely without leaking keys.
- You need `rnstatus`, `rnpath`, or `rnid` help output.

## How to use

1. Build rns-mcp: `cd rns-mcp && go test ./... && go build`.
2. Add the binary to your MCP client config as `reticulum`.
3. Call `list_topics` and `search_docs {query}` for the manual, or `rns_status` for local state.
4. Community tools and `fetch_page` use HTTPS with a cache.

## Examples

- "Search the Reticulum manual for `IFAC`."
- "Show the `rns_status` output for interface health."
- "Fetch the `nomadnet_context` prompt."
