# lxmfy

Stdio MCP server for LXMFy: searchable docs from `lxmfy.quad4.io`, bot
scaffolding, static bot diagnostics, and test guidance. Go, stdlib only.

## Tools

- `list_topics`, `get_topic`, `list_sections`, `get_section`, `search_docs`,
  `fetch_page` - navigate the LXMFy documentation.
- `list_templates`, `scaffold_bot`, `scaffold_cog` - generate starter bot and
  cog files.
- `list_test_scenarios` - overview of the reliability/stress test suite.
- `diagnose_bot` - static checks on a bot Python file for common mistakes.

## Build

```sh
make all
```

License: 0BSD.