---
name: wails
description: >
  This skill covers Wails, the Go + webview desktop app framework, with
  focus on v3. Use it for v3 beta status and version pinning, the wails3
  CLI, the application/services/manager-API model, bindings generation,
  Taskfile-based builds, platform requirements (WebView2, WKWebView,
  GTK4/WebKitGTK 6), packaging and signing, and v2-to-v3 migration.
---

## When to use this skill

- You are starting a Wails project or choosing between v2 and v3.
- You are writing Go services exposed to the frontend (bindings).
- You are wiring `wails3 dev`, `wails3 build`, or `wails3 package`.
- You are debugging Linux webkit deps, WebView2 issues, or dev-server
  port problems.
- You are migrating a v2 app to v3.

## How to use

1. Read this file for the release status, the v3 mental model, and the
   build/dev flow.
2. Load [references/api.md](references/api.md) for the manager API,
   service/binding rules, events, and window options.
3. Fall back to https://v3.wails.io/ for v3 docs (v2 docs stay at
   https://wails.io/).

## Examples

- "Scaffold a v3 app and add a service with a method the frontend calls."
- "Why does wails3 dev fail with 'unable to connect to frontend server'?"
- "What Linux packages do I need for a GTK4 build vs a GTK3 build?"
- "Explain the v3 build system and where wails.json went."
- "Package a signed macOS universal binary from the Taskfile."

# Wails v3

Wails wraps a Go backend in a native webview: WebView2 on Windows,
WKWebView on macOS, WebKitGTK on Linux. The frontend is any web stack,
served from embedded assets in production or a Vite dev server in
development.

## Release status

- **v3 is beta, not GA.** Promoted from alpha on 2026-08-02 with
  `v3.0.0-beta.0`. Nightly beta tags are cut from master. Confirmed
  tags include `v3.0.0-beta.20`/`beta.21` (check
  github.com/wailsapp/wails/releases for the newest). The desktop API
  is declared stable in the beta, but **v2 remains the current stable
  release** (latest `v2.11.0`, Nov 2025).
- **Beta scope:** Windows 10/11 amd64+arm64, macOS Intel+Apple Silicon,
  Linux amd64+arm64. **Go 1.25+ required.** Android/iOS are
  experimental and outside the compatibility promise.
- Releases are tag-only since ~beta.8. Install the CLI with
  `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`, or pin
  the tag matching your `go.mod` since betas cut nightly and can drift.
- v3 docs live at https://v3.wails.io/.

## The v3 model

v2's monolithic `wails.Run(&options.App{...})` becomes three phases:

```go
//go:embed all:frontend/dist
var assets embed.FS

app := application.New(application.Options{
    Name: "My App",
    Services: []application.Service{
        application.NewService(&GreetService{}),
    },
    Assets: application.AssetOptions{
        Handler: application.AssetFileServerFS(assets),
    },
})
app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title: "My App", Width: 1024, Height: 768,
})
err := app.Run()
```

- `application.New(application.Options{...})` constructs the app.
- `app.Window.New()`/`NewWithOptions(WebviewWindowOptions)` creates
  windows at any time, before or after `app.Run()`. Multi-window is
  first-class.
- `app.Run()` blocks and returns an error.

`application.Options` holds `Name`, `Description`, `Icon`, `Services`,
`Assets`, platform blocks (`Mac`, `Windows`, `Linux`, `IOS`), plus
`ShouldQuit func() bool` and `OnShutdown func()`. There is no
`OnStartup` option. Startup work goes in `ServiceStartup(ctx,
options)` on a service or before `app.Run()`.

## Services replace Bind

v2 `Bind: []interface{}{...}` becomes
`Services: []application.Service{application.NewService(&svc)}`.
Rules:

- The value must be a pointer to a named struct type. Plain functions
  and generic types are rejected.
- Only exported (PascalCase) methods bind. A leading
  `context.Context` param is auto-injected. Variadic params and
  multiple return values work. `error` goes last.
- A Go `error` surfaces as a Promise rejection (`RuntimeError` with
  `.message` and `.cause`, while arg-count/conversion failures produce
  `TypeError`).
- Services can implement `ServiceStartup(ctx, options) error` and
  `ServiceShutdown() error`, and can mount HTTP handlers on the asset
  server.

`wails3 generate bindings` emits `frontend/bindings/<pkg>/<service>.js`
plus `index.js` and `models.js`/`models.ts` (TS via `-ts`). Calls are
hash-based (`$Call.ByID(...)`) unless `-names` is passed. The frontend
runtime is the `@wailsio/runtime` npm package.

Manager API, events, and window options:
[references/api.md](references/api.md).

## CLI

`wails3` is a separate binary from v2's `wails`.

- `wails3 setup` environment wizard, `wails3 init -n app -t vanilla`,
  `wails3 dev`, `wails3 build`, `wails3 package`, `wails3 doctor`.
- `wails3 dev` starts Vite on port **9245** (flag `-port`, env
  `WAILS_VITE_PORT`) and sets `FRONTEND_DEVSERVER_URL` so the asset
  handler proxies the dev server. Vite needs `strictPort` on that
  port or dev fails with "unable to connect to frontend server". A
  watcher rebuilds and relaunches the Go binary on `*.go` changes.
- `wails3 build` is a thin wrapper over the Taskfile `build` task.
  Only `--tags`, `--obfuscated`, `--garbleargs` remain as flags. The
  old `-platform`, `-o`, `-ldflags`, `-clean`, `-debug`, `-package`
  flags are gone. That config lives in Taskfiles and
  `build/config.yml`.
- `wails3 task [name]` runs any Taskfile task. `wails3 generate
  bindings` produces frontend bindings. Other `generate` subs cover
  `build-assets`, `constants`, `template`, `.desktop`, `appimage`,
  icons.
- `wails3 setup signing`/`setup entitlements`, `wails3 sign`, and
  `wails3 tool sign` handle signing and notarization.

## Project layout

`wails.json` is gone. Orchestration is `Taskfile.yml` at the root
plus a `build/` directory:

- `build/config.yml`: project metadata (name, identifier, version,
  Info.plist, NSIS, .desktop, protocols).
- `build/Taskfile.yml`: common tasks.
- `build/{windows,darwin,linux}/Taskfile.yml`: platform tasks.

Regenerate with `wails3 generate build-assets` or `wails3 update
build-assets`. Binaries land in `bin/<APP_NAME>`, not `build/bin/`.

## Platform requirements

| OS | Engine | Notes |
|---|---|---|
| Windows | WebView2 | Preinstalled on Win10/11. NSIS installer bundles a bootstrapper. Offline needs the Evergreen Standalone Installer. |
| macOS | WKWebView | Custom scheme via `WKURLSchemeHandler`. Min target `-mmacosx-version-min=12.0`. |
| Linux | GTK4 + WebKitGTK 6.0 | Default since beta. GTK3 + WebKit2GTK 4.1 is legacy opt-in via `-tags gtk3`, supported through v3.0.x, removal planned v3.1. |

- Linux dev deps (default): `libgtk-4-dev libwebkitgtk-6.0-dev`
  (Debian/Ubuntu), `gtk4-devel webkitgtk6.0-devel` (Fedora),
  `gtk4 webkitgtk-6.0` (Arch). Legacy `-tags gtk3` path:
  `libgtk-3-dev libwebkit2gtk-4.1-dev` (Ubuntu 22.04, Debian 12,
  Fedora <=39, RHEL 9). `wails3 doctor` diagnoses both.
- CGO is required on Linux for the GTK bindings.

## Packaging and cross-builds

- Windows: NSIS installer (`build/windows/nsis/<app>-installer.exe`)
  or MSIX via `FORMAT=msix`. The installer carries a WebView2
  bootstrapper.
- macOS: `.app` bundle in `bin/`. Universal binary via
  `darwin:package:universal` (`wails3 tool lipo`, works from
  Linux/Windows). DMG via `darwin:package:dmg`.
- Linux: AppImage, DEB, RPM, Arch packages via `linux:create:*`
  tasks. Nfpm config in `build/linux/nfpm/nfpm.yaml`. PGP signing via
  `linux:sign:*`.
- Cross builds run through per-platform Taskfiles that fall back to a
  Docker `wails-cross` image when CGO or SDKs are missing. Linux
  always needs CGO so it always crosses through Docker. Windows
  crossing uses native Go when `CGO_ENABLED=0`.
- Signing: `wails3 setup signing` stores creds in the OS keychain.
  Windows EXE and Linux DEB/RPM sign from any OS. MacOS
  signing/notarization runs on macOS only.
- Obfuscation: `wails3 build --obfuscated` uses Garble (needs
  `go install mvdan.cc/garble@v0.16.0`). Generate bindings with
  `-obfuscated` for stable IDs. UPX post-build is available but not
  recommended on macOS (breaks signing).

## v2 -> v3 migration

Not a drop-in upgrade. Key changes:

1. `wails.Run(&options.App{...})` -> `application.New(...)` +
   `app.Window.NewWithOptions(...)` + `app.Run()`.
2. `Bind` -> `Services`. `OnStartup`/`OnShutdown` ->
   `ServiceStartup(ctx, options)`/`ServiceShutdown()`.
3. `runtime.*` calls -> methods on `app`/window objects
   (`app.Event.Emit`, `window.SetTitle`).
4. Events become typed objects (`CustomEvent`, `ApplicationEvent`,
   `WindowEvent`). Frontend uses `Events.On/Emit` from
   `@wailsio/runtime`.
5. Menus: `app.NewMenu()` + `app.Menu.Set(menu)` replaces the `Menu:`
   option.
6. `wails.json` -> `build/config.yml` + Taskfiles. Bindings move to
   `frontend/bindings/` with the `@wailsio/runtime` dep.

Parity gaps in the beta: no general plugin system (the service model
is designed for it), clipboard is text-only, Android/iOS are
experimental, and the automated migration assistant is unreleased.
The GTK3 path is deprecated and slated for removal in v3.1.

## Gotchas

- Pin the CLI tag to your `go.mod` (`go install
  .../cmd/wails3@v3.0.0-beta.N`). Betas cut nightly and drift.
- DevTools open via right-click -> Inspect in dev, or
  `application.OpenDevTools()` behind the `runtimedevtools` build
  tag (macOS 12+, dev builds only).
- `embed.FS` is still required (`//go:embed all:frontend/dist`). In
  dev the FS handler is bypassed by the `FRONTEND_DEVSERVER_URL`
  proxy.
- Service must be `*NamedStruct`. No bound functions, no generics.
- `models.ts` only appears with `-ts`.
- The docs' install page still says Go 1.24. The release contract is
  **1.25+**.
