---
name: workspace
description: >
  This skill wraps repo dev commands through workspace. Use when you
  need to run Taskfile targets, git commands, test mapping, or test
  triage.
metadata:
  server: workspace
---

## When to use this skill

- You want Taskfile targets listed or run without a shell.
- You need git status/log/diff/blame through tools.
- You want the repo's test mapping or a triaged test failure.

## How to use

1. Build workspace: `cd mcp/workspace && go test ./... && go build`.
2. Add the binary to your MCP client config as `workspace`.
3. `task_list` then `task_run {name}`, `find_test` then `run_tests`
   or `test_triage`.

## Examples

- "List Taskfile targets and run the test target."
- "Which test file covers `internal/docs/docs.go`?"
- "Triage the failing test for meshchatx backend manager."

# workspace

workspace executes repo dev commands as fixed-argv subprocesses: no
shell, `cmd.Dir = root`, 180 s timeout, 256 KiB output cap. Root is
`MCP_REPO_ROOT` or the nearest `.git`.

## Environment

- `MCP_REPO_ROOT`: explicit root.
- `MCP_READ_ONLY=1` / `READ_ONLY=1` / `--read-only`: hides the three
  Write tools (`task_run`, `run_tests`, `test_triage`).
- Needs `task`, `git`, `uv`, `pnpm`, `go` on PATH depending on the
  tool.

## Tool inventory

| tool | args | purpose |
|---|---|---|
| `task_list` | none | `task --list` |
| `task_run` | `name` (Write) | run a target, name regex-gated |
| `git_status` | none | branch + porcelain status |
| `git_log` | `n` (15) | `git log --oneline` |
| `git_diff_stat` | `ref` | `git diff --stat [ref]` |
| `run_tests` | `file` (Write) | run mapped test file |
| `git_blame_context` | `file`, `start`, `end` (<=500 lines) | blame summary per commit |
| `test_triage` | `file` (Write) | mapped test + condensed failure report |
| `find_test` | `file` | candidate test paths with exists markers |

## Test mapping

- `.py` -> `tests/backend/test_<s>.py` or `<s>_test.py`
- `.ts/.tsx/.js/.jsx/.vue/.svelte` -> `tests/frontend/<s>.test.js`,
  `.test.ts`, `.spec.js`
- `.go` -> `<dir>/<s>_test.go`
- Generic stems (manager, index, core, main, utils) also try the
  parent dir name.

Runners: `uv run pytest <f> -q --tb=short`, `pnpm exec vitest run
<f>`, `go test ./<dir>`. File args must be repo-relative.

## Notes and quirks

- stderr merges into stdout output.
- README drift: `run_tests` is implemented but undocumented.
- Errors arrive as `error: ` tool content, not JSON-RPC errors.
