---
name: gh
description: >
  This skill covers the GitHub CLI (gh, v2.101.x as of Sept 2026). Use
  it for auth and token handling, gh api REST/GraphQL patterns, repo/pr/
  issue/workflow/run/release commands, secrets and variables, attestation
  verification, extensions, and scripting flags (--json/--jq/--template).
---

## When to use this skill

- You are scripting GitHub operations in shell or CI.
- You need `gh api` for a REST or GraphQL endpoint, with pagination.
- You are watching runs, downloading artifacts, or managing releases.
- You are verifying artifact attestations or managing secrets/vars.

## How to use

1. Read this file for the command map and scripting patterns.
2. Load [references/api.md](references/api.md) for `gh api` REST and
   GraphQL recipes, pagination, output shaping, and token hygiene.
3. Run `gh <cmd> --help` or check https://cli.github.com/manual/gh for
   flag-level detail.
4. Run `--json` bare on a list/view command to enumerate its fields.

## Examples

- "Watch the failing CI run and dump only the failed job log."
- "Page through all repos in an org via gh api graphql."
- "Verify the provenance attestation on a release tarball."
- "Set a repo secret for Actions from stdin."

# GitHub CLI

## Version and install

- Latest **v2.101.0** (Sept 15, 2026). Releases are immutable since
  2.93 and ship SLSA attestations: `gh at verify -R cli/cli <asset>`.
- Install: `brew install gh`, `winget install GitHub.cli`, official
  APT repo (cli.github.com/packages), RPM, or release binaries.
- **v2.101.0 caveat:** the APT/RPM repo signing key rotated. Installs
  made before April 8, 2026 must refresh the keyring at
  `/etc/apt/keyrings/githubcli-archive-keyring.gpg`.
- **v2.97 security fixes:** terminal escape injection in `gh api`/`pr
  diff`/`gist view`, partial token leaks in `gh auth status`,
  attestation `--signer-*` regex bypass. Keep gh current.

## Auth

- `gh auth login` - browser OAuth device flow (code auto-copied to
  clipboard since 2.101. `gh config set clipboard disabled` opts out)
  or `--with-token` on stdin. `--hostname` for GHES.
- Token precedence: `GH_TOKEN` > `GITHUB_TOKEN` > stored
  `~/.config/gh/hosts.yml`. GHES: `GH_ENTERPRISE_TOKEN` >
  `GITHUB_ENTERPRISE_TOKEN`.
- `gh auth token` prints the token for scripts:
  `git clone https://x-access-token:$(gh auth token)@github.com/o/r`.
- `gh auth setup-git` installs gh as the git credential helper.
  `gh auth switch` for multi-account. `gh auth refresh -s <scope>`.

## Core commands

- `gh repo create|clone|fork|view|list|sync|rename|archive|read-file|
  read-dir` - read-file/read-dir fetch repo content without cloning.
- `gh pr create|list|view|checkout|merge|diff|review|checks|ready|
  status` - `pr checkout --worktree PATH` (2.98+), `pr merge --squash
  --delete-branch --auto`.
- `gh issue ...` - `issue develop --checkout --worktree` (2.99),
  `--attach` on create/edit/comment for uploads (2.99).
- `gh workflow run <name> -f input=value`, `workflow enable|disable`.
- `gh run list|view|watch|rerun|download|cancel` - `run watch --exit-
  status` for CI gating, `run view --log-failed`, `-j <job>` for one
  job's log.
- `gh release create v1.2.3 --generate-notes`,
  `release download <tag> -p '*.tar.gz'` (public assets need no auth).
- `gh secret set NAME -b "$(cat file)" -a actions` / `gh variable set`.
  Scopes: `-R repo`, `--org`, `--env`.
- `gh attestation verify <file|oci://ref> -R owner/repo`,
  `--signer-workflow`, `at download`, `at trusted-root`.
- `gh search repos|issues|prs|code|commits` - semantic/hybrid issue
  search via `--search-type` (2.98).
- `gh ruleset check`, `gh cache list|delete`, `gh status`,
  `gh agent-task` (Copilot coding agent), `gh skill` (agent skills).

## gh api

- `gh api repos/{owner}/{repo}/issues` - `{owner}`/`{repo}`/`{branch}`
  placeholders resolve from the git remote. `-X`, `-H`, `-f` (raw
  string field), `-F` (typed: `=int`, `@file`, `@-` stdin).
- GraphQL: `gh api graphql -f query='...' -F owner=$OWNER`.
- Pagination: `--paginate` follows REST Link headers and GraphQL
  `pageInfo.endCursor` automatically. Add `--slurp` to merge pages.
- Output: `--jq '.items[].name'` (built-in jq), `--template` Go
  templates (`{{range}}`, `{{tablerow}}`, `{{pluck}}`, `{{timeago}}`).
- `gh api rate_limit` for quota. `GH_DEBUG=api` traces requests.

## Extensions and scripting

- `gh ext install owner/repo`, `upgrade --all`, `browse`. Extensions
  are `gh-*` executables. Install needs no auth since 2.90. Notable:
  gh-dash (TUI), gh-poi (prune merged branches), gh-models, gh-aw,
  official `webhook` (2.100). `gh copilot` now forwards to the Copilot
  CLI (gh-copilot extension deprecated Oct 2025).
- Every list/view supports `--json <fields>` + `--jq`/`--template`.
  `GH_FORCE_TTY`, `GH_PAGER`, `GH_REPO`, `GH_HOST` steer scripting.

## Sources

- Manual: https://cli.github.com/manual
- Releases: https://github.com/cli/cli/releases
