---
name: anti-hallucination
description: >
  This skill is a discipline checklist for not fabricating facts.
  Use it whenever an answer depends on specific names, versions,
  URLs, flags, APIs, statistics, or citations: verify against a
  live source when tools allow, and ask the user when they do not.
  Covers the failure modes - recalled versions, invented packages,
  plausible URLs, stale APIs - and the fallback chain.
metadata:
  sources:
    - https://github.com/github/advisory-database
---

## When to use this skill

- Any claim that has a ground truth: a flag name, a function
  signature, a version, a date, a URL, a CVE ID, a quote.
- The environment lacks web search, curl, or registry access and
  the answer depends on current external facts.
- You catch yourself writing "as of" without having checked.

## The core rule

If a fact is checkable and unchecked, it is a guess. A guess stated
confidently is worse than a question asked.

## Failure modes to catch

- Recalled versions and package names. Model memory of "latest" is
  frozen at training time and often wrong even then. Invented names
  are a real attack surface: slopsquatting registers the plausible
  names LLMs hallucinate. Always look up. See the dep-versions
  skill for per-registry commands.
- Plausible URLs. Never construct a URL you have not seen. A wrong
  domain is a phishing vector. If a URL is needed and cannot be
  fetched or verified, ask the user for it.
- Flag and API drift. CLIs rename flags between majors. Check
  `--help` output or man pages on the installed binary, not memory
  of another version.
- Statistics and dates. "In 2024, X%" claims need a source. If you
  cannot name the source, drop the number or mark it approximate.
- Fabricated citations. Never produce a paper title, CVE, or link
  you did not retrieve. Citing a real-looking but wrong reference
  is worse than no citation.
- Confident extrapolation. "This repo probably uses X because
  similar repos do." Check the repo.

## Verification ladder

1. Local ground truth first: the repo, lockfiles, `--help`, the
   vendored source. Read it.
2. Then live checks: `npm view`, `go list -m`, `curl` the page,
   `gh api`, web search.
3. Then the user: when no tool can reach the fact, ask. Provide the
   specific question, not a vague "I need info".
4. Last resort, state the uncertainty inline: "I believe X but
   could not verify. Check before relying on it." Do this rarely.
   A habit of hedging everything is its own failure mode.

## When there is no way to check

Say so directly and ask. One sentence: what you need, why, and the
exact form (a file, a version, a paste of `--version` output, a URL).
Do not produce a filled-in guess to look productive. The user's
preference, stated in this repo's AGENTS.md, is a specific
clarifying question over a wrong answer.

## Output hygiene

- Separate CONFIRMED facts from assumptions in research write-ups.
  The research-methods skill has the labeling contract.
- Mark time-sensitive claims (versions, prices, API status) with
  the date checked so staleness is visible later.
- When a claim came from an unverified secondary source, say
  "reported by" not "is".

## Related

- `dep-versions` - registry lookups and release-age rules.
- `research-methods` - source hierarchy and currency checks.
- `no-trickery` - trusting fetched web content safely.
- `no-slop` - output quality rules in generated artifacts.
