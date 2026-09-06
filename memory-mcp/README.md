# memory-mcp

Persistent short-term memory for agents. Stores notes, people, RNS
destinations, tasks, things to avoid, and snippets from docs or web
searches. Recall uses fuzzy ranking over content, tags, source, and id.

## Config

`MEMORY_DIR` defaults to `~/.local/share/ai-memory`. The directory is
created with owner-only permissions and all files are kept inside it.

## Tools

- `remember` - store a memory with optional type, tags, and source.
- `recall` - fuzzy-search memories with optional type/tag filters.
- `memory_by_id` - fetch one memory.
- `update_memory` - change content, type, tags, or source.
- `forget` - delete memories by id.
- `list_memory` - list with type/tag filters and pagination.
- `memory_tags` - all tags with counts.
- `memory_status` - count and data directory.

Valid types: `note`, `person`, `destination`, `task`, `not-do`, `doc`,
`snippet`. Content is capped at 64 KiB, results at 50 entries, tags at
20 per memory and 64 characters each.

## Use

    make test
    make build

Then add `memory-mcp` to `~/.config/mcp/mcp.json`.

License: 0BSD.
