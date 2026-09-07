# Contributing

Small, focused pull requests.

## Rules

- Go stdlib only. One new dependency needs a written justification in
  the PR body.
- `make all` (fmt, vet, test, build) must pass in every touched dir.
  `make gosec` must stay clean. Annotate with `#nosec` plus a reason,
  never blanket-suppress.
- New servers start from `mcp/scaffold/` and keep its error-handling
  contract: missing args name the argument and the valid values, tool
  failures return `isError` content, nothing panics out of a request.
- Tools are read-only by default. A mutating tool needs an explicit
  policy in its description.
- Bound every result. Say when output is capped and why.
- No secrets, paths, or machine-specific data in source or tests.
- SPDX `0BSD` header on every file.

## Testing expectations

Real targets, not endpoint reachability. A new tool should be exercised
against a real repository, config, or document, and the output judged
for usefulness, not just for a 200-shaped response.
