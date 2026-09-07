---
name: i18n
description: >
  This skill checks locale coverage through i18n-mcp.
  Use when you need to find missing keys, hardcoded strings, key usage, or
  locale statistics.
metadata:
  server: i18n-mcp
---

## When to use this skill

- You are adding a new locale or key.
- You want to ensure every key is translated.
- You are hunting hardcoded UI strings.

## How to use

1. Build i18n-mcp: `cd i18n-mcp && go test ./... && go build`.
2. Add the binary to your MCP client config as `i18n`.
3. Call `list_locales` to see files, `missing_keys {reference}` for gaps, and `key_usage {key}` for references.
4. Use `hardcoded_strings {path}` to find untranslated text.

## Examples

- "List locales and show keys missing from `es` against `en`."
- "Find hardcoded strings in `templates/`."
- "Show where `save_button` is used in source."
