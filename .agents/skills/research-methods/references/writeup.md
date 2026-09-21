# Write-up contract and agent workflow

## Output contract

- Confidence tiers: CONFIRMED (primary source or direct observation,
 version-pinned), OBSERVED (two or more independent secondaries),
 UNVERIFIED (single tertiary source or recall). Claims never move up
 without new evidence.
- Version-sensitive claims: "as of <date>, <project> vX.Y does Z",
 with the doc version or commit recorded.
- Fast-moving claims (latest versions, roadmaps, pricing, CVE status)
 always carry a retrieval date.
- Cite primary URLs only in the final write-up. Tertiary sources may
 be named as discovery paths, not as authorities.
- Contradictions get documented pairwise with dates, never silently
 resolved.
- Prefer pinned/versioned URLs over floating `latest` aliases. Archive
 volatile pages via SavePageNow.

## Discovery order for a new project

1. Repo root: README, docs/, CHANGELOG, UPGRADING/MIGRATION,
 SECURITY.md, CONTRIBUTING.
2. `llms.txt` at the base URL, plus `.md` variants of doc pages.
3. Releases/tags for currency signals.
4. Issue tracker for known problems (open and closed).

## Search tactics

- `site:` to restrict to primary domains (`site:rfc-editor.org`,
 `site:github.com/<org>/<repo>`).
- `before:`/`after:` to bracket release windows (beta operator, most
 reliable for news).
- Query with exact error strings, flag names, and version numbers,
 not prose questions.
- GitHub: `is:issue is:closed "<exact error>"`.
- Search finds which artifact matters. Fetch the artifact itself
 before citing. A snippet quoting a primary source is still
 tertiary.

## When code beats docs

Default values, error behavior, edge cases, ordering guarantees, and
anything version-sensitive: read the implementation or the diff
between tags. `git log -S`/`git blame` on the symbol beats prose.
Hyrum's Law: observable behavior is the contract regardless of docs.

## Common rationalizations

- "The docs say so" -> docs describe intent. Verify against the
 installed version.
- "The issue is closed" -> check completed vs not-planned vs stale.
- "It is on the releases page" -> check the tag date vs your claim's
 window.
- "Everyone says so" -> trace to the one upstream source everyone is
 citing.
- "I remember it" -> recall is a tertiary source. Verify or label.
