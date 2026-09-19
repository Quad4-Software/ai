# OpenSubsonic API reference

Base protocol: Subsonic REST API v1.16.1. OpenSubsonic API version is
`1`, the only valid value of the `version` param on
`getOpenSubsonicExtensions`. Spec: https://opensubsonic.netlify.app/,
source repo: github.com/opensubsonic/open-subsonic-api. An OpenAPI
schema ships at /docs/openapi/ (JSON only, `format=json`).

## Request shape

Every call is `GET /rest/<endpoint>` (or POST with the `formPost`
extension) with query params:

- `u` username
- `p` password, clear or `enc:`-prefixed hex (legacy, testing only)
- `t` + `s` token + salt, where `t = md5(password + s)` and `s` is a
  random string >=6 chars
- `apiKey` (extension auth, replaces all of the above)
- `v` client's protocol version, `c` client name
- `f` response format: `xml` (default), `json`, `jsonp`

`apiKeyAuthentication` rules: `apiKey` stands alone. Passing `u` with
it returns error 43. Keys are server-defined, under 2048 URL-encoded
chars. Servers should return `helpUrl` in auth errors.

## Response envelope and errors

```json
{ "subsonic-response": {
    "status": "ok", "version": "1.16.1",
    "type": "Navidrome", "serverVersion": "0.64.0",
    "openSubsonic": true,
    "openSubsonicExtensions": [{"name": "transcodeOffset", "versions": [1]}]
}}
```

`type`, `serverVersion`, `openSubsonic` are required. Error envelope:

```json
{ "subsonic-response": {
    "status": "failed",
    "error": {"code": 40, "message": "Wrong username or password."}
}}
```

Error codes: 0 generic, 10 missing param, 20 client must upgrade,
30 server must upgrade, 40 wrong credentials, 41 token auth not
supported, **42 auth mechanism not supported, 43 conflicting auth
mechanisms, 44 invalid API key** (42-44 are OpenSubsonic additions),
50 not authorized, 60 trial expired, 70 not found.

## `getOpenSubsonicExtensions`

System endpoint, **must be publicly accessible** (no auth). Takes
`version` (only `1` valid). Returns the list of extensions the server
supports, each with `name` and `versions`. Clients should call it once
at setup and re-check after a server upgrade.

## Endpoint families

- **System:** `ping`, `getLicense`, `getOpenSubsonicExtensions`,
  `tokenInfo`
- **Browsing:** `getMusicFolders`, `getIndexes` (non-ID3 file/dir
  view), `getMusicDirectory`, `getGenres`, `getArtists`/`getArtist`/
  `getAlbum`/`getSong` (ID3 view), `getVideos`, `getVideoInfo`,
  `getArtistInfo(2)`, `getAlbumInfo(2)`, `getSimilarSongs(2)`,
  `getTopSongs`
- **Album/song lists:** `getAlbumList(2)`, `getRandomSongs`,
  `getSongsByGenre`, `getNowPlaying`, `getStarred(2)`
- **Searching:** `search`, `search2`, `search3`
- **Playlists:** `getPlaylists`, `getPlaylist`, `createPlaylist`,
  `updatePlaylist`, `deletePlaylist`
- **Media retrieval:** `stream`, `download`, `hls.m3u8`,
  `getCoverArt`, `getLyrics`, `getLyricsBySongId` (ext), `getAvatar`,
  `getCaptions`, `getTranscodeDecision` (ext), `getTranscodeStream`
  (ext)
- **Media annotation:** `star`, `unstar`, `setRating`, `scrobble`,
  `reportPlayback` (ext)
- **Sharing:** `getShares`, `createShare`, `updateShare`,
  `deleteShare`
- **Podcast:** `getPodcasts`, `getNewestPodcasts`, `refreshPodcasts`,
  `createPodcastChannel`, `deletePodcastChannel`,
  `deletePodcastEpisode`, `downloadPodcastEpisode`, `getPodcastEpisode`
  (ext)
- **Jukebox:** `jukeboxControl`
- **Internet radio:** `getInternetRadioStations`,
  `create/update/deleteInternetRadioStation`
- **Bookmarks/queue:** `getBookmarks`, `createBookmark`,
  `deleteBookmark`, `getPlayQueue`, `savePlayQueue`,
  `getPlayQueueByIndex`/`savePlayQueueByIndex` (ext)
- **Chat/user/scanning:** `getChatMessages`, `addChatMessage`,
  `getUser(s)`, `create/update/deleteUser`, `changePassword`,
  `getScanStatus`, `startScan`

## Extensions in detail

- **`apiKeyAuthentication`** (v1): `apiKey=<key>` query param. Adds
  errors 42/43/44 and the `helpUrl` field.
- **`formPost`** (v1): request params as POST body
  `application/x-www-form-urlencoded`.
- **`getPodcastEpisode`** (v1): single-episode metadata without
  fetching the whole channel.
- **`indexBasedQueue`** (v1): `savePlayQueueByIndex` and
  `getPlayQueueByIndex` use an index for `current` instead of a song
  ID, which allows duplicates.
- **`playbackReport`** (v1): `reportPlayback` (GET + form POST) with
  `mediaId`/`songId`, `mediaType` (song/podcast), `position`/
  `positionMs`, `state` (starting/playing/paused/stopped), optional
  `duration`, `playbackRate`, `ignoreScrobble`. Also adds `state`,
  `position`, `playbackRate`, `duration` to `getNowPlaying` entries.
- **`songLyrics`** (v1, v2): structured lyrics via `getLyricsBySongId`.
  V2 adds word/syllable-level timing (karaoke) and agent/voice layers
  gated behind `enhanced=true`.
- **`sonicSimilarity`** (v1): `getSonicSimilarTracks` and
  `findSonicPath` returning `sonicMatch` entries with similarity
  scores.
- **`topSongsByArtistId`** (v1): top-songs lookup by artist ID.
- **`transcodeOffset`** (v1): makes `timeOffset` valid for music on
  `stream`, so clients can seek inside a transcoded stream.
- **`transcoding`** (v1): client POSTs a JSON `ClientInfo` body
  (directPlayProfiles, transcodingProfiles, codecProfiles) to
  `getTranscodeDecision`, then GETs `getTranscodeStream` with the
  server-issued `transcodeParams` token plus `mediaId`, `mediaType`,
  `offset`.

## Streaming params

`stream?id=<id>` accepts `maxBitRate` (kbps, 0 = no limit), `format`
(`raw` disables transcoding), `timeOffset` (video by default, music
only if `transcodeOffset` is advertised), `size` (video WxH),
`estimateContentLength` (estimated Content-Length on transcoded
output), `transcodedContentType`. Errors come back as a
`subsonic-response` envelope.

`scrobble` registers playback. `submission=true` marks it for
forwarding to Last.fm/ListenBrainz, which the server handles.

## Example session

```bash
BASE="https://navidrome.example.com/rest"
AUTH="u=demo&t=$(echo -n "secretNaCl42" | md5sum | cut -d' ' -f1)&s=NaCl42&v=1.16.1&c=myclient&f=json"

# Extension discovery (no auth needed).
curl -s "$BASE/getOpenSubsonicExtensions?version=1&f=json"

# Ping.
curl -s "$BASE/ping?$AUTH"

# Search.
curl -s "$BASE/search3?$AUTH&query=blues&songCount=5"

# Stream.
curl -s "$BASE/stream?$AUTH&id=<songId>&maxBitRate=128" -o out.mp3
```
