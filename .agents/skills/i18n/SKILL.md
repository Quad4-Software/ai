---
name: i18n
description: >
  This skill checks locale coverage through i18n. Use when you need
  missing keys, hardcoded strings, key usage, or locale statistics
  for a repo's translation files.
metadata:
  server: i18n
---

## When to use this skill

- You need missing locale keys relative to the reference locale.
- You are hunting hardcoded strings that bypass the i18n layer.
- You want key usage sites or per-locale statistics.

## How to use

1. Build i18n: `cd mcp/i18n && go test ./... && go build`.
2. Add the binary to your MCP client config as `i18n`.
3. Call `list_locales` to confirm detection, then `missing_keys`.

## Examples

- "Which keys does de.json lack relative to en.json?"
- "Find hardcoded template strings under src/frontend/src."
- "Where is `settings.theme` used in source?"

# i18n

i18n audits locale files against a reference locale (default `en`)
and scans source for strings that bypass the i18n layer. Read-only.

## Environment

- `MCP_REPO_ROOT`: explicit root. Otherwise walks up for
  `meshchatx/src/frontend/locales`, `src/frontend/locales`, or
  `locales`.
- `MCP_LOCALES_DIR`: override the locales dir (absolute or
  root-relative).
- `MCP_READ_ONLY=1` / `READ_ONLY=1` / `--read-only`: no-op. All tools
  are read-only.

## Tool inventory

| tool | args | purpose |
|---|---|---|
| `list_locales` | none | locale files + flattened key counts |
| `missing_keys` | `reference` (en) | keys in reference missing per locale |
| `key_lookup` | `key` | dotted key's value in every locale |
| `hardcoded_strings` | `path`, `limit` (40/300) | template text/attrs not routed through `t()`/`$t` |
| `key_usage` | `key`, `path` | substring search in source, first 30 hits |
| `key_stats` | `limit` (10/100) | counts, longest values, dup values, placeholder mismatches |
| `unused_keys` | `path`, `limit` (50/500) | en keys never quoted in source |

`hardcoded_strings` scans `.vue .svelte .ts .js .tsx .jsx` and
`unused_keys` adds `.py`. `key_stats` and `unused_keys` error if the
reference locale is absent.

## Notes and quirks

- Jailing is a prefix check without symlink resolution, unlike
  agents/context. Keep `MCP_REPO_ROOT` real-path clean.
- `hardcoded_strings` and `unused_keys` are heuristic. Dynamically
  constructed keys produce false positives.
- Placeholder mismatch detection covers `{var}` and `%s` styles.
