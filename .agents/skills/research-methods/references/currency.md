# Version and currency checks

## Docs versioning traps

- Read the Docs: `latest` tracks the default branch, `stable` tracks
 the greatest semver tag, and non-stable versions show out-of-date
 banners. Cite the stable URL, note which you read.
- Docusaurus: unreleased docs live at `/docs/next/` while "latest" is
 a navigation label.
- Unversioned vendor docs (HashiCorp style) describe the newest
 release. If your install is older, find the pinned folder or the
 repo's docs at that tag.
- Git blame the doc file and compare its last-modified date against
 the feature's release date when claims look stale.

## Changelog archaeology

Order of reading for a new project: README, docs/, CHANGELOG,
UPGRADING/MIGRATION/BREAKING-CHANGES, SECURITY.md, releases page,
then issues. Deprecations and breaking changes land in release notes
long before prose docs get updated. Distinguish "current default"
from "historically documented".

## Standards status

- RFC metadata first: Status (Informational, Experimental, Proposed
 Standard, Internet Standard, BCP, Historic), stream (only IETF
 stream makes standards), Updates/Obsoletes/Errata. Since RFC 6410
 there are two maturity levels and Draft Standard is retired.
- Never cite a Historic or Obsoleted RFC as current behavior.
- Internet-Drafts expire and mutate. Cite name + version + date.

## Issue and repo state

- `is:issue is:closed` searches find solved problems. Check whether
 closure was completed, not-planned, or stale-bot.
- Repo pulse heuristics: commits within 90 days, first-response under
 7 days, close rate above 0.7. Red flags: last release over 2 years
 old, response over 30 days, bot-only activity, single maintainer.
- endoflife.date tracks lifecycle per product (API exposes isEol,
 isMaintained, eolFrom, latest per cycle).
- OpenSSF Scorecard gives automated Maintained/vulnerability checks.

## Vanished docs

- Wayback Availability API:
 `https://archive.org/wayback/available?url=<url>` returns the
 closest snapshot. CDX API lists all captures for diffing.
 SavePageNow pins today's evidence for volatile pages.

## Temporal discipline

Models treat context as stationary. Date-stamp every freshness-
sensitive claim ("as of 2026-09, latest is vX") and re-verify at
decision time rather than trusting an old note.
