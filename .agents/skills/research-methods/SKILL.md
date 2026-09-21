---
name: research-methods
description: >
  This skill covers disciplined technical research for engineering
  work: source hierarchies, version and currency verification,
  evidence discipline (vendor benchmarks, CVE/advisory conflicts,
  statistics literacy), agent-specific search and fetch strategies,
  and a write-up contract with confidence tiers. Use it whenever a
  claim will drive a decision, when citing docs or versions, when
  sources conflict, or when answering "what is the current state of
  X".
---

## When to use this skill

- You are about to state a version, default, or behavior as fact.
- You are evaluating a library, tool, or approach from docs and blogs.
- Sources disagree and you need to decide what to write down.
- You are compiling research into a doc that others will trust.

## How to use

1. Read this file for the core rules.
2. Load a reference for the task at hand:
   [references/currency.md](references/currency.md) for version and
   freshness checks, [references/advisories.md](references/advisories.md)
   for CVE/GHSA/NVD/KEV/EPSS verification, and
   [references/writeup.md](references/writeup.md) for the output
   contract (confidence tiers, date-stamping, citation rules).
3. The load-bearing rule: every retrieved claim is a hypothesis, not
   a fact. It becomes a fact only when a primary source or direct
   observation confirms it for the version in question.

## Examples

- "Is feature X in the latest release or only on main?"
- "Two blogs disagree on the default - which is right?"
- "Verify this CVE actually affects our pinned version."

# Research methods

## Source hierarchy

- **Tier 0, behavioral truth**: the installed artifact and its source
  code. Docs describe intent, code describes behavior. When they
  disagree, the diff wins and the doc gap is itself a finding.
- **Tier 1, primary**: official docs for the version in use, source
  and git history, RFCs/standards with checked status, changelogs,
  maintainer statements, registry metadata, papers.
- **Tier 2, secondary**: vendor engineering blogs, talks, issue
  threads as evidence of a problem. Fine for landscape mapping, not
  sole support for load-bearing claims.
- **Tier 3, tertiary**: SEO blogs, forum answers, AI summaries,
  including your own recall. Hypothesis generation only.

## Lateral reading

Fact-checkers beat historians at evaluating sources because they
leave the page fast and check what independent sources say about it
(Wineburg and McGrew). Default move: after locating a claim-bearing
page, the next action is an independent source, not deeper reading of
the same site. Two blogs summarizing one changelog count as one
source.

## Currency checks (summary)

- "latest" docs usually track the default branch, not the release.
  Pin doc URLs to the installed version. RTD `stable` vs `latest`
  and Docusaurus `/docs/next/` differ.
- Read CHANGELOG/BREAKING-CHANGES before prose docs. Deprecations
  land in release notes first.
- An RFC number is not a standard: check status, stream, errata, and
  Obsoleted-By before citing.
- Closed issue does not mean fixed: check completed vs not-planned
  vs stale-bot.
- Repo pulse: commits in <90 days, issue response <7 days, close
  rate >0.7 are healthy signals. Bot-only activity and single
  maintainers are flags.

## Evidence discipline

- Classify claims: directly observed, primary-asserted, secondary-
  asserted, promotional. Only the first two may be written as fact,
  and primary assertions still need version pinning.
- Two independent sources for any decision-driving claim. Actively
  try to disprove, not just confirm.
- Vendor benchmarks: who paid, who wrote the items, is the harness
  public, what version and reasoning effort were used.
- "Up to X%" is a bound, not an expectation. Ask for the base rate
  and the denominator.
- Sources in conflict: quote both with dates. Never silently resolve.

## Write-up contract

- CONFIRMED (primary source or direct observation, version-pinned),
  OBSERVED (2+ independent secondaries), UNVERIFIED (single tertiary
  or recall). Claims never move up a tier without new evidence.
- Version-sensitive claims read "as of <date>, <project> vX.Y does
  Z". Fast-moving claims always carry a retrieval date.
- Cite primary URLs only in the final write-up.

## Sources

- Wineburg and McGrew, Lateral Reading (2019):
  cor.stanford.edu/research/lateral-reading
- Caulfield SIFT: hapgood.us/2019/06/19/sift-the-four-moves/
- CRAAP test. SPJ/Reuters sourcing norms
- endoflife.date API. OpenSSF Scorecard. Archive.org Wayback API
