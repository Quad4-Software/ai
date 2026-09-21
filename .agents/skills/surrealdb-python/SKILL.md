---
name: surrealdb-python
description: "Using SurrealDB with the Python SDK, covering both client/server mode (WebSocket) and embedded mode (in-memory or file-based). Use when connecting to SurrealDB from Python, using the surrealdb Python package, running SurrealDB embedded without a server, or performing CRUD operations from Python code. Triggers: surrealdb Python, Surreal(), AsyncSurreal(), Python SDK, embedded SurrealDB, mem://, file://."
metadata:
  author: surrealdb
  version: "0.1.0"
---

# SurrealDB Python SDK

## Running SurrealDB (Server Mode)

Persist data with RocksDB:

```bash
surreal start -u root -p root rocksdb:database
```

In-memory:

```bash
surreal start -u root -p root
```

## Client/Server Mode

Connect via WebSocket using the `Surreal` context manager:

```python
from surrealdb import Surreal

with Surreal("ws://localhost:8000/rpc") as db:
    db.signin({"username": "root", "password": "root"})
    db.use("namespace_test", "database_test")

    db.create(
        "person",
        {
            "user": "me",
            "password": "safe",
            "marketing": True,
            "tags": ["python", "documentation"],
        },
    )

    print(db.select("person"))

    print(db.update("person", {
        "user": "you",
        "password": "very_safe",
        "marketing": False,
        "tags": ["Awesome"],
    }))

    print(db.delete("person"))

    db.query("""
    insert into person {
        user: 'me',
        password: 'very_safe',
        tags: ['python', 'documentation']
    };
    """)

    print(db.query("select * from person"))

    print(db.query("""
    update person content {
        user: 'you',
        password: 'more_safe',
        tags: ['awesome']
    };
    """))

    print(db.query("delete person"))
```

## Embedded Mode

Run SurrealDB directly inside your Python process - no server required.

- **In-memory**: `Surreal("mem://")` / `AsyncSurreal("mem://")`
- **File-based persistence**: `Surreal(f"file://{db_path}")` / `AsyncSurreal(f"file://{db_path}")`

See [references/embedded.md](references/embedded.md) for a complete async
example with file-based persistence.

## Async variant

`AsyncSurreal` mirrors `Surreal` method-for-method and is the right pick
under asyncio or ASGI frameworks (FastAPI, Litestar). The sync class
wraps a blocking runtime and is fine for scripts and notebooks.

```python
from surrealdb import AsyncSurreal

async with AsyncSurreal("ws://localhost:8000/rpc") as db:
    await db.signin({"username": "root", "password": "root"})
    await db.use("ns", "db")
    result = await db.query("SELECT * FROM person WHERE age > $min",
                            {"min": 30})
```

## Method map

| method | purpose |
|---|---|
| `signin`/`signup` | root, namespace, database, record, or scope auth |
| `use` | select namespace and database |
| `authenticate` | resume a session from a token |
| `invalidate` | end the session |
| `query` | raw SurrealQL. Always prefer `$var` params over f-strings |
| `select`/`create`/`insert` | reads and creates by table or record id |
| `update`/`merge`/`patch` | content replace, partial merge, JSON Patch |
| `upsert` | create-or-update by id |
| `delete` | delete table or record |
| `live`/`subscribe_live` | live query streams |
| `let`/`unset` | session parameters |

## Safety and testing

- Parameterize every query with `$var` bindings. Never f-string user
  input into SurrealQL.
- `mem://` embedded mode is the standard test fixture: no server, no
  cleanup, deterministic. File mode under `tempfile` covers
  persistence tests.
- Signin credentials belong in env vars or your secret store, not in
  source. See `surrealdb` (references/security.md) for the auth model
  and PERMISSIONS.

## Cross-links

- `surrealdb` hub: connection URLs, storage engines, deployment.
- `surrealql`: statement syntax for `query` calls.
- `surrealdb-vector`: HNSW indexes and KNN queries for embeddings.
- `surrealkit`: schema migration and typing workflow.
