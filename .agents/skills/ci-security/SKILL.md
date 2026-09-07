---
name: ci-security
description: >
  This skill guides CI/CD security with ci-security-mcp. Use when you are
  editing GitHub Actions, hardening Dockerfiles, pinning actions to SHAs, or
  scanning CI files for security flaws.
compatibility: stdio-mcp
metadata:
  server: ci-security-mcp
---

## When to use this skill

- You are editing `.github/workflows/*.yml` or a Dockerfile.
- You need to pin an action or resolve a tag to a SHA.
- You want a security audit of a workflow or container.

## How to use

1. Build ci-security-mcp: `cd ci-security-mcp && go test ./... && go build`.
2. Add the binary to your MCP client config as `ci-security`.
3. Call `scan_workflow {yaml}` or `scan_dockerfile {dockerfile}` for a security pass.
4. Use `pin_workflow {yaml}` to rewrite `uses:` lines to pinned SHAs.

## Examples

- "Scan `.github/workflows/ci.yml` for unpinned actions and script injection."
- "Pin all `uses:` lines in a workflow to commit SHAs."
- "Run `audit_workflow` on a new workflow file."

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
