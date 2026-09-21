# gh api and scripting recipes

## REST

```bash
gh api repos/{owner}/{repo}/pulls            # placeholders resolve from remote
gh api -X POST repos/{owner}/{repo}/issues -f title=x -f body=y
gh api -F count=42 -F flag=true -f label=@labels.txt endpoint
gh api --input body.json endpoint            # raw JSON body
gh api -H "Accept: application/vnd.github+json" -i endpoint
gh api --cache 10m endpoint                  # client-side cache
```

`-f` is always a string (supports `k[sub]=v` nesting). `-F` is typed:
`=int`, `=bool`, `@file`, `@-` stdin. `--hostname` for GHES.

## GraphQL

```bash
gh api graphql -f query='query($owner:String!,$repo:String!){
  repository(owner:$owner,name:$repo){issues(first:10){nodes{title}}}
}' -F owner=octocat -F repo=hello-world
gh api graphql --paginate --slurp -f query='...'   # walks pageInfo.endCursor
```

`--paginate` follows REST Link headers and GraphQL cursors. `--slurp`
wraps paginated arrays into one array.

## Output shaping

`--jq '.items[] | .name'` uses built-in jq. `--template` uses Go
templates with helpers: `{{range}}`, `{{tablerow}}`, `{{pluck}}`,
`{{timeago}}`, `{{color}}`. List/view commands take `--json <fields>`.
run with bare `--json` to enumerate field names.

Env vars: `GH_FORCE_TTY` (interactive output in scripts), `GH_PAGER`,
`GH_REPO`, `GH_HOST`, `GH_DEBUG=api` (request tracing).

## Recipes

```bash
gh run watch <run-id> --exit-status --interval 5   # gate CI in a script
gh run view <run-id> --log-failed                  # only failing jobs
gh run download <run-id> -n artifact-name
gh pr checkout 123 --worktree ../pr-123            # 2.98+
gh release download v1.2.3 -p '*.tar.gz' -R owner/repo
gh secret set MY_TOKEN -b "$TOKEN" -R owner/repo -a actions
gh at verify artifact.tgz -R owner/repo --signer-workflow release.yml
gh api rate_limit --jq '.resources.core'
gh search code 'repo:owner/repo filename:.env'     # secret sweep
```

## Auth and token hygiene

- `GH_TOKEN` > `GITHUB_TOKEN` > `~/.config/gh/hosts.yml`. GHES:
 `GH_ENTERPRISE_TOKEN` > `GITHUB_ENTERPRISE_TOKEN`.
- Never put tokens on the command line - use `gh auth login
 --with-token` on stdin or env vars. Tokens on argv land in shell
 history and process listings.
- `gh auth token` prints the token. In scripts prefer
 `$(gh auth token)` subshells over storing it in a variable that
 persists or gets echoed.
- `gh auth status -t` shows tokens - keep it out of logs (v2.97 fixed
 a partial token leak here).
- Fine-grained PATs over classic PATs. Scope to single repos.

## Security notes

- v2.97.0 (Jul 2026) fixed terminal escape injection in `gh api`/`pr
 diff`/`gist view`, a partial token leak in `gh auth status`, and an
 attestation `--signer-*` regex bypass. Keep gh current.
- Extensions are executables (`gh-*`) with full user privileges -
 install only from trusted publishers, `gh ext list` audits what is
 installed.
- `gh workflow run -f` inputs are attacker-reachable surface on
 public repos. Treat workflow_dispatch inputs as untrusted.
