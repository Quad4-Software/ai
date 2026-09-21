---
name: android-performance
description: >
  This skill covers Android UI performance: finding and fixing Compose
  scroll jank, frame-level profiling with JankStats and gfxinfo, eager
  vs lazy list containers, recomposition scoping, network timeout bugs
  that stall playback, and adding measurable regression checks.
---

## When to use this skill

- A screen stutters on scroll and you need the cause, not a guess.
- You want on-device jank detection wired to screen names in logcat.
- You are deciding between `Row+horizontalScroll` and `LazyRow`, or
  debugging why lazy lists recompose too much.
- Playback or downloads stall and you suspect network timeouts.
- You want a measurable before/after on a performance fix.

## How to use

1. Read this file for the methodology and the common jank causes.
2. Load [references/jankstats.md](references/jankstats.md) for the
   JankStats integration code, throttled logging, and the unit-testable
   helper pattern.
3. Pair with `android-dev` for the adb verification loop and
   `android-media` for streaming-specific buffering fixes.

## Examples

- "Home lags when I scroll up and down, find the cause."
- "Add jank logging so I can see which screen drops frames."
- "Why does this track stop playing at 75 seconds?"
- "Prove the fix actually reduced janky frames."

# Android performance methodology

## Method

1. Reproduce reliably. Script the interaction with `adb shell input
   swipe` so before and after runs are identical.
2. Measure a baseline. Count janky frames over the scripted run.
3. Find the smallest hot path. Read the composition structure first,
   profile second.
4. Fix one thing. Re-measure with the same script.
5. Keep the instrumentation so regressions are visible later.

## Compose jank causes, ranked by how often they are the real bug

### 1. Eager scroll containers inside lazy lists

`Row` + `horizontalScroll` or `Column` + `verticalScroll` inside a
`LazyColumn` item composes every child eagerly. A shelf of 20 cards
with images means 20 compositions and 20 image loads in one frame when
the shelf enters the viewport, and again when it scrolls back in.

Fix: `LazyRow`/`LazyColumn` with `items(list, key = { it.id },
contentType = { "type" })`. Only visible children compose.

### 2. Missing keys and contentType

Without `key`, reordering or inserting items throws away remembered
state and forces full recomposition. Without `contentType`, items of
different shapes cannot reuse each other's composition slots, so the
list allocates fresh nodes on scroll instead of recycling.

### 3. State reads at the wrong scope

Reading a `StateFlow` in the `LazyListScope` builder lambda re-runs the
whole DSL on every emission. Read it inside `item {}` content so only
that item recomposes. Same rule for `remember` keys: key on the stable
id, not the whole object.

### 4. Work in composition

Bitmap decode, JSON parse, palette extraction, sorting, string building
in a loop. Move to `Dispatchers.Default`/`IO` behind a `LaunchedEffect`
or `produceState`. Palette extraction specifically: run
`Palette.from(bitmap)` off main, it scans pixels.

### 5. Per-frame StateFlow emission

Polling `player.position` into a `StateFlow` at 10-60Hz recomposes
every collector every tick. For smooth progress, sample per frame with
`withFrameMillis` inside `LaunchedEffect` and only update a local
`MutableState`:

```kotlin
@Composable
fun rememberSmoothPositionMs(positionMs: () -> Long): State<Long> {
    val position = remember { mutableLongStateOf(0L) }
    LaunchedEffect(Unit) {
        while (isActive) {
            withFrameMillis { position.longValue = positionMs() }
        }
    }
    return position
}
```

Collectors still recompose, but the frame-aligned loop avoids extra
invalidations and quantizes to vsync. Pause the loop when `isPlaying`
is false and when the composable is off screen.

## Buffering and stall causes

- OkHttp `callTimeout` spans the whole call including the body. Any
  stream or download longer than the timeout dies mid-transfer. Use a
  dedicated client with `callTimeout(0)` for media. See
  `android-media`.
- Retry interceptors that retry non-idempotent or streaming requests
  cause stutters and duplicate downloads. Retry GET/HEAD only, cap
  attempts, exponential backoff.
- `DefaultHttpDataSource` vs `OkHttpDataSource`: the default misses
  your interceptors (auth headers, logging). If artwork or streams need
  headers, inject the OkHttp-backed factory.
- `CacheDataSource.FLAG_IGNORE_CACHE_ON_ERROR` prevents a single bad
  cache entry from hard-failing playback.

## On-device jank detection

`androidx.metrics:metrics-performance` (JankStats) reports per-frame
`isJank` and duration, and `PerformanceMetricsState` tags frames with
app state (screen name, scroll in progress) so reports say where jank
happened. Gate it on `BuildConfig.DEBUG` and throttle logging. Full
integration code and the testable throttle helper are in
[references/jankstats.md](references/jankstats.md).

Verification loop:

```
adb install -r app-debug.apk
adb shell monkey -p <pkg> -c android.intent.category.LAUNCHER 1
# scripted scroll
adb shell input swipe 540 1800 540 400 400
adb logcat -d -s Jank
```

A good fix shows as fewer janky frames on the same swipe script.

## Heavier tools when the simple answer is wrong

- `adb shell dumpsys gfxinfo <pkg> framestats` for raw per-stage
  nanosecond timing.
- Perfetto/`systrace` for CPU vs GPU vs binder breakdown.
- Macrobenchmark (`androidx.benchmark.macro`) with `FrameTimingMetric`
  for a repeatable scroll test that runs in CI. Needs a separate
  benchmark module and a non-debuggable build variant, so it is
  heavyweight, but it is the real regression gate.
- Compose compiler metrics (`composeOptions` reportsDestination) to
  audit stability annotations if recompositions look wrong.

## Rules of thumb

- If a list stutters on entry of a section, count what composes. Eager
  containers are almost always it.
- If a track stops at a round number of seconds, it is a timeout. Check
  `callTimeout` first.
- If jank is random and small, look for per-frame allocations or
  unremembered `ImageRequest`s.
- Measure before optimizing. One janky frame in 66 is not the same as
  one in 6.
