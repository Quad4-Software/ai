---
name: micron
description: >
  Use when working with Micron markup or the micron-mcp server: syntax
  reference, the nine MCP tools (parse, lint, render, extract, search,
  template, reference), the vendored micron-parser-go, and HTML safety rules.
---

# Micron and micron-mcp

## Micron syntax

Micron is NomadNet's lightweight markup language. It has headings at three
levels, bold/italic/underline spans, color directives, alignment, link
fields, form inputs, tables, partials, and literal blocks.

- Headings are single, double, and triple greater-than signs.
- Bold, italic, and underline spans each have their own delimiters.
- Colors are backtick-quoted directives and named blocks.
- Link fields are backtick-wrapped and follow the url|label format.
- Forms expose input fields and submit actions.
- Tables are pipe-delimited and can carry data descriptors.
- Partials are included by name.
- Literal blocks are left as preformatted text.

For exact syntax, call the micron_reference tool or read
micron-mcp/third_party/micron-parser-go/micron/doc.go.

## micron-mcp tools

Registered in micron-mcp/main.go:

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

## Vendored parser

The parser is vendored at micron-mcp/third_party/micron-parser-go with

```
require micron-parser-go v1.1.0
replace micron-parser-go => ./third_party/micron-parser-go
```

in micron-mcp/go.mod. To update: replace the vendored tree, bump the
require version, keep the replace line, run go mod tidy offline-safe and
go test ./... in micron-mcp. Never go get the parser from the network.

## Limits and HTML safety

- Source input capped at 1 MiB (maxSourceBytes in main.go). Queries
  capped at maxQueryBytes. NUL bytes rejected.
- HTML output is escaped. javascript:, vbscript:, data:, and file:
  link schemes are rejected and neutralized in the parser (url.go, lint.go,
  html_build.go). Do not weaken these checks.
- No raw HTML passthrough: Micron never emits script tags, event-handler
  attributes, or unescaped markup.
