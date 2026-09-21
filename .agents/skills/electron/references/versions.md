# Electron versions

Cadence: 8 weeks per major, following every other Chromium milestone.
Since Chrome moved to a 2-week cycle with Chrome 153 (Sept 8, 2026),
Electron aligns to Extended Stable milestones (M156, M160, every 4th).
Alpha 4 weeks, beta 4 weeks. Support window: latest 3 stable majors.

## Last six stable majors

| Electron | Stable | EOL | Chromium | Node | V8 |
| --- | --- | --- | --- | --- | --- |
| 39 | Oct 28 2025 | May 5 2026 | 142 | 22.20.0 | 14.2 |
| 40 | Jan 13 2026 | Jun 30 2026 | 144 | 24.11.1 | 14.4 |
| 41 | Mar 10 2026 | Aug 25 2026 | 146 | 24.14.0 | 14.6 |
| 42 | May 5 2026 | Oct 20 2026 | 148 | 24.15.0 | 14.8 |
| 43 | Jun 30 2026 | Jan 5 2027 | 150 | 24.17.0 | 15.0 |
| 44 | Aug 25 2026 | Mar 2 2027 | 152 | 24.18.1 | 15.2 |

Supported now: 42, 43, 44. Latest patch mid-Sept 2026: 44.4.0.

## Per-major breaking changes

**E39**: OffscreenSharedTexture signature change (unified `handle`),
window.open popups always resizable. Added accent color on
Windows/Linux, `webFrameMain.fromFrameToken`, USBDevice.configurations,
HDR RGBAF16 offscreen output.

**E40**: renderer `clipboard` access deprecated (contextBridge it),
macOS dSYMs moved to tar.xz. Added `app.isHardwareAccelerationEnabled()`,
WebSocket auth via `login` event.

**E41**: PDFs no longer create a separate WebContents (OOPIF viewer,
use the frame tree). Cookie `changed` event cause values renamed,
`showHiddenFiles` deprecated on Linux. Added WasmTrapHandlers fuse,
MSIX auto-updating, `app.configureWebAuthn`, extended Windows toast
actions.

**E42**: macOS notifications migrated to UNNotification (unsigned
apps get `failed`). The `electron` npm package no longer downloads
the binary in postinstall - it lazy-downloads on first `bin` run, so
`npm --ignore-scripts` now works and ELECTRON_SKIP_BINARY_DOWNLOAD
was removed. Offscreen rendering default deviceScaleFactor is 1.0,
`quotas` removed from Session.clearStorageData.

**E43**: downloads default to the user Downloads folder. NativeImage
normalizes to sRGB. Frameless windows on Linux get rounded corners
by default (`roundedCorners: false` to opt out). `showHiddenFiles`
removed on Linux. Added `WebContents.clone()`,
`globalShortcut.setSuspended()`, `Notification.getHistory()` on macOS.

**E44** (biggest recent break set): removed all 32-bit builds (win
ia32, linux armv7l). Dropped macOS 12 (needs 13+). ANGLE statically
linked (no more libEGL/libGLESv2 shipped). `clipboard` fully removed
from renderers. `net.request`/`net.fetch` gained
`select-client-certificate`. Sec-Fetch-Dest nav restrictions. Unity
DE support removed. `openAsHidden` login options removed.

## Upcoming

**E45** - beta Sept 29 2026, stable Oct 20 2026, Chromium M156, Node
24.19. Planned breaks: Node module shims and Buffer/setImmediate/
clearImmediate globals removed from sandboxed preloads (`require`
will only load `electron` - use Web APIs or bundle polyfills),
ipcRenderer/process in sandboxed preloads switch to a native
EventEmitter. New: localAIHandler in UtilityProcess (Prompt API),
iCloud Keychain passkeys, cross-platform save/restore window state,
`webFrameMain.printToPDF()`, `disableWakeLocks` webPreference.

**E46** - stable Jan 5 2027, Chromium M160.
