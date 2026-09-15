# melovian

MCP server for a running Melovian instance: library search and stats,
playlist and smart-playlist management, metadata lookups and autofix,
extension inspection and settings, and optional SearXNG web search.

## Configuration

| Env var | Purpose |
|---|---|
| `MELOVIAN_URL` | Base URL of the instance. https required; http allowed for loopback only. |
| `MELOVIAN_USERNAME` | Optional. With `MELOVIAN_PASSWORD`, logs in over `/api/auth/login` on first request. |
| `MELOVIAN_PASSWORD` | Optional. Session cookie is held in memory only and never returned in output. |
| `MELOVIAN_SEARXNG_URL` | Optional. Enables `web_search` against a SearXNG instance. |

## Tools

Read-only:

- `status`, `library_stats`
- `search`, `artists`, `albums`, `album_tracks`, `genres`,
  `genre_tracks`, `random_tracks`, `track`, `similar_tracks`, `starred`
- `playlists`, `playlist`, `smart_playlist_support`
- `metadata_summary`, `metadata_tracks`, `metadata_lookup`,
  `metadata_suggestions`
- `extensions`, `extension_registry`, `extension_settings`
- `web_search` (SearXNG, env-gated)
- `scaffold_extension` (returns source files for a new registry
  extension; nothing written to disk)

Mutating (disabled under `--read-only` or `MCP_READ_ONLY=1`):

- `playlist_create`, `playlist_rename`, `playlist_delete`,
  `playlist_set_tracks`, `playlist_add_track`, `playlist_remove_track`
- `smart_playlist_create`
- `metadata_autofix`
- `extension_enable`, `extension_set_settings`,
  `extension_install_remote`

`playlist_set_tracks` and `playlist_add_track` accept track IDs and
resolve full track records through the API automatically.

## Client config

```json
{
  "command": "/path/to/mcp/melovian/melovian",
  "env": {
    "MELOVIAN_URL": "http://localhost:4533",
    "MELOVIAN_USERNAME": "you",
    "MELOVIAN_PASSWORD": "secret"
  }
}
```

License: 0BSD.
