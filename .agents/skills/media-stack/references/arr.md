# Radarr, Sonarr, Lidarr, Prowlarr reference

All four share the same architecture: an ASP.NET backend, a SQLite state
DB under `/config`, a `config.xml` for bind/auth settings, and a REST API
key in Settings -> General. Release channels are `master`/`main` (stable)
and `develop` (pre-release). `nightly` exists for bleeding-edge builds.

## Radarr

- **Latest stable:** v6.4.4.10685 (Sept 2026). Port 7878. API v3.
- The v6 line reached stable Sept 2025 (v6.0.0.10217). Notable additions:
  Postgres DB option (v6.2), RQBit download client (v6.4.1).
- **Auth:** `X-Api-Key: <key>` header, or `?apikey=<key>` query param.
  Find the key in Settings -> General -> Security.
- Key endpoints under `/api/v3`:
  - `GET /movie/lookup?term=`, `/movie/lookup/tmdb`, `/movie/lookup/imdb`
    search TMDB before adding.
  - `POST /movie` adds a movie. The payload needs `tmdbId`, `title`,
    `qualityProfileId`, `rootFolderPath`, `monitored: true`, and
    `addOptions: {searchForMovie: true}` to trigger an immediate search.
  - `GET /rootfolder`, `GET /qualityprofile` supply the IDs the POST needs.
  - `GET /queue` (params `page`, `pageSize`, `includeUnknownMovieItems`),
    `DELETE /queue/{id}` with `removeFromClient` and `blocklist` params.
  - `GET /wanted/missing`, `GET /wanted/cutoff`, `GET /history`.
  - `POST /command` runs named commands:
    `{"name": "MoviesSearch", "movieIds": [123]}`,
    `{"name": "RefreshMovie"}`, `{"name": "RssSync"}`.
  - `GET /system/status`, `GET /health`, `GET /remotepathmapping`,
    `GET /downloadclient`, `GET /indexer`, `GET /notification`,
    `GET /tag`.
- API docs: https://radarr.video/docs/api/

### Adding a movie over the API

```bash
KEY="<radarr-api-key>"
BASE="http://radarr:7878/api/v3"

# 1. Find the TMDB entry.
curl -s "$BASE/movie/lookup?term=heat%201995" -H "X-Api-Key: $KEY"

# 2. Get the root folder and quality profile IDs.
curl -s "$BASE/rootfolder" -H "X-Api-Key: $KEY"
curl -s "$BASE/qualityprofile" -H "X-Api-Key: $KEY"

# 3. Add it.
curl -s -X POST "$BASE/movie" -H "X-Api-Key: $KEY" \
  -H "Content-Type: application/json" -d '{
    "tmdbId": 949,
    "title": "Heat",
    "year": 1995,
    "qualityProfileId": 1,
    "rootFolderPath": "/data/media/movies",
    "monitored": true,
    "addOptions": {"searchForMovie": true}
  }'
```

## Sonarr

- **Latest stable:** v4.0.19.2979 (June 2026), `main` branch. Develop is
  ahead (v4.0.19.30xx). Port 8989. API v3 (the v3 API docs apply to v4).
- **v3 is EOL.** Last v3 was 3.0.10.1567. v4 broke compatibility:
  Preferred Words became Custom Formats, Release Profiles were removed
  (delete them before upgrading), ffprobe replaced MediaInfo for file
  analysis, and language handling moved into Custom Formats (the v3
  `languageprofile` endpoint is gone).
- **Auth:** same `X-Api-Key` pattern as Radarr.
- Key endpoints under `/api/v3`:
  - `GET /series/lookup?term=` (accepts a `tvdb:` prefix for exact IDs),
    `GET/POST/PUT/DELETE /series`.
  - `GET /episode?seriesId=`, `GET /episodefile`.
  - `GET /wanted/missing`, `GET /wanted/cutoff`,
    `GET /calendar?start=&end=`.
  - `POST /command`: `SeriesSearch`, `SeasonSearch`, `EpisodeSearch`,
    `RenameFiles`, `RefreshSeries`, `RescanSeries`, `RssSync`.
  - `GET /rename?seriesId=` previews the naming scheme's output, useful
    before committing to `RenameFiles`.
  - `GET /release`, `GET /parse` for inspecting how a release title parses.
- Anime: use series type `anime`, the `{absolute:000}` naming token, and
  separate anime naming/custom-format rules.
- API docs: https://sonarr.tv/docs/api/

## Lidarr

- **Latest stable:** v3.1.0.4875 (Nov 2025). Develop ahead at v3.1.4.x,
  nightly at 3.1.5.x. Port 8686. **API v1**, not v3.
- The first v3 stable (v3.0.1.4866, Oct 2025) carried breaking changes:
  .NET 8 runtime, Basic Auth removed (forms auth only), linux-x86 builds
  dropped, and a SourceGear sqlite3 dependency that requires GLIBC 2.29+
  (breaks Debian 10, DSM, Ubuntu 18.04 hosts).
- Key endpoints under `/api/v1`:
  - `GET/POST /artist`, `GET/POST /album`, `GET /artist/lookup?term=`.
  - `GET /rootfolder`, `GET /qualityprofile`, `GET /metadataprofile`.
    Lidarr adds a metadata profile on top of the quality profile. An
    artist POST needs both.
  - `GET /queue`, `GET /history`, `GET /wanted/missing`,
    `GET /calendar`, `POST /command`, `GET /system/status`.
- In-container updates break audio fingerprinting. Always recreate the
  container to upgrade.
- API docs: https://lidarr.audio/docs/api/

## Prowlarr

- **Latest stable:** v2.5.2.5491 (July 2026). Develop at v2.6.x. Port
  9696. API v1.
- Role: one place to hold indexer credentials, sync them into every *arr
  app, and proxy searches. Adding an indexer in Prowlarr and an app in
  Settings -> Apps pushes the indexer config downstream automatically.
- Key endpoints under `/api/v1`:
  - `GET/POST/PUT/DELETE /indexer`, `GET /indexer/schema` (all indexer
    types and their field schemas), `POST /indexer/test` and
    `/indexer/testall`.
  - `GET/POST /applications` (the *arr sync targets), `GET /appprofile`.
  - `GET/POST /search` for direct queries through Prowlarr.
  - `GET /downloadclient`, `GET /history`, `GET /system/status`.
- **Per-indexer proxy:** every configured indexer gets a Torznab/Newznab
  endpoint at `http://<host>:9696/<indexerId>/api?t=search&q=<term>
  &apikey=<key>`. Any Newznab-capable client can query through Prowlarr,
  which is how apps that Prowlarr does not sync to directly still benefit
  from centralized indexer credentials.
- App profiles control which indexer categories sync to which *arr app,
  so a music indexer does not land in Radarr.
- API docs: https://prowlarr.com/docs/api/

## Quality profiles vs Custom Formats (Sonarr v4 / Radarr v6 / Lidarr v3)

Two scoring systems that work together:

- **Quality profiles** define the ordered list of allowed qualities
  (Bluray-1080p above WEBDL-1080p above HDTV-1080p, etc.) and a cutoff.
  Anything at or above the cutoff stops upgrading.
- **Custom Formats** are named regex/metadata matchers on the release
  title and mediainfo (e.g. "x265", "Remux", "Repack/Proper", release
  groups). They carry a score **per quality profile**, not globally. Each
  profile sets a minimum CF score (reject anything below), an
  upgrade-until CF score, and can use negative scores for unwanted
  formats.
- The TRaSH convention sets Proper/Repack handling to "Do Not Prefer" in
  settings and controls it through a Repack/Proper custom format instead.
- Naming tokens should embed the CF and quality so a file's provenance
  survives renames. Example Sonarr pattern:

  ```
  {Series CleanTitleWithoutYear} {(Series Year)} - S{season:00}E{episode:00} - {Episode CleanTitle:90} {[Custom Formats]}{[Quality Full]}{[MediaInfo 3D]}{[MediaInfo VideoDynamicRangeType]}{[Mediainfo AudioCodec}{ Mediainfo AudioChannels]}{[Mediainfo VideoCodec]}{-Release Group}
  ```

- `recyclarr` syncs TRaSH custom formats and quality profiles into
  Radarr/Sonarr from YAML. Pin a major version tag. The `latest` image
  tag is no longer published.

## Upgrade and migration caveats

- Never update inside the container. Pull a new image tag. The in-app
  updater corrupts image-bundled binaries (Lidarr fingerprinting breaks
  outright).
- Back up `/config` before major upgrades. The SQLite schema migrates
  forward and there is no supported rollback. For Postgres-backed *arr
  installs, `pg_dump` the database before upgrading.
- Sonarr v3 -> v4 is a one-way migration that drops Release Profiles and
  rewrites Preferred Words into Custom Formats.
- Radarr/Sonarr/Lidarr accept `POSTGRES_HOST`/`POSTGRES_PORT`/
  `POSTGRES_USER`/`POSTGRES_PASSWORD`/`POSTGRES_MAIN_DB` env vars on
  recent builds for a Postgres backend instead of SQLite.
