# lxmf

Stdio MCP server reading local Reticulum instance state (`~/.reticulum`,
override with `MCP_RNS_CONFIG`). Read-only. Private key material is never
returned: identity files are listed by name only, config secrets redacted.

## Tools

- `store_config` - sanitized config (rpc_key, passphrases, ifac keys redacted)
- `store_inventory` - storage inventory with sizes and file counts
- `store_identities` - identity names under identities/ and storage/identities
- `store_destinations {limit}` - decoded announce store: hash, name, first/last seen

Msgpack decoding uses `python3` (override `MCP_RNS_PYTHON`), everything else is pure Go.
