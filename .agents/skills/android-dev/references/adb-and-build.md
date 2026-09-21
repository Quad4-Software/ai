# adb and build cookbook

## Device setup

```
adb devices -l                 # confirm device authorized
adb shell settings put global window_animation_scale 0.5   # optional
```

## Install and launch

```
./gradlew assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell monkey -p <pkg> -c android.intent.category.LAUNCHER 1
```

`monkey` with the LAUNCHER category is the reliable way to start the
default activity when you do not know its name. `am start -n` requires
the fully qualified component.

## Smoke test loop

```
sleep 5
adb shell pidof <pkg>                              # alive?
adb exec-out screencap -p > /tmp/shot.png          # visual check
adb logcat -d | grep -E "<tag>|FATAL|AndroidRuntime"
```

- `adb exec-out` streams raw bytes, use it for screencap/pull binary.
- `input tap x y` and `input swipe x1 y1 x2 y2 durationMs` drive UI
 without a test framework. 400ms is a natural-feel scroll fling.
- `dumpsys activity <pkg>` shows the resumed activity/fragment state.

## Reading jank on device

```
adb shell dumpsys gfxinfo <pkg> framestats   # per-frame timing dump
adb logcat -s Jank                           # if JankStats is wired
```

`gfxinfo framestats` columns are nanosecond timestamps per pipeline
stage. Frame deadline misses show up as large `vsync`-to-`completed`
gaps. For scripted comparisons, run the same swipe sequence before and
after a fix and count janky frames.

## Compile error triage

```
./gradlew :app:compileDebugKotlin -q   # quiet, errors only
```

For API mismatches in a dependency, decompile the exact AAR:

```
unzip -o -q <lib>.aar classes.jar
unzip -o -q classes.jar -d classes
javap -classpath classes <fully.qualified.Class>
```

This beats guessing constructor overloads from memory, especially for
Media3 where `@UnstableApi` surfaces churn between minor versions.

## Gradle hygiene

- New deps: pin an exact version published at least 7 days ago, add
 to `libs.versions.toml`, reference with `libs.<alias>`.
- `testImplementation` for JVM tests, `debugImplementation` for
 anything profiling or tooling related.
- Do not edit `gradle.properties` R8 or caching flags to work around a
 failure. Find the real error first.
