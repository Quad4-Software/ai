# mcp-scaffold

Template for new MCP servers in this toolkit. Not a server itself.

## Use

```sh
cp -r mcp-scaffold my-mcp
cd my-mcp
# rename module + binary:
#   go.mod: module quad4.io/mcp/my-mcp
#   internal/mcp import path in main.go
#   NewServer("my-mcp", ...)
#   .gitignore: binary name
go build -o my-mcp .
go test ./...
```

## What the template gives you

- `internal/mcp` — hardened stdio JSON-RPC layer: protocol negotiation
  (echoes known client versions), sorted cacheable `tools/list`
  (`listChanged: false`), `inputExamples`, experimental Tasks
  (`tasks/list|get|result|cancel`, `task` param on `tools/call`),
  panic recovery per call, `-32700`/`-32602`/`-32601`/`-32603` codes,
  notification silence, clean EOF exit, buffered synchronized output.
- `main.go` — schema helpers (`obj`, `strArg`, `intArg`) and two
  example tools showing the validation and error conventions.
- `main_test.go` — table-free handler tests for both tools.
- `go.mod` — Go 1.27, zero dependencies. Keep it that way.

## Rules for new servers

- Stdlib only, or one justified dependency in go.mod.
- Read-only tools by default. Mutators need an explicit policy.
- Bound every result. Say when output is capped and why.
- Jail path arguments. Allowlist hosts and commands. Never shell out
  with a string.
- Errors name the missing argument and the valid values.
- SPDX `0BSD` header on every file.
- Register in `~/.config/mcp/mcp.json`.

License: 0BSD.
