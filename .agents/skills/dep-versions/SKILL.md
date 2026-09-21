---
name: dep-versions
description: >
  This skill covers choosing dependency versions correctly. Use it
  whenever adding, bumping, or pinning a package: look the version up
  in the registry instead of recalling it, prefer releases older than
  24-72 hours, check known-vulnerability feeds, and never invent a
  version when offline. Covers npm/pnpm, Go modules, crates.io, PyPI,
  and the repo's offline constraint.
metadata:
  sources:
    - https://osv.dev/
    - https://github.com/github/advisory-database
---

## When to use this skill

- Adding a dependency or bumping one.
- A version number is about to be written into a manifest, lockfile,
  Dockerfile, or skill doc.
- Choosing between latest, pinned, or ranged versions.

## Rules

1. Never write a version from memory. Look it up in the live registry.
   Recalled versions are stale at best and slopsquat bait at worst.
2. Prefer a release older than 24-72 hours over the newest. Every
   major npm/PyPI compromise of 2025-2026 was live for under a day.
   This repo's pnpm config already enforces `minimumReleaseAge`.
3. Check the candidate against a vulnerability feed before pinning.
4. If there is no network or no registry client, do not guess. Read
   the lockfile and vendor dir, or ask the user for the version.
5. Pin exactly in lockfiles. In libraries use ranges only when the
   ecosystem convention calls for it (npm peer ranges, Go semver).

## Registry lookups by ecosystem

- npm/pnpm: `npm view <pkg> version`, `npm view <pkg> versions`,
  `npm view <pkg> time` (publish timestamps per version).
- Go: `go list -m -versions <module>` and `go list -m -u -json <mod>`
  inside a module. pkg.go.dev for docs and discovery date.
- crates.io: `cargo search <name>` and `cargo info <name>`.
- PyPI: `uv pip index versions <pkg>` or `pip index versions <pkg>`,
  or the JSON API `https://pypi.org/pypi/<pkg>/json`.
- Maven, RubyGems, NuGet: their search APIs, or the site.

## Vulnerability and freshness checks

- `osv-scanner scan --lockfile=<file>` or the OSV API for a single
  package+version query. Works across ecosystems.
- `govulncheck ./...` for Go reachability, `pip-audit` for PyPI,
  `npm audit` / `pnpm audit` for npm.
- GitHub Advisory Database web or `gh api` search for GHSA entries.
- `npm view <pkg> time --json` gives publish dates. A version
  published minutes ago by a new maintainer is a red flag, not a
  feature.

## Offline and no-tool fallback

This repo builds offline. When you cannot reach a registry:

- Use the version already in the lockfile or vendor tree.
- State plainly that you cannot verify currency and ask the user to
  provide the target version or run the lookup.
- Never substitute a plausible-looking version number.

## Related

- `pnpm` skill: minimumReleaseAge, trustPolicy, allowBuilds.
- `python-packaging` skill: --exclude-newer cooldowns, hash pinning.
- `sast` skill: scanner table for osv-scanner, grype, trivy.
- `bug-hunting` supply-chain reference: why fresh releases are the
  risk surface.
