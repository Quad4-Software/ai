# Micron syntax (NomadNet 1.4.0)

Authority: NomadNet `MicronParser.py` and Guide.py. Escaping uses a
leading backslash before a tag sequence. A bare double-backtick pair
resets bold, italic, underline, colors, and alignment.

## Page headers

Leading `#` lines that start with `#!` are directives (not comments):

| Directive | Meaning |
| --- | --- |
| `#!c=SECONDS` | Peer cache TTL. Use `0` to disable cache. Must be first when present. Default peer cache is 12 hours. |
| `#!fg=XXX` | Default foreground, 3 or 6 hex digits |
| `#!bg=XXX` | Default background, 3 or 6 hex digits. Place after `#!c=` when both are used. |
| `#!fold OPEN [CLOSED]` | Fold glyphs. One token uses the same glyph for open and closed. Defaults: open ▾, closed ▸ |

## Sections

| Markup | Meaning |
| --- | --- |
| `> Title` | Level 1 heading / section |
| `>> Title` | Level 2 |
| `>>> Title` | Level 3+ (palette defined for levels 1..3) |
| `>>>>` | Indent-only section with no heading text |
| `<` | Reset section depth to 0 for the rest of the line |

Any depth is allowed. Section content is indented. Heading text also
becomes an auto-anchor slug (lowercased, non-alnum runs to `-`).

## Collapsible sections (1.4.0)

| Markup | Meaning |
| --- | --- |
| `` `+> Title `` | Foldable, expanded by default |
| `` `-> Title `` | Foldable, collapsed by default |
| `` `+>> Nested `` | Any `>` depth after the `+` / `-` |

Folds cover everything until the next heading of the same or higher
level. NomadNet toggles with Enter, Space, or click. micron-parser-go
HTML uses `<details class="Mu-fold">` with `data-mu-fold` and
`Mu-fold-glyph`.

## Dividers

| Markup | Meaning |
| --- | --- |
| `-` | Horizontal rule (─) |
| `-X` | Divider using character X (control characters rejected) |

## Alignment (line-start)

| Tag | Align |
| --- | --- |
| `` `c `` | center |
| `` `l `` | left |
| `` `r `` | right |
| `` `a `` | document default |

## Inline formatting

| Tag | Effect |
| --- | --- |
| `` `! `` ... `` `! `` | bold |
| `` `* `` ... `` `* `` | italic |
| `` `_ `` ... `` `_ `` | underline |
| `` `` `` | reset style, colors, and align |

## Colors

| Tag | Effect |
| --- | --- |
| `` `Fxxx `` | foreground 3-digit hex (each nibble expanded) |
| `` `FTxxxxxx `` | foreground truecolor 6-digit hex |
| `` `f `` | restore default foreground |
| `` `Bxxx `` | background 3-digit hex |
| `` `BTxxxxxx `` | background truecolor 6-digit hex |
| `` `b `` | restore default background |
| `gNN` | grayscale token (digits 0-9), accepted as a color value |

### Default theme palettes

Dark (NomadNet `STYLES_DARK` / Go `DarkTheme`):

| Role | FG | BG |
| --- | --- | --- |
| plain | `ddd` | default |
| heading1 | `222` | `bbb` |
| heading2 | `111` | `999` |
| heading3 | `000` | `777` |

Light (`STYLES_LIGHT`):

| Role | FG | BG |
| --- | --- | --- |
| plain | `222` | default |
| heading1 | `000` | `777` |
| heading2 | `111` | `aaa` |
| heading3 | `222` | `ccc` |

NomadNet maps 3-digit and 6-digit colors onto mono / 16 / 256 / truecolor
Urwid palettes via `low_color` and `high_color` in MicronParser.py. Go
`ColorToCSS` emits `#rgb`, `#rrggbb`, or grayscale `#hhhhhh`.

## Links

| Form | Example |
| --- | --- |
| URL only | `` `[hash:/page/index.mu] `` |
| Labeled | `` `[label`hash:/page/index.mu] `` |
| With fields / vars | `` `[Go`:/page/x.mu`user|token|action=view] `` |
| Submit all fields | `` `[Go`:/page/x.mu`*] `` |
| Same-page anchor | `` `[Jump`#name] `` |
| Next heading | `` `[Continue`#] `` |
| Cross-page anchor | `` `[Go`hash:/page.mu`anchor=name] `` |

## Anchors (NomadNet browser)

| Form | Meaning |
| --- | --- |
| `` `:name `` | Explicit zero-width marker (`A-Z` `a-z` `0-9` `_` `-`) |
| heading text | Auto-slug anchor shared with explicit names |

micron-parser-go currently keeps `` `:name `` as visible text in HTML. Prefer
`#name` links for portable jump targets.

## Fields

| Form | Meaning |
| --- | --- |
| `` `<name`value> `` | Text field (default width 24) |
| `` `<name`> `` | Empty text field |
| `` `<16|name`> `` | Width in columns (max 256) |
| `` `<40x5|notes`> `` | Width x rows |
| `` `<!|secret`hidden> `` | Masked |
| `` `<!32|all`hidden> `` | Masked with width |
| `` `<?|name|value`>label `` | Checkbox |
| `` `<?|name|value|*`>label `` | Pre-checked checkbox |
| `` `<^|name|value`>label `` | Radio |
| `` `<^|name|value|*`>label `` | Pre-selected radio |

Checkboxes that share a field name join checked values with commas.
Radios with the same name are mutually exclusive.

## Tables

Fence with `` `t ``. Optional align letter and max width on the opener
(`` `tl ``, `` `tc30 ``, `` `tr ``). Body is GitHub-flavored pipes:

```
`t
| Name | Price | Qty |
| ---- | :---: | --: |
| Apple | Free | 5 |
`t
```

## Images (NomadNet)

Block-level, own line, WebP only:

```
`(alt text`w=n`a=c`:/media/demo.webp)
```

Options: `w=` / `h=` as columns, rows, percent, or `n` (near native).
`a=` is `l`, `r`, or `c`. micron-parser-go does not render images yet.

## Partials

| Form | Meaning |
| --- | --- |
| `` `{url} `` | Async include |
| `` `{url`10} `` | Refresh every 10 seconds |
| `` `{url`0`pid=32|field} `` | Fields and partial id |

## Literals and comments

| Form | Meaning |
| --- | --- |
| `` `= `` | Toggle literal block (content not interpreted) |
| `# comment` | Line comment when `#` is first and not `#!` |

## MicronParser.py surface (NomadNet)

Useful entry points and helpers in the Python authority:

- `markup_to_attrmaps` builds Urwid widgets from markup
- `parse_line`, `make_output`, `make_style`, `parse_partial`, `parse_image`
- `CollapsibleHeading` widget with fold glyphs and toggle
- `slugify_micron` for heading auto-anchors
- `STYLES_DARK` / `STYLES_LIGHT`, `DEFAULT_FOLD_GLYPHS` (`▾`, `▸`)
- `DEFAULT_FG_DARK` `ddd`, `DEFAULT_FG_LIGHT` `222`
