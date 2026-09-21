---
name: void-theme
description: >
  The Quad4 "void" design system: a dark-space, monochrome UI theme
  (near-black canvas, hairline borders, Space Grotesk + Space Mono, subtle
  starfield) with a matching light "paper" mode. Use it when styling or
  building web UIs that should match quad4.io, reworking an app to look
  like a professional console (Sentry-like density), or when you need the
  token palette, typography, component patterns, or stacking rules.
---

## When to use this skill

- You are building or restyling a web UI for a Quad4 project.
- You want the dark "void" theme, the light "paper" theme, or both via a
  dark-mode toggle.
- You need ready-made tokens: colors, fonts, radii, shadows, z-index
  layers, and component recipes (sidebar, cards, tables, forms, toasts,
  dropdowns, modals).
- You are converting an existing utility-class UI to this look without
  rewriting every template.

## Design language

- Restrained, dense, console-like. Think Sentry or Linear, not marketing
  pages.
- Monochrome accent. No brand colors in the chrome. Links and active
  states use ink-on-paper contrast, weight, or a hairline underline, not
  hue. One exception allowed: `#0073ff` at low alpha for ::selection and
  the smallest possible focus/detail touches.
- Surfaces communicate elevation by subtle lightness steps, never by
  colored tint. Borders are hairlines (1px) at low contrast.
- Typography: Space Grotesk for UI text, Space Mono for numbers, code,
  IDs, timestamps, and stats. Vendor the woff2 files, no CDN.
- Corners: `rounded-md` (6px) for controls and menus, `rounded-lg` (8px)
  for cards and panels. No pills for interactive controls. Pills only for
  read-only status badges.
- Motion: `transition-colors` on hover states only. No transform or
  layout animations. Honor `prefers-reduced-motion`.

## Tokens

Dark void scale (canvas to raised):

```
void-950  #0a0a0b   canvas / page background
void-900  #0d0d10   recessed background, tab strips
void-850  #101013   subtle hover on canvas, secondary surface
void-800  #16161a   cards, panels, inputs
void-700  #1f1f24   hairline borders
void-600  #2a2a31   strong borders, dividers
void-500  #3f3f46   disabled text
void-400  #71717a   secondary text
void-300  #a1a1aa   muted-emphasis text, icons
void-200  #d9d9de   body text on dark
void-100  #e9e9ec   emphasis text
void-50   #f4f4f5   inverted accent surface (buttons)
```

Light paper scale (same slots, inverted roles):

```
paper-50   #fafafa  canvas
paper-100  #f4f4f5  recessed / hover
paper-200  #e9e9ec  hairline borders
paper-300  #d9d9de  strong borders
paper-400  #a1a1aa  secondary text
paper-500  #71717a  muted-emphasis
paper-900  #18181b  body text
paper-950  #0a0a0b  emphasis, inverted accent surface
```

Rule: body text is `void-200`-range on dark, `paper-900` on light.
Secondary text is one step dimmer. Never use pure `#000`/`#fff` except
for inverted button faces.

## Typography

```css
@font-face {
  font-family: 'Space Grotesk';
  src: url('space-grotesk-latin-400-normal.woff2') format('woff2');
  /* also 500 and 700 */
}
@font-face {
  font-family: 'Space Mono';
  src: url('space-mono-latin-400-normal.woff2') format('woff2');
  /* also 700 */
}
body { font-family: 'Space Grotesk', system-ui, sans-serif; }
```

- `font-mono` for: numbers in stat cards, IDs/hashes, timestamps,
  exception values, code, DSN strings.
- Headings: `text-xl font-semibold tracking-tight` for page titles.
  Section labels: `text-xs font-semibold uppercase tracking-wider` in
  muted color.

## Backgrounds

The dark canvas may carry a very subtle starfield (tiny 1-1.5px radial
dots at ~30-55% white alpha, fixed, `pointer-events: none`). Use it on
auth/splash surfaces. Inside the app keep the canvas plain. Example
layering: several `radial-gradient(1px 1px at <x>% <y>%, #ffffff73 50%,
transparent 51%)` layers composited on the void canvas.

## Layout

- Desktop: fixed left sidebar (`w-56`, `sticky top-0 h-screen`, own
  border-right hairline) with a mark + title header row, icon+label nav
  items, and a bottom cluster (admin, tokens, API, preferences, log out).
- Mobile: top header with mark and a disclosure menu (checkbox-hack or
  JS). The sidebar is `hidden md:flex`.
- Content column: `flex-1 min-w-0`, page padding `px-4 md:px-6 py-5`.
- Active nav item: `bg-<surface-hover>` + `font-semibold`, inactive:
  muted text with hover to full contrast.

## Component recipes

Buttons:

- Primary: inverted mono. Dark mode: `bg-void-50 text-void-950`. Light
  mode: `bg-paper-950 text-paper-50`. `text-sm font-semibold px-3 py-1.5
  rounded-md shadow-sm`, hover one step lighter/darker.
- Secondary: `border` hairline + surface bg, muted text, hover raises.
- Danger: same shape, `text-red-500`/`border-red-800` tint, subtle
  `hover:bg-red-950/40`.
- Disabled: muted text + `cursor-not-allowed`, no hover change.
- Joined groups use `rounded-s-md`/`rounded-e-md` on the ends.

Cards:

- `rounded-lg border border-<hairline> bg-<surface> shadow-sm`.
- Do not put `overflow-hidden` on cards that contain dropdown menus. It
  clips the menu. Round the inner content instead (first/last row
  corner radii) or accept the 1px corner bleed.

Tables/lists:

- Hairline row separators (`border-t first:border-t-0`), `px-4 py-3`
  cells, `hover:bg-<surface-850>` + `transition-colors` on rows.
- Numeric columns right-aligned and `font-mono`.

Forms:

- Inputs: `rounded-md border border-<hairline> bg-<surface>`,
  `focus:ring-2 focus:ring-<muted> focus:border-<strong>`, `px-3 py-2`.
- Labels: `text-sm` muted. Errors: red-tinted border + `text-sm` red
  message.
- Selects: same border/radius treatment as inputs.

Dropdowns:

- Trigger: `position: relative` wrapper. Menu: `position: absolute;
  top: 100%; padding-top: 2px` (padding, not margin, so hover does not
  break across the gap). `right: 0` to align right.
- Menu container: `z-index: 50`, `rounded-md`, `overflow: hidden`,
  `box-shadow: 0 10px 30px rgba(0,0,0,.28)`. Items carry their own
  hairline `border-b` separators and surface bg.
- Hover-only menus are acceptable for action menus but prefer
  click-toggle for touch targets.

Toasts:

- Fixed bottom-right stack, `z-60` if dropdowns are `z-50`, surface card
  with icon + message + dismiss X, auto-dismiss ~5s.

Modals:

- `fixed inset-0 z-50` scrim (`bg-void-950/60` or paper equivalent),
  centered rounded-lg card, focus the primary action.

Badges:

- Read-only pills: `rounded-full text-xs font-medium px-2 py-0.5` in
  muted surface colors. Semantic colors only for status (ok=green,
  warn=amber, error=red), all at muted/low-alpha tints.

## Z-index layers

```
base content        0
dropdown menus      50
modals / scrims     50
toasts              60
```

Keep it flat. If a dropdown renders under another element, check for an
`overflow-hidden` ancestor clipping it (overflow clipping beats z-index)
before raising the layer.

## Accessibility

- Contrast: body text must hit WCAG AA on its surface (void-200 on
  void-950 passes. Do not go dimmer than void-400 for readable text).
- Focus: every interactive element needs a visible focus state
  (ring-2 in a muted neutral).
- Keyboard: menus and modals must be escapable. Inputs keep `label`
  associations.
- `prefers-reduced-motion`: disable starfield drift and transitions.
- Icons are 1.5px-stroke outline style (Heroicons outline or similar),
  `size-5` in nav, `size-4` inline.

## Tailwind adoption pattern

When a codebase already uses a different palette in thousands of places,
do not rewrite the templates. Remap the existing scale names in
`tailwind.config.js` to the void/paper values so `slate-*`/`gray-*`/
accent classes pick up the theme globally, then hand-fix only the chrome
(layout shell, nav, forms, key screens). Add a `*-850` step for the
between-surface hover state.
