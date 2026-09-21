---
name: agents
description: >
  This skill exposes a repo's agent knowledge graph through agents.
  Use when you need to list skills, read conventions, search .agents docs,
  find module owners, or resolve which skill fits a task.
metadata:
  server: agents
---

## When to use this skill

- You need to know what skills or conventions are available in a repo.
- You want to read .agents/ docs or AGENTS.md without leaving your editor.
- You are choosing a skill for a new task.

## How to use

1. Build agents: `cd mcp/agents && go test ./... && go build`.
2. Add the binary to your MCP client config as `agents`.
3. Call `repo_root` to confirm the jailed root, then `list_skills` or `search_docs`.
4. Use `skill_for {task}` or `fuzzy_search {query}` when the right skill is not obvious.

## Examples

- "List all .agents skills and pick the best one for memory handling."
- "Search .agents/ and docs/ for the path-jailing convention."
- "Show the module ownership for context."

# agents

agents exposes a repo's `.agents` knowledge graph and docs as MCP
tools. It is read-only and jailed to the detected repo root.

## Environment

- `MCP_REPO_ROOT`: explicit root. Otherwise the server walks up from
  cwd looking for a directory containing `.agents/`.
- `MCP_READ_ONLY=1` or `READ_ONLY=1` or `--read-only`: standard
  read-only gate. This server has no Write-marked tools, so it is a
  no-op here.

## Tool inventory

| tool | args | purpose |
|---|---|---|
| `repo_root` | none | detected root |
| `list_skills` | none | `.agents/skills/*/SKILL.md` entries |
| `get_skill` | `name` | SKILL.md by dir name.`/` and `\` rejected |
| `list_conventions` | none | `.agents/conventions/` plus README/overview/module-ownership/AGENTS.md |
| `read_file` | `path` | any repo file, jailed, 1 MiB cap |
| `search_docs` | `query`, `limit` (30/200) | case-insensitive regex over `.agents/`, `docs/`, `AGENTS.md`. Md/mdc/txt only |
| `tree` | `path`, `depth` (2/6) | dir listing. Skips node_modules/.git/dist/vendor/build |
| `skill_for` | `task`, `limit` (5/15) | ranks skills by word overlap on name+description |
| `module_owners` | none | `.agents/module-ownership.md` |
| `ask` | `question`, `limit` (8/20) | top-3 skill bodies plus fuzzy snippets |
| `fuzzy_search` | `query`, `limit` (20/100) | token search over `.agents`, `docs`, `AGENTS.md` |

## Notes and quirks

- Path jailing resolves symlinks with `EvalSymlinks`. Traversal is
  rejected. File reads are capped at 1 MiB.
- Errors arrive as tool content prefixed `error: ` with
  `isError: true`, not JSON-RPC errors.
- Oversized tool output (>8 MiB) is an error, never truncated.
- README drift: the `tree` tool is implemented but not documented
  there.
