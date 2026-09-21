# Running SurrealDB

Ordered from least to most setup. Docs root:
https://surrealdb.com/docs/running/overview

## Ways to run

1. **Studio Sandbox** - browser, no install, non-persistent. For quick
 SurrealQL experiments.
2. **SurrealDB Cloud** - managed instances, free tier, email sign-in.
 Plans: Start (single node, vertical scaling) to Scale (multi-node
 cluster on distributed storage, minimum three compute units, HA).
3. **surreal binary** - single Rust binary, self-hosted. Install:
 https://surrealdb.com/docs/running/installation (macOS, Linux,
 Windows, nightly).
4. **Docker** - `surrealdb/surrealdb` image.

```bash
docker run --rm -p 8000:8000 surrealdb/surrealdb:latest \
  start --user root --pass secret rocksdb://data/database.db
```

## Storage engines

The query layer (SurrealQL, auth, permissions, indexing, transactions)
sits on a pluggable storage layer that owns durability, concurrency and
replication.

| Engine | Shape | Status / use |
| --- | --- | --- |
| RocksDB | LSM-tree KV store on disk | Mature, recommended for write-heavy single-node servers. `rocksdb://path` |
| SurrealKV | SurrealDB's own LSM engine | Beta. Small config surface, aimed at embedded and local-first. `surrealkv://path` |
| SurrealMX | In-memory | `memory`. Tests, scratch, Redis-like persistence mode exists. Supports VERSION temporal reads |
| IndexedDB | Browser | WASM/embedded in web apps |
| Distributed storage | Shared transactional store | Multi-node clusters only. Managed Scale plan or self-hosted Enterprise |

Temporal versioning (SELECT ... VERSION) was built on SurrealKV first
and now works on SurrealMX and RocksDB where supported.

## Single node

```bash
surreal start --user root --pass secret --bind 127.0.0.1:8000 memory
surreal start rocksdb://path/to/database
```

Default endpoint is port 8000. Serves the REST API, RPC, GraphQL, and
(3.1+) `POST /mcp` on the same bind. Vertical scaling only, no built-in
fault tolerance. Use filesystem backups. Prefer RocksDB over SurrealKV
for conservative production.

## Multi-node

Compute nodes are stateless and share one distributed transactional
storage cluster, so nodes join or leave without data redistribution.
Self-hosted multi-node needs SurrealDB Enterprise (Kubernetes operator,
EKS/GKE/AKS runbooks). Managed multi-node is the Scale plan. A single
RocksDB pod is not a cluster.

## SurrealDB Cloud

- Managed backups, resizing without rebuild, HA on Scale, monitoring.
- Object-storage backing rolling out on Scale: transactional data on
 object storage, hot data on local disk.
- Manage via dashboard, surrealctl CLI, or the hosted MCP server
 (see mcp.md).

## surreal binary essentials

```bash
surreal start [opts] <storage>   # run a server: memory | rocksdb:// | surrealkv:// | file://
surreal sql                      # interactive REPL / piped queries
surreal import / export          # .surql backup and restore
surreal isready                  # readiness check
surreal mcp                      # embedded MCP server over stdio (3.1+)
surreal version                  # version check
surreal upgrade                  # upgrade the binary
```

Full CLI detail lives in the surrealdb-cli skill and
https://surrealdb.com/docs/reference/cli

## SurrealKit

Separate schema-management CLI: scaffold projects, sync schema in dev,
plan and roll out production migrations with rollback, generate
JSON/TypeScript types from a live database, TOML test suites. See the
surrealkit skill.
