## lxmfy tools

Registered in mcp/lxmfy/main.go. Docs are fetched live from
lxmfy.quad4.io (1 MiB page cap, 32-entry LRU, 10 min TTL, 15s timeout),
so the server needs network at runtime though builds and tests stay
offline.

- `list_topics`, `get_topic`, `list_sections`, `get_section` -
  navigate the three doc pages above. Prefer `get_section` over
  `get_topic` on large pages.
- `search_docs` - ranked excerpts with topic and section anchors.
- `fetch_page` - any URL on the lxmfy.quad4.io host (allowlisted), for
  deep links outside the topic index.
- `list_templates`, `scaffold_bot`, `scaffold_cog` - generate starter
  files. Names must match `^[a-z][a-z0-9_-]{0,48}$`. Templates:
  minimal, echo, note, reminder, rrc, cogtest.
- `list_test_scenarios` - reliability and stress test categories.
- `diagnose_bot` - static checks on bot source (max 64 KiB): missing
  admins, Landlock disabled, unsafe threading, missing run guard, etc.
