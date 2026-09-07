---
name: workspace
description: >
  This skill wraps repo dev commands through workspace-mcp.
  Use when you need to run Taskfile targets, git commands, test mapping,
  or test triage.
metadata:
  server: workspace-mcp
---

## When to use this skill

- You want to run a Taskfile target or check git status/log/diff.
- You need to find the test file for a source file.
- You want a compact test failure report for a changed file.

## How to use

1. Build workspace-mcp: `cd workspace-mcp && go test ./... && go build`.
2. Add the binary to your MCP client config as `workspace`.
3. Call `task_list` to see targets and `task_run {name}` to execute.
4. Use `find_test {file}` and `test_triage {file}` for test workflows.

## Examples

- "Run the `test` target for this repo."
- "Show the git status and recent log."
- "Find and run the tests for `internal/mcp/server.go`."
