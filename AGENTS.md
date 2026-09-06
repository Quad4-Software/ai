# AGENTS.md

Multi-module Go repo of stdio MCP servers. Remote:
git@github.com:Quad4-Software/ai.git, branch master. Module prefix:
github.com/Quad4-Software/ai/.

## Commands

From repo root:

```
make all       # fmt + go-fix + vet + test + build for every *-mcp dir
make gosec     # gosec security scan per server
make golangci  # golangci-lint run ./... per server
make build test vet fmt clean
```

Per server: cd <name>-mcp && make test, or go test ./....

## Conventions

- Stdlib-first: no third-party dependencies unless vendored or justified.
- Read-only by default: servers never mutate host state.
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

micron-mcp vendors micron-parser-go under
micron-mcp/third_party/micron-parser-go via:

```
require micron-parser-go v1.1.0
replace micron-parser-go => ./third_party/micron-parser-go
```

To update: replace the vendored tree, bump the require version to match,
keep the replace line, run go mod tidy and go test ./... inside
micron-mcp. Never go get the parser from the network. Lint targets
exclude /third_party/.

## Adding an MCP server

1. Copy mcp-scaffold/ to <name>-mcp/.
2. Set module to github.com/Quad4-Software/ai/<name>-mcp in its go.mod.
3. Register tools in main.go via mcp.NewServer(name, version, tools, nil).
4. Keep it stdlib-only, read-only, offline-capable.
5. The root Makefile discovers it automatically (wildcard *-mcp).
6. Add README.md and follow SECURITY.md rules.

## CI

Workflows in .github/workflows/: ci.yml, dependency-review.yml,
gosec.yml, race.yml, fuzz.yml, bench.yml, leak.yml, release.yml.
All actions pinned to full SHAs. Every job begins with
step-security/harden-runner (pinned). No arbitrary shell on untrusted
input. Dependabot handles gomod weekly.

See .agents/skills/ for detailed guides: mcp-toolkit, reticulum-mesh,
micron, ci-security, release.
