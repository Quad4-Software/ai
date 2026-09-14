# kaneo

Stdio MCP server for the [Kaneo](https://kaneo.app) project management
API. Works against cloud.kaneo.app or a self-hosted instance.

## Tools

Read-only:

- `auth_status` - whether an API key is configured and where it came
  from. The key itself is never shown.
- `list_projects` - projects in a workspace.
- `get_board` - a project's columns with compact task rows.
- `get_task` - one task with full description and labels.
- `find_tasks` - title-substring search for dedupe before creating.
- `list_labels` - workspace labels.
- `get_task_comments` - comments on a task.

Write tools (all gated on a configured API key, marked `Write` so
`--read-only` or `MCP_READ_ONLY=1` hides them):

- `create_task` - title, description, priority, status, due date.
- `update_task` - partial update via the granular endpoints.
- `move_task` - change column, or move across projects.
- `add_comment` - post a comment.
- `set_label` - attach or detach a label by id or exact name.
- `delete_task` - permanent; confirm with the user first.

## Configuration

Environment variables:

| Var | Purpose |
|---|---|
| `KANEO_API_KEY` | API key. Never returned in output. |
| `KANEO_API_URL` | API base, default `https://cloud.kaneo.app/api` |
| `KANEO_WORKSPACE_ID` | Default workspace |
| `KANEO_PROJECT_ID` | Default project |

Or `~/.config/kaneo/config.json`:

```json
{"apiUrl": "https://todo.example.com/api", "apiKey": "...", "workspaceId": "...", "projectId": "..."}
```

Environment wins over the file. A config file containing a key must be
`0600`; anything looser is rejected.

## Storing a key securely

Run the built binary's setup once, by hand:

```bash
./kaneo setup
```

It prompts on the terminal with echo disabled (platforms without `stty`
warn that input is visible) and writes the config file with owner-only
permissions. The key never touches shell history, logs, or MCP tool
output.

## MCP client entry

```json
{"command": "/path/to/mcp/kaneo/kaneo"}
```

Write access without a key: tools return a clear "no API key
configured" error. `MCP_READ_ONLY=1` hides all write tools entirely.

## Tests

`make test` or `go test ./...`. Tests use in-process `httptest`
servers only, no external network.
