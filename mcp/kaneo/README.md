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
- `get_board` - a project's columns with compact task rows.
- `get_task` - one task with full description and labels.
- `find_tasks` - title-substring search for dedupe before creating.
- `list_labels` - workspace labels.
- `get_task_comments` - comments on a task.
- `use_project` - set the session default project by name, slug, or id.

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
