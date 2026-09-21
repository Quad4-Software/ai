---
name: surrealdb-vector
description: "Vector search with SurrealDB using HNSW indexes, KNN queries, and similarity scoring. Use when creating vector indexes, querying vectors with KNN distance operators, building semantic search or RAG pipelines, tuning HNSW parameters (EFC, M, M0, distance function, type), or implementing recommendation systems with SurrealDB. Triggers: HNSW, vector, embedding, KNN, cosine, euclidean, semantic search, RAG, vector::distance."
metadata:
  author: surrealdb
  version: "0.1.0"
---

# SurrealDB Vector Search

## HNSW Index

Create a basic HNSW index:

```surql
DEFINE INDEX hnsw_idx ON pts FIELDS point HNSW DIMENSION 4;
```

With specific distance function and type:

```surql
DEFINE INDEX hnsw_idx ON pts FIELDS point HNSW DIMENSION 4 DIST EUCLIDEAN TYPE F64;
```

Available types: `F64`, `F32`, `I64`, `I32`, `I16`.

### Full Table Example

```surql
DEFINE TABLE OVERWRITE document SCHEMALESS;
DEFINE FIELD OVERWRITE embedding ON document TYPE array<float>;
DEFINE INDEX OVERWRITE hnsw_idx_document ON document
    FIELDS embedding
    HNSW DIMENSION 384
    DIST COSINE
    TYPE F32
    EFC 150 M 12 M0 24;
```

### HNSW Parameters

| Parameter | Description |
| --------- | ------------------------------------------------ |
| DIMENSION | Vector dimensionality (must match your embeddings)|
| DIST | Distance function: `COSINE`, `EUCLIDEAN`, etc. |
| TYPE | Numeric type: `F64`, `F32`, `I64`, `I32`, `I16` |
| EFC | Construction search effort (higher = better index)|
| M | Max connections per node |
| M0 | Max connections at layer 0 |

## Querying Vectors

The `<|K, EF|>` operator performs KNN search. `K` is the number of results,
`EF` is the search effort (higher = more accurate, slower).

Recommended effort values:
- `40` - default, good accuracy
- `17` - fast but may miss some results

### Basic KNN Query

```surql
SELECT
    *,
    vector::distance::knn() AS dist
FROM document
WHERE embedding <|10, 40|> $vector;
```

`vector::distance::knn()` uses the distance function defined by the index.

### Scored Results with Threshold

```surql
SELECT *, score
FROM (
    SELECT *, (1 - vector::distance::knn()) AS score
    FROM document
    WHERE embedding <|20, 40|> $vector
)
WHERE score >= $threshold
ORDER BY score DESC;
```

## Practical notes

- DIMENSION must equal the embedding model's output size. A mismatch
  fails at index build or query time, not at DEFINE time.
- Distance functions: COSINE for most embedding models (OpenAI,
  sentence-transformers), EUCLIDEAN for spatial data, MANHATTAN and
  MINKOWSKI also exist. Match what the model was trained or
  normalized for.
- EFC and M trade build time and memory for recall. Start with the
  defaults, raise EFC if recall misses, raise M/M0 for high-dimension
  vectors.
- EF at query time is the recall knob: raise it when KNN misses
  obvious neighbors before you touch index params.
- The KNN operator only uses the index when it appears in a WHERE
  clause against an indexed field. Verify with EXPLAIN.
- Combine with a scalar WHERE (e.g. `AND category = $cat`) to get
  filtered KNN. Confirm the index actually engages under EXPLAIN for
  your SurrealDB version.
- RAG pattern: store chunk text plus embedding in one record, KNN for
  retrieval, graph edges for source linkage. See `surrealql` for the
  RELATE syntax and `surrealql-performance` for index tuning.

## Cross-links

- `surrealql`: statement syntax for queries and DEFINE.
- `surrealql-functions`: `vector::` function signatures via the LSP.
- `surrealql-performance`: EXPLAIN and index strategy.
- `surrealdb-python` / `surrealdb-js`: SDK-side embedding pipelines.
