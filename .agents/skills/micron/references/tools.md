## micron tools

Registered in mcp/micron/main.go:

- parse_micron - source to structured JSON AST (blocks, inlines, colors,
  spans).
- lint_micron - syntax diagnostics via micron.DiagnosticsJSON.
- render_html - source to safe HTML fragment.
- render_ansi - source to ANSI terminal output.
- extract_links - URL, label, source line, and field names.
- extract_headings - headings with depth and source line.
- search_micron - matching line numbers and snippets.
- generate_template - starter .mu template: page, color, form, table, or
  minimal.
- micron_reference - syntax quick reference.
