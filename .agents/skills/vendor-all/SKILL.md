---
name: vendor-all
description: >
  This skill helps you vendor web assets locally. Use when you want to
  avoid CDNs, remove trackers, support offline use, or pin versions of
  fonts, scripts, styles, and images.
---

## When to use this skill

- You are building a web project and it loads assets from a CDN.
- You want to remove trackers and keep the site offline-capable.
- You need to pin a version of a JS, CSS, or font dependency.

## How to use

1. Audit the HTML for remote URLs, analytics, and social embeds.
2. Download assets with `wget` or `curl`, or install from npm and copy out the needed files.
3. Rewrite links to point at `assets/vendor/`, keep license/attribution notes, and add subresource integrity.
4. Use `npm`/`pnpm`/`yarn` where you need lockfiles and version pinning.

## Examples

- "Replace a Google Fonts link with self-hosted `woff2` files."
- "Vendor `htmx` and `bootstrap` into `assets/vendor/` with SRI hashes."
- "Build a `LICENSES.md` for all vendored assets."

# Vendor-all web assets

Modern web projects often pull in fonts, scripts, icons and styles from third-party
CDNs. This leaks visitor data to those hosts, breaks offline use, and creates a
dependency on networks you do not control. This skill explains how to vendor all
of that content locally.

## Why vendor

- Privacy: no requests to Google Fonts, jsDelivr, cdnjs, unpkg or other trackers
- Offline: pages work without an internet connection or inside an intranet
- Reliability: no downtime from a remote CDN
- Reproducibility: the assets in your repo are the assets your users get
- Anti-tracking: fewer third parties see your traffic and your users' IP addresses

## What to vendor

- Web fonts (Google Fonts, Bunny Fonts, Font Awesome, etc.)
- CSS frameworks (Bootstrap, Tailwind CDN, etc.)
- JavaScript libraries (React, Vue, jQuery, lodash, etc.)
- Icons and icon fonts (Font Awesome, Material Icons, Lucide, etc.)
- Images, logos and avatars loaded from remote hosts
- Analytics, tag managers and comment widgets (replace or remove, do not just
  proxy)

## Fetching assets

Use `wget` or `curl` to download the exact files your HTML currently requests.

With wget:

```
wget --no-clobber --convert-links -P assets/vendor/ URL
```

With curl:

```
curl -L -o assets/vendor/filename URL
```

For a page with many remote assets, use a list file:

```
wget --no-clobber -P assets/vendor/ -i urls.txt
```

## Storing assets

Keep vendored files under a single directory, for example `assets/vendor/` or
`static/vendor/`. Preserve the original filename or rename to a clear,
version-pinned name such as `jquery-3.7.1.min.js`.

Suggested layout:

```
assets/vendor/
  fonts/
    inter-400.woff2
    inter-700.woff2
  css/
    bootstrap-5.3.2.min.css
  js/
    htmx-1.9.12.min.js
```

## Rewriting HTML

Replace remote URLs with local paths:

```
<link rel="stylesheet" href="assets/vendor/css/bootstrap-5.3.2.min.css">
<script src="assets/vendor/js/htmx-1.9.12.min.js"></script>
```

For CSS that imports fonts from a remote URL, edit the `@font-face` `src` lines
to point at local files and remove the original `url()` to the CDN.

## Subresource integrity

Keep an `integrity` attribute on every local script and stylesheet if the
original had one. When you download a new version, generate a fresh hash:

```
openssl dgst -sha384 -binary FILE | openssl base64 -A
```

Then add `sha384-HASH` as the `integrity` value and set `crossorigin="anonymous"`.

## Google Fonts

Google Fonts is a tracker. Download the font files and host them yourself.

1. Pick a font from a source that gives direct file downloads (fontsource,
   the foundry's own repo, or a package manager like npm)
2. Place `woff2` files under `assets/vendor/fonts/`
3. Define `@font-face` rules in your own CSS
4. Remove the `<link href="https://fonts.googleapis.com/...">` tag

Do not use the Google Fonts CSS API even if you serve it through a proxy.

## Fonts from npm

Many open fonts ship on npm. Install them locally and copy the font files into
your vendor directory:

```
npm install @fontsource/inter
cp node_modules/@fontsource/inter/files/* assets/vendor/fonts/
```

Then import or define `@font-face` in your own CSS.

## Self-hosted package managers

For JS and CSS, prefer installing from npm (or pnpm/yarn/bun) and copying the
needed files out of `node_modules` into `assets/vendor/`, instead of linking to
a CDN.

```
cp node_modules/htmx.org/dist/htmx.min.js assets/vendor/js/
```

This keeps your build offline and gives you lockfiles for version control.

## Removing telemetry

Audit the page for these common trackers and remove them:

- Google Fonts, Google Analytics, Google Tag Manager
- Cloudflare Insights, Browser Insights
- Hotjar, Mixpanel, Segment, Amplitude
- Social media embeds (Twitter, Facebook, Instagram)
- Comment widgets that load remote scripts (Disqus, etc.)
- CDN-hosted analytics or crash reporters

If a feature requires a tracker, either self-host an equivalent or drop the
feature.

## License and attribution

Vendored files are still covered by their original licenses. Keep a
`LICENSES.md` or `NOTICES` file in `assets/vendor/` listing each asset, its
source URL, its license and the version you downloaded.

## Tools

- `wget` or `curl` for one-off downloads
- `npm`/`pnpm`/`yarn` for versioned package installs
- `openssl dgst -sha384` for integrity hashes
- A simple static HTTP server for local testing, such as `python -m http.server`
  or `darkhttpd`
