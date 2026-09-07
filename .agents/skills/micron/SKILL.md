---
name: micron
description: >
  This skill covers Micron markup, the vendored parser, and the tools in
  micron. Use when you need to parse, lint, render, extract, or search
  Micron, or update the vendored parser.
compatibility: stdio-mcp
metadata:
  server: micron
  dialect: NomadNet 1.4.0
  parser: micron-parser-go v1.1.4
---

## When to use this skill

- You are writing or linting Micron markup.
- You need to render, extract, search, or template Micron.
- You are updating the vendored micron-parser-go.

## How to use

1. Build micron: `cd mcp/micron && go test ./... && go build`.
2. Add the binary to your MCP client config as micron.
3. Call micron_reference or parse_micron for syntax questions, lint_micron
   for checks, render_html for output.
4. Update third_party/micron-parser-go offline, bump the require version,
   and keep the replace line.

## Examples

- Parse a Micron document and render it to HTML.
- Lint a Micron file for unsafe link schemes.
- Show the Micron syntax for collapsible headings and colors.

# Micron and micron

## Dialect authority

Micron is NomadNet's markup language. Primary authority is NomadNet 1.4.0
`nomadnet/ui/textui/MicronParser.py` and Guide.py topic Outputting
Formatted Text. micron-parser-go is the Go/HTML/ANSI implementation
vendored here. When Go and NomadNet disagree, prefer NomadNet and note the
gap.

Full tag and color tables live in
[references/syntax.md](references/syntax.md). Tool details are in
[references/tools.md](references/tools.md).

## Feature map (NomadNet 1.4.0)

- Sections: `>`, `>>`, `>>>`, ... and depth reset with `<`
- Collapsible sections (1.4.0): `` `+> `` open, `` `-> `` collapsed,
  `#!fold OPEN [CLOSED]` glyphs (defaults ▾ / ▸)
- Dividers: `-` and `-X`
- Inline style: bold `` `! ``, italic `` `* ``, underline `` `_ ``, reset ``
- Colors: `` `Fxxx `` / `` `FTxxxxxx `` / `` `f ``, `` `Bxxx `` /
  `` `BTxxxxxx `` / `` `b ``, grayscale `gNN`
- Page headers: `#!c=`, `#!fg=`, `#!bg=`, `#!fold`
- Alignment: `` `c `` `` `l `` `` `r `` `` `a ``
- Links, request fields, same-page `#` anchors
- Explicit anchors `` `:name `` (NomadNet browser)
- Fields: text, masked, multi-row, checkbox `` `? ``, radio `` `^ ``
- Tables: GitHub-style pipes inside `` `t `` fences
- Images: `` `(alt`w=`h=`a=`:url.webp) `` (NomadNet / Kitty, WebP only,
  not rendered by micron-parser-go yet)
- Partials `` `{url`refresh`fields} ``
- Literals `` `= `` and `#` comments

## Heading theme colors

Dark theme defaults: plain fg `ddd`, heading1 `222`/`bbb`, heading2
`111`/`999`, heading3 `000`/`777`. Light theme: plain fg `222`, heading1
`000`/`777`, heading2 `111`/`aaa`, heading3 `222`/`ccc`.

## Tool reference

Nine tools: parse_micron, lint_micron, render_html, render_ansi,
extract_links, extract_headings (includes fold state), search_micron,
generate_template (page, color, form, table, fold, minimal),
micron_reference.

## Vendored parser

Vendored at mcp/micron/third_party/micron-parser-go:

```
require micron-parser-go v1.1.4
replace micron-parser-go => ./third_party/micron-parser-go
```

Upstream: Quad4-Software/Micron-Parser-Go. Read CHANGELOG.md in the
vendored tree before bumping. To update: replace the vendored tree, bump
the require version, keep the replace line, run go mod tidy offline-safe
and go test ./... in mcp/micron. Never go get the parser from the network.

### Recent changelog (v1.1.0 through v1.1.4)

- v1.1.4: NomadNet 1.4.0 folding headings (`` `+> ``, `` `-> ``, `#!fold`)
  and HTML `<details class="Mu-fold">` output.
- v1.1.3: Go 1.27.1+, ForceMonospace link-label escape fix, wasm_exec.js
  in release checksums.
- v1.1.2: SHASUMS256.txt in release packaging.
- v1.1.1: Nerd Font / PUA glyph span for WASM.
- v1.1.0: Document IR, Lint, ANSI, bindings, NomadNet as dialect authority.

Note: ConvertMicronToHTML handles folds. Document IR / Parse still treats
fold lines as paragraphs, so extract_headings scans source lines for fold
prefixes instead of relying on the AST alone.

## Limits and HTML safety

- Source capped at 1 MiB, queries at maxQueryBytes, NUL rejected.
- HTML escaped. javascript:, vbscript:, and file: schemes are prefixed with
  nomadnetwork:// on href. Lint also flags data: and javascript:.
- No raw HTML passthrough.
