# gateway-mcp

One MCP server that multiplexes every other server in your client
config behind four tools, so agents load a tiny tool surface instead
of dozens of schemas. Same idea as Atlassian's mcp-compressor: small
index first, schemas on demand.

## Tools

- `servers` - which children are configured and reachable
- `tools {server}` - compact index: `<server>.<name> - one-line desc`
- `tool_schema {name}` - full schema for one tool, e.g. `reticulum.list_topics`
- `invoke {name, arguments}` - call a tool through the gateway

## Config

Reads `GATEWAY_CONFIG` or `~/.config/mcp/mcp.json`. Any
`command`-based server is proxied; `url`-only and `disabled` entries
and the gateway itself are skipped. Children spawn lazily on first
use and respawn once on transport failure.

Register it instead of the individual servers:

```json
"gateway": {"command": "/path/gateway-mcp/gateway-mcp"}
```

License: 0BSD.
- `gateway_stats` - per-server calls, errors, respawns, avg latency
