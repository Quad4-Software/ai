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
