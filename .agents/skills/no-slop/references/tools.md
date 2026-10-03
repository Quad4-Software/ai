## Tools

- `check_text {text, kind}`: lint text and return JSON findings
  - `kind` can be `prose` (default), `doc` or `comment`
  - `doc` bans semicolons
  - `comment` also bans backticks inside comments
- `check_file {path, kind}`: lint a file on disk (max 1 MiB), auto-detecting
  kind from the extension
- `check_dir {path, glob, kind}`: lint every file matching a glob under a
  directory, returns per-file finding counts with the first 5 findings each
- `check_diff {diff, kind}`: lint only the added lines of a unified diff.
  Findings report new-file line numbers
- `fix_text {text, kind}`: same findings plus rewrite instructions
- `list_rules`: the full embedded ruleset
- `check_commit {ref, kind}`: lint added lines of a commit or range.
  `ref` defaults to HEAD, a `a..b` range runs `git diff`. Runs git in
  `MCP_REPO_ROOT` or the current directory

## Prompts

- `review_prose {text}`: returns the full ruleset plus instructions to list
  each violation by rule number and produce a clean rewrite that keeps all
  verifiable facts
