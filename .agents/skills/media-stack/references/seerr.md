# Seerr reference

Seerr is the unified successor to Overseerr and Jellyseerr. Both upstream
projects were merged into `seerr-team/seerr` in Feb 2026. Overseerr is
archived (last release v1.35.0, Feb 2026, which exists only to prep the
migration). Jellyseerr's last branded release was v2.7.3 (Aug 2025) and
its Docker image is marked "do not use, migrate to seerr/seerr".

- **Latest stable:** v3.4.1 (July 2026). v3.4.0 fixed CVE-2026-73291, a
  path-traversal-to-RCE in the ImageProxy. Run 3.4.x or later.
- **Port:** 5055 (env `PORT`). Healthcheck: `GET /api/v1/settings/public`.
- **Migration:** pointing the Seerr image at an existing Overseerr or
  Jellyseerr config volume migrates the SQLite DB on first start. The
  `jellyseerr`/`overseerr` image tags forward to `seerr/seerr` builds.
- **Media server binding:** one Seerr instance binds to **one** media
  server: Plex, Jellyfin, or Emby (a `MediaServerType` enum, chosen at
  setup). Simultaneous multi-server is not supported. For a Plex+Jellyfin
  household, run two Seerr instances.
- Jellyfin/Emby **Quick Connect** login was added in v3.4.0.

## Auth

Two models, both under `/api/v1`:

1. **`X-Api-Key` header.** The full admin key lives in Settings -> General
   (pin it via the `API_KEY` env var). Calls with `X-Api-Key` act as
   admin (user ID 1) unless an `X-API-User` header names another user ID.
2. **Cookie session** (`connect.sid`) from `POST /auth/plex`,
   `POST /auth/jellyfin` (covers Jellyfin and Emby), or
   `POST /auth/local`.

API spec: `seerr-api.yml` in the repo, served through
https://docs.seerr.dev/api/seerr-api/.

## Endpoint map

All paths are relative to `/api/v1`.

- `GET /status`, `GET /status/appdata` (public).
- Requests: `GET/POST /request`, `GET/PUT/DELETE /request/{id}`,
  `POST /request/{id}/approve`, `/decline`, `/retry`,
  `POST /request/{requestId}/{status}`, `GET /request/count`.
- Users: `GET/POST /user`, `GET/PUT/DELETE /user/{id}` plus
  `/user/{id}/settings` and quota subpaths. `GET /auth/me`,
  `POST /auth/logout`,
  `POST /auth/jellyfin/quickconnect/initiate|check`.
- Discovery: `GET /search`, `/discover/movies`, `/discover/tv`,
  `/movie/{id}`, `/tv/{id}`, `/collection/{id}`, `/person/{id}`,
  `/watchlist`, `/media`, `GET/POST /issue`, `/blocklist`.
- Services/settings: `GET/POST /settings/radarr`, `POST
  /settings/radarr/test` (same shape for `/settings/sonarr`),
  `GET /service/radarr`, `GET /service/sonarr`.

### Example: approve a request

```bash
# Find pending requests.
curl -s "http://seerr:5055/api/v1/request?filter=pending" \
  -H "X-Api-Key: $SEERR_KEY"

# Approve request 42.
curl -s -X POST "http://seerr:5055/api/v1/request/42/approve" \
  -H "X-Api-Key: $SEERR_KEY"
```

## Wiring to the *arr apps

Settings -> Services takes one Radarr and one Sonarr entry (plus optional
separate 4K entries per type). Each entry holds hostname, port, SSL flag,
API key, URL base, and a "default server" flag. The test button calls the
*arr's `/system/status` and returns its quality profiles and root folders
so they can be picked in the UI. Only Radarr/Sonarr **v3 and v4** are
supported. Lidarr and Prowlarr are not wired here.

On approval, Seerr creates the item through the *arr's add endpoint, so
the movie/series appears in the *arr library monitored and searched.

## Notifications

Per-user and per-event toggles across these agents:

- **Email** (SMTP).
- **Discord** via webhook URL. Supports role IDs, thread IDs (v3.4.0+),
  and multiple Discord IDs (v3.3.0+).
- **Generic webhook** with a custom JSON payload. Template variables:
  `{{notification_type}}`, `{{media}}`, `{{request}}`, `{{extra}}`.
  Custom headers and dynamic URL placeholders supported since v3.0.0.
- **Gotify**, **Pushover**, **Telegram**.

## Operational notes

- State is a SQLite DB in `/app/config`. Back it up before upgrades. There
  is no downgrade path.
- The container image is `seerr/seerr` (also published as
  `ghcr.io/seerr-team/seerr`). The `fallenbagel/jellyseerr` and
  `sct/overseerr` images are frozen.
- Behind a reverse proxy, Seerr wants the full origin passed through.
  WebSocket-based notifications go over the same port.
- Seerr does not manage downloads. If a request is approved but nothing
  downloads, the failure is in the *arr layer, not Seerr. Check the *arr's
  queue and indexer health first.
