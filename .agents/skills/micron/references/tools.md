## micron tools

Registered in mcp/micron/main.go. Dialect: NomadNet 1.4.0 via vendored
micron-parser-go v1.1.4.

- parse_micron - source to structured JSON AST (blocks, inlines, page
  colors including fold glyphs, spans). Fold headings are not yet
  first-class blocks in the IR.
- lint_micron - syntax diagnostics via micron.DiagnosticsJSON.
- render_html - source to safe HTML fragment. Fold headings become
  `<details class="Mu-fold">`.
- render_ansi - source to ANSI terminal output.
- extract_links - URL, label, source line, and field names.
- extract_headings - headings with depth, source line, collapsible, and
  collapsed flags (source scan so `` `+> `` / `` `-> `` are included).
- search_micron - matching line numbers and snippets.
- generate_template - starter .mu template: page, color, form, table,
  fold, or minimal.
- micron_reference - NomadNet 1.4.0 syntax quick reference.

See [syntax.md](syntax.md) for the full tag and color tables.
