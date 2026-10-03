---
name: release
description: >
  This skill describes the release process for the repo. Use when you are
  cutting a tag, running GoReleaser, publishing SHA-256 tables, or editing
  the release workflow.
compatibility: goreleaser
---

## When to use this skill

- You are tagging a release with a `v*.*.*` tag.
- You need to run GoReleaser or verify `dist/` artifacts.
- You are editing `.github/workflows/release.yml`.

## How to use

1. Ensure `make all` and `make gosec` are clean on master.
2. Create a semantic `vX.Y.Z` tag and push it.
3. Let `.github/workflows/release.yml` run GoReleaser and publish the notes.md SHA-256 table.
4. Verify assets and the immutable tag. Never force-move a tag.

## Examples

- "Cut release v0.3.0 and verify the dist/ checksums."
- "Generate a notes.md table from `dist/checksums.txt`."
- "Check that release.yml pins all actions to full SHAs."

# Release Process

## Tagging

- Tags are semantic and match v*.*.* (for example v1.2.3). No v1.2 or
  latest-style tags.
- Tags are immutable once pushed: never force-move or re-tag. If a release
  is bad, cut a new patch tag.
- Tag from master after CI is green.

## GoReleaser

- Configuration lives in .goreleaser.yml at repo root.
- Build artifacts land in dist/ (gitignored).
- Because this is a multi-module repo of independent binaries, the release
  matrix covers every mcp/* server (scaffold excluded): linux, darwin,
  windows on amd64 and arm64, CGO_ENABLED=0 with -s -w ldflags.
- Archives are tar.gz (zip on windows) plus a sha256 checksums.txt.
- Local dry run: goreleaser release --snapshot --clean, then inspect
  dist/. A real release is goreleaser release --clean (make release).

## notes.md artifact table

Each release publishes a notes.md containing a sha256 checksum table of
every released artifact:

```
| Artifact | SHA-256 |
| --- | --- |
| rns_linux_amd64.tar.gz | `<sha256>` |
```

scripts/release-notes.sh builds it from dist/checksums.txt and prepends
dist/CHANGELOG.md when GoReleaser emitted one. GoReleaser uploads
checksums.txt as a release asset, and the workflow puts notes.md in the
release body via gh release edit.

## Release workflow

The release workflow is .github/workflows/release.yml:

- Triggers on v*.*.* tags and workflow_dispatch.
- A validate-tag job rejects refs that are not strict vX.Y.Z before the
  release job runs.
- Includes step-security/harden-runner with a pinned SHA (see ci-security
  skill).
- Pins all actions to full commit SHAs.
- Runs make all, make gosec, then goreleaser release --clean.
- Calls scripts/release-notes.sh to build notes.md from dist/checksums.txt.
- Uses gh release edit to publish the SHA-256 table to the release body.
- Uses a GITHUB_TOKEN of least privilege (contents: write only).

## Checklist

1. Update per-server README.md / CHANGELOG notes as needed.
2. make all and make gosec clean on master.
3. Tag vX.Y.Z, push tag.
4. Verify release assets, sha256 table in notes.md, and immutable tag.
