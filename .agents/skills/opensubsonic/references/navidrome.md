# Navidrome reference

Latest stable **v0.64.0** (Sept 2026). Single Go binary, embedded
SQLite (`navidrome.db` in `DataFolder`, no external DB), React web UI
on port 4533. GPL-3.0. The `master` branch can be unstable. Use
release builds.

## Configuration

Precedence: env (`ND_*`) > CLI flag > `navidrome.toml`. Common keys:

- **Core:** `MusicFolder` (`ND_MUSICFOLDER`, default `./music`),
  `DataFolder` (`./data`), `CacheFolder`, `DbPath`, `Port` (4533),
  `Address`, `BaseURL`/`BasePath`/`BaseHost`/`BaseScheme`
  (reverse-proxy subpath), `TLSCert`/`TLSKey`, `LogLevel`, `LogFile`,
  `SessionTimeout` (48h), `UILoginBackgroundURL`, `UIWelcomeMessage`,
  `DefaultTheme`, `DefaultLanguage`, `DefaultUIVolume`.
- **Scanner:** `Scanner.Enabled`, `Scanner.Schedule` (cron, `0` = off, renamed from `ScanSchedule` in 0.55), `Scanner.WatcherWait`,
  `Scanner.ScanOnStartup`, `Scanner.Extractor` (`taglib` default,
  `legacy-taglib` fallback), `Scanner.FollowSymlinks`,
  `Scanner.PurgeMissing`, `Scanner.ArtistJoiner`,
  `Scanner.ArtistSplitExceptions`, `Scanner.IgnoreDotFolders` (0.63+),
  `PID.Album`/`PID.Track` (ID persistence rules), `Tags` (custom tag
  map with Aliases/Type/Split).
- **Transcoding/images:** `EnableTranscodingConfig`,
  `Transcoding.EnableCancellation`, `TranscodingCacheSize` (100MB),
  `ImageCacheSize`, `FFmpegPath`, `CoverArtPriority`,
  `CoverJpegQuality`, `ArtistArtPriority`, `LyricsPriority` (0.63+:
  `.lrc,.ttml,.yaml/.yml,.elrc,.srt,embedded` + plugin names),
  `EnableArtworkPrecache`, `EnableCoverAnimation`.
- **Subsonic:** `Subsonic.AppendSubtitle`,
  `Subsonic.ArtistParticipations`, `Subsonic.AppendAlbumVersion`,
  `Subsonic.DefaultReportRealPath`, `Subsonic.EnableAverageRating`,
  `Subsonic.LegacyClients`, `Subsonic.MinimalClients`.
- **Features:** `EnableSharing` (true since 0.63), `ShareURL`,
  `DefaultShareExpiration`, `DefaultDownloadableShare`,
  `EnableFavourites`, `EnableStarRating`, `EnableUserEditing`,
  `EnableGravatar`, `EnableDownloads`, `EnableExternalServices`,
  `EnableInsightsCollector`, `EnableReplayGain`, `EnableNowPlaying`,
  `EnableScrobbleHistory`, `AutoImportPlaylists`, `PlaylistsPath`,
  `SmartPlaylistRefreshDelay`, `DefaultPlaylistPublicVisibility`,
  `AutoTranscodeDownload`, `DefaultDownsamplingFormat`,
  `SearchFullString`, `RecentlyAddedByModTime`, `PreferSortTags`,
  `IgnoredArticles`, `IndexGroups`, `AlbumPlayCountMode`,
  `SimilarSongsMatchThreshold`.
- **Auth/security:** `PasswordEncryptionKey`, `AuthRequestLimit`,
  `AuthWindowLength`, `ExtAuth.UserHeader` (`Remote-User`),
  `ExtAuth.TrustedSources` (CIDRs or `@` for unix socket),
  `ExtAuth.LogoutURL`, `EnforceNonRootUser` (0.62+). `ExtAuth.*` was
  renamed from `ReverseProxyUserHeader`/`ReverseProxyWhitelist` in
  0.59. Old names still map.
- **Integrations:** `LastFM.Enabled/ApiKey/Secret/Language`,
  `ListenBrainz.Enabled/BaseURL`, `Deezer.Enabled`, `Agents`,
  `Jellyfin.Enabled` (0.64 experimental), `Jukebox.*`, `Plugins.*`,
  `HTTPHeaders`, `Prometheus.*`. `Spotify.*` removed in 0.61.

## Plugins (0.60+)

WebAssembly sandbox via Extism. Capabilities: MetadataAgent,
Scrobbler, Lyrics, SonicSimilarity, TaskWorker, Lifecycle,
SchedulerCallback, WebSocketCallback. Host services cover HTTP, KV
store, task queues, and the Subsonic API. PDKs for Go/Rust. Python
and JS supported. Config: `Plugins.Enabled` (default true),
`Plugins.Folder`, `Plugins.CacheSize` (200MB), `Plugins.AutoReload`,
`Plugins.LogLevel`. Since 0.64, plugin HTTP/WS to private or
loopback addresses is blocked unless declared in `requiredHosts`.

## Smart playlists

`.nsp` JSON files in the music folder, imported at scan. Fields:
`all`/`any` criteria arrays, `sort`, `order`, `limit`. Operators:
`is`, `isNot`, `gt`, `lt`, `contains`, `notContains`, `startsWith`,
`endsWith`, `inTheRange`, `before`, `after`, `inTheLast`,
`notInTheLast`, `inPlaylist`, `notInPlaylist`, `isMissing`,
`isPresent`. Since 0.63.2, top-level `any`+`all` cannot be mixed.

## External services

`Agents` config (default `deezer,lastfm,listenbrainz` plus local
fallback) supplies artist art, bios, and similar artists. Scrobbling
goes to Last.fm, ListenBrainz, and plugin scrobblers. Without agents,
`getTopSongs`, `getSimilarSongs`, `getArtistInfo`, and `getAlbumInfo`
return nothing useful.

## Deployment notes

- Docker `deluan/navidrome`: `4533:4533`, `/data` rw, `/music` ro.
  Run as `user: 1000:1000`. No `PUID`/`PGID` env vars exist.
- `ND_CONFIGFILE=/data/navidrome.toml` keeps config on the data
  volume.
- Subpath hosting: `BaseURL=/music` plus `ShareURL` for public share
  links. The proxy must pass `/share` paths.
- Jukebox needs MPV (`MPVPath`) and `Jukebox.Enabled`.
- Prometheus: `Prometheus.Enabled`, `Prometheus.MetricsPath`
  (`/metrics`), `Prometheus.Password` for basic-auth user `navidrome`.
- CLI: backup, `search rebuild`, `plugin`, inspect/missing-fix
  subcommands.
- Multi-library (0.58+): per-user library grants. `getMusicFolders`
  returns only what the user can see.

## Security detail

- Passwords use reversible encryption keyed by
  `PasswordEncryptionKey`. Required because Subsonic token auth needs
  the plaintext to verify `md5(password + salt)`. The default key is
  in the repo, so without a custom key stored passwords are only
  obfuscated.
- JWT secret stored plaintext in `navidrome.db` `property` table
  (fixed advisory GHSA-xwx7-p63r-2rj8).
- Share tokens are JWTs carrying a `sid` claim. Deleted shares 404,
  expired tokens 400. Share ownership is enforced on update/delete
  (0.61+).
- `EnforceNonRootUser` (0.62+) refuses to run as root.
- `ExtAuth` validates trusted proxy sources by CIDR or `@` unix
  socket. A proxy that can be bypassed defeats it.

## Breaking changes by release

- **0.64.0 (Sept 2026):** all IDs migrated to canonical 128-bit
  base62. Back up `DataFolder` first. Clients caching IDs need a
  re-sync. Extism built-in HTTP removed (plugins must use
  `host.HTTPSend`). Shares always owned by creator. Unknown config
  keys now warn. Security fixes for SQLi, share IDOR, plugin SSRF,
  rate-limit bypass. Experimental Jellyfin Music API
  (`Jellyfin.Enabled=true` lets Finamp/Jellify connect).
- **0.63.0:** sidecar lyrics (TTML/ELRC/SRT/YAML), `songLyrics` v2,
  smart search. `EnableSharing` default flipped to true.
- **0.62.0:** `sonicSimilarity` + `playbackReport` extensions,
  transcode concurrency cap, `EnforceNonRootUser`.
- **0.61.0:** artwork overhaul, FTS5 BM25 search, server-managed
  transcoding + `transcoding` extension. Spotify integration removed.
- **0.60.0:** Wasm plugin system rewrite, go-taglib extractor
  default, Instant Mix.
- **0.58.0:** multi-library. Full scan required post-upgrade or
  annotations are lost. DB migration irreversible.
- **0.55.0:** multi-valued tags and artist roles. New scanner is
  TagLib-only (ffmpeg extractor removed). `ScanSchedule` renamed to
  `Scanner.Schedule`. `Scanner.Extractor`, `GenreSeparators`,
  `GroupAlbumReleases` removed in favor of `Tags`/`PID`.

## Known rough edges

- `getPodcasts` returns 501, which breaks some clients' server
  capability tests.
- The native `/api/*` JSON API is undocumented and unstable. Use the
  Subsonic API for integrations.
- `search2`/`search3` are autocomplete only, no Lucene syntax.
- `getIndexes` lacks `shortcuts` and direct children.
- `getAvatar` returns a Gravatar redirect or placeholder.
