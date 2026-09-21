---
name: context
description: >
  This skill exposes token-efficient repo context through context. Use
  when you need file outlines, symbol reads, caller/callee maps, or
  diff-aware symbol lookup without loading whole files.
metadata:
  server: context
---

## When to use this skill

- You need a symbol body or file outline without pulling the file.
- You are tracing callers/callees before a refactor.
- You want `git diff` hunks mapped to the enclosing declarations.

## How to use

1. Build context: `cd mcp/context && go test ./... && go build`.
2. Add the binary to your MCP client config as `context`.
3. Start with `repo_map` or `file_outline`, then `read_symbol` or
   `read_window` for detail.

## Examples

- "Outline `internal/docs/docs.go` and read the `Search` symbol."
- "Find callers of `NewServer` under mcp/."
- "Map the staged diff to the functions it touches."

# context

context gives token-efficient access to repo code: outlines, symbol
bodies, references, and diff mapping. Every response carries an
`[est. tokens: N]` suffix (chars/4). Read-only, jailed to the root.

## Environment

- `MCP_REPO_ROOT`: explicit root. Otherwise walks up for a `.git` dir.
- `MCP_READ_ONLY=1` / `READ_ONLY=1` / `--read-only`: no-op here. All
  tools are read-only.

## Tool inventory

| tool | args | purpose |
|---|---|---|
| `file_outline` | `path` | signature skeleton, decls + line ranges |
| `read_symbol` | `path`, `name`, `number` (1) | one symbol body |
| `read_window` | `path`, `start`, `end` | numbered lines, max 2000-line window |
| `repo_map` | `path`, `limit` (30/200) | files ranked by cross-refs, top 12 symbols each |
| `find_references` | `name`, `path`, `limit` (40/300) | word-boundary identifier uses |
| `search_symbols` | `query`, `kind`, `path`, `limit` (20/100) | ranked symbol search over in-memory index |
| `diff_symbols` | `ref`, `staged` | git diff hunks mapped to enclosing decls |
| `read_callsite` | `path`, `line` | which symbol encloses a line |
| `callers` | `name`, `path`, `limit` (30/200) | heuristic `name(` callers outside the def |
| `callees` | `path`, `name`, `number` | calls inside a symbol resolved via index |
| `search_code` | `pattern`, `path`, `context` (3/20), `limit` | regex search with context lines |

## Notes and quirks

- Symbol extraction is regex-based. Outlines only support
  `.py .go .ts .js .vue .svelte`. Scans cover more extensions but
  resolution is heuristic, not LSP-grade.
- Files over 2 MiB are skipped. The index is in-memory and dies with
  the process.
- `diff_symbols` shells out to `git` with a 60 s timeout and needs
  git on PATH.
- Errors arrive as `error: ` tool content, not JSON-RPC errors.
