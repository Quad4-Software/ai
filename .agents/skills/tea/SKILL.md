---
name: tea
description: >
  This skill covers tea, the official Gitea CLI (v0.16.x, Sept 2026),
  which also works against Forgejo and Codeberg. Use it for login
  management across instances, issues/pulls/releases/milestones from
  the terminal, actions secrets and variables, notifications, JSON
  output for scripts, and Forgejo compatibility caveats.
---

## When to use this skill

- You are working against Gitea, Forgejo, or Codeberg from a terminal
  or CI.
- You need repo, issue, PR, release, or notification operations without
  the web UI.
- You are scripting against one or more Gitea instances.

## How to use

1. Read this file for auth and the command map.
2. Load [references/recipes.md](references/recipes.md) for CI login,
   common flows, and edge cases.
3. `tea <cmd> --help` for flags. The repo README tracks the full list:
   https://gitea.com/gitea/tea

## Examples

- "Add a CI login for our Gitea instance without a config file."
- "List open PRs on this repo as JSON."
- "Create a release with attached binaries on Forgejo."

# tea (Gitea CLI)

## Version and install

- Latest **v0.16.0** (Sept 10, 2026). Still 0.x: expect interface churn
  between releases, pin the version in CI.
- Install: `go install gitea.dev/tea@latest`, release binaries from
  https://dl.gitea.com/tea/ (with checksums.txt/.sig), `brew install
  tea`, Docker image `gitea/tea:0.16.0`, distro packages (often lag).

## Auth and logins

- Config: `$XDG_CONFIG_HOME/tea/config.yml` (older: `~/.tea/tea.yml`).
  Since ~0.15 secrets go to the OS keychain where available.
- `tea login add -n name -u https://gitea.example.com -t <token>` -
  interactive mode also offers an OAuth browser flow. Validates the
  token before saving. `--insecure` for self-signed TLS.
- `tea login list|default|edit|status|logout`, `tea whoami`.
  `logout` only removes the local entry. Revoke the token in the web
  UI separately.
- **Env login:** `GITEA_SERVER_URL` + `GITEA_SERVER_TOKEN` create an
  ephemeral login usable in CI without a config file.
- Token scopes: Gitea application token with `read:user` minimum. Add
  `write:issue`, `write:repository` etc. as needed.

## Remote discovery

Inside a git worktree, tea reads the remote URL, matches the host
against configured logins, and derives owner/repo. Overrides: `--repo
owner/repo|path|URL`, `--login <name>`, `--remote <name>`. Outside a
repo you must pass `--login` + `--repo`.

## Commands (v0.16)

- `tea issues|i` list/create/view/update (`--state`, `--labels`,
  `--assignee`, `--milestone`). `issues create -o json` since 0.16,
  body via stdin or `--file`.
- `tea pulls|pr` list/create/checkout/merge/close/review,
  `pulls clean` prunes merged branches.
- `tea releases|r` `create v1.4.0 --asset ./file`, list, edit, delete.
- `tea repos` details/create/fork/search/migrate. `tea clone`/`C`.
- `tea labels`, `milestones|ms`, `times|t`, `organizations|org`,
  `branches|b`, `comment|c`, `webhooks` (incl. auth header/branch
  filter).
- `tea actions secrets list|create` and `actions variables` - the only
  Gitea Actions coverage.
- `tea open|o` (browser: `tea open 189`, `tea open issues`),
  `notifications|n` (`-o simple`, pipe to notify-send),
  `admin users`, `ssh-keys`.

## Scripting

- `-o/--output`: `table`, `simple`, `json`, `yaml`, `csv`, `tsv`
  (varies per command. JSON on creates is new in 0.16).
- JSON is tea's own shape, not the raw REST API, and keys shift between
  releases. Pipe `-o json` to `jq`. There is no `--jq`, no `api`
  passthrough, no GraphQL.

## Forgejo compatibility

Works: Forgejo serves the Gitea `/api/v1` surface and treats tea as
the standard CLI (Codeberg included). Caveats:

- Forgejo reports versions like `7.0.0+gitea-1.22.0`. Tools should
  parse the `+gitea-` suffix for feature detection.
- Known gaps: no commit-status checks for PRs, `pr create` may print
  no URL, JSON keys vary across tea versions.

## Limitations vs gh

No extension system, no arbitrary `api` command, no attestation or
rate-limit helpers, no `run watch`. Multi-instance named logins are
its strength over gh.

## Sources

- Repo: https://gitea.com/gitea/tea (module gitea.dev/tea)
- Binaries: https://dl.gitea.com/tea/
