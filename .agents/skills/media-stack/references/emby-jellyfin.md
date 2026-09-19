# Emby and Jellyfin reference

Jellyfin forked from Emby at 3.5.2 in 2018, when Emby closed its source.
The API shape stayed nearly identical: same endpoint families, same auth
scheme names. Anything written against one port to the other with mostly
header-name changes.

| | Emby | Jellyfin |
|---|---|---|
| Latest stable (Sept 2026) | 4.9.5.0 (May 2026) | 10.11.11 (June 2026) |
| Beta line | 4.10.0.x | 12.0 RC (renumbered, drops "10.") |
| Source | Closed. `Emby.Releases` repo ships binaries + changelogs | Open (GPLv2) |
| HTTP port | 8096 | 8096 |
| HTTPS port | 8920 | 8920 (when configured) |
| LAN discovery | UDP 7359 | UDP 7359 |

## Auth

Both use the same scheme names but different recommended headers.

**Jellyfin (current, 10.11.x):**

- Preferred: `Authorization: MediaBrowser Client="<app>", Device="<dev>",
  DeviceId="<id>", Version="<ver>", Token="<token>"`. The scheme name is
  literally `MediaBrowser`.
- Token-only fallback: `?ApiKey=<key>` query param.
- Legacy headers still work but are gated by `EnableLegacyAuthorization`
  in `system.xml` (default true, slated to flip off and be removed in the
  12.x line): `X-Emby-Token`, `X-MediaBrowser-Token`,
  `X-Emby-Authorization`, `api_key` query param, and the `Emby` scheme.
- Login: `POST /Users/AuthenticateByName` with `{"Username","Pw"}` in
  JSON returns `{User, SessionInfo, AccessToken, ServerId}`. Admin API
  keys come from Dashboard -> API Keys or `POST /Auth/Keys?app=<name>`.

**Emby:**

- `X-Emby-Token: <key>` header, `?api_key=` query param, or the
  `Authorization` header with the `Emby` scheme:
  `Authorization: Emby UserId="...", Client="...", Device="...", DeviceId="...",
  Version="...", Token="..."`.
- `POST /Users/AuthenticateByName` issues session tokens the same way.

For new automation against Jellyfin, use the `MediaBrowser` scheme or
`?ApiKey=`, not the legacy `X-Emby-Token` headers.

## Endpoint families

Same shape on both. A short map:

- `GET /System/Info`: server version, ID, OS. Public-ish.
- `POST /Library/Refresh`: kick a full library scan (admin).
- `GET /Library/MediaFolders`: configured libraries.
- `GET /Users`, `GET /Users/{id}`, `POST /Users/{id}/Authenticate`.
- `GET /Items`, `GET /Users/{userId}/Items`: library queries with
  `Recursive`, `IncludeItemTypes`, `ParentId`, `Filters`, `Fields`,
  `SortBy`, `Limit`/`StartIndex` params.
- `GET /Items/{id}/PlaybackInfo`: transcoding/direct-play decision data.
- `GET /Sessions`: active sessions. `POST /Sessions/{id}/Playing/...` and
  `/Sessions/{id}/Command` remote-control them.
- `POST /Items/{id}/Refresh`: rescan one item.
- `GET /Shows/{id}/Seasons`, `/Shows/{id}/Episodes`.
- `GET /QuickConnect/...`: Quick Connect pairing, built into Jellyfin and
  used by Seerr for login.
- `GET /DisplayPreferences`, `/UserViews`, `/Playstate` for user state.

### Example: scan the library and check sessions

```bash
# Jellyfin
curl -s -X POST "http://jellyfin:8096/Library/Refresh" \
  -H "Authorization: MediaBrowser Client=\"ops\", Device=\"scripts\", DeviceId=\"ops-1\", Version=\"1.0\", Token=\"$JF_KEY\""

curl -s "http://jellyfin:8096/Sessions" \
  -H "Authorization: MediaBrowser Token=\"$JF_KEY\", Client=\"ops\", Device=\"scripts\", DeviceId=\"ops-1\", Version=\"1.0\""
```

## Jellyfin specifics

- **Hardware transcoding** via bundled jellyfin-ffmpeg: Intel QSV, NVIDIA
  NVENC/NVDEC, AMD AMF, VA-API (Linux), VideoToolbox (macOS), RKMPP
  (Rockchip), V4L2. Full-pipeline accel (scale, deinterlace, tonemap,
  subtitle burn-in) on Intel/AMD/Nvidia since 10.8. Dolby Vision tonemap
  on RKMPP since 10.11.
- **Trickplay** (seek thumbnails) is native since 10.9. **Intro skipping**
  is a plugin (`intro-skipper` via manifest
  `https://intro-skipper.org/manifest.json`), usually paired with the
  File Transformation plugin for the web-UI skip button.
- **Plugins** install from repository manifest URLs under Dashboard ->
  Plugins -> Repositories. Official repo:
  `https://repo.jellyfin.org/files/plugin/manifest.json`. Unstable:
  `https://repo.jellyfin.org/files/plugin-unstable/manifest.json`. The
  server filters the catalog by `targetAbi`, so a major upgrade hides
  incompatible plugins.
- **Database path:** `/config/data` (SQLite on EF Core since 10.9:
  `library.db` plus the EF migration history). `system.xml`,
  `encoding.xml`, and `logging.default.json` live in the same config
  tree.
- **Upgrade path:** 10.9 started the EF Core rewrite with a one-way
  migration and auto `library.db.bak` backups. 10.10 required stepping
  through 10.10.z for the EF migrations. 10.11 re-added a legacy
  migration path so pre-10.10 installs can upgrade directly (PR 14051).
  The 10.11 rollout shipped several rapid security/regression fixes,
  so pin to a recent 10.11.x.
- **12.0 renumbers the project** (no "10." prefix) and moves to .NET 10.
  Plugin ABI breaks: every plugin must be rebuilt against
  `targetAbi 12.0.0.0`, so a 12.0 upgrade means disabling all plugins
  first and pulling them from the unstable repo. 12.0 was still in RC as
  of this writing. Treat 10.11.x as the stable line.
- **Desktop client:** `jellyfin-media-player` ended at v1.12.0 (last Qt5
  build, March 2025). The project is now `jellyfin-desktop` on Qt6
  (v2.0.0+, Dec 2025). The web client remains the reference UI. Swiftfin
  covers Apple platforms, Findroid/Streamyfin cover Android/mobile.

## Emby specifics

- The server source is closed. The `MediaBrowser.Emby.Releases` repo on
  GitHub carries binaries and changelogs. Stable line is 4.9.5.0.
  4.10.0.x is beta and 4.11.0.1 has appeared in aggregators (treat
  anything above 4.9.5 as pre-release).
- Emby Premiere gates hardware transcoding, mobile sync, and some apps.
  The free tier covers direct play and software transcoding.
- Config lives under `/config` (`system.xml`, `encoding.xml`,
  `users` DB, `library` DB). The DB is still the pre-EF SQLite stack
  Jellyfin inherited.
- Plugin catalog is smaller than Jellyfin's but more stable across
  upgrades because there is no ABI break schedule.

## Wiring into the stack

- The *arr apps reach the media server through Settings -> Connect ->
  Emby/Jellyfin with an API key. Events on grab, import, upgrade, rename,
  and delete trigger a targeted library refresh. The media server needs
  only read access to `/data/media`.
- Seerr binds to **one** media server per instance. Jellyfin and Emby are
  near-interchangeable through the same API enum. Pick the one the
  household actually uses.
- For Jellyfin, create the *arr/Seerr API key under Dashboard -> API
  Keys rather than using a user's session token, so automation survives
  password changes and logout sweeps.
