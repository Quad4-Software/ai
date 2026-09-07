# Supply chain and dependency audit (2025-2026)

The repo is only half the attack surface; the build is the other half.
Run `supply_chain_scan` first, then verify the flagged items against this
list.

## What the scanner checks

- npm `preinstall`/`postinstall`/`install`/`prepare` lifecycle scripts and
  `setup.py` `cmdclass` or embedded `exec`/`base64` payloads
- Loose dependency specs: `*`, `latest`, `git+`, `http:`, `file:` in
  package.json; requirements.txt lines without `==`; go.mod `replace`
  directives
- GitHub Actions referenced by mutable tag or branch (`uses: x@v5`,
  `uses: x@main`) instead of a full 40-char SHA
- Fork-reachable triggers (`pull_request_target`, `workflow_run`,
  `repository_dispatch`, `issue_comment`) in workflows that read
  `secrets.*` or checkout `github.event.pull_request.head.*`
- `curl|sh` / `wget|bash` installers in scripts, Makefiles, Dockerfiles
- Missing lockfiles: go.sum, package-lock.json, requirements hashes

## Incident shapes to pattern-match

- Typosquat: one or two character edits of a popular name. Verify every
  dependency name against the registry record, not the spelling.
- Slopsquat: a package name invented by an AI completion that an attacker
  then registers. Any dep that first appeared in a generated diff gets
  extra scrutiny: check its age, download counts, and maintainer history
  before trusting it.
- Maintainer takeover: the npm "qix" phishing campaign (Sept 2025)
  republished chalk, debug and a dozen dependents with a crypto-clipper.
  Indicator: a release with no matching git tag or a maintainer whose
  account changed hands.
- Tag rewriting: tj-actions/changed-files (Mar 2025, CVE-2025-30066)
  moved version tags to a malicious commit that dumped CI secrets into
  public logs. Only full SHAs are immutable.
- Tarball-vs-git divergence: XZ Utils (CVE-2024-3094) shipped a backdoor
  that existed only in release tarballs and test files. Diff published
  artifacts against tagged source.
- Self-propagating worms: Shai-Hulud (2025) harvested tokens with
  TruffleHog and republished further packages. Post-publish review and
  scoped, expiring tokens limit the blast radius.
- Proxy caching: the Go module proxy keeps immutable copies of a
  typosquat even after the source repo is cleaned (boltdb-go/bolt).
  `go.sum` presence is necessary but not sufficient; check provenance.
- Dependency confusion: internal package names published publicly with a
  higher version. Check for `.npmrc`/`pip.conf` private-index overrides
  and reserved public names for internal scopes.
- Repo-jacking: transferred or renamed GitHub repos leave the old
  namespace claimable. Flag old org names in go.mod imports and README
  install URLs.
- CDN poisoning: script tags from mutable CDNs (polyfill.io incident).
  Vendor assets or pin with SRI.

## Verification tooling

- `govulncheck` for known Go vulns, `pip-audit`/`safety` for PyPI,
  `osv-scanner` for cross-ecosystem
- `cosign verify` / `slsa-verifier` for signed artifacts and SLSA
  provenance where offered
- `gitleaks` / `trufflehog git file://.` for secrets in history
- `actionlint` for workflow correctness; review workflow diffs like
  dependency diffs

## CI/CD pipeline review (PPE)

Poisoned Pipeline Execution (MITRE T1677, OWASP CICD-SEC-4): the attacker
modifies what the pipeline runs, not the app. Hunt:

- `run:` blocks interpolating `${{ github.event.* }}` fields (title,
  branch name, comment body) directly into shell: script injection.
- `pull_request_target` or `workflow_run` jobs that check out untrusted
  code or run `make`/`npm install`/`go build` on it.
- Broad `permissions:` on `GITHUB_TOKEN`; default should be read-only.
- Caches and artifacts shared between untrusted and trusted jobs.
- Third-party actions from unknown publishers, especially those
  requesting `id-token: write` or deployment permissions.
- Release jobs reachable from non-tag triggers.

## Repo hygiene signals

- Dependabot or Renovate config present and current.
- `minimumReleaseAge`-style cooldowns on new dependency versions where
  the ecosystem supports it.
- Signed commits/tags for release-critical repos.
- `.gitignore` covering `.env*`, `*.pem`, key material; `git_secrets`
  clean on recent history.
