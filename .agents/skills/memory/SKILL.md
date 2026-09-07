---
name: memory
description: >
  This skill persists short-term agent memory through memory.
  Use when you need to remember, recall, update, or forget notes, people,
  destinations, tasks, or snippets.
metadata:
  server: memory
---

## When to use this skill

- You need to store a fact, person, or task across sessions.
- You want fuzzy recall from a large memory set.
- You need to update or delete an existing memory.

## How to use

1. Build memory: `cd mcp/memory && go test ./... && go build`.
2. Add the binary to your MCP client config as `memory`.
3. Call `remember {content}` with type and tags, then `recall {query}` for fuzzy search.
4. Use `memory_by_id {id}` to fetch or `update_memory` / `forget` to change.

## Examples

- "Remember that `rns` stores config in `~/.reticulum` with tag `config`."
- "Recall all notes about `LXMF` with type `note`."
- "Update memory 12 to mark the task as done."
