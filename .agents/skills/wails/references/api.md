# Wails v3 API reference

Import root: `github.com/wailsapp/wails/v3/pkg/application`.

## Manager API

`*application.App` groups functionality under public fields:

- `app.Window.*`: `New()`, `NewWithOptions(WebviewWindowOptions)`,
  `Current()`, `GetByName(name)`, `GetAll()`, `OnCreate(fn)`.
- `app.Event.*`: `Emit(name, data)`, `On(name, func(*CustomEvent))`,
  `OnApplicationEvent(eventType, func(*ApplicationEvent))`.
- `app.Menu.*`: `app.NewMenu()` builds, `app.Menu.Set(menu)` applies.
- `app.Dialog.*`, `app.Clipboard.*`, `app.Browser.*`, `app.Autostart.*`,
  `app.Updater.*`, `app.Logger.*`, `app.Env.*`, `app.Screen.*`.

v2 `runtime.*` ctx-threaded calls map to these fields. A service holds
`app *application.App` (set in `ServiceStartup`) and calls
`s.app.Window.Current().SetTitle(...)` style methods.

## Services and bindings

```go
type GreetService struct{ app *application.App }

func (s *GreetService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
    s.app = options.App
    return nil
}
func (s *GreetService) ServiceShutdown() error { return nil }

func (s *GreetService) Greet(ctx context.Context, name string) (string, error) {
    return "hello " + name, nil
}
```

- `application.NewService(&GreetService{})`: pointer to a named
  struct only. Functions and generic types are rejected.
- Only exported methods bind. Leading `context.Context` is
  auto-injected. Variadic params and multiple returns work. `error`
  conventionally last.
- Services can mount HTTP routes on the asset server
  (`AttachServiceHandler(route, handler)`).

## Bindings generation

`wails3 generate bindings` writes:

- `frontend/bindings/<pkg>/<servicename>.js` plus `index.js`
- `models.js` or `models.ts` (`-ts`. Interfaces via `-i`, classes get
  `createFrom`)

Flags: `-d` output dir (default `frontend/bindings`), `-names` for
`$Call.ByName("pkg.Struct.Method", ...)` instead of hash IDs
(`$Call.ByID(3576998831, ...)`), `-b` bundle the runtime, `-dry`,
`-clean`, `-obfuscated`/`-obfuscated-output` for Garble builds.

Frontend runtime: `import { Call as $Call, Events, Browser, Window }
from '@wailsio/runtime'`. Optional Vite plugin
`@wailsio/runtime/plugins/vite` for typed events.

## Error model

A Go `error` return rejects the frontend Promise. Post-#5690 the
rejections are typed:

- `TypeError`: arg count or conversion failure.
- `RuntimeError`: Go error or panic (exported by
  `@wailsio/runtime`).
- `Error`: anything else.

Each carries `.name`, `.message` (the Go error string), and `.cause`
(the JSON-serialised Go error, or an array for multiple errors).

## Events

```go
import "github.com/wailsapp/wails/v3/pkg/events"
```

- Custom events: `app.Event.Emit("name", data)` and
  `app.Event.On("name", func(e *application.CustomEvent) { ... })`
  (`.Name`, `.Data`, `.Sender`). Frontend mirrors via `Events.On`/
  `Events.Emit`.
- Typed system events: `events.Common.*`, `events.Windows.*`,
  `events.Mac.*`, `events.Linux.*`, generated in
  `v3/pkg/events/events.go`. Namespaces `common:`, `windows:`,
  `mac:`, `linux:` (e.g. `common:WindowFocus`,
  `windows:APMSuspend`, `mac:ApplicationDidBecomeActive`).
- Registration: `app.Event.OnApplicationEvent(events.Common.
  ApplicationStarted, func(e *application.ApplicationEvent) {...})`
  and `window.OnWindowEvent(events.Common.WindowFocus, func(e
  *application.WindowEvent) {...})`.
- Cancellable hooks: `window.RegisterHook(events.Common.WindowClosing,
  func(e) { e.Cancel() })`: a listener cannot cancel, a hook can.
- Event types: `ApplicationEvent`, `WindowEvent`, `CustomEvent`.

## Windows

`app.Window.New()` or `NewWithOptions(application.WebviewWindowOptions{...})`.
Windows can be created before or after `app.Run()`.

`WebviewWindowOptions` fields include `Name`, `Title`, `Width`,
`Height`, `Frameless`, `AlwaysOnTop`, `StartHidden`, `Hidden`,
`HideOnFocusLost`, `HideOnEscape`.

- **Frameless:** `Frameless: true` + CSS drag regions. Windows extras:
  `NonClientRegionSupport` (WebView2 `app-region: drag/no-drag`) and
  `WebView2CompositionHosting` (composition-hosted caption buttons,
  incl. Snap Layouts). macOS has `squareCorners`/`cornerRadius`.
- **macOS quit behavior:** `Mac: application.MacOptions{
  ApplicationShouldTerminateAfterLastWindowClosed: ... }`.

## Systray

`systray := app.NewSystemTray()`, `systray.SetMenu(...)`,
`systray.AttachWindow(window)`, `systray.WindowOffset`,
`WindowDebounce`, `OnClick`/`OnRightClick` overrides, light/dark/
template icons. `HideOnFocusLost` auto-disables on
focus-follows-mouse Linux WMs (Hyprland, Sway, i3).

## Asset server

`application.Options{ Assets: application.AssetOptions{ Handler:
<http.Handler>, Middleware: <Middleware>, DisableLogging: bool } }`.

- Handlers: `application.AssetFileServerFS(fs.FS)` or
  `application.BundledAssetFileServer(fs.FS)`.
- `Middleware` is `func(next http.Handler) http.Handler`. Chain with
  `application.ChainMiddleware(...)`. Either `Handler` or
  `Middleware` is required.
- Dev mode: env `FRONTEND_DEVSERVER_URL` makes the handler a reverse
  proxy to Vite.
- `embed.FS` still required (`//go:embed all:frontend/dist`).
