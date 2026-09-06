# i18n-mcp

Stdio MCP server for locale coverage checks. Read-only. Stdlib only.

Root: `MCP_REPO_ROOT` or nearest ancestor containing a locales dir.
Override dir with `MCP_LOCALES_DIR`.

## Tools

- `list_locales` - locale files with flattened key counts
- `missing_keys {reference}` - keys present in reference (default en) missing elsewhere
- `key_lookup {key}` - a dotted key's value in every locale
- `hardcoded_strings {path, limit}` - template text and attribute literals not via i18n
- `key_usage {key, path}` - where a key is referenced in source
- `unused_keys {path, limit}` - reference-locale keys never quoted in source (candidates)
- `key_stats {limit}` - longest values, duplicate values, placeholder mismatches
