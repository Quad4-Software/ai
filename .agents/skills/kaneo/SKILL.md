---
name: kaneo
description: Kaneo task board access via the kaneo MCP server. Use when you need to list, create, update, move, or comment on Kaneo tasks, or sync a task list to a board.
---

# kaneo

Stdio MCP server at `mcp/kaneo`. Wraps the Kaneo REST API for cloud or
self-hosted instances (todo.quad4.io is the local instance).

## Auth

The API key never appears in tool output and is never stored in the
config file. Sources, in order:

1. `KANEO_API_KEY` env var
2. OS keyring via `secret-tool` (libsecret) or `pass`, stored by
   `mcp/kaneo/kaneo setup`, which prompts on the terminal with echo
   off, then autodetects workspaces and projects for a default picker

A legacy plaintext `apiKey` in `~/.config/kaneo/config.json` is
migrated into the keyring on startup and stripped from the file.

`KANEO_API_URL` / `apiUrl` defaults to `https://cloud.kaneo.app/api`.
`KANEO_WORKSPACE_ID` and `KANEO_PROJECT_ID` supply defaults so most
calls need no ids. `auth_status` reports what is configured.

## Multiple projects

`config.json` can hold a `projects` list (name, slug, workspaceId,
projectId). Anywhere a tool takes `projectId` it also accepts a
configured project name or slug. `use_project` switches the session
default; `list_workspaces` + `list_projects` discover ids the config
does not know yet.

## Tool map

| Tool | Write | Purpose |
|---|---|---|
| `auth_status` | | configured?, source, apiUrl, default project, known projects |
| `list_workspaces` | | workspaces the key can access |
| `use_project` | | switch session default project by name/slug/id |
| `list_projects` | | projects in a workspace |
| `get_board` | | columns + compact task rows (no descriptions) |
| `get_task` | | full task incl. description, labels |
| `find_tasks` | | title substring search, use before create for dedupe |
| `list_labels` | | workspace labels |
| `get_task_comments` | | comments on a task |
| `create_task` | yes | title, description, priority, status, dueDate |
| `update_task` | yes | partial update: title, description, priority, status, dueDate |
| `move_task` | yes | column move, or cross-project via destinationProjectId |
| `add_comment` | yes | post markdown comment |
| `set_label` | yes | attach/detach by label id or exact name |
| `delete_task` | yes | permanent, confirm first |

Write tools fail cleanly without a key. `MCP_READ_ONLY=1` or
`--read-only` hides them.

## Conventions

- Statuses: `backlog`, `to-do`, `in-progress`, `in-review`, `done`,
  `cancelled`. Priorities: `no-priority`, `low`, `medium`, `high`,
  `urgent`.
- When the user says "move X to in progress", use `update_task` with
  `status`, or `move_task`.
- Always `find_tasks` before `create_task` to avoid duplicates.
- Task numbers shown in the UI (`MEL-12`) are display-only; tools use
  the long `id`.
