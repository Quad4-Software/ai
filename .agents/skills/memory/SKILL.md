---
name: memory
description: >
  This skill exposes a persistent JSONL memory store through memory.
  Use when you need to store, recall, update, or forget tagged notes,
  people, destinations, tasks, and snippets across sessions.
metadata:
  server: memory
---

## When to use this skill

- You need facts to survive across sessions.
- You want tagged recall over notes, people, tasks, or snippets.
- You need a scratchpad with fuzzy search.

## How to use

1. Build memory: `cd mcp/memory && go test ./... && go build`.
2. Add the binary to your MCP client config as `memory`.
3. `remember {content, type, tags}` to store, `recall {query}` to
   search, `memory_status` to inspect.

## Examples

- "Remember that the gateway socket lives under XDG_RUNTIME_DIR."
- "Recall everything tagged `deploy`."
- "Forget these two stale memory ids."

# memory

memory is the one write-capable server in the set. It stores JSONL
records in `MEMORY_DIR` (default `~/.local/share/ai-memory`) as
`memories.jsonl` with dir 0700 and file 0600, written atomically via
tmp+rename.

## Types and limits

Types: `note`, `person`, `destination`, `task`, `not-do`, `doc`,
`snippet`. Caps: 64 KiB content, 20 tags x 64 chars (normalized to
lowercase hyphenated), 50 results max per query.

## Environment

- `MEMORY_DIR`: store location.
- `MCP_READ_ONLY=1` / `READ_ONLY=1` / `--read-only`: hides the three
  Write tools (`remember`, `update_memory`, `forget`).

## Tool inventory

| tool | args | purpose |
|---|---|---|
| `remember` | `content`, `type`, `tags`, `source` (Write) | store, id is sha256 + random suffix |
| `recall` | `query`, `type`, `tags`, `limit` (10/50) | fuzzy ranked, empty query lists recent |
| `memory_by_id` | `id` | fetch one |
| `update_memory` | `id`, `content`, `type`, `tags`, `source` (Write) | partial update |
| `forget` | `ids` (Write) | delete by id |
| `list_memory` | `type`, `tags`, `limit`, `offset` | filtered listing, newest first |
| `memory_tags` | none | tag counts |
| `memory_status` | none | count + data dir |

## Notes and quirks

- Secret-looking values are redacted in output only, via a
  `(token|key|password|secret|private)[:=]` pattern. Content is
  stored plaintext, so do not put real secrets in memories.
- Every operation loads and rewrites the whole file under a mutex,
  O(n) per call, fine for personal-scale stores.
- Corrupted lines are skipped silently. No dedupe on `remember`.
