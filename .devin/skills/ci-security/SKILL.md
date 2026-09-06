---
name: ci-security
description: >
  Use when editing GitHub Actions workflows, CI jobs, or security gates in
  this repo: harden-runner, pinned SHAs, Dependabot, dependency review,
  gosec, go vet, and the race/fuzz/bench/leak workflows.
---

# CI and Security Gates

## Workflows

.github/workflows/ contains: ci.yml, dependency-review.yml, race.yml,
fuzz.yml, bench.yml, leak.yml, gosec.yml, and release.yml.

- ci.yml - fmt/vet/test/build plus gosec via
  go install github.com/securego/gosec/v2/cmd/gosec@latest then make gosec.
- dependency-review.yml - dependency review on PRs.
- race.yml - go test -race.
- fuzz.yml - fuzz targets per server.
- bench.yml - benchmark regression checks.
- leak.yml - goroutine/resource leak checks.
- gosec.yml - standalone gosec badge workflow.
- release.yml - v*.*.* tag triggered release (see release skill).

## Rules for editing workflows

- Every job starts with step-security/harden-runner at a pinned SHA
  (current: 5c6294f65b6e27a9f0b85a469efb074d601c8c26 # v2.0.0). Use
  egress audit or blocking as appropriate.
- All third-party actions must be pinned to full commit SHAs with a version
  comment, never floating tags or branches.
- Dependabot (.github/dependabot.yml) manages gomod weekly, limit 5 PRs.
  Keep it covering every *-mcp dir.
- Never add steps that execute arbitrary shell from PR-controlled input
  (no run: on untrusted context, no pull_request_target with checkout of
  PR code).
- No secrets in logs. Rely on redaction, never echo tokens. Secrets in
  workflow files only via ${{ secrets.* }} with least privilege.
- Builds and tests must stay offline-capable. Avoid steps that fetch
  unpinned tools. Tool installs (like gosec) should be version-pinned where
  feasible.

## Go-level security

- go vet and gosec run per server via make.
- #nosec annotations must carry an inline justification explaining why the
  finding is a false positive or accepted risk.
- Servers stay read-only, path-jailed, and redact secrets (see mcp-toolkit
  skill).
- Keep third_party/ vendored trees out of lint scans (the Makefile filters
  /third_party/ from go fix. Apply the same exclusion elsewhere).
