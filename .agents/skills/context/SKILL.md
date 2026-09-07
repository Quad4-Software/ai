---
name: context
description: >
  This skill provides token-efficient code access through context.
  Use when you need outlines, symbol bodies, repo maps, or references for
  large files without reading whole files.
metadata:
  server: context
---

## When to use this skill

- A file is too large to read in full.
- You need only declarations, one symbol, or a window of lines.
- You want a ranked repo map or cross-file references for a symbol.

## How to use

1. Build context: `cd mcp/context && go test ./... && go build`.
2. Add the binary to your MCP client config as `context`.
3. Call `file_outline {path}` for a skeleton, then `read_symbol {path, name}` for a body.
4. Use `repo_map {path}` or `find_references {name}` for cross-file context.

## Examples

- "Show the outline of `internal/mcp/server.go`."
- "Read the body of `NewServer` from `mcp/context/internal/mcp/server.go`."
- "Find all references to `HandleMessage` in the repo map."
