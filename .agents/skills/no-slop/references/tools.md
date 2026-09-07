## Tools

- `check_text {text, kind}`: lint text and return JSON findings
  - `kind` can be `prose` (default), `doc` or `comment`
  - `doc` bans semicolons
  - `comment` also bans backticks inside comments
- `check_file {path, kind}`: lint a file on disk, auto-detecting kind from the
  extension
- `check_diff {diff, kind}`: lint only the added lines of a unified diff
- `fix_text {text, kind}`: same findings plus rewrite instructions
- `list_rules`: the full embedded ruleset
- `check_commit {ref}`: lint added lines of a commit or range
