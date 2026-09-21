# tea recipes and edge cases

## CI login without a config file

```bash
export GITEA_SERVER_URL=https://gitea.example.com
export GITEA_SERVER_TOKEN=<token>
tea issues list --repo owner/name -o simple
```

The env pair creates an ephemeral login that participates in
remote-URL matching. Alternatively `tea login add --name ci --url
$GITEA_SERVER_URL --token $GITEA_SERVER_TOKEN` writes a real entry.

## Common flows

```bash
tea issues list --state open -o json | jq '.[].title'
tea issues create --title "x" --file body.md -o json     # json since 0.16
tea pulls create --title "x" --head feature --base main
tea pulls merge 42 --style squash
tea releases create v1.4.0 --asset ./dist/app-linux-amd64.tar.gz
tea notifications -o simple | notify-send -            # desktop hookup
tea actions secrets list --repo owner/name             # Gitea Actions
tea open 189                                           # issue in browser
```

`--repo` accepts `owner/repo`, a local path, or a URL. `--remote`
picks a non-default git remote. `--login` selects the named login
when several exist for one host.

## Known edge cases

- JSON output is tea's own shape, not raw REST, and keys shift
 between releases. Pin the tea version in CI.
- `pr create` on some versions prints only a confirmation, no URL.
 Fetch it with `pulls list` + jq if scripting needs the link.
- No commit-status/checks view for PRs, no draft toggle, no `api`
 passthrough - fall back to `curl $GITEA_SERVER_URL/api/v1/...` with
 the token for anything tea does not cover.
- `tea logout` removes the local entry only. Revoke the token in the
 web UI.
- Forgejo versions report as `7.0.0+gitea-1.22.0`. Parse the
 `+gitea-` suffix for feature detection.
- Multiple logins on one host: tea picks via remote-URL match or
 `--login`. Ambiguous cases error with "no Gitea login found".
