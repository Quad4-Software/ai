# JankStats integration

`androidx.metrics:metrics-performance:1.0.0` reports every frame with
`isJank` and duration. Tag frames with the current screen so reports
say where the drop happened.

## Dependency

```toml
# libs.versions.toml
metricsPerformance = "1.0.0"
androidx-metrics-performance = { module = "androidx.metrics:metrics-performance", version.ref = "metricsPerformance" }
```

```kotlin
implementation(libs.androidx.metrics.performance)
```

## Monitor

```kotlin
class JankMonitor private constructor(
    private val stats: JankStats,
    private val metricsState: PerformanceMetricsState?,
) {
    private var frames = 0
    private var jankFrames = 0
    private var worstMs = 0.0
    private var screen = "unknown"

    private fun onFrame(frameData: FrameData) {
        frames++
        if (!frameData.isJank) return
        jankFrames++
        val ms = frameData.frameDurationUiNanos / 1_000_000.0
        if (ms > worstMs) worstMs = ms
        if (jankShouldLog(jankFrames)) {
            AppLog.w("Jank", "janky ${ms.toInt()}ms ${jankSummary(screen, jankFrames, frames, worstMs)}")
        }
    }

    fun setScreen(name: String) {
        screen = name.substringAfterLast('.').substringBefore('/')
        metricsState?.putState("screen", screen)
    }

    companion object {
        fun install(activity: Activity): JankMonitor? {
            if (!BuildConfig.DEBUG) return null
            return runCatching {
                val holder = PerformanceMetricsState.getHolderForHierarchy(
                    activity.window.decorView,
                )
                var monitor: JankMonitor? = null
                val stats = JankStats.createAndTrack(activity.window) { frameData ->
                    monitor?.onFrame(frameData)
                }
                JankMonitor(stats, holder.state).also { monitor = it }
            }.getOrNull()
        }
    }
}
```

## Wire into the activity

```kotlin
// in onCreate, after setContent { }
jankMonitor = JankMonitor.install(this)

// inside setContent, tag frames with the nav route
val backStack by navController.currentBackStackEntryAsState()
LaunchedEffect(backStack?.destination?.route) {
    jankMonitor?.setScreen(backStack?.destination?.route ?: "unknown")
}
```

## Throttle helper (unit-testable)

```kotlin
internal fun jankShouldLog(jankCount: Int): Boolean =
    jankCount == 1 || (jankCount > 0 && jankCount % 8 == 0)

internal fun jankSummary(screen: String, jank: Int, total: Int, worstMs: Double): String =
    "screen=$screen jank=$jank/$total worst=${worstMs.toInt()}ms"
```

Trap to avoid: `0 % 8 == 0` is true, so a bare `count % 8 == 0` logs
count 0. Guard with `count > 0` or test for it. The first jank frame
always logs, then every 8th, so a scrolling session produces a handful
of lines not a flood.

## Reading output

```
adb logcat -d -s Jank
# Jank: janky 45ms screen=HomeRoute jank=1/66 worst=45ms
```

## Macrobenchmark skeleton (for a real regression test)

```kotlin
@get:Rule val rule = MacrobenchmarkRule()

@Test
fun scrollHome() = rule.measureRepeated(
    packageName = "<pkg>",
    metrics = listOf(FrameTimingMetric()),
    iterations = 5,
    startupMode = StartupMode.COLD,
) {
    pressHome()
    startActivityAndWait()
    val list = device.findObject(By.scrollable(true))
    list.fling(Direction.DOWN)
    list.fling(Direction.UP)
}
```

Needs a `benchmark` module, `benchmark` build type (non-debuggable,
release-ish), and `androidx.benchmark.macro.junit4`. Only set it up if
you need CI-level regression gating. JankStats covers day-to-day.
