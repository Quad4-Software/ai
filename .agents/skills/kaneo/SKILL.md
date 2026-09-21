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
default.`list_workspaces` + `list_projects` discover ids the config
does not know yet.

## Tool map

| Tool | Write | Purpose |
|---|---|---|
| `auth_status` | | configured?, source, apiUrl, default project, known projects |
| `list_workspaces` | | workspaces the key can access |
| `use_project` | | switch session default project by name/slug/id |
| `list_projects` | | projects in a workspace |
| `get_project` | | one project by name/slug/id |
| `get_board` | | columns + compact task rows (no descriptions) |
| `list_columns` | | board columns with ids, positions, isFinal |
| `get_task` | | full task incl. description, labels |
| `find_tasks` | | title substring search, use before create for dedupe |
| `search` | | workspace-wide search (q, type, projectId, limit) |
| `list_labels` | | workspace labels |
| `get_task_comments` | | comments on a task |
| `get_task_relations` | | subtask/blocks/related links |
| `list_time_entries` | | time logged on a task |
| `get_task_activity` | | task event history |
| `github_app_info` | | instance GitHub App slug, or null when unconfigured |
| `github_repositories` | | repos reachable via the installed App |
| `get_github_integration` | | project's repo link, or null |
| `create_task` | yes | title, description, priority, status, dueDate, assignee |
| `update_task` | yes | partial update incl. assignee (user id or "none") |
| `move_task` | yes | column move, or cross-project via destinationProjectId |
| `add_comment`, `update_comment`, `delete_comment` | yes | comment CRUD |
| `set_label` | yes | attach/detach by label id or exact name |
| `create_label`, `update_label`, `delete_label` | yes | label CRUD (hex color) |
| `link_tasks`, `unlink_tasks` | yes | relations: subtask, blocks, related |
| `log_time`, `update_time_entry` | yes | ISO 8601. Omit endTime for a running timer |
| `create_project` | yes | name + slug (auto-derived), icon, description |
| `update_project` | yes | name, slug, icon, description, isPublic |
| `archive_project` | yes | hide or restore (archive=false), keeps data |
| `reorder_projects` | yes | sidebar order via [{id, position}] |
| `delete_project` | yes | permanent incl. all tasks, confirm first |
| `create_column`, `update_column` | yes | name, icon, color, isFinal |
| `reorder_columns` | yes | [{id, position}] |
| `delete_column` | yes | confirm first when it holds tasks |
| `github_verify` | yes | check App install + permissions on owner/repo |
| `connect_github` | yes | link project to owner/repo |
| `update_github_integration` | yes | isActive / commentTaskLinkOnGitHubIssue |
| `disconnect_github` | yes | unlink repo |
| `import_github_issues` | yes | issues -> tasks, skips linked ones |
| `delete_task` | yes | permanent, confirm first |

Write tools fail cleanly without a key. `MCP_READ_ONLY=1` or
`--read-only` hides them.

## Conventions

- Statuses: `backlog`, `to-do`, `in-progress`, `in-review`, `done`,
  `cancelled`. Priorities: `no-priority`, `low`, `medium`, `high`,
  `urgent`.
- GitHub integration needs a GitHub App on the server first
  (`GITHUB_APP_ID`, `GITHUB_PRIVATE_KEY`, `GITHUB_WEBHOOK_SECRET`,
  `GITHUB_APP_NAME` env vars on the API). `github_app_info` reports
  whether one is set. Linking endpoints need workspace
  manage_settings permission.
- When the user says "move X to in progress", use `update_task` with
  `status`, or `move_task`.
- Always `find_tasks` before `create_task` to avoid duplicates.
- Task numbers shown in the UI (`MEL-12`) are display-only. Tools use
  the long `id`.
