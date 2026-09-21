---
name: web-perf
description: >
  This skill covers web performance profiling and testing as of
  September 2026: Core Web Vitals (LCP, INP, CLS), lab tools
  (Lighthouse, WebPageTest, DevTools Performance panel,
  chrome-devtools-mcp, Playwright tracing), slow-connection testing
  (DevTools throttling vs tc/netem vs Toxiproxy), heap and leak
  profiling (heap snapshots, memlab, Node --heap-prof), RUM
  collection with web-vitals and PerformanceObserver, and heatmap
  or analytics tooling (Microsoft Clarity, PostHog, Plausible,
  Umami). Use it for slow pages, jank, memory leaks, low-end-device
  or bad-network testing, and picking analytics.
metadata:
  sources:
    - https://web.dev/articles/vitals
    - https://developer.chrome.com/docs/devtools
    - https://www.webpagetest.org/
    - https://github.com/facebookincubator/memlab
---

## When to use this skill

- A page or app is slow, janky, or leaking memory.
- Testing behavior on slow networks or low-end devices.
- Setting up RUM collection or picking analytics/heatmap tooling.
- Gating performance in CI.

## First principles

- Field data is truth, lab data is for diagnosis. A perfect
  Lighthouse score can still fail in the field. Judge by p75 of
  real users, segment mobile and desktop.
- Measure before and after. Record the baseline numbers.
- Throttle to match your users' reality: DevTools calibrated
  mid-tier mobile preset plus Slow 4G is the defensible default.

## Core Web Vitals

Judged at p75 of field data (CrUX, 28-day rolling). Lab scores do
not affect ranking.

| Metric | Good | Needs work | Poor |
| --- | --- | --- | --- |
| LCP | <= 2.5s | 2.5-4.0s | > 4.0s |
| INP | <= 200ms | 200-500ms | > 500ms |
| CLS | <= 0.1 | 0.1-0.25 | > 0.25 |

INP replaced FID in March 2024. It measures full interaction
latency across all interactions, reporting a near-worst value.
LCP sub-parts (TTFB, resource load delay, resource load duration,
render delay) are surfaced in DevTools and Lighthouse. LoAF (long
animation frames, >50ms, per-script attribution) is the successor
to long tasks. Ignore SEO blogs claiming tighter thresholds: the
table above is current.

## Lab tools

- Lighthouse: `npm install -g lighthouse`, then
  `lighthouse <url> --preset=desktop`. Default mobile profile is
  ~150ms RTT, 1.6 Mbps down, 4x CPU. Use
  `--throttling-method=devtools` for packet-level realism. Gate CI
  with `lhci autorun` plus a `budget.json`.
- WebPageTest: real devices (Moto G class) are the gold standard
  for low-end testing. Scripting language supports navigate,
  setCookie, block, click. Waterfall for render-blocking chains,
  filmstrip for paint progression.
- Chrome DevTools Performance panel: Live Metrics screen shows
  real-time LCP/INP/CLS with optional CrUX overlay. Insights
  sidebar flags LCP phases, layout-shift culprits, third parties,
  and the network dependency tree.
- `npx chrome-devtools-mcp@latest` (public preview since Sept
  2025): trace capture and analysis from an agent session.
- Playwright tracing (`trace: 'on-first-retry'` in CI) is a
  debugging tool, not a profiler. Pair with `page.metrics()` or
  CDP for numbers.

## Slow connections

Three levels, use the right one:

1. DevTools request-level throttling: Fast 4G, Slow 4G (~Lighthouse
   mobile), 3G, Offline, custom profiles. CPU throttle 4x/6x/20x or
   the calibrated low/mid-tier presets. Quick and renderer-only.
2. tc/netem for real packets:

```bash
sudo tc qdisc add dev eth0 root netem delay 100ms 20ms distribution normal
sudo tc qdisc change dev eth0 root netem loss 5% 25%
sudo tc qdisc change dev eth0 root netem rate 1.5mbit
sudo tc qdisc del dev eth0 root
```

   Catches behavior DevTools cannot (real packet loss, reordering,
   burst loss). Test h2 vs h3 under loss, QUIC does not magically
   rescue lossy networks.
3. Toxiproxy for service-to-service resilience: latency, bandwidth,
   timeout, reset toxics behind a TCP proxy. Tests retry, timeout,
   and circuit-breaker logic, not browser rendering.

Facebook ATC is dead (archived 2018). For real low-end devices use
WebPageTest's Moto devices or a cheap physical phone over
chrome://inspect.

## Heap and leaks

- DevTools Memory panel: heap snapshot, filter `Detached` for
  detached DOM trees, use Comparison view between two snapshots
  around the suspected leak action. Allocation sampling for longer
  sessions, instrumentation timeline is dev-only.
- memlab (Meta): scripted E2E leak detection, works on Chromium,
  Node, Electron. `memlab run --scenario test.js`, then
  `memlab analyze` subcommands. CI-friendly.
- Node: `--inspect` plus the Memory tab, `--heap-prof` for sampled
  profiles, `--heapsnapshot-signal=SIGUSR2` for signal-triggered
  snapshots. Full snapshots pause the process and can double heap,
  avoid on busy production processes.

## RUM collection

- `web-vitals` v5 (~2KB): onLCP, onINP, onCLS. The
  `web-vitals/attribution` build adds sub-parts and LoAF script
  attribution.
- Ship with `navigator.sendBeacon` (survives unload), batch by
  metric.id, send on visibilitychange hidden or pagehide since INP
  and CLS only finalize then.
- Collect nav type (incl. bfcache restores and SPA soft navs),
  `effectiveType`, `saveData`, `deviceMemory`, page template.
  Aggregate to p75 server-side to match CrUX.
- Cross-origin resource timing is coarse without
  `Timing-Allow-Origin`. LoAF attribution is empty for cross-origin
  scripts. Do not beacon URLs containing PII.
- JS Self-Profiling API (`new Profiler`) for production CPU
  sampling in Chromium, roughly 1% overhead at 10ms intervals.

## Heatmaps and analytics

- Microsoft Clarity: free, unlimited sessions, click/scroll
  heatmaps, session replay, rage-click and dead-click detection.
  30-day retention, data processed by Microsoft, read the ToS.
- Hotjar is now Contentsquare. The free tier samples ~5% of sessions.
  PostHog includes heatmaps, session replay, feature flags, 1M
  events/mo free tier.
- Privacy-friendly pageview analytics (cookie-free, no consent
  banner needed): Umami (MIT, easy self-host), GoatCounter (Go,
  single binary), Plausible CE (AGPL, heavier). Matomo for
  GA-style depth.
- Session replay and behavioral analytics generally need consent
  under GDPR/ePrivacy. Mask form inputs and passwords.

## Pitfalls

- Do not run axe during a perf measurement. It injects ~0.5MB and
  takes 400-900ms on dense pages. Keep accessibility and perf as
  separate CI jobs.
- DevTools emulation misses real GPU, thermal, and memory pressure.
  INP problems on low-end hardware only reproduce on real devices.
- Source maps in production: generate hidden maps, upload to your
  error tracker, do not ship .map files publicly.

## Unverified

Exact current Lighthouse major version, PostHog and Contentsquare
plan details, and chrome-devtools-mcp flags shift quickly. Check
`npm view` and docs before pinning a workflow to them.
