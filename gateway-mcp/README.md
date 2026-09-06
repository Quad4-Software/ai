# gateway-mcp

One MCP server that multiplexes every other server in your client
config behind five tools, so agents load a tiny tool surface instead
of dozens of schemas. Same idea as Atlassian's mcp-compressor: small
index first, schemas on demand.

## Tools

- `servers` - which children are configured and reachable
- `tools {server}` - compact index: `<server>.<name> - one-line desc`
- `tool_schema {name}` - full schema for one tool, e.g. `reticulum.list_topics`
- `invoke {name, arguments}` - call a tool through the gateway
- `gateway_stats` - per-server calls, errors, respawns, avg latency

## Config

Reads `GATEWAY_CONFIG` or `~/.config/mcp/mcp.json`. Any
`command`-based server is proxied; `url`-only and `disabled` entries
and the gateway itself are skipped. Children spawn lazily on first
use and respawn once on transport failure.

Register it instead of the individual servers:

```json
"gateway": {"command": "/path/gateway-mcp/gateway-mcp"}
```

## Shared daemon mode

Stdio MCP is one process per client window, so by default every
window gets its own gateway and its own set of children. To share one
gateway across all windows, run it as a daemon and point clients at a
thin `--attach` shim:

```json
"gateway": {"command": "/path/gateway-mcp/gateway-mcp", "args": ["--attach"]}
```

`--attach` bridges stdio to a unix socket and auto-starts the daemon
(`--daemon`) on first connect. The socket defaults to
`$XDG_RUNTIME_DIR/gateway-mcp.sock` (or `/tmp/gateway-mcp-$UID.sock`),
is owner-only, and can be overridden with `--socket <path>`. The
daemon writes `<socket>.log` and removes the socket on SIGTERM/SIGINT.
All windows then share one gateway, one set of child processes, and
one `gateway_stats` view. Daemon state (tool lists, child processes)
persists across client sessions until the daemon is stopped.

License: 0BSD.
