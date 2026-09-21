---
name: sast
description: >
  This skill covers free and open-source local static analysis,
  linting, secret scanning, dependency scanning, and IaC scanning tools
  as of September 2026: ruff, bandit, opengrep, semgrep CE, gosec,
  staticcheck, golangci-lint, ESLint v10, oxlint, biome, gitleaks,
  trufflehog, detect-secrets, osv-scanner, trivy, grype, syft,
  cargo-audit, cargo-deny, checkov, hadolint, kube-linter, conftest,
  shellcheck, shfmt, codeql CLI, zizmor, and meta runners like
  pre-commit and MegaLinter. Use it when picking a scanner, wiring
  pre-commit or CI gates, or auditing a repo locally.
---

## When to use this skill

- You are picking a linter or SAST tool for a language.
- You are scanning a repo for secrets, vulnerable deps, or bad IaC.
- You are building a pre-commit or CI scanning gate.
- You need to know which tools run fully offline vs need a DB or
  registry.

## How to use

1. Read this file for the landscape and decision rules.
2. Load a reference file by scope:
   [references/python.md](references/python.md) for ruff, bandit, and
   the Python type-checker wars.
   [references/scanners.md](references/scanners.md) for secrets, SCA,
   IaC, shell, Rust, Go, and JS tools with versions and offline notes.
   [references/multi.md](references/multi.md) for opengrep vs semgrep,
   codeql licensing, SonarQube, MegaLinter, and pre-commit glue.
3. Verify versions against the linked release pages. These tools ship
   weekly.

## Examples

- "Which tools should gate this repo's pre-commit?"
- "Semgrep or opengrep for a self-hosted scan?"
- "Scan this Dockerfile and the k8s manifests locally."

# Local security tooling landscape

## Decision rules

- **Python**: ruff (`S` rules port most of bandit) + a type checker
  (basedpyright or pyrefly stable, ty beta, mypy 2.x). Bandit is
  redundant if ruff `S` is on.
- **Multi-language SAST**: opengrep (LGPL, fully offline with local
  rules, cross-function taint, cross-file in v2 alphas) or semgrep CE
  (registry rules need network and are under a restrictive license).
- **Go**: go vet + staticcheck + gosec + govulncheck (reachability
  cuts false positives). golangci-lint v2.x wraps most of them.
- **JS/TS**: ESLint v10 (flat config only) or oxlint (type-aware via
  tsgolint on TypeScript 7). Biome if you want one binary for lint +
  format.
- **Secrets**: gitleaks for offline speed (feature-frozen, fine),
  trufflehog when you need live verification (AGPL, needs network to
  verify), detect-secrets for baseline workflows.
- **Dependencies**: osv-scanner (offline DB mode) or trivy fs/repo.
  npm audit alone is a floor, not a strategy. govulncheck and reach-
  ability analysis remove ~85% of SCA noise.
- **IaC**: checkov (Terraform/K8s/Helm/Dockerfile), trivy config
  (tfsec is absorbed), KICS for breadth. Terrascan is archived.
- **Containers**: hadolint for Dockerfiles, trivy image or grype for
  contents, kube-linter for manifests/Helm.
- **Shell**: shellcheck + shfmt.
- **CI security**: zizmor for GitHub Actions (offline mode, SARIF).
- **Orchestration**: pre-commit for local gates, MegaLinter for CI
  fan-out (images only on ghcr.io since v9.5).
- **CodeQL**: deep but the engine is proprietary - free only for
  public repos, OSS research, and query testing. Not for private CI
  without GitHub Code Security.

## Output plumbing

Most tools emit SARIF (ruff `--output-format=sarif`, gosec `-fmt
sarif`, opengrep, zizmor, trivy, checkov). Pipe SARIF into GitHub code
scanning or a local aggregator. `--baseline` mechanisms exist in
detect-secrets and gitleaks for legacy-repo adoption.

## Reference files

- [references/python.md](references/python.md)
- [references/scanners.md](references/scanners.md)
- [references/multi.md](references/multi.md)
