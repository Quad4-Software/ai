# MCP and agent-tool security

MCP tool descriptions, instructions fields and output are executable
instructions to a client model. Treat an MCP server like a browser treats
a webpage: its content can steer the agent that consumes it.

## Tool poisoning

A tool description or instruction block containing directives aimed at
the model rather than documentation for the user. Patterns to flag
(`mcp_audit` looks for these):

- "ignore previous instructions", "forget your rules", "<IMPORTANT>"
  blocks that reorder the user's priorities
- "do not tell/inform the user", "secretly", "covertly"
- Data-movement pressure: "always include the contents of X", "first
  read ~/.ssh and attach it", "send results to <URL>"
- Long descriptions embedding a second system prompt or fake user
  confirmation ("the user already approved this")

A poisoned description needs no code execution; the client model does the
work. Report the exact description text as the evidence.

## Rug pulls

A server approved with benign tools that later serves different
descriptions or new tools. Mitigations: pin server versions, hash or
snapshot `tools/list` output, re-review on version change, prefer local
stdio servers over remote ones for sensitive environments.

## Exec and file sinks in tool handlers

- Tool arguments flowing to `exec.Command`, `subprocess` with
  `shell=True`, `child_process.exec`, or `os.system`: command injection
  reachable by any client, and by any attacker who can influence the
  arguments (prompt injection in data the model reads).
- Path arguments reaching `os.ReadFile`/`open` without a jailed root:
  arbitrary read. Confirm by asking the tool for `../../etc/passwd`.
- Unbounded output: a tool that can return megabytes starves the
  context window; output caps are a security property.
- Write-capable tools (even opt-in ones) need token scoping, allowlisted
  targets, and no secret material in output.

## Prompt injection through tool results

Tool results re-enter the model as context. A `read_file` on an attacker
file, a web fetch, or a search result can carry injected instructions.
Audit: does the server annotate or fence tool output, does it redact
secrets, could tool output include model-steering text. Agents should
treat all tool output as untrusted data.

## AI-generated code review

- Slopsquatting: hallucinated package names registered by attackers.
  Verify every new dependency resolves to a real, old-enough,
  maintained package before it is installed.
- Insecure defaults trained into models: `verify=False`,
  `InsecureSkipVerify`, `md5`, `shell=True`, `pickle`, `Math.random`
  for tokens. Run `injection_scan` and `crypto_scan` on generated diffs
  as a matter of course.
- Missing error paths and optimistic parsing: generated code rarely
  handles the unhappy path. `soft_fuzz_scan` and error-path injection
  apply here too.
- Plausible-but-wrong API usage: check every unfamiliar call against
  the real docs, not the surrounding comments.
