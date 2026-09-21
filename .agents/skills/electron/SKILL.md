---
name: electron
description: >
  This skill covers Electron, the Chromium + Node desktop app
  framework, as of September 2026 (Electron 44 stable, 45 beta, 46
  nightly). Use it for version selection and EOL windows, the Windows
  sandbox and Job Object failure modes (V8 cage, Chromium broker,
  LPAC/AppContainer), packaging and auto-update plumbing (Forge,
  electron-builder, electron-updater, Squirrel, MSIX, signing,
  notarization), and the renderer hardening checklist (contextIsolation,
  sandbox, fuses, CSP, IPC validation).
---

## When to use this skill

- You are building, packaging, or updating an Electron app.
- An Electron app fails to start under Windows Job Objects, Windows
  Sandbox, AppContainer, Session 0, or EDR containment.
- You are choosing an update path: autoUpdater vs electron-updater vs
  update servers.
- You are hardening renderer/preload security or flipping fuses.

## How to use

1. Read this file for version and support facts plus the sandbox
   warning.
2. Load [references/versions.md](references/versions.md) for the last
   six majors, per-version breaking changes, and what is coming in
   Electron 45 and 46.
   Load [references/windows-sandbox.md](references/windows-sandbox.md)
   for the Job Object, V8 sandbox, LPAC, and AppLocker failure
   mechanics and workarounds.
   Load [references/packaging.md](references/packaging.md) for Forge
   vs electron-builder, update servers, signing, fuses, and MSIX.
3. Primary docs: https://www.electronjs.org/docs/latest and
   https://releases.electronjs.org

## Examples

- "Electron app renders blank inside our Job Object - why?"
- "Set up auto-updates for Windows and macOS."
- "What breaks when I upgrade from Electron 41 to 44?"

# Electron

## Versions and support

- Major versions follow Chromium milestones on an **8-week cadence**
  (every other Chromium release. Since Chrome moved to 2-week cycles
  in Sept 2026, Electron aligns to Extended Stable milestones). Alpha
  4 weeks, beta 4 weeks, stable lands with the Chrome stable.
- **Only the latest 3 stable majors get fixes.** Currently Electron
  42, 43, 44. Electron 44.0.0 shipped Aug 25, 2026 (Chromium 152,
  Node 24.18, V8 15.2). Electron 45 stable Oct 20, 2026 (M156),
  Electron 46 stable Jan 5, 2027 (M160).
- Notable recent breaks: E42 stopped downloading the binary in
  postinstall (lazy download, enables `npm --ignore-scripts`) and
  moved macOS notifications to UNNotification. E44 removed 32-bit
  builds, dropped macOS 12, statically linked ANGLE, and removed the
  renderer `clipboard` module. E45 removes Node shims and
  Buffer/setImmediate globals from sandboxed preloads.

## Windows sandbox warning

Do not run an Electron app inside a Windows Job Object that sets UI
restrictions, inside Windows Sandbox, inside a real AppContainer, or
as a service in Session 0. The Chromium broker spawns renderers
suspended, then assigns them to a sandbox job with
JOB_OBJECT_UILIMIT_* restrictions. Windows only permits job nesting
when neither job sets UI limits, so `AssignProcessToJobObject` fails
and the child is terminated: `render-process-gone
launch-failed`, GPU crash loops, or a blank window. The V8 sandbox
also reserves a ~1 TB address-space cage and commit limits from an
outer job can starve it. LPAC sandboxing for the GPU and network
processes additionally requires the install dir ACL'd to
ALL RESTRICTED APPLICATION PACKAGES (S-1-15-2-2) - missing or
unresolvable ACEs cause `GPU process isn't usable. Goodbye.`
Workarounds, escalating: fix the outer job (no UI limits, allow
nesting/breakaway, grant the LPAC SID), `app.disableHardwareAcceleration()`,
`--disable-gpu-sandbox`, and only as a last resort `--no-sandbox`
(which forfeits renderer isolation). Details and real-world issue
links are in references/windows-sandbox.md.

## Updates and packaging

- Forge is the official toolchain (v7.x stable, v8 on `next`),
  electron-builder v27 is the community standard with the broadest
  target matrix. `@electron/packager` is the low-level lib Forge
  wraps.
- Built-in `autoUpdater`: Squirrel.Mac (needs signing) and
  Squirrel.Windows or MSIX updater on Windows. Nothing built-in on
  Linux.
- `electron-updater` (electron-builder sibling): NSIS, AppImage, deb,
  rpm, mac zip. Differential downloads, staged rollouts via
  `stagingPercentage`, channels, signature verification. It does NOT
  support Squirrel.Windows - use NSIS.
- Free hosted updater for OSS: update.electronjs.org +
  update-electron-app. Self-host: Hazel, Nuts, electron-release-
  server, Nucleus, or static hosting of the latest*.yml manifests.
- Signing: Windows needs Authenticode (Azure Trusted Signing is the
  modern cheap path). MacOS needs Developer ID + notarization via
  @electron/notarize. MSIX must be signed and runs full-trust, never
  AppContainer.
- Fuses (`@electron/fuses`) flip post-build security bits in the
  binary: RunAsNode, cookie encryption, NODE_OPTIONS, embedded asar
  integrity validation (turn it on), OnlyLoadAppFromAsar. Set
  `strictlyRequireAllFuses` so upgrades fail loudly on new fuses.

## Renderer hardening baseline

Defaults are sane in current majors but verify every webPreferences
block: `nodeIntegration: false`, `contextIsolation: true`,
`sandbox: true` (default on since E20), permission handlers
deny-by-default, CSP `script-src 'self'`, `setWindowOpenHandler`
allowlists, `will-navigate` limits, validate `event.senderFrame` on
every IPC handler, never `shell.openExternal` on untrusted URLs, and
prefer `utilityProcess` + contextBridge over exposing Node to
renderers. Full 20-item checklist: docs/tutorial/security.md.
