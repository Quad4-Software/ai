---
name: android-media
description: >
  This skill covers the Media3 audio stack on Android: ExoPlayer,
  MediaLibraryService and MediaSession, lock screen and notification
  artwork, OkHttp data sources, SimpleCache stream caching, offline
  DownloadManager, and synced lyrics. Use it for playback services,
  buffering bugs, offline downloads with progress, and media metadata.
---

## When to use this skill

- You are building or fixing an audio player with Media3/ExoPlayer.
- Lock screen or notification controls lack cover art or artwork fails
  to load behind authenticated URLs.
- Streams or downloads die mid-playback and you suspect timeouts.
- You need offline downloads with queued/active/failed state and
  progress reporting.
- You are implementing synced lyrics, seek-to-line, or media browsing
  for Android Auto and system controllers.

## How to use

1. Read this file for the architecture and the known pitfalls.
2. Load [references/media3-recipes.md](references/media3-recipes.md)
   for working code: authenticated bitmap loader, download state
   publishing, synced lyrics composable.
3. Pair with `android-performance` for jank and buffering methodology,
   and `android-dev` for build/adb workflow.

## Examples

- "Why does my stream cut out after about a minute?"
- "Lock screen shows no album art even though artworkUri is set."
- "Add a downloads screen with live progress and cancel."
- "Highlight the current lyric line and seek when tapped."

# Media3 audio stack

## Architecture

A working setup has three pieces:

- A `PlayerManager` singleton owns the `ExoPlayer`, the queue as
  `List<Song>` (your domain model), `StateFlow`s for `nowPlaying`,
  `isPlaying`, `isBuffering`, `positionMs`, `durationMs`, `shuffle`,
  `repeatMode`, and `playbackError`. It builds `MediaItem`s from songs
  with full `MediaMetadata`.
- A `PlaybackService extends MediaLibraryService` hosting a
  `MediaLibrarySession`. System controls (lock screen, notification,
  Auto, Bluetooth AVRCP) bind through the session. `onPlaybackResumption`
  lets the session restart the queue after process death.
- A `MediaSessionHost` keeps the app-side `MediaController` connected so
  UI state stays in sync.

## The callTimeout streaming bug

OkHttp `callTimeout` spans the entire call, including consuming the
response body. A shared client with `callTimeout(75s)` kills any track
that streams longer than 75 seconds. Symptom: playback dies mid-song or
downloads fail on large files with no obvious error.

Fix: keep two clients.

```kotlin
val shared = OkHttpClient.Builder()
    .connectTimeout(18, SECONDS)
    .readTimeout(45, SECONDS)
    .callTimeout(75, SECONDS)
    .build()

val media = shared.newBuilder()
    .callTimeout(0, SECONDS)   // streams and downloads
    .build()
```

Point `OkHttpDataSource.Factory(media)` at the media client for both
the stream cache and the `DownloadManager`. Keep `callTimeout` on the
shared client for API calls.

## Lock screen and notification artwork

Two requirements, both easy to miss:

1. `MediaMetadata.Builder().setArtworkUri(uri)` on each `MediaItem`.
2. A bitmap loader on the session that can actually fetch the URI.

`MediaLibrarySession.Builder.setBitmapLoader(...)`. The trap:
`DataSourceBitmapLoader(context)` builds a `DefaultHttpDataSource`
internally, which does NOT carry your OkHttp interceptors. Auth headers
(bearer tokens, cookies, API keys in headers) never reach the artwork
request and the load 401s silently, so the lock screen shows no art.

Fix: inject your authenticated factory. In Media3 1.8 the factory
constructor takes an explicit executor:

```kotlin
val bitmapLoader = CacheBitmapLoader(
    DataSourceBitmapLoader(
        DataSourceBitmapLoader.DEFAULT_EXECUTOR_SERVICE.get(),
        OkHttpDataSource.Factory(mediaClient),
    ),
)
MediaLibrarySession.Builder(this, player, callback)
    .setBitmapLoader(bitmapLoader)
    .build()
```

`CacheBitmapLoader` caches decoded bitmaps so repeated artwork fetches
do not hit the network.

## Stream cache vs download cache

Use two separate `SimpleCache` instances:

- Stream cache: `LeastRecentlyUsedCacheEvictor` (or default) bounded to
  a few hundred MB for progressive playback caching. Read-through so
  replayed tracks skip the network.
- Download cache: `NoOpCacheEvictor`, persistent, owned by
  `DownloadManager`. Playback of downloaded files reads through a
  `CacheDataSource.Factory` with
  `setCacheWriteDataSinkFactory(null)` (read-only) and
  `FLAG_IGNORE_CACHE_ON_ERROR` so a corrupt entry falls back to the
  network.

## Offline downloads with progress

`DownloadManager` + a `DownloadService` foreground service. Store your
domain object in the `DownloadRequest` data blob so the UI can render
rows without a second lookup:

```kotlin
DownloadRequest.Builder(song.id, Uri.parse(downloadUrl(song.id)))
    .setData(json.encodeToString(song).toByteArray())
    .setCustomCacheKey(song.id)
    .build()
```

Expose ALL download entries, not just completed ones. Iterate
`downloadManager.downloadIndex.getDownloads()` in the
`DownloadManager.Listener` callbacks (`onInitialized`,
`onDownloadChanged`, `onDownloadRemoved`) and publish a snapshot list
with `state`, `percentDownloaded`, `bytesDownloaded`, `contentLength`.
Map `Download.STATE_*` to UI states: queued, downloading, restarting,
stopped (paused), failed, completed. `percentDownloaded` can be
`C.PERCENTAGE_UNSET` (-1) when content length is unknown, so render an
indeterminate bar in that case.

## Synced lyrics

For servers exposing structured lyrics (OpenSubsonic
`getLyricsBySongId`, LRC, TTML):

- Prefer entries where `synced == true` and at least one line has a
  `start` timestamp (milliseconds).
- Apply the entry `offset` (ms) when computing the active line:
  `activeIndex = lines.indexOfLast { (it.start ?: MAX) + offset <= pos }`.
- Drive highlighting from the player's position sampled per frame (see
  `android-performance` for `withFrameMillis`), not a polled StateFlow.
- Auto-scroll to `activeIndex - 2` so the current line sits above
  center, but guard with `listState.isScrollInProgress` so manual
  scrolls are not fought.
- Tap-to-seek: `player.seekTo(line.start + offset)`.
- Fall back in order: synced structured, unsynced structured, legacy
  plain-text lyrics.

## Pitfalls

- `replaceMediaItem` on the current item can restart playback in older
  Media3. 1.8+ supports metadata-only in-place updates when uri and
  cacheKey match, but verify on your version before relying on it.
- `Download.STATE_*` constants are `@UnstableApi`. Add the file-level
  OptIn.
- Do not run `onPlaybackResumption` media item building on the main
  thread if it touches disk or network. Return a future.
