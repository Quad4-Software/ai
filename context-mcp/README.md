# context-mcp

Token-efficient code access for agents working large files. Read a
signature skeleton first, then pull only the symbol bodies or line
windows you need. Every response reports an estimated token cost.

Inspired by Aider's repo map: skeleton over files, references over grep.

## Tools

- `file_outline {path}` - decls with line ranges, never bodies
- `read_symbol {path, name, number}` - one function/class body only
- `read_window {path, start, end}` - numbered lines, max 2000
- `repo_map {path, limit}` - files ranked by cross-file refs, top symbols
- `find_references {name, path, limit}` - word-boundary identifier uses
- `search_code {pattern, path, context, limit}` - regex search with context

## Env

- `MCP_REPO_ROOT` - repo root; falls back to nearest `.git` above cwd

Read-only. Path-jailed, symlink-safe. SPDX: 0BSD.

New: `search_symbols` (cached repo index, ranked) and `diff_symbols`
(map a git diff to enclosing declarations instead of reading it raw).
- `read_callsite {path, line}` - which symbol encloses a line
- `callers {name}` / `callees {path, name}` - heuristic call graph
