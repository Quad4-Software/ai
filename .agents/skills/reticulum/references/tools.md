## Servers

- `rns-mcp` serves the Reticulum manual as searchable, section-aware tools with
  low memory usage.
- `lxmf-mcp` reads `~/.reticulum` directly for sanitized config, storage
  inventory, identity names and decoded known destinations.
- `lxmf-mcp` is strictly read-only. It must never write to `~/.reticulum`,
  restart rnsd or expose private keys. Identity files are listed by name only
  and config secrets are redacted.
- `lxmfy-mcp` provides LXMFy docs, bot scaffolding, diagnostics and test
  guidance. Framework detail lives in the lxmfy skill.
- `meshchatx-mcp` provides MeshChatX documentation tools for lookups and
  browsing, plus opt-in GitHub issue write tools (template-aware
  issue_create in the repo's issue-form style, issue_update for
  edit/close/reopen, issue_comment, issue_search, issue_references)
  gated behind a GitHub token env var. Issue text auto-links ecosystem
  and spec terms (BCP 47, LXMF, RNode, KISS, ...) to canonical URLs so
  filed issues stay readable without assumed knowledge.

## Useful rns-mcp tools

- `search_docs`, `get_topic`, `rns_status`, `rns_path_table`,
  `rns_path_lookup`, `rns_probe`, `rns_config_check`, `community_read`,
  `interface_directory`.

## Conventions for Reticulum MCP tools

- Read-only by default. Tools never start, stop or reconfigure the daemon.
  The one exception is meshchatx-mcp's GitHub issue tools, which write only
  to the issue tracker and only when a token env var is configured.
- Path-jail all file access to `~/.reticulum` (or a configured root), reject
  traversal.
- Redact secrets: passphrases, private keys, tokens, credentials and any field
  that could identify or compromise a mesh peer.
- Return destination hashes in canonical angle-bracket hex form, e.g.
  `<13425ec15b621c1d928589718000d814>`, and never display private key data.
- Cap and paginate large outputs; RNS networks may be low bandwidth and the
  consumers of these tools may be remote.
- Use `#nosec` sparingly and only with an inline justification comment.

## Blackhole tool rules

When building MCP tools around blackhole data, keep responses bounded, redact
operator and victim identities, and never expose private key material. Only read
state; do not restart rnsd or rewrite `~/.reticulum/config`.

## Discovery tool rules

Treat unknown discovery info as untrusted. Invalid discovery stamps are cached and
corrupted discovery files are handled, but an agent should never fabricate or
replay discovery data. Redact passphrases and identity material.

## IFAC tool rules

When building MCP tools around IFAC config, redact `ifac_key`, passphrases and
any credential fields. Keep responses short for low-bandwidth links.
