# SurrealDB MCP servers

Two MCP surfaces publish the same data tools:

- **Hosted** at https://mcp.surrealdb.com - fronts SurrealDB Cloud:
 organisation, instance, billing and Agent Memory management on top of
 the data tools. Nothing to install.
- **Embedded** inside the surreal binary (v3.1+) - for databases you run
 yourself, over stdio (`surreal mcp`) or HTTP (`POST /mcp`).

## Hosted server (SurrealDB Cloud)

Client config shape:

```json
{ "mcpServers": { "surrealdb": { "url": "https://mcp.surrealdb.com" } } }
```

Deviations: VS Code uses `servers` with `"type": "http"`, Windsurf and
Antigravity use `serverUrl`, Zed uses `context_servers`, Devin CLI takes
`devin mcp add surrealdb https://mcp.surrealdb.com` then
`devin mcp login surrealdb`.

### Auth

- OAuth: the client opens a browser, user signs in with their Surreal
 ID. No credentials in the config file. The server acts as the user and
 sees only their organisations and role.
- Personal access token (no-browser or unattended): created at
 https://account.surrealdb.com/tokens, sent as
 `Authorization: Bearer <token>` header. Always include `read:cloud` or
 the token cannot resolve which org/instance to act on.

| PAT scope | Allows |
| --- | --- |
| `read:cloud` | Read orgs, instances, contexts, usage, logs |
| `write:cloud-instances` | Deploy, resize, pause, upgrade, delete instances |
| `query:cloud-instances` | Read/write data inside instances |
| `write:cloud-organization` | Manage orgs, members, roles, invitations |
| `write:cloud-billing` | Billing details and plans |
| `write:cloud-spectron` | Create Agent Memory contexts, manage access |
| `query:spectron-contexts` | Store and recall agent memory |

Tokens do not expire on their own and stand in for the whole account.
Keep them out of committed files.

### Tool groups

Profile and organisations, members and invitations, instances
(list/deploy/pause/resume/resize/upgrade/backups/delete), instance data,
monitoring and usage, billing, catalogue (regions, instance types,
versions as tools and MCP resources), terms, Agent Memory contexts, and
Agent Memory itself (remember, recall, reflect, forget, upload,
inspect). Clients with prompts get a Deploy a Cloud Instance wizard.

### Hard limits (server-side, cannot be overridden)

- Instance deletion requires the exact current name. Backups die with
 the instance.
- No tool accepts card numbers. Checkout is a secure page.
- Terms acceptance is the user's, the server only fetches documents.
- New keys and tokens are shown once at creation.

## Embedded MCP (your own database, v3.1+)

### stdio

```bash
surreal mcp --user root --pass secret --ns main --db main memory
```

The editor spawns SurrealDB as a child process. Every call runs with
owner-level access: no handshake, nothing to narrow. Trusted solo dev
machines only. Prefer SURREAL_USER/SURREAL_PASS env vars over literals
in args. Swap `memory` for `rocksdb://path` to persist.

### HTTP

```bash
surreal start --user root --pass secret --bind 127.0.0.1:8000 memory
# endpoint: http://127.0.0.1:8000/mcp
```

Same auth as the REST API (Bearer JWT, Basic). Put it behind TLS in
production: the session header works as a bearer token until the session
expires (five minutes after last request by default). Non-loopback
Host headers get `403 Forbidden: Host header is not allowed` unless
SURREAL_MCP_ALLOWED_HOSTS lists the hostname or
SURREAL_MCP_ALLOW_ALL_HOSTS=true is set behind a trusted proxy
(both since 3.2.1).

### Published tools

query, gql (default on from 3.3.0, needs --allow-experimental gql on
3.2.x), graphql, select, create, insert, upsert, update, delete, relate,
run, list, use, info. Legacy names (list_tables, use_database, version)
are gone. `gql` is annotated neither read-only nor safe since 3.3.0.
Before that it was marked read-only despite being able to write.

COMMENT text on tables and fields is surfaced through info/list - put
record-ID conventions and graph paths in comments so agents do not
guess.

### Protocol notes (v3.3.0+)

Advertises MCP revision 2026-07-28 (stateless: no initialize handshake,
no sessions. Every tool except `use` takes optional `namespace` and
`database` args). Older handshake revisions still work on the same
endpoint. Scope resolution order: call args, `surreal-ns`/`surreal-db`
headers, session `use`, server defaults.

### SURREAL_MCP_* configuration

| Variable | Default | Effect |
| --- | --- | --- |
| SURREAL_MCP_QUERY_TIMEOUT_SECS | 60 | Per-tool timeout (0 disables) |
| SURREAL_MCP_MAX_RESULT_BYTES | 256 KiB | Cap on serialised output |
| SURREAL_MCP_RUN_MAX_ARGS | 64 | Max args to `run` |
| SURREAL_MCP_PARAMS_MAX_KEYS | 256 | Max top-level param keys |
| SURREAL_MCP_PARAMS_MAX_QL_BYTES | 4 KiB | Max `$ql` string length in *_data payloads |
| SURREAL_MCP_SCHEMA_RESOURCE_MAX_TABLES | 200 | Tables enriched in schema resource |
| SURREAL_MCP_ALLOWED_HOSTS | loopback only | Host allowlist for HTTP /mcp |
| SURREAL_MCP_ALLOW_ALL_HOSTS | false | Accept any Host (trusted proxy only) |
| SURREAL_HTTP_MAX_MCP_BODY_SIZE | 4 MiB | HTTP body cap for /mcp |
| SURREAL_MCP_NS / SURREAL_MCP_DB | - | Default ns/db for `surreal mcp` |

Metrics: `surrealdb.mcp.*` counters and histograms since 3.1.0. Audit:
forward the `surrealdb::mcp::audit` tracing target to a SIEM. Records
carry tool, subject, ns, db, outcome, never query text or payloads.

### Security checklist for embedded MCP

- Least-privilege DEFINE USER over root credentials for agent clients.
- Deny high-risk functions (`--deny-funcs`, `--allow-net`).
- Explicit `--allow-origin`, no `*` in production.
- stdio mode is owner-level on every call. Keep it local and trusted.
