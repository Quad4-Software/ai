---
name: gateway
description: >
  This skill multiplexes MCP servers through gateway.
  Use when you want one small tool surface and lazy child spawn for many
  configured servers.
metadata:
  server: gateway
---

## When to use this skill

- You want to load one MCP server instead of many.
- You need a tool index and on-demand schemas for all configured children.
- You want shared daemon mode across client windows.

## How to use

1. Build gateway: `cd mcp/gateway && go test ./... && go build`.
2. Add the binary to your MCP client config as `gateway` or use `--attach` for daemon mode.
3. Call `servers` to list children, `tools {server}` for an index, and `tool_schema {name}` for a schema.
4. Invoke `invoke {name, arguments}` to run a child tool through the gateway.

## Examples

- "List configured servers and show the tools for `reticulum`."
- "Fetch the schema for `reticulum.list_topics`."
- "Call `rns.search_docs` through the gateway for `interface types`."
