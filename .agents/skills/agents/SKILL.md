---
name: agents
description: >
  This skill exposes a repo's agent knowledge graph through agents-mcp.
  Use when you need to list skills, read conventions, search .agents docs,
  find module owners, or resolve which skill fits a task.
metadata:
  server: agents-mcp
---

## When to use this skill

- You need to know what skills or conventions are available in a repo.
- You want to read .agents/ docs or AGENTS.md without leaving your editor.
- You are choosing a skill for a new task.

## How to use

1. Build agents-mcp: `cd agents-mcp && go test ./... && go build`.
2. Add the binary to your MCP client config as `agents`.
3. Call `repo_root` to confirm the jailed root, then `list_skills` or `search_docs`.
4. Use `skill_for {task}` or `fuzzy_search {query}` when the right skill is not obvious.

## Examples

- "List all .agents skills and pick the best one for memory handling."
- "Search .agents/ and docs/ for the path-jailing convention."
- "Show the module ownership for context-mcp."
