# AGENTS.md

Multi-module Go repo of stdio MCP servers. Remote:
git@github.com:Quad4-Software/ai.git, branch master. Module prefix:
github.com/Quad4-Software/ai/.

## Commands

From repo root:

```
make all       # fmt + go-fix + vet + test + build for every mcp/* dir
make gosec     # gosec security scan per server
make golangci  # golangci-lint run ./... per server
make build test vet fmt clean
```

Per server: cd mcp/<name> && make test, or go test ./....

## Conventions

- Stdlib-first: no third-party dependencies unless vendored or justified.
- Read-only by default: servers never mutate host state. Exception:
  meshchatx ships opt-in GitHub issue tools (create, update, close,
  comment, search) that only activate when a token env var is set
  (MESHCHATX_GITHUB_TOKEN, GITHUB_TOKEN, GH_TOKEN). Tokens are never
  returned in tool output and the target repo is owner/name validated.
- Path jailing: all file access resolved under an allowed root, traversal
  rejected.
- Secrets: never return key material or config secrets, redact them.
- #nosec annotations require an inline justification comment.
- Offline builds and tests: no network fetches in build or test paths.
- Keep main.go thin, domain logic lives in internal/ packages.

## Toolchain

Go 1.27 with a custom toolchain go1.27.1-X:nodwarf5. golangci-lint is
incompatible with this toolchain (it fails on the custom version string /
nodwarf5 build), so make golangci may not work locally. Rely on make vet,
make gosec, and CI instead. Do not switch go.mod toolchain lines to fix it.

## Vendored Micron parser

micron vendors micron-parser-go under
mcp/micron/third_party/micron-parser-go via:

```
require micron-parser-go v1.1.4
replace micron-parser-go => ./third_party/micron-parser-go
```

To update: replace the vendored tree, bump the require version to match,
keep the replace line, run go mod tidy and go test ./... inside
micron. Never go get the parser from the network. Lint targets
exclude /third_party/. Read the vendored CHANGELOG.md when bumping
(NomadNet 1.4.0 fold headings landed in v1.1.4).

## Adding an MCP server

1. Copy mcp/scaffold/ to mcp/<name>/.
2. Set module to github.com/Quad4-Software/ai/mcp/<name> in its go.mod.
3. Register tools in main.go via mcp.NewServer(name, version, tools, nil).
4. Keep it stdlib-only, read-only, offline-capable.
5. The root Makefile discovers it automatically (wildcard mcp/*, excluding scaffold).
6. Add README.md and follow SECURITY.md rules.

## CI

Workflows in .github/workflows/: ci.yml, dependency-review.yml,
gosec.yml, race.yml, fuzz.yml, bench.yml, leak.yml, release.yml, scorecard.yml.
All actions pinned to full SHAs. Every job begins with
step-security/harden-runner (pinned). No arbitrary shell on untrusted
input. Dependabot handles gomod weekly.

See .agents/skills/ for detailed guides: mcp-toolkit, reticulum,
lxmfy, micron, ci-security, release.

## Agent Skills distribution

The `.agents/skills/` directory follows the Agent Skills specification. Once
this repo is pushed to GitHub, anyone can install the skills with:

```
npx skills add Quad4-Software/ai
```

Use `--skill <name>` to install only one, `-g` for global installation, or
`--list` to preview. The `skills.sh.json` at the repo root lists the skill
groups for the skills.sh marketplace.
