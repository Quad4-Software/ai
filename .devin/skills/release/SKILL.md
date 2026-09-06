---
name: release
description: >
  Use when cutting a release: GoReleaser dist/ artifacts, semantic v*.*.*
  tags, immutable tags, notes.md sha256 artifact tables, .goreleaser.yml,
  and the release workflow.
---

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
  matrix covers each *-mcp server binary (or a documented subset).
- Local dry run: goreleaser release --snapshot --clean, then inspect dist/.

## notes.md artifact table

Each release publishes a notes.md containing a sha256 checksum table of
every released artifact:

```
| artifact | sha256 |
|----------|--------|
| rns-mcp_linux_amd64.tar.gz | <sha256> |
```

Generate with sha256sum over dist/ outputs and verify against the
checksums.txt GoReleaser emits. Attach notes.md and checksums.txt to the
GitHub release.

## Release workflow

The release workflow is .github/workflows/release.yml:

- Triggers on v*.*.* tags and workflow_dispatch.
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
