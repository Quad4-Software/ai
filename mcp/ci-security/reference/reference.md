# CI security reference

GitHub Actions and container hardening, distilled from GitHub's secure-use docs
and the OWASP Docker Security Cheat Sheet.

## Action pinning

Third-party actions are code you run with your secrets. A tag (v4, main) is a
mutable pointer: the maintainer or an attacker can move it. Pin to the full
40-char commit SHA and keep the tag as a comment so updates stay reviewable:

    - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2

Resolve a tag or branch to a SHA via the GitHub API:
GET https://api.github.com/repos/<owner>/<repo>/commits/<ref> -> .sha
or resolve a tag ref via git ls-remote <repo> <ref>.

Enable immutable releases / tag protection on your own actions so tags cannot
be re-pointed after publish (GitHub repository setting: immutable releases).

## pull_request_target

pull_request_target runs in the base repo context with secrets and a write
token, even for forks. The trap: checking out or running code from the PR head
executes attacker-controlled code with elevated privileges.

Safe uses: labeling, commenting, approving, metadata checks that never run PR
code. Unsafe: any step that checks out the head, builds it, or evals its
contents (scripts, Makefile, package.json hooks).

If you must run PR code, do it in the pull_request trigger or check out the
merge ref, gate on an authorization label/comment from a maintainer, and scope
permissions: read-only.

## Secrets

- Never print or interpolate secrets into logs or run blocks, masking is a
  last resort, not a control.
- Pass secrets via env on the specific step that needs them, not job env.
- secrets are not available to workflows from forks under pull_request, under
  pull_request_target they ARE, which is why the trigger is dangerous.
- Prefer OIDC federation (id-token: write) over long-lived cloud secrets.
- Rotate any secret that appeared in a log, even masked.

## permissions and GITHUB_TOKEN

Default the token to read-only and elevate per job:

    permissions: {}            # top of file
    jobs:
      build:
        permissions:
          contents: read

Any job missing an explicit permissions block inherits the repo default, which
is often read-write. pin_actions: require SHA pinning at the org/repo Actions
settings level where available.

## Script injection

Never interpolate github.event.* / github.head_ref / issue titles / PR bodies
into run: directly, the value is attacker-controlled. Use an intermediate env:

    env:
      TITLE: ${{ github.event.pull_request.title }}
    run: echo "$TITLE"

## Docker pinning and hardening

- Pin images by digest: image@sha256:<digest>. Tags (including version tags and
  especially latest) are mutable. Resolve: docker buildx imagetools inspect
  <img> or the registry API.
- Set a non-root USER in every image, never run as the implicit root.
- No ADD from remote URLs, use COPY for local files, fetch+verify for remote.
- No curl | sh, no --privileged, no host mounts of docker.sock in CI unless
  the job is the image build itself.
- Drop capabilities, read-only rootfs, and a HEALTHCHECK where applicable.
- Scan built images, keep base images minimal and rebuilt on a schedule so
  pinned digests stay patchable.

## Dependency and egress

- Pin package manager lockfiles, prefer --frozen-lockfile / --immutable.
- Scope outbound network where the runner allows it (egress audit or
  step-level hardening).
- Treat caches as untrusted input across trust boundaries (PR caches can
  poison default-branch builds).

## zizmor

zizmor (github.com/zizmorcore/zizmor) is a static analyzer for CI/CD
definitions: GitHub Actions workflows and action.yml files, Dependabot
configs, and pre-commit configs. Its audits cover template-injection,
unpinned-uses, excessive-permissions, cache-poisoning, dangerous-triggers,
artipacked (credential persistence via actions/checkout), ref-confusion,
known-vulnerable-actions, self-hosted-runner, secrets-in-herit, and more.
See docs.zizmor.sh/audits for the full index.

Install:

    uvx zizmor --help              # runs the latest wheel from PyPI
    uvx zizmor@1.30.1 .            # pinned version
    pipx install zizmor            # or: brew install zizmor
    cargo install --locked zizmor
    docker pull ghcr.io/zizmorcore/zizmor:latest

Local runs:

    zizmor .                       # audit a whole repo (workflows, actions,
                                   # dependabot, pre-commit)
    zizmor .github/workflows/      # workflows only
    zizmor owner/repo              # audit a remote repo (needs a token)
    cat workflow.yml | zizmor -    # stdin

Operating modes: offline is the default when no token is present. Setting
GH_TOKEN, GITHUB_TOKEN, or ZIZMOR_GITHUB_TOKEN enables online audits (for
example known-vulnerable-actions, which queries the GitHub advisory
database). --offline forces offline, --no-online-audits fetches inputs but
skips online audits. Local tip: zizmor --gh-token $(gh auth token) ...

Output and exit codes: --format plain (default), json, sarif, or github
(workflow annotations, capped at 10 per step by GitHub). Exit code 0 means
clean; 11-14 mean findings, where the code maps to the highest severity
(informational, low, medium, high). --format=sarif always exits 0 so SARIF
consumers see results inside the file, not via exit code. Filter noise with
--min-severity and --min-confidence. Personas: regular (default, high
signal), --persona=pedantic (code smells), --persona=auditor (everything).
--fix rewrites fixable findings in place (safe fixes only; --fix=all also
applies unsafe ones).

Ignoring findings:

    run: | # zizmor: ignore[template-injection] reason here
      echo "${{ github.event.issue.title }}"

or a zizmor.yml (repo root or .github/zizmor.yml) with per-rule ignore
lists keyed by file[:line[:col]]:

    rules:
      template-injection:
        ignore:
          - safe.yml
          - wf.yml:123
          - wf.yml:123:45

### zizmor in GitHub Actions

Easiest: zizmorcore/zizmor-action (pin to SHA like any action). It runs a
digest-pinned ghcr.io image and exposes persona, min-severity,
min-confidence, version, collect, and online-audits inputs.

    - uses: zizmorcore/zizmor-action@<sha> # vX.Y.Z
      with:
        advanced-security: false   # plain output, fails the job on findings

Modes:

- advanced-security: true (default) writes SARIF and uploads it via
  github/codeql-action/upload-sarif to the Security tab. Needs
  security-events: write (plus contents/actions: read on private repos).
  It never fails on findings; use rulesets ("code scanning merge
  protection") to block merges on alerts.
- advanced-security: false uses plain output and propagates zizmor's exit
  code, so findings fail the job. Add annotations: true for inline
  annotations instead (remember the 10-annotation cap).
- token defaults to github.token and enables online audits; set
  online-audits: false for a fully offline run.

Manual equivalent without the action:

    - uses: astral-sh/setup-uv@<sha> # vX.Y.Z
    - run: uvx "zizmor@1.30.1" --format=sarif . > results.sarif
      env:
        GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    - uses: github/codeql-action/upload-sarif@<sha> # vX.Y.Z
      with:
        sarif_file: results.sarif
        category: zizmor
