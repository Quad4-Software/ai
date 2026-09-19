---
name: opensubsonic
description: >
  This skill covers the OpenSubsonic API specification and its main
  server implementation, Navidrome. Use it for the Subsonic 1.16.1 base
  protocol, OpenSubsonic extensions, auth models, the response envelope,
  streaming and transcoding flow, server/client compatibility, and
  Navidrome deployment, configuration, security, and upgrade caveats.
---

## When to use this skill

- You are writing a client against the Subsonic or OpenSubsonic API.
- You are checking which extensions a server advertises or which auth
  model to send.
- You are deploying, configuring, or upgrading Navidrome.
- You are debugging client compatibility (missing endpoints, auth
  errors, ID changes).
- You are choosing between OpenSubsonic servers.

## How to use

1. Read this file for the spec model, extension map, and the Navidrome
   deployment essentials.
2. Load [references/opensubsonic-api.md](references/opensubsonic-api.md)
   for the envelope, auth, error codes, and endpoint families.
3. Load [references/navidrome.md](references/navidrome.md) for config
   keys, deployment, security notes, and breaking changes.
4. Fall back to https://opensubsonic.netlify.app/ for endpoint text and
   https://www.navidrome.org/docs/ for Navidrome options.

## Examples

- "Write a ping + search3 request against a Navidrome server."
- "Which OpenSubsonic extensions does Navidrome advertise?"
- "Why does my client's seek bar not work over transcoded streams?"
- "Set up Navidrome behind a reverse proxy at /music."
- "Explain the Navidrome 0.64 ID migration and what it breaks."

# OpenSubsonic and Navidrome

OpenSubsonic is an open extension of the Subsonic API. The base protocol
is Subsonic **v1.16.1**, the last official Subsonic API version. The
OpenSubsonic API itself stays at **version 1** forever. New capability
ships as optional extensions inside that version rather than as version
bumps. A server says what it supports through two channels: extra fields
in every response envelope, and a public `getOpenSubsonicExtensions`
endpoint.

Navidrome is the most complete implementation and the reference server
most clients test against. Latest stable **v0.64.0** (Sept 2026).

## The response envelope

Every call returns a `subsonic-response` envelope. OpenSubsonic adds
three required fields:

```json
{ "subsonic-response": {
    "status": "ok",
    "version": "1.16.1",
    "type": "Navidrome",
    "serverVersion": "0.64.0",
    "openSubsonic": true,
    "openSubsonicExtensions": [
      {"name": "transcodeOffset", "versions": [1]}
    ]
}}
```

`type`, `serverVersion`, and `openSubsonic` are mandatory. Clients use
them to identify the server and to re-check extension support after an
upgrade instead of hard-coding feature detection.

## Extensions

Extensions are the only versioning mechanism. A server advertises each
with a name and supported version list. Current defined extensions:

| Extension | What it adds |
|---|---|
| `apiKeyAuthentication` | `apiKey` query param replacing u/t/s/p. Adds errors 42/43/44 and `helpUrl` |
| `formPost` | POST bodies as `application/x-www-form-urlencoded` |
| `getPodcastEpisode` | Single-episode metadata without fetching the channel |
| `indexBasedQueue` | `savePlayQueueByIndex` + `getPlayQueueByIndex` (index-based `current`, allows duplicates) |
| `playbackReport` | `reportPlayback` endpoint with `state`/`position`/`playbackRate`. Enriches `getNowPlaying` |
| `songLyrics` | Structured lyrics via `getLyricsBySongId`. V2 adds word-level timing and agent layers behind `enhanced=true` |
| `sonicSimilarity` | `getSonicSimilarTracks` + `findSonicPath` |
| `topSongsByArtistId` | Top-songs lookup by artist ID |
| `transcodeOffset` | Makes `timeOffset` valid for music on `stream` (seeking during transcode) |
| `transcoding` | `getTranscodeDecision` (client posts capabilities) + `getTranscodeStream` (server-issued `transcodeParams`) |

Things that are not extensions: `getAvatar`, `hls.m3u8`, `getBookmarks`,
`getInternetRadioStations`, `jukeboxControl`, `getPodcasts`, shares,
play queue, `getLyrics`. Those are formalized vanilla Subsonic
endpoints. `mediaAnnotation` is the name of an endpoint family, not an
extension.

Full endpoint and auth detail:
[references/opensubsonic-api.md](references/opensubsonic-api.md).

## Auth

Three models, all as query params (or POST form with `formPost`):

- **Salt + token (standard):** `u`, `t`, `s` where `t =
  md5(password + s)` and `s` is a random salt >=6 chars. Every
  OpenSubsonic server supports this.
- **Plain password (legacy):** `u`, `p` with the password clear or
  `enc:`-prefixed hex. Testing only.
- **API key (extension):** `apiKey=<key>` alone. Sending `u` alongside
  it returns error 43 (conflicting mechanisms). Navidrome does not
  implement this extension yet (PR open).

Every call also carries `v` (client's protocol version), `c` (client
name), and `f` (`json` recommended, `xml` is the default).

## Streaming

`stream?id=<id>` returns audio bytes. Relevant params: `maxBitRate`
(0 = unlimited), `format` (`raw` disables transcoding), `timeOffset`
(video only unless the server advertises `transcodeOffset`),
`estimateContentLength`, `transcodedContentType`. Errors on a stream
URL come back as a `subsonic-response` envelope, not raw bytes.

The `transcoding` extension inverts the flow: the client POSTs its
capability profile to `getTranscodeDecision`, the server returns a
decision plus an opaque `transcodeParams` token, and the client GETs
`getTranscodeStream` with it. Navidrome implements this with a JWT in
`transcodeParams`.

## Server and client landscape

- **Most complete:** Navidrome (implements nearly every extension),
  LMS (epoupon/lms, most extensions, legacy auth), gonic (>=0.21,
  jukebox, salt+token).
- **Partial:** Ampache, Funkwhale, Nextcloud Music (no token auth),
  Supysonic. Newer full-spec servers: Melodee, HexSonic, Sonata.
- **Not OpenSubsonic:** Airsonic-Advanced is a functional Subsonic
  1.16.1 server but unmaintained. Clients need compat mode.
- **Clients driving the spec:** Symfonium (co-designs extensions),
  Tempus/Tempo, Feishin, Supersonic, Substreamer, Amperfy, Ultrasonic,
  DSub, play:Sub.

# Navidrome

Single-binary Go server, embedded SQLite, React web UI on **port
4533**. Latest stable **v0.64.0** (Sept 2026).

## Deployment

- Docker image `deluan/navidrome`. Ports `4533:4533`, volumes `/data`
  (DB + cache, rw) and `/music` (ro is fine). Run as `user: 1000:1000`
  or `--user $(id -u):$(id -g)`. There are **no** `PUID`/`PGID` env
  vars.
- Config via `navidrome.toml` (inside the container at
  `ND_CONFIGFILE=/data/navidrome.toml`), env vars prefixed `ND_`, or
  CLI flags. Precedence: env > flag > file.
- Reverse proxy: `BaseURL=/music` for a subpath. `ShareURL` for public
  share links. The proxy must pass `/share` for sharing to work.
- Bare-metal binaries for linux/darwin/windows (tar.gz, .deb, .rpm,
  .msi). A systemd service template ships in the repo.

## Architecture

- **Scanner:** TagLib-based via go-taglib (pure-Go default since 0.60).
  Watcher mode, resumable scans, symlink resolution, `Scanner.PurgeMissing`.
- **Transcoding:** ffmpeg (`FFmpegPath`), cached
  (`TranscodingCacheSize`), decision engine behind the `transcoding`
  extension.
- **Plugins (0.60+):** WebAssembly sandbox via Extism. Capabilities:
  MetadataAgent, Scrobbler, Lyrics, SonicSimilarity, TaskWorker,
  Lifecycle, SchedulerCallback, WebSocketCallback. PDKs for Go/Rust. Python/JS also supported.
- **Multi-library (0.58+):** separate libraries with per-user access. `getMusicFolders` returns what the user can see.
- **Jukebox:** MPV over IPC (`Jukebox.Enabled`, `Jukebox.Devices`).
- **Smart playlists:** `.nsp` JSON files imported at scan.
- **Metrics:** Prometheus endpoint off by default
  (`Prometheus.Enabled`, `Prometheus.MetricsPath=/metrics`,
  `Prometheus.Password` basic-auth user `navidrome`).
- **Telemetry:** anonymous Insights collector is **on by default**.
  Opt out with `EnableInsightsCollector=false`.

## OpenSubsonic support

Navidrome advertises API 1.16.1, `openSubsonic: true`, `type:
"Navidrome"`. `getOpenSubsonicExtensions` is public. Advertised
extensions: `transcodeOffset`, `formPost`, `songLyrics` v1+v2,
`indexBasedQueue`, `transcoding`, `playbackReport`, and
`sonicSimilarity` (the last only when a SonicSimilarity plugin such as
AudioMuse-AI is loaded). Not implemented: `apiKeyAuthentication` and
`topSongsByArtistId`.

Endpoint gaps by design: no video at all, no podcasts (`getPodcasts`
returns 501), `search2`/`search3` are autocomplete only, `getTopSongs`
and `getSimilarSongs` need Last.fm, `getArtistInfo`/`getAlbumInfo`
need external agents. There is also an undocumented native JSON API
under `/api/*` powering the web UI. Treat it as unstable.

## Security notes

- Passwords are stored with **reversible encryption**, not bcrypt.
  Token auth needs the plaintext. `PasswordEncryptionKey` re-encrypts
  once and cannot be changed afterward. The default key is in the
  codebase, so treat stored passwords as obfuscated.
- Sessions are JWTs. The JWT secret sits in `navidrome.db` plaintext.
- Shares are `/share/{id}` links with JWT stream tokens carrying a
  `sid` claim. Deleted shares 404, expired tokens 400.
- Rate limiting via `AuthRequestLimit`/`AuthWindowLength`.
  `EnforceNonRootUser` refuses root operation (0.62+). `ExtAuth`
  trusted-source IP validation.
- v0.64.0 fixed SQLi in the native API's artist `role` sort/filter, a
  share IDOR, plugin SSRF, and a rate-limit bypass.

## Upgrade caveats

- **v0.64.0 (Sept 2026):** all IDs re-encoded to canonical 128-bit
  base62. Touches every table, so back up first. Clients that cache
  IDs need a re-sync. Extism built-in HTTP disabled for plugins. Plugin HTTP/WS to private/loopback blocked unless declared in
  `requiredHosts`. Experimental Jellyfin Music API added
  (`Jellyfin.Enabled`).
- **v0.63.0:** sidecar lyrics (TTML/ELRC/SRT/YAML) + `songLyrics` v2.
  `EnableSharing` flipped to true.
- **v0.62.0:** `sonicSimilarity` + `playbackReport` extensions,
  transcode concurrency cap, `EnforceNonRootUser`.
- **v0.61.0:** artwork overhaul, FTS5 BM25 search, server-managed
  transcoding. Spotify integration removed.
- **v0.55.0:** multi-valued tags + artist roles, new scanner
  (TagLib-only, ffmpeg extractor removed), `ScanSchedule` renamed to
  `Scanner.Schedule`.
- **v0.58.0:** multi-library. Run a full scan after upgrade or lose
  annotations. DB migration is irreversible.

Config keys and per-area detail: [references/navidrome.md](references/navidrome.md).
