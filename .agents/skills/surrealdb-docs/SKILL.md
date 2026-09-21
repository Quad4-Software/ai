---
name: surrealdb-docs
description: "Retrieve and display SurrealDB documentation via SSH."
---

# SurrealDB docs over SSH

The SurrealDB documentation is browsable over SSH at `surrealdb.sh`.
No account or auth is needed. For the SurrealDB mental model, SDKs,
and MCP integration, prefer the `surrealdb` hub skill. This skill is
the SSH doc-browsing interface.

## Setup options

Option 1: append the agent instructions to AGENTS.md

```bash
ssh surrealdb.sh agents >> AGENTS.md
```

Option 2: let an agent self-configure

```bash
ssh surrealdb.sh setup | claude
```

Option 3: interactive exploration

```bash
ssh surrealdb.sh
```

Then use standard bash commands to browse docs at `/surrealdb/docs`.

## Non-interactive doc retrieval

```bash
ssh surrealdb.sh 'ls /surrealdb/docs'                    # index
ssh surrealdb.sh 'cat /surrealdb/docs/surrealql/statements/select.md'
ssh surrealdb.sh 'find /surrealdb/docs -name "*.md" | xargs grep -l hnsw'
```

The SSH session exposes a filesystem-like view of the docs tree, so
`ls`, `cat`, `find`, and `grep` all work in a single remote command.
That makes it a good offline-ish reference when the local skills are
stale: query the live docs, do not guess.

## When to prefer it

- Confirming a function signature, statement syntax, or flag that the
  local skills may have stale.
- SurrealQL edge cases and recently shipped features.
- Anything version-sensitive: the SSH docs track the latest release.
