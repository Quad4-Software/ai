---
name: android-dev
description: >
  This skill covers Android app development with Kotlin, Gradle KTS,
  and Jetpack Compose. Use it for project layout and version catalogs,
  the build/test/install/verify loop over adb, Compose conventions
  (state, effects, navigation, lazy lists), Media3 and Coil
  integration patterns, and the common pitfalls that break builds.
---

## When to use this skill

- You are writing or fixing Kotlin Android code with Jetpack Compose.
- You need the Gradle build, unit test, adb install, and smoke-test loop.
- You are wiring Media3 playback, Coil images, DataStore settings, or
  Navigation Compose routes.
- You hit a Compose or Media3 compile error and want the known cause.

## How to use

1. Read this file for the toolchain, build commands, and conventions.
2. Load [references/adb-and-build.md](references/adb-and-build.md) for
   the adb install, launch, screenshot, input, and logcat cookbook.
3. Pair with the `android-performance` skill for jank and profiling,
   and `android-media` for the Media3 playback stack.

## Examples

- "Build the app, install it on my phone, and check it launches."
- "Why does my Downloads screen fail to compile on Download.STATE_*?"
- "Add a Compose screen with a settings toggle backed by DataStore."
- "Wire a media session so lock screen controls show cover art."

# Android + Kotlin + Compose

## Toolchain snapshot

- Kotlin 2.x, Android Gradle Plugin 9.x, JVM target 17.
- compileSdk 37, targetSdk 36+, minSdk 26 is a sane floor for media
  apps (Media3, per-app language, themed icons all work).
- Version catalog at `gradle/libs.versions.toml`, single `:app`
  module, `gradle.properties` for AndroidX/R8 flags.
- Prefer `debugImplementation` for tooling-only deps so release builds
  stay clean.

## Build, test, verify loop

```
./gradlew :app:compileDebugKotlin   # fastest compile check
./gradlew test                      # JVM unit tests
./gradlew assembleDebug             # produces app-debug.apk
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell monkey -p <pkg> -c android.intent.category.LAUNCHER 1
adb shell pidof <pkg>               # confirm it is still alive
adb exec-out screencap -p > shot.png
```

Always compile before installing. `monkey -c LAUNCHER 1` fires the
launcher intent without needing the activity name. Screenshot with
`exec-out` (not `adb shell screencap`, which mangles binary output).

## Compose conventions that matter

- Collect flows with `collectAsStateWithLifecycle()` and an explicit
  `initialValue` for non-StateFlow sources.
- Keep state reads inside `item {}` content lambdas, not in the
  `LazyListScope` builder. Reads in the builder re-run the whole list
  DSL. Reads in item content only recompose that item.
- Every lazy list item gets a stable `key` and a `contentType`. Keys
  preserve scroll/remembered state across reorder, contentType lets
  heterogeneous rows recycle.
- `remember(key)` for anything derived per item. Never build
  ImageRequests, formatters, or parsed models fresh in composition.
- Use `withFrameMillis` sampling for smooth playback progress instead
  of emitting a StateFlow every poll tick. See `android-performance`.
- `ModalBottomSheet` for lyrics, queues, and pickers keeps nav flat.

## Stack notes

- Media3 1.8: `UnstableApi` annotation is required for many classes
  (`Download`, `CacheDataSource`, `DataSourceBitmapLoader`). File-level
  opt-in:
  `@file:androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)`
- Coil: build `ImageRequest` inside `remember(cacheKey)` and use your
  own `memoryCacheKey`/`diskCacheKey` when the URL embeds rotating auth
  params, or every recomposed image gets a new cache entry.
- DataStore Preferences for settings. Expose a `Flow<AppSettings>` and
  typed setters. Read once with `.first()` for one-shot decisions.
- Navigation Compose: `@Serializable` route objects/data classes,
  `composable<Route> { entry -> entry.toRoute<Route>() }`.
- OkHttp: one shared client plus a dedicated media client. See
  `android-media` for the `callTimeout` trap.

## Common build breakers

- `OkHttpDataSource.Factory(client)` returns `HttpDataSource.Factory`,
  which is a `DataSource.Factory`. Passing it where `Context` is
  expected is a signature mistake. Check constructor overloads on the
  exact dependency version with `javap` on the AAR `classes.jar`.
- `Download.STATE_*` and most exoplayer offline classes are
  `@UnstableApi`. Missing opt-in is a compile error, not a warning.
- Kotlin does not support `::localFunction` callable references for
  local functions. Pass `{ localFn() }` instead.
- `LinearProgressIndicator(progress = { ... })` takes a lambda in
  recent Material3. The `Float` overload is deprecated.
- Dagger/Hilt-free apps work fine with a manual dependency container
  on the `Application` class. Keep it lazy so cold start stays fast.

## Media3 manifest checklist

Playback needs a `MediaLibraryService` (or `MediaSessionService`)
declared with `androidx.media3.session.MediaLibraryService` intent
filter, `FOREGROUND_SERVICE`, and `FOREGROUND_SERVICE_MEDIA_PLAYBACK`
permissions plus `android:foregroundServiceType="mediaPlayback"`.
Offline downloads need a second service extending `DownloadService`
with `foregroundServiceType="dataSync"`.
