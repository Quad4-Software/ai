---
name: osint
description: >
  This skill covers defensive OSINT: collecting publicly available
  information for investigations, attack-surface monitoring, and threat
  intel. Use it for username, domain, email, IP, or infrastructure
  reconnaissance, breach-exposure checks, metadata and archive
  research, and building a free OSINT toolkit. Covers Maigret,
  Sherlock, theHarvester, subfinder, Amass, dnsx, crt.sh, HIBP,
  IntelligenceX, Shodan, Censys, GreyNoise, ExifTool, Wayback,
  SpiderFoot, and crypto tracing tools, plus legal boundaries and
  source reliability.
metadata:
  sources:
    - https://github.com/soxoj/maigret
    - https://github.com/laramies/theHarvester
    - https://www.osintframework.com/
---

## When to use this skill

- Mapping what an attacker can learn about your org or a target.
- Validating an IOC, email, username, or wallet against public data.
- Checking breach exposure for your own domains and accounts.
- Investigating a domain, IP, or document during IR or research.

## Boundaries first

- Passive collection (search engines, crt.sh, Wayback, Shodan, breach
  DBs, WHOIS/RDAP) is generally lawful. Respect ToS and rate limits.
- Semi-passive (resolving discovered names, fetching public pages)
  generates observable traffic but is low-risk.
- Active recon (port scans, directory brute force, credential testing,
  contacting the target) requires written authorization, same footing
  as pentesting.
- PII collection falls under GDPR/CCPA even when public. Minimize,
  secure, document purpose.
- Checking your own org against breach dumps is standard practice.
  Redistributing or trading dumps is not.
- Viewing dark-web sources is legal in most places. Purchasing or
  interacting with actors is not. Use a dedicated VM and Tor.

## Workflow

1. Define scope: entity types, authorization memo, exclusions.
2. Passive collection: dorks, crt.sh, Wayback, passive DNS, breach
   lookups, public code, job postings, Shodan/Censys.
3. Pivot and enrich: username to profiles, email to breaches, domain
   to infra, document to metadata.
4. Active, only when authorized: DNS brute force, probing.
5. Document everything: timestamps, source URLs, screenshots/WARC,
   confidence per finding.

Rate every finding with the Admiralty Code (source reliability A-F x
information credibility 1-6). A username match is a lead, not identity
proof: corroborate with two or more independent signals. Record
retrieval dates, public data goes stale.

## Tool map (Sept 2026)

| Category | Best picks | Notes |
|---|---|---|
| Search/dorks | Google operators, GHDB, Bing, Brave | `cache:` operator is dead, use Wayback |
| Username | Maigret (3k+ sites, maintained), Sherlock v0.16 (pipx install) | WhatsMyName is now data-only |
| Domain/DNS | subfinder, Amass v5 (major rewrite), dnsx, crt.sh, SecurityTrails | subfinder wants free API keys |
| Email | HIBP (paid API), holehe (fragile modules), GHunt, Epieos | h8mail is unmaintained |
| Breach data | HIBP, IntelligenceX (free tier incl. darkweb), Hudson Rock, DeHashed | BreachForums was taken down Oct 2025 |
| Infra intel | Shodan (InternetDB is free, no key), Censys, GreyNoise, FOFA | GreyNoise separates scanner noise from targeted |
| Geolocation | Google Earth, SunCalc, Overpass Turbo, WiGLE | Method over tools: shadows, terrain, signage |
| Metadata | ExifTool, oletools (olevba/olemeta), pdfinfo | Platforms strip EXIF on upload |
| Archives | Wayback + CDX API, archive.today, Common Crawl | Common Crawl columnar index is underused |
| Frameworks | theHarvester v4.11 (active), SpiderFoot forks (upstream stalled at v4.0), Maltego CE (limited) | Recon-ng is dormant |
| Crypto | Blockchair, Etherscan, mempool.space, Arkham free tier, bitcoinabuse | Ransom tracing, OFAC checks |
| Dark web | Ahmia, IntelligenceX, DarkSearch | Dedicated VM, never authenticate |

Zero-dollar starter stack: Maigret + theHarvester + subfinder/dnsx +
crt.sh + HIBP + IntelligenceX + Shodan/GreyNoise free + ExifTool +
Wayback covers most defensive OSINT work.

## Pitfalls

- Username overlap does not prove identity. Two independent signals
  minimum before reporting.
- Free tiers shift constantly (Shodan, Censys, Maltego CE). Re-verify
  quotas before relying on them.
- holehe/reset-flow tools decay as sites change. Treat misses as
  inconclusive, not negative results.
- Breach data itself is unverified until cross-checked. Forum dumps
  contain planted and padded data.
