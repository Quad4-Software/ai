# agents-mcp

Stdio MCP server exposing a repo's agent knowledge graph and docs. Read-only,
path-jailed to the repo root (symlinks resolved). Stdlib only.

Root resolution: `MCP_REPO_ROOT` env, else nearest ancestor with `.agents/`.

## Tools

- `repo_root` - detected repo root
- `list_skills` / `get_skill {name}` - `.agents/skills/*/SKILL.md`
- `list_conventions` - `.agents/conventions/` plus top-level agent docs
- `read_file {path}` - any repo file, jailed, 1 MiB cap
- `search_docs {query, limit}` - regex over .agents/, docs/, AGENTS.md
- `module_owners` - `.agents/module-ownership.md`
- `skill_for {task}` - rank .agents skills against a task description
