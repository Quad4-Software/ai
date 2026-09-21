# Multi-language and meta tools

## OpenGrep vs Semgrep CE

Both are LGPL-2.1 engines with compatible rule syntax. The split is
about features and rule licensing, not the engine license.

- **OpenGrep** (v1.28-v1.30 stable. V2.0.0-interfile alphas, Aug-Sep
 2026): consortium fork of semgrep v1.100.0 by 10+ appsec vendors.
 Restored cross-function taint (12 langs), fingerprinting, native
 Windows. OCaml 5 shared-memory parallelism (claimed 25-74% faster).
 Adds VB, Apex, Elixir. Cross-file analysis landing in v2 alphas.
 Fully offline with local rule YAML. JSON and SARIF out.
- **Semgrep CE** (v1.177.x): engine still LGPL, but cross-function
 taint, cross-file analysis, and Pro rules are commercial. Registry
 rules are under the Semgrep Rules License - internal use only, not
 shippable. `--config auto`/`--config p/<ruleset>` needs network.
 Homebrew install dropped in v1.176. Use pip or binary.

Default to opengrep for self-hosted/offline scanning unless a
semgrep-only feature is required.

## CodeQL CLI

Deep cross-file SAST, but the engine binary is **proprietary**: free
for public repos, OSI-licensed codebases, academic research, and
testing OSS queries. Private code and most CI use requires GitHub
Code Security. Queries in github/codeql are MIT. Distribute via
codeql-bundle-*.tar.zst from codeql-action releases. No musl/Alpine.
Workflow: `codeql database create` then `codeql database analyze`.

## Bearer CLI

v2.0.1, ELv2 (source-available, not OSI). SAST + sensitive-dataflow
for JS/TS, Ruby, Java, Go, Python, PHP. Secrets scanner wraps
gitleaks. Cross-file analysis is Bearer Pro only.

## SonarQube Community Build

26.9.x monthly (2026.1 LTA), LGPL-3.0. Server + SonarScanner CLI
(Java 21 required since 26.8). Analysis stays on your server.

## MegaLinter

v9.x, AGPL-3.0. Orchestrates ~100 linters over 69 languages via one
GitHub Action or `npx mega-linter-runner`. Since v9.5.0 images are
only on ghcr.io (Docker Hub frozen at v9.4.0).

## pre-commit

v4.6.x, MIT. The standard glue: `.pre-commit-config.yaml` pins each
hook by `rev`, `pre-commit autoupdate` bumps them, `pre-commit run
--all-files` for full sweeps. Wire ruff, gitleaks, shellcheck, zizmor
here for local gating before CI sees the commit.

## SARIF as the common bus

ruff, gosec, opengrep, zizmor, trivy, checkov, codeql all emit SARIF.
Upload to GitHub code scanning or diff locally with sarif-tools.
