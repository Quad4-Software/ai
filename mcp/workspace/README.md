# workspace

Stdio MCP server wrapping repo dev commands. Validated args, bounded output,
180 s timeout. Stdlib only.

Root: `MCP_REPO_ROOT` or nearest `.git` ancestor.

## Tools

- `task_list` / `task_run {name}` - Taskfile targets (name allowlist charset)
- `git_status` - branch plus porcelain status
- `git_log {n}` - recent oneline commits
- `git_diff_stat {ref}` - diffstat of working tree or vs a ref
- `find_test {file}` - maps a source file to probable test paths, marks existing
- `test_triage {file}` - run the mapped test, return compact failure report (file:line, assertions)
- `git_blame_context {file, start, end}` - per-commit blame summary for a range
