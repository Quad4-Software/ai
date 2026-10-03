---
name: mediawiki
description: >
  This skill exposes one MediaWiki instance through mediawiki. Use when
  you need to search, read, or traverse a wiki that may sit behind
  Anubis, Cloudflare, or other anti-bot interstitials.
compatibility: stdio-mcp
metadata:
  server: mediawiki
---

## When to use this skill

- You need to search or read pages on a configured MediaWiki site.
- You want page metadata, backlinks, or category members.
- A wiki is behind an anti-bot interstitial (Anubis, Cloudflare,
  DDoS-Guard) and plain fetches get challenged.

## How to use

1. Build mediawiki: `cd mcp/mediawiki && go test ./... && go build`.
2. Add the binary to your MCP client config as `mediawiki` and set
   `MEDIAWIKI_BASE_URL` to the wiki root.
3. Call `status` first when calls fail: it reports reachability, the
   detected protection vendor, the active transport, and FlareSolverr
   health.
4. Use `search`, `page`, `info`, `backlinks`, and `category` for reads.

## Examples

- "Search the wiki for `cargo` and show the top 5 hits."
- "Fetch the Main Page as plain text."
- "List members of Category:Maintenance that are subcategories."
- "Why does `page` fail? Run `status`."

# mediawiki

Read-only stdio server for the MediaWiki Action API on exactly one
configured wiki. Six tools, no prompts. The client is built lazily, so
a missing `MEDIAWIKI_BASE_URL` never breaks tools/list, it fails at
first call.

## Tools

- `status`: probe the wiki. Returns `api_url`, `reachable`,
  `transport` (direct, anubis:<algorithm>, or flaresolverr),
  `protection`/`protection_bypassed`, FlareSolverr health, and site
  metadata (sitename, generator, base).
- `search {query, limit}`: full-text search. `limit` is 1-50, default
  10. Returns title, HTML-stripped snippet, size, wordcount, timestamp,
  plus `searchinfo`.
- `page {title, format, section}`: page content. `format` is `text`
  (default, rendered HTML stripped to readable text), `wikitext`, or
  `html`. `section` fetches one section index. Redirects are followed.
- `info {title}`: pageid, missing flag, length, last touched, full
  URL, and category membership.
- `backlinks {title, limit}`: pages linking to a title, redirect
  pages excluded. `limit` is 1-500, default 50.
- `category {name, limit, type}`: category members. `name` may omit
  the `Category:` prefix. `type` filters to `page`, `subcat`, or
  `file`. `limit` is 1-500, default 50.

Every response carries a `transport` field naming the path used.

## Environment

| Env var | Purpose |
|---|---|
| `MEDIAWIKI_BASE_URL` | Required. Wiki root, for example https://example.org. https required, http allowed for loopback only. No embedded credentials. |
| `MEDIAWIKI_API_URL` | Optional Action API endpoint. Default BASE_URL + /api.php. |
| `MEDIAWIKI_FLARESOLVERR_URL` | Optional FlareSolverr instance such as http://127.0.0.1:8191. Falls back to `FLARESOLVERR_URL`. Prefer 127.0.0.1 over localhost: podman port maps are IPv4-only and an ::1 dial may reset. |
| `MEDIAWIKI_COOKIES` | Optional "name=value; ..." cookies for the wiki host, for example a solved cf_clearance. Registered for redaction, never returned in output or errors. |
| `MEDIAWIKI_USER_AGENT` | Optional UA override. Pair it with MEDIAWIKI_COOKIES when pasting a browser clearance. |
| `MEDIAWIKI_TIMEOUT` | Optional per-request timeout in seconds, default 30. |

## Protection handling

The transport detects the interstitial vendor (cloudflare, anubis,
ddos-guard, sucuri, aws-waf, akamai, or generic) and acts:

- Anubis `fast` and `slow` proof-of-work plus `metarefresh` are solved
  natively, no external dependency.
- Everything else goes through FlareSolverr when configured. Issued
  cookies and the solver user agent are adopted for direct follow-up
  requests. If the clearance is bound to the solver egress IP, the
  solver-rendered body is used as-is (browser `<pre>` wrapping on bare
  JSON is stripped).
- With no solver configured, errors name the detected vendor and the
  env vars that would route around it. A `block` kind is a deny rule,
  not a solvable challenge.

## Notes and quirks

- Single-host jail: every request must stay on the configured host,
  so the transport cannot be used as an open proxy.
- Response bodies are capped at 8 MiB and rejected, not truncated.
- Challenge solving retries at most 3 rounds per request. The server
  tool-call timeout is 150 s because solving can take tens of seconds.
- Solver-issued cookies persist in memory for the life of the process,
  so a challenge is solved once per session.
- All tools are read-only GETs against api.php. Nothing writes to the
  wiki.
