# SurrealDB SDKs

Ten official SDKs, one per language. All speak the same RPC protocol
over WebSocket or HTTP, so the flow is identical everywhere: connect,
Use() a namespace and database, sign in, then query. What differs is
idiom: types, async model and error handling follow the host language.

Docs root: https://surrealdb.com/docs/languages

## Official SDKs

| Language | Package / repo | Notes |
| --- | --- | --- |
| Rust | `surrealdb` crate, github.com/surrealdb/surrealdb | The engine itself. Full embedded support (memory, SurrealKV, RocksDB, TiKV backends as features) |
| JavaScript / TypeScript | `surrealdb` on npm | Node, Deno, Bun, browser. Embedded engines via @surrealdb/node and WASM/IndexedDB in the browser. See the surrealdb-js skill |
| Python | `surrealdb` on PyPI | Client/server over WebSocket and embedded (mem://, file://). See the surrealdb-python skill |
| Go | `github.com/surrealdb/surrealdb.go` | Wire client, v1.7.0, Go 1.23+, SurrealDB v2.x/v3.x. Embedded via `github.com/surrealdb/surrealdb.c.go` (v0.1.0, cgo binding) |
| .NET | `SurrealDb.Net` on NuGet | Client plus embedded support |
| Java | `surrealdb` on Maven Central | Client plus embedded via JNI |
| Kotlin | surrealdb Kotlin SDK | JVM and Android targets |
| PHP | `surrealdb/surrealdb.php` on Packagist | Client SDK |
| Swift | `surrealdb` Swift package | Client plus embedded for Apple platforms |
| Mojo | surrealdb Mojo package | Client SDK |

Community SDKs cover more languages:
https://surrealdb.com/docs/languages/community

Framework guides exist for Expo and React Native.

## Connection URL schemes

The scheme on the endpoint picks the transport:

| Scheme | Transport | Use |
| --- | --- | --- |
| `ws://`, `wss://` | WebSocket RPC | Stateful, required for live queries, sessions, transactions |
| `http://`, `https://` | HTTP RPC | Stateless, one connection per request |
| `mem://` | Embedded in-memory | No server process. Data dies with the connection |
| `surrealkv://<path>` | Embedded SurrealKV | File persistence, embedded builds |
| `rocksdb://<path>` | Embedded RocksDB | File persistence. Needs the engine built with RocksDB |

## Go SDK: surrealdb.go (client)

```bash
go get github.com/surrealdb/surrealdb.go
```

```go
import (
    "context"

    surrealdb "github.com/surrealdb/surrealdb.go"
    "github.com/surrealdb/surrealdb.go/pkg/models"
)

ctx := context.Background()
db, err := surrealdb.FromEndpointURLString(ctx, "ws://localhost:8000")
defer db.Close(ctx)

db.Use(ctx, "namespace", "database")
db.SignIn(ctx, surrealdb.Auth{Username: "root", Password: "secret"})
```

Key shape: top-level generic functions, not methods on db. The type
parameter says what to unmarshal into.

```go
users, err := surrealdb.Select[[]User](ctx, db, models.Table("users"))
one, err   := surrealdb.Select[User](ctx, db, models.NewRecordID("users", "alice"))
u, err     := surrealdb.Create[User](ctx, db, models.Table("users"), User{Name: "A"})
res, err   := surrealdb.Query[[]User](ctx, db,
    "SELECT * FROM users WHERE age < $max",
    map[string]any{"max": 30})
// res is *[]QueryResult[[]User] - iterate res for per-statement results
```

- `models.Table` generates a random record ID, `models.NewRecordID`
 addresses a specific record. Structs serialise with `json` tags.
- ws:// is required for live queries and multi-statement transactions.
- API reference: https://surrealdb.com/docs/reference/golang

## Go embedded: surrealdb.c.go

`surrealdb.c.go` wraps the engine through a C binding (cgo, needs a C
toolchain). Open an in-process database:

```go
import surrealdb "github.com/surrealdb/surrealdb.c.go"

db, err := surrealdb.Open(ctx, "mem://")        // or surrealkv://path
defer db.Close()
db.Use(ctx, "main", "main")
surrealdb.Query[Person](ctx, db, "SELECT * FROM person", nil)
```

- Types use `cbor` tags, e.g. `surrealdb.RecordID[string]` with
 `cbor:"id,omitempty"`.
- `rocksdb://` requires a manual build of the binding with RocksDB:
 https://github.com/surrealdb/surrealdb.c.go/blob/main/docs/rocksdb.md
- Repo note for this codebase: it is an external dependency with cgo,
 so it conflicts with the stdlib-first and offline-build conventions in
 AGENTS.md. Vendor or justify before adding it to any mcp/* module.

## Choosing an interface instead of an SDK

- RPC over WebSocket is what SDKs use and the only interface with live
 queries.
- HTTP REST suits environments that cannot hold a socket open.
- The CLI covers import, export and one-off queries (surrealdb-cli
 skill).
