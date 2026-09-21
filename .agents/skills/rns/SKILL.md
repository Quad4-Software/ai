---
name: rns
description: >
  This skill exposes the Reticulum manual and local utilities through
  rns. Use when you need searchable manual topics, local RNS state, or
  utility help.
metadata:
  server: rns
---

## When to use this skill

- You need Reticulum manual sections or the community wiki.
- You want local RNS state: status, path table, destination lookup.
- You need help with an rns utility or the config file.
- You are browsing the rns.recipes forum or interface directory.

## How to use

1. Build rns: `cd mcp/rns && go test ./... && go build`.
2. Add the binary to your MCP client config as `rns`.
3. `list_topics`/`search_docs` for the manual, `rns_status` for local
   state, community tools for forum and discussions.

## Examples

- "Search the Reticulum manual for announce mechanics."
- "Show the local path table, max 4 hops."
- "Lint my ~/.reticulum/config."

# rns

rns wraps the Reticulum manual (16 topics including the miraheze
community wiki), local rnsd utilities, community sources, and the
rngit git-over-Reticulum tool. No repo root needed.

## Docs tools (7)

`list_topics`, `get_topic`, `list_sections`, `get_section`,
`search_docs`, `fetch_page` (allowlisted to reticulum.network and
reticulum.miraheze.org), `wiki_welcome`. Fetches need network, and
the in-session cache never expires (declared `cacheTTL` is unused).

## Local utility tools (7)

Exec wrappers, 20 s timeout, 256 KiB output cap. Binaries must be on
PATH (`pipx install rns`):

- `rns_util_help {utility}`: `rnsd rnstatus rnpath rnprobe rnid rncp
  rnx rnsh rngit rnodeconf nomadnet`
- `rns_status`, `rns_path_table`, `rns_path_lookup {destination}`,
  `rns_destination_hash {identity, aspects}`, `rns_probe`
- `rns_config_check {path}`: lints sections, interface types,
  ifac_size, rpc_key. It is not repo-jailed and reads any user path
  (default `~/.reticulum/config`).

## Community tools (9 entries)

15 s timeout, 10 min cache: `unsigned_posts`, `forum_categories`,
`forum_threads`, `forum_latest`, `forum_search`, `community_read`
(allowlist: rns.recipes/forum/, unsigned.io/, github.com/markqvist/
Reticulum/discussions/), `github_discussions` (HTML scrape, fragile),
`github_search` (unauthenticated REST, 10 req/min),
`interface_directory`.

## rngit (Write)

`rns_rngit {subcommand, rns_url, args}`: subs `create release fork
mirror sync perms work info ls log show verify install`. URL must
match `^rns://[0-9a-fA-F]{32}(/seg){1,3}$` and the flag allowlist is
enforced. Hidden by `MCP_READ_ONLY=1`/`--read-only`.

## Prompts

`nomadnet_context` (static context) and `zen_review {design}` (Zen of
Reticulum finish gates).

## Notes and quirks

- Community and docs tools need network. Only the local utils and
  prompts work offline.
- `github_discussions` scrapes HTML and breaks when markup changes.
- README drift: it claims rngit is not wrapped (it is, as a Write
  tool) and says 15 manual chapters (there are 16).
