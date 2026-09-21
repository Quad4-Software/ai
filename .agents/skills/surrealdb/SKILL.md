---
name: surrealdb
description: >
  This skill covers SurrealDB, the multi-model database written in Rust
  (documents, graphs, vectors, full-text, time series, geospatial and
  relational in one ACID engine), plus its SDKs, deployment models,
  authentication model, and the SurrealDB MCP servers. Use it as the hub
  for anything SurrealDB: picking an SDK (Go, Rust, JS, Python, .NET,
  Java, Kotlin, PHP, Swift, Mojo), embedding the engine, running a server
  or cluster, connecting agents through mcp.surrealdb.com or the embedded
  surreal mcp server, and finding the right specialist skill or doc page.
---

## When to use this skill

- You are choosing or wiring a SurrealDB SDK, especially Go
  (surrealdb.go over the wire, surrealdb.c.go embedded).
- You are deciding how to run SurrealDB: embedded, single node,
  distributed, Docker, or SurrealDB Cloud.
- You are setting up authentication or permissions.
- You are connecting an AI agent through the hosted or embedded MCP
  server.
- You need the map to the official skills and doc pages.

## How to use

1. Start here for the mental model and version facts.
2. Load a file from references/ for SDK, deployment, security, or MCP
   detail.
3. Load the specialist skills for query work: surrealql for statements,
   surrealql-functions for built-in functions, surrealql-performance for
   indexing and record ID design, surrealdb-vector for HNSW and KNN,
   surrealdb-js or surrealdb-python for those SDKs, surrealdb-cli for the
   surreal binary, surrealkit for schema migrations, and surrealdb-docs
   to pull doc pages.
4. Fall back to https://surrealdb.com/docs for anything else. Every page
   is available as markdown by appending .md to the path, and
   https://surrealdb.com/docs/llms.txt is the full index.

## Examples

- "Connect this Go service to SurrealDB and add a users table."
- "Should I use RocksDB or SurrealKV for a single-node server?"
- "Give the coding agent tools against my local database."
- "Write a DEFINE ACCESS block so users can only read their own records."

# SurrealDB

## Core concepts

- **SurrealDB** is a multi-model database written in Rust. One engine
  stores document, graph, vector, full-text, relational, time-series,
  geospatial and key-value data, all inside one ACID transaction. Latest
  stable is v3.2.x as of September 2026.
- **SurrealQL** is the one query language. It keeps SQL shape (SELECT,
  CREATE, UPDATE, DELETE) and adds arrow syntax for graph paths, dot
  notation for nested fields, and vector similarity operators. Target 3.x
  syntax. 2.x syntax is a common source of stale queries.
- **Other interfaces** reach the same data: GraphQL (schema generated
  from tables), a REST API over HTTP, an RPC protocol over WebSocket or
  HTTP, ISO GQL, and DEFINE API for custom HTTP endpoints written in
  SurrealQL.
- **Schema is your choice.** Schemaless tables accept anything.
  Schemafull tables enforce defined fields, types and assertions on
  write. You can tighten a table later without rewriting data.
- **Real-time is built in.** Live queries push changes to subscribers,
  changefeeds stream table history, and DEFINE EVENT fires triggers on
  writes (ASYNC events run after commit).
- **Compute is separate from storage.** The same SurrealQL and SDKs work
  embedded, single-node, distributed, or managed, so deployment changes
  do not touch queries.
- **Multi-tenant by design.** Namespaces and databases separate tenants,
  access control reaches down to individual fields, and record users let
  a frontend connect directly.

## Version notes

- SurrealDB 3.x is current (v3.2.4 latest). Data tools on the MCP
  servers and per-call namespace arguments need 3.1 or later.
- `surreal mcp` (embedded MCP over stdio) exists since v3.1.0.
  `SURREAL_MCP_ALLOWED_HOSTS` exists since v3.2.1. ISO GQL via MCP is on
  by default from v3.3.0.
- Go client SDK surrealdb.go v1.7.0 supports SurrealDB v2.x and v3.x and
  needs Go 1.23+. surrealdb.c.go v0.1.0 is the embedded engine binding.

## Reference files

- [references/sdks.md](references/sdks.md): all ten official SDKs,
  community SDKs, connection URL schemes, and a detailed Go section
  covering surrealdb.go and the embedded surrealdb.c.go.
- [references/running.md](references/running.md): install paths, surreal
  start, storage engines (RocksDB, SurrealKV, SurrealMX, IndexedDB),
  Docker, single vs multi-node, SurrealDB Cloud plans.
- [references/security.md](references/security.md): the four auth
  methods, RBAC levels, record users, JWT and bearer access, table and
  field permissions, capabilities flags.
- [references/mcp.md](references/mcp.md): hosted mcp.surrealdb.com
  (tools, OAuth, PAT scopes) and the embedded MCP server (surreal mcp,
  POST /mcp, tool list, SURREAL_MCP_* env vars, security checklist).

## Sources

- Docs: https://surrealdb.com/docs (index: /docs/llms.txt)
- Agent setup: https://surrealdb.com/docs/agents
- Official skills: https://github.com/surrealdb/agent-skills (installed
  alongside this one via skills-lock.json)
- Code: https://github.com/surrealdb/surrealdb,
  https://github.com/surrealdb/surrealdb.go,
  https://github.com/surrealdb/surrealdb.c.go
- Hosted MCP: https://mcp.surrealdb.com
- Account portal (tokens): https://account.surrealdb.com/tokens
