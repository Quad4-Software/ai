---
name: micron
description: >
  This skill covers Micron markup, the vendored parser, and the tools in
  micron. Use when you need to parse, lint, render, extract, or search
  Micron, or update the vendored parser.
compatibility: stdio-mcp
metadata:
  server: micron
---

## When to use this skill

- You are writing or linting Micron markup.
- You need to render, extract, search, or template Micron.
- You are updating the vendored `micron-parser-go`.

## How to use

1. Build micron: `cd mcp/micron && go test ./... && go build`.
2. Add the binary to your MCP client config as `micron`.
3. Call `micron_reference` or `parse` for syntax questions, `lint` for checks, `render` for output.
4. Update `third_party/micron-parser-go` offline, bump the require version, and keep the `replace` line.

## Examples

- "Parse a Micron document and render it to HTML."
- "Lint a Micron file for unsafe link schemes."
- "Show the Micron syntax reference for headings."

# Micron and micron

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
mcp/micron/third_party/micron-parser-go/micron/doc.go.

## Tool reference

Tool details are in [references/tools.md](references/tools.md). This section is optional if micron is installed.

## Vendored parser

The parser is vendored at mcp/micron/third_party/micron-parser-go with

```
require micron-parser-go v1.1.0
replace micron-parser-go => ./third_party/micron-parser-go
```

in mcp/micron/go.mod. To update: replace the vendored tree, bump the
require version, keep the replace line, run go mod tidy offline-safe and
go test ./... in mcp/micron. Never go get the parser from the network.

## Limits and HTML safety

- Source input capped at 1 MiB (maxSourceBytes in main.go). Queries
  capped at maxQueryBytes. NUL bytes rejected.
- HTML output is escaped. javascript:, vbscript:, data:, and file:
  link schemes are rejected and neutralized in the parser (url.go, lint.go,
  html_build.go). Do not weaken these checks.
- No raw HTML passthrough: Micron never emits script tags, event-handler
  attributes, or unescaped markup.
