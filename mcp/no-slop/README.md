# no-slop

Stdio MCP server that lints prose against the
[no_ai_slop_writing_rules](https://github.com/realrossmanngroup/no_ai_slop_writing_rules)
rule set plus the MeshChatX style rules. Fully offline, zero dependencies, single static binary.

## Tools

- `check_text` `{text, kind}` - lint text, returns JSON findings (rule, line, match, fix).
  `kind` is `prose` (default), `doc`, or `comment`. `doc` bans semicolons,
  `comment` also bans backticks inside comments.
- `check_file` `{path, kind}` - lint a file on disk, kind auto-detects from extension.
- `check_diff` `{diff, kind}` - lint only the added lines of a unified diff, with new-file line numbers.
- `fix_text` `{text, kind}` - same findings plus rewrite instructions, ready to act on.
- `list_rules` - the full embedded ruleset.

## Prompts

- `review_prose` `{text}` - returns the ruleset plus the text as a review task.

## What it catches

Emojis and decorative unicode arrows, em/en dashes, semicolon density, backticks in
comments, exclamation marks, all 24 no-slop rules (filler phrases, AI verbs and
transitions, intensifiers, weasel words, academic tells, inflated symbolism,
hallucinated citation markup, research narration), dramatic and vague headings,
"It's not X, it's Y" parallelism, hedging-per-paragraph overload, and flat
sentence-length variance. Fenced code blocks and quoted text are excluded.

## Build and test

```
go test ./...
go build -ldflags="-s -w" -o no-slop .
```

## MCP config

```json
{
  "mcpServers": {
    "prose": { "command": "/path/to/no-slop" }
  }
}
```
- `check_commit {ref}` - lint added lines of a commit or range
