# ci-security-mcp

Stdio MCP server for CI/CD security: GitHub Actions secure-use guidance,
workflow and Dockerfile scanning, and SHA pinning. Stdlib only.

## Tools

- `list_guides` / `get_guide {id}` - GitHub secure-use, pull_request_target,
  secrets, workflow syntax, security hardening, OWASP Docker, immutable releases
- `reference` - embedded offline cheat sheet (pinning, secrets, permissions,
  injection, Docker hardening)
- `scan_workflow {yaml}` - unpinned actions, pull_request_target + head checkout,
  script injection (including inside run blocks), missing permissions, secret
  echo, pipe-to-shell, hardcoded credentials
- `scan_dockerfile {dockerfile}` - unpinned/latest images, missing USER,
  ADD remote, privileged flags, pipe-to-shell, chmod 777
- `lint_yaml {yaml}` - tabs, trailing whitespace, CRLF, duplicate keys
- `resolve_ref {repo, ref}` - tag/branch to commit SHA via api.github.com
- `pin_workflow {yaml}` - rewrite uses: lines to SHA pins with the tag as a
  comment (max 10 lookups per call)

## Prompts

- `audit_workflow {yaml}` - audit against the embedded reference.

## Build and test

```
go test ./...
go build -ldflags="-s -w" -o ci-security-mcp .
```
