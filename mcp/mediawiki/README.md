# mediawiki

Read-only MCP server for the MediaWiki Action API on one configured
wiki. It fetches through a challenge-aware transport: Anubis
proof-of-work (fast, slow, metarefresh) is solved natively, and
Cloudflare, DDoS-Guard, and other JS challenges route through a
FlareSolverr instance when one is configured. Cookies issued by a
solver persist in memory for the life of the process.

## Configuration

| Env var | Purpose |
|---|---|
| `MEDIAWIKI_BASE_URL` | Required. Wiki root, for example https://example.org. https required; http allowed for loopback only. |
| `MEDIAWIKI_API_URL` | Optional Action API endpoint override. Default is BASE_URL + /api.php. |
| `MEDIAWIKI_FLARESOLVERR_URL` | Optional. FlareSolverr instance such as http://127.0.0.1:8191. Falls back to `FLARESOLVERR_URL`. Prefer 127.0.0.1 over localhost: podman port maps are IPv4-only and an ::1 dial may reset. |
| `MEDIAWIKI_COOKIES` | Optional "name=value; ..." cookies for the wiki host, for example a solved cf_clearance. Never returned in output. |
| `MEDIAWIKI_USER_AGENT` | Optional UA override. Pair it with MEDIAWIKI_COOKIES when pasting a browser clearance. |
| `MEDIAWIKI_TIMEOUT` | Optional per-request timeout in seconds, default 30. |

## Protection handling

When a response looks like an interstitial the server detects the
vendor (Cloudflare, Anubis, DDoS-Guard, Sucuri, AWS WAF, Akamai, or
generic) and acts:

- Anubis `fast`/`slow` proof-of-work and `metarefresh` are solved
  locally with no external dependency.
- Everything else goes through FlareSolverr when configured. Issued
  cookies and the solver user agent are adopted for direct follow-up
  requests; if a clearance is bound to the solver egress IP, the
  solver-rendered body is used as-is.
- With no solver configured, errors name the detected vendor and the
  env vars that would route around it.

## Tools

All read-only:

- `status` - probe reachability, detected protection, transport in
  use, and FlareSolverr health
- `search` - full-text search (title, snippet, size, timestamp)
- `page` - page content as text, wikitext, or html; optional section
- `info` - existence, length, last touched, URL, categories
- `backlinks` - pages linking to a title
- `category` - category members, filterable by page/subcat/file

## Client config

```json
{
  "command": "/path/to/mcp/mediawiki/mediawiki",
  "env": {
    "MEDIAWIKI_BASE_URL": "https://example.org",
    "MEDIAWIKI_FLARESOLVERR_URL": "http://127.0.0.1:8191"
  }
}
```

License: 0BSD.
