---
name: cdn
description: >
  This skill covers Cloudflare and BunnyCDN (bunny.net) as of September
  2026: CDN caching, authoritative DNS and registrar, object storage
  (R2, Bunny Storage), edge compute (Workers, Edge Scripts), tunnels
  (cloudflared), WAF, and APIs/Terraform. Use it when putting a site or
  service behind a CDN, choosing object storage, exposing a self-hosted
  service, registering domains, or picking between the two providers.
metadata:
  sources:
    - https://developers.cloudflare.com/
    - https://www.cloudflare.com/plans/
    - https://docs.bunny.net/
    - https://bunny.net/pricing/
---

## When to use this skill

- Putting a site, API, or self-hosted service behind a CDN.
- Picking object storage with S3-compatible access.
- Exposing a home lab or private service without port forwarding.
- Registering or migrating domains and DNS.
- Deciding Cloudflare vs BunnyCDN, or running both.

## Provider cheat sheet

| Need | Pick |
|---|---|
| Zero cost static site + API | Cloudflare Free |
| Domain registration at cost | Cloudflare Registrar (locks DNS to CF) |
| High-volume or video bandwidth | Bunny ($0.005-0.01/GB, no content ToS risk) |
| Edge compute depth | Cloudflare Workers (KV, D1, DO, Queues, Workflows) |
| EU residency, GDPR simplicity | Bunny (EU company, region-pinned storage) |
| Free DNS hosting | Both. Bunny adds scriptable DNS records |
| Tunnel to a private origin | Cloudflare Tunnel (free) |
| Predictable pay-as-you-go | Bunny, one rate card |

The common pattern is running both: Cloudflare free for DNS, proxy, WAF
and Workers on the main site, and Bunny for media, downloads and video. Do
not double-proxy the same traffic through both.

## Cloudflare

Free plan is genuinely usable: unmetered CDN bandwidth (fair-use),
authoritative DNS with free DNSSEC, universal SSL, 5 WAF custom rules
plus the managed ruleset, Bot Fight Mode, unmetered DDoS, Tunnel, and
Zero Trust free to 50 users.

Critical rules of thumb:

- The free CDN only caches a fixed extension list (css, js, jpg, png,
  pdf, zip and friends). HTML and JSON are never cached by default.
  Use Cache Rules to override.
- Page Rules are legacy and being auto-migrated. Cache Rules are
  stackable: selecting "eligible for cache" enables cache-everything,
  so order a bypass rule first if you only want default extensions.
- Orange-cloud proxying is HTTP/HTTPS only on fixed ports
  (80, 443, 8080, 8443, 2052-2053, 2082-2083, 2086-2087, 2095-2096,
  8880). Mail, SSH, game servers and wildcard records must stay
  DNS-only. Arbitrary ports need Spectrum (Enterprise).
- ToS: the CDN is for web content. Video and disproportionately large
  files must be served via paid services (R2, Stream, Images) unless
  Enterprise. This is why Bunny pairs so well.
- Registrar sells domains at cost, but the domain must use Cloudflare
  nameservers.
- Auth with scoped API Tokens, never the Global API Key. Terraform via
  the official cloudflare provider and cf-terraforming. Wrangler for
  Workers.
- Workers Free: 100k req/day shared across all scripts, 10 ms CPU per
  request, 128 MB. Workers Paid ($5/mo) raises CPU to minutes and adds
  Containers. No egress fees ever.
- Pages is feature-frozen. New work goes to Workers Static Assets, and
  _headers and _redirects files are silently ignored there, use
  Transform Rules.

## BunnyCDN

No free bandwidth tier but effectively free at small scale (about
$1/month minimum). Pay-as-you-go, no contracts.

- Pull Zones front any origin. Standard tier: 119 PoPs,
  $0.01/GB EU/NA, $0.03 Asia, $0.045 SA, $0.06 MEA. Volume tier: flat
  $0.005/GB on 10 PoPs for big files and video. You can disable
  expensive regions per zone.
- Storage Zones: replicated object storage via HTTP API and FTP/SFTP.
  Standard HDD $0.01/GB per region. S3-compatible API is still public
  preview: enable at zone creation only, 8 S3 regions, reduced
  replication. rclone and aws-cli work. Traffic from Storage to a Bunny
  pull zone is free.
- Token authentication gives signed URLs: basic MD5 or advanced
  HMAC-SHA256 with geo rules, directory prefixes for HLS, IP locking.
- Perma-Cache keeps a permanent replica layer for near-100% hit ratio.
  Optimizer does image transforms at $9.50/pull-zone/month flat.
- Bunny Shield is the WAF/DDoS layer: a free basic tier plus paid
  advanced tiers.
- Bunny DNS is free (unlimited queries, 500 zones) and unique in
  offering scriptable DNS records through Edge Scripting.
- Edge Scripts are Deno-based isolates, 30 s CPU per request, priced
  per request plus CPU. Bunny Stream is a full video pipeline with
  free standard transcoding. Magic Containers run Docker images across
  41+ regions.
- API: single account AccessKey against api.bunny.net. Official
  Terraform provider BunnyWay/bunnynet and an official bunny CLI.
- Gotchas: thinner network than Cloudflare in exotic locales, S3 API
  is preview-grade, storage replication regions cannot be removed after
  creation, Edge SSD tier forces Frankfurt as primary.

## Decision detail

- Cloudflare wins when the bill must be zero, when edge compute or the
  Workers ecosystem matters, and for tunnels to private origins.
- Bunny wins on bandwidth-heavy workloads, video, EU residency, and
  when you want one simple rate card with no ToS traps.
- Both are credible DNS hosts. Cloudflare ties DNS to its registrar.
  Bunny DNS stays independent.
