# Security

## Model

These are local stdio servers. There is no network listener, no auth
surface, no remote input beyond what the calling agent sends. Trust
boundary: the MCP client and the user whose config registers the binary.

## What the code does on purpose

- Reads files under a configured repo root (path-jailed, `..` rejected).
- Reads local Reticulum state under `~/.reticulum` (config lint is
  read-only; storage decoders never write).
- Runs fixed-argv subprocesses (`git`, `task`, `uv`, `pnpm`, RNS
  utilities). No shell strings, no user-controlled executable names.
- Fetches from host allowlists only (reticulum.network, meshchatx.com,
  rns.recipes, unsigned.io, github.com, api.github.com).
- `gateway-mcp` spawns servers from the client's own config file. It
  cannot be pointed at arbitrary binaries through tool arguments.

## Reporting

Open a private security advisory on the repository rather than a public
issue. Do not include real identity keys, RPC keys, or destination
hashes in reports; the tools redact these in output by design.
