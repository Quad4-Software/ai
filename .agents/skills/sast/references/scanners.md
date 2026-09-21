# Scanner catalog (versions as of Sept 2026)

## Secrets

| Tool | Version | License | Notes |
| --- | --- | --- | --- |
| gitleaks | 8.30.1 | MIT | Feature-frozen (author moved to Betterleaks). `gitleaks git`, `dir`, `stdin`. Fast, fully offline, no verification. `.gitleaks.toml`, `--baseline-path` |
| trufflehog | 3.97.x | AGPL-3.0 | 800+ detectors, live verification `--only-verified` (needs network). Git, GitHub/GitLab orgs, S3, images |
| detect-secrets | 1.5.x | Apache-2.0 | Baseline model `.secrets.baseline`, audit-then-gate workflow for legacy repos |

## Dependency and container scanning (SCA)

| Tool | Version | License | Notes |
| --- | --- | --- | --- |
| osv-scanner | 2.5.1 | Apache-2.0 | Lockfiles across ecosystems, `scan image`, guided `fix`, `--offline-vulnerabilities` with local DB cache for air-gap |
| trivy | 0.74.x | Apache-2.0 | `fs`, `image`, `config` (IaC, absorbed tfsec), `repo`, `k8s`, `sbom`, `vm`. DB downloads on first run. Air-gap needs `--skip-db-update` + private mirror |
| grype | 0.118.x | Apache-2.0 | Image/dir/SBOM scan (`grype sbom:sbom.json`), pairs with syft |
| syft | 1.52.x | Apache-2.0 | SBOM generation CycloneDX/SPDX, can sign in-toto attestations |
| govulncheck | 1.1.4 | BSD-3 | Go reachability against vuln.go.dev, `-mode binary`, `-db` for local copy, SARIF out |
| vexctl | 0.4.x | Apache-2.0 | OpenVEX documents: create, merge, filter. `grype --vex` and `trivy --vex` consume them to suppress triaged findings |
| cargo-audit | 0.22.2 | Apache/MIT | Cargo.lock vs RustSec DB, `audit bin`, `audit fix` |
| cargo-deny | 0.19.x | Apache/MIT | Advisories, licenses, bans, sources policy via deny.toml |

npm audit caveat: lockfile matching against one DB, no reachability,
counts devDeps, `fix --force` installs breaking majors, misses
typosquats entirely. Layer OSV or Trivy on top.

## IaC and containers

| Tool | Version | License | Notes |
| --- | --- | --- | --- |
| checkov | 3.3.x | Apache-2.0 | 1000+ policies, graph checks, TF/CFN/K8s/Helm/Dockerfile/OpenTofu, custom YAML/Python policies, AI-infra rules (Bedrock, Vertex, OpenAI) |
| KICS | 2.x | Apache-2.0 | 2400+ Rego queries, 22 platforms |
| hadolint | 2.15.x | GPL-3.0 | Dockerfile AST lint + ShellCheck on RUN |
| kube-linter | 0.8.3 | Apache-2.0 | K8s YAML + Helm, `.kube-linter.yaml` |
| conftest | 0.69 | Apache-2.0 | Rego policy tests on any config, `conftest verify` unit-tests policies |
| Terrascan | ARCHIVED Nov 2025 | - | Final v1.19.9, read-only. Migrate to checkov/trivy config/KICS |

## Shell

- **shellcheck 0.11.0** (GPL-3.0) - SC codes, `# shellcheck
 disable=SC2086`, `.shellcheckrc`.
- **shfmt 3.13.1** (BSD-3) - format/parse, zsh support since 3.13,
 `shfmt -l -w`, EditorConfig-aware.

## Go

- **go vet** - bundled, always run.
- **staticcheck 0.8.1** - moved from date versioning to 0.x.
- **gosec 2.28.0** - `#nosec Gxxx` needs justification (repo rule).
- **golangci-lint 2.13.x** - v2 schema (`version: "2"` in
 .golangci.yml), `migrate` converts v1 configs. Fails on this repo's
 custom toolchain per AGENTS.md - rely on vet/gosec locally.

## JS/TS

- **ESLint 10.x** (MIT) - flat config only, `.eslintrc` removed.
- **typescript-eslint 8.69.x** - type-aware via
 `recommendedTypeChecked` + `parserOptions.projectService`.
- **oxlint 1.81.x** (MIT) - Rust, type-aware via tsgolint on
 TypeScript 7 (59 of 61 type-aware rules at native speed),
 `npx @oxlint/migrate` from ESLint.
- **Biome 2.5.x** (MIT/Apache-2.0) - lint + format + import sort in
 one binary, own type inference for type-aware rules.

## Rust

- **clippy** via rustup - `cargo clippy --all-targets -- -D warnings`.
 Security-adjacent lints: unwrap_used, undocumented_unsafe_blocks.
- **cargo-geiger** - unsafe usage stats, sporadic development.

## zizmor

MIT. GitHub Actions static analysis: `zizmor . --offline`, personas
(`--persona pedantic`), SARIF output, first-class in this repo's CI
(zizmor.yml, `make zizmor`).
