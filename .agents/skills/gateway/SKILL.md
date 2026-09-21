---
name: gateway
description: >
  This skill multiplexes other stdio MCP servers through gateway. Use
  when you want one MCP endpoint that proxies tools from many
  configured servers, with lazy spawning and per-server stats.
metadata:
  server: gateway
---

## When to use this skill

- You want one MCP config entry instead of N server entries.
- You need a unified tool index across several stdio servers.
- You want daemon/socket or SSE access to a set of MCP servers.

## How to use

1. Build gateway: `cd mcp/gateway && go test ./... && go build`.
2. Point `GATEWAY_CONFIG` at an mcp.json (default
   `~/.config/mcp/mcp.json`) listing the child servers.
3. Call `servers` to verify children, `tools` for the index, and
   `invoke {name: "<server>.<tool>", arguments: {...}}` to proxy.

## Examples

- "List which child servers are reachable and their tool counts."
- "Invoke `context.file_outline` on internal/docs/docs.go."
- "Show per-server call stats after a session."

# gateway

gateway multiplexes stdio MCP servers behind five tools. Children
spawn lazily on first use, calls are serialized per child, and a
failed transport gets one respawn. Child stderr is discarded.

## Configuration

- `GATEWAY_CONFIG`: path to the MCP config JSON. Default
  `~/.config/mcp/mcp.json`. Entries with only `url`, `disabled`
  flags, or a self-reference to gateway are skipped.
- `--socket` (default `$XDG_RUNTIME_DIR/gateway.sock` or
  `/tmp/gateway-$UID.sock`), `--attach` (stdio-to-socket shim that
  auto-spawns the daemon), `--daemon`, `--read-only`.
- `HTTP_PORT` exposes `GET /`, `GET /healthz`, `GET /sse`, and
  `POST /messages?session=<id>`.

Daemon mode: socket chmod 0600, log at `<socket>.log`, socket
removed on SIGTERM/SIGINT.

## Tool inventory

| tool | args | purpose |
|---|---|---|
| `servers` | none | children + reachability + tool count |
| `tools` | `server` (opt) | compact `<server>.<tool>` index |
| `gateway_stats` | none | per-server calls, errors, spawns, avg_ms |
| `tool_schema` | `name` | full schema for one `<server>.<tool>` |
| `invoke` | `name`, `arguments` | proxy a `tools/call` to a child |

## Notes and quirks

- Gateway tools carry no Write flag, so `--read-only` does not block
  `invoke` of a child's mutating tool directly. Protection relies on
  children inheriting `MCP_READ_ONLY` via the environment. Set
  `MCP_READ_ONLY=1` in the gateway environment before relying on it.
- Each `/sse` connection builds its own child set rather than
  sharing the daemon's children.
- Response matching is by `"id":N` substring over up to 10000 lines
  with no read deadline. A hung child stalls until the server-side
  30 s tool timeout fires.
- Child `tools/list` is cached for the process lifetime. Restarting
  a child does not refresh it.
