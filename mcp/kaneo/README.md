# kaneo

Stdio MCP server for the [Kaneo](https://kaneo.app) project management
API. Works against cloud.kaneo.app or a self-hosted instance.

## Tools

Read-only:

- `auth_status` - whether an API key is configured, where it came
  from, and which projects are configured. The key itself is never
  shown.
- `list_workspaces` - workspaces (organizations) the key can access.
- `list_projects` - projects in a workspace.
- `get_project` - one project by name, slug, or id.
- `get_board` - a project's columns with compact task rows.
- `list_columns` - a project's board columns.
- `get_task` - one task with full description and labels.
- `find_tasks` - title-substring search for dedupe before creating.
- `search` - workspace-wide search over tasks, projects, comments,
  and activities.
- `list_labels` - workspace labels.
- `get_task_comments` - comments on a task.
- `get_task_relations` - a task's subtask/blocks/related links.
- `list_time_entries` - time logged on a task.
- `get_task_activity` - a task's event history.
- `use_project` - set the session default project by name, slug, or id.
- `github_app_info` - the instance's GitHub App slug, or empty when
  the admin has not configured one.
- `github_repositories` - repos reachable through the installed App.
- `get_github_integration` - a project's repo link, or null.

Write tools (all gated on a configured API key, marked `Write` so
`--read-only` or `MCP_READ_ONLY=1` hides them):

- `create_task` - title, description, priority, status, due date,
  assignee.
- `update_task` - partial update via the granular endpoints; assignee
  takes a user id or `none` to unassign.
- `move_task` - change column, or move across projects.
- `add_comment`, `update_comment`, `delete_comment` - comment CRUD.
- `set_label` - attach or detach a label by id or exact name.
- `create_label`, `update_label`, `delete_label` - label CRUD.
- `link_tasks`, `unlink_tasks` - task relations.
- `log_time`, `update_time_entry` - time tracking.
- `create_project`, `update_project`, `archive_project`,
  `reorder_projects`, `delete_project` - project lifecycle.
- `create_column`, `update_column`, `reorder_columns`,
  `delete_column` - board columns.
- `github_verify`, `connect_github`, `update_github_integration`,
  `disconnect_github`, `import_github_issues` - repo linking and
  issue import. These need a GitHub App configured on the server
  (`GITHUB_APP_ID`, `GITHUB_PRIVATE_KEY`, `GITHUB_WEBHOOK_SECRET`,
  `GITHUB_APP_NAME`) and workspace manage_settings permission.
- `delete_task` - permanent; confirm with the user first.

## Configuration

Environment variables:

| Var | Purpose |
|---|---|
| `KANEO_API_KEY` | API key. Never returned in output. |
| `KANEO_API_URL` | API base, default `https://cloud.kaneo.app/api` |
| `KANEO_WORKSPACE_ID` | Default workspace |
| `KANEO_PROJECT_ID` | Default project |

Or `~/.config/kaneo/config.json` (no key material ever lands here):

```json
{
  "apiUrl": "https://todo.example.com/api",
  "keyBackend": "secret-service",
  "defaultProject": "melovian",
  "projects": [
    {"name": "Melovian", "slug": "melovian", "workspace": "Quad4",
     "workspaceId": "...", "projectId": "..."}
  ]
}
```

Environment wins over the file. Tools that take `projectId` or
`workspaceId` also accept a configured project name or slug, and fall
back to `defaultProject` when omitted.

## Storing a key

The API key lives in the OS keyring, not in the config file. Two
backends are supported via their CLIs, no extra dependencies:

- `secret-tool` (libsecret Secret Service: GNOME Keyring, KWallet)
- `pass` (the standard unix password store)

Run the built binary's setup once, by hand:

```bash
./kaneo setup
```

It prompts for the API URL, reads the key with terminal echo disabled,
stores it in the first available backend, then lists the workspaces
and projects the key can see so you can pick a default project. The
key never touches shell history, logs, config files, or MCP tool
output.

A legacy plaintext `apiKey` left in `config.json` is migrated into the
keyring automatically on startup and stripped from the file.

## MCP client entry

```json
{"command": "/path/to/mcp/kaneo/kaneo"}
```

Write access without a key: tools return a clear "no API key
configured" error. `MCP_READ_ONLY=1` hides all write tools entirely.

## Tests

`make test` or `go test ./...`. Tests use in-process `httptest`
servers only, no external network.
