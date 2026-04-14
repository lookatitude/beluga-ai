---
title: "Memory Stores API — Redis, Postgres, Neo4j"
description: "Memory store providers API for Beluga AI. MessageStore and GraphStore backends: Redis, PostgreSQL, SQLite, MongoDB, Neo4j, Memgraph, Dragonfly."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "memory stores API, MessageStore, GraphStore, Redis, PostgreSQL, Neo4j, MongoDB, SQLite, Beluga AI, Go, reference"
---

## dragonfly

```go
import "github.com/lookatitude/beluga-ai/memory/stores/dragonfly"
```

Package dragonfly provides a DragonflyDB-backed implementation of [memory.MessageStore].
DragonflyDB is fully Redis-compatible, so this implementation uses the same
go-redis client library and sorted set storage approach as the Redis store.

## Usage

```go
import "github.com/lookatitude/beluga-ai/memory/stores/dragonfly"

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
store, err := dragonfly.New(dragonfly.Config{
    Client: client,
    Key:    "beluga:dragonfly:messages", // optional, this is the default
})
if err != nil {
    log.Fatal(err)
}

err = store.Append(ctx, msg)
results, err := store.Search(ctx, "query", 10)
all, err := store.All(ctx)
```

## DragonflyDB vs Redis

DragonflyDB is a modern in-memory data store that provides a Redis-compatible
API with higher throughput and lower memory usage. This store is functionally
identical to the Redis store but uses a distinct default key prefix
("beluga:dragonfly:messages") to avoid collisions in mixed deployments.

Messages are stored as JSON in a sorted set (ZSET) with a monotonically
increasing sequence number as the score to preserve insertion order.

This implementation requires github.com/redis/go-redis/v9.

---

## inmemory

```go
import "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
```

Package inmemory provides in-memory implementations of the memory store
interfaces. These implementations are suitable for development, testing,
and short-lived agent sessions. Data is not persisted across process
restarts.

## MessageStore

`MessageStore` implements [memory.MessageStore] with a thread-safe slice.
Messages are searched via case-insensitive substring matching on text
content parts:

```go
store := inmemory.NewMessageStore()
err := store.Append(ctx, msg)
results, err := store.Search(ctx, "hello", 10)
all, err := store.All(ctx)
```

## GraphStore

`GraphStore` implements [memory.GraphStore] with in-memory maps for entities
and relations. It supports basic type-based queries ("type:person") and
breadth-first neighbor traversal:

```go
graph := inmemory.NewGraphStore()
err := graph.AddEntity(ctx, memory.Entity{ID: "alice", Type: "person"})
err = graph.AddRelation(ctx, "alice", "bob", "knows", nil)
entities, relations, err := graph.Neighbors(ctx, "alice", 2)
```

For full Cypher query support, use the neo4j or memgraph store providers.

Both stores are safe for concurrent use.

---

## memgraph

```go
import "github.com/lookatitude/beluga-ai/memory/stores/memgraph"
```

Package memgraph provides a Memgraph-backed [memory.GraphStore] implementation for the
Beluga AI memory system. Memgraph uses the Bolt protocol (same as Neo4j),
so this implementation uses the Neo4j Go driver with Cypher queries.

## Usage

```go
import "github.com/lookatitude/beluga-ai/memory/stores/memgraph"

store, err := memgraph.New(memgraph.Config{
    URI:      "bolt://localhost:7687",
    Username: "",
    Password: "",
})
if err != nil {
    log.Fatal(err)
}
defer store.Close(ctx)

err = store.AddEntity(ctx, memory.Entity{
    ID:   "alice",
    Type: "person",
    Properties: map[string]any{"age": 30},
})
err = store.AddRelation(ctx, "alice", "bob", "knows", nil)
results, err := store.Query(ctx, "MATCH (n:Entity) RETURN n")
entities, relations, err := store.Neighbors(ctx, "alice", 2)
```

## Memgraph vs Neo4j

Memgraph is an in-memory graph database optimized for real-time analytics
and streaming workloads. It is Bolt-compatible with Neo4j and uses the
same Cypher query language. This store uses the same graph model as the
neo4j store: entities as "Entity" nodes and relations as "RELATION" edges.

## Testability

The store uses an internal sessionRunner interface that abstracts session
operations, enabling mock-based testing without a live database.

This implementation requires github.com/neo4j/neo4j-go-driver/v5.

---

## mongodb

```go
import "github.com/lookatitude/beluga-ai/memory/stores/mongodb"
```

Package mongodb provides a MongoDB-backed implementation of [memory.MessageStore].
Messages are stored as BSON documents in a MongoDB collection with a
monotonically increasing sequence field for chronological ordering.

## Usage

```go
import "github.com/lookatitude/beluga-ai/memory/stores/mongodb"

store, err := mongodb.New(mongodb.Config{
    Collection: client.Database("beluga").Collection("messages"),
})
if err != nil {
    log.Fatal(err)
}

err = store.Append(ctx, msg)
results, err := store.Search(ctx, "query", 10)
all, err := store.All(ctx)
```

## Collection Interface

The store accepts any value satisfying the `Collection` interface, which
is implemented by *mongo.Collection and can be mocked for testing. The
interface requires InsertOne, Find, and DeleteMany methods.

## Storage Format

Each message is stored as a BSON document with fields for sequence number,
role, parts, metadata, tool calls, and timestamp. Documents are sorted by
the sequence field for chronological retrieval.

---

## neo4j

```go
import "github.com/lookatitude/beluga-ai/memory/stores/neo4j"
```

Package neo4j provides a Neo4j-backed [memory.GraphStore] implementation for the
Beluga AI memory system. It uses Cypher queries for graph operations and
supports the full [memory.GraphStore] interface including entity management,
relationship creation, querying, and neighbor traversal.

## Usage

```go
import "github.com/lookatitude/beluga-ai/memory/stores/neo4j"

store, err := neo4j.New(neo4j.Config{
    URI:      "neo4j://localhost:7687",
    Username: "neo4j",
    Password: "password",
    Database: "",  // empty for default database
})
if err != nil {
    log.Fatal(err)
}
defer store.Close(ctx)

err = store.AddEntity(ctx, memory.Entity{
    ID:   "alice",
    Type: "person",
    Properties: map[string]any{"age": 30},
})
err = store.AddRelation(ctx, "alice", "bob", "knows", nil)
results, err := store.Query(ctx, "MATCH (n:Entity) RETURN n")
entities, relations, err := store.Neighbors(ctx, "alice", 2)
```

## Graph Model

Entities are stored as Neo4j nodes with the label "Entity" and properties
set from the entity's Properties map. The entity ID and type are stored
as node properties. Relations are stored as "RELATION" edges with a "type"
property.

## Testability

The store uses an internal sessionRunner interface that abstracts Neo4j
session operations, enabling mock-based testing without a live database.

This implementation requires github.com/neo4j/neo4j-go-driver/v5.

---

## postgres

```go
import "github.com/lookatitude/beluga-ai/memory/stores/postgres"
```

Package postgres provides a PostgreSQL-backed implementation of
[memory.MessageStore]. Messages are stored in a table with columns for role,
content (JSONB), metadata (JSONB), and created_at timestamp. Search uses
case-insensitive ILIKE queries on the content column.

This implementation uses github.com/jackc/pgx/v5 as the PostgreSQL driver.

## Usage

```go
import "github.com/lookatitude/beluga-ai/memory/stores/postgres"

store, err := postgres.New(postgres.Config{
    DB:    pgxConn,       // *pgx.Conn, pgxpool.Pool, or pgxmock
    Table: "messages",    // optional, this is the default
})
if err != nil {
    log.Fatal(err)
}
err = store.EnsureTable(ctx) // auto-create table if needed
```

## DBTX Interface

The store accepts any value satisfying the `DBTX` interface, which is
implemented by pgx.Conn, pgxpool.Pool, and pgxmock for testing.

## Schema

The auto-created table has the following columns:

- id: SERIAL PRIMARY KEY
- role: TEXT NOT NULL
- content: JSONB NOT NULL
- metadata: JSONB
- created_at: TIMESTAMPTZ NOT NULL DEFAULT NOW()

Use `MessageStore.EnsureTable` to create it automatically.

---

## redis

```go
import "github.com/lookatitude/beluga-ai/memory/stores/redis"
```

Package redis provides a Redis-backed implementation of [memory.MessageStore].
Messages are stored as JSON in a Redis sorted set, scored by a monotonically
increasing sequence number to preserve insertion order. Search uses
case-insensitive substring matching on text content parts.

This implementation requires a Redis server (v5.0+) and uses
github.com/redis/go-redis/v9 as the client library.

## Usage

```go
import "github.com/lookatitude/beluga-ai/memory/stores/redis"

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
store, err := redis.New(redis.Config{
    Client: client,
    Key:    "beluga:messages", // optional, this is the default
})
if err != nil {
    log.Fatal(err)
}

err = store.Append(ctx, msg)
results, err := store.Search(ctx, "query", 10)
all, err := store.All(ctx)
```

The sorted set key defaults to "beluga:messages" and can be overridden
via `Config`.Key to support multi-tenant or multi-agent deployments.

---

## sqlite

```go
import "github.com/lookatitude/beluga-ai/memory/stores/sqlite"
```

Package sqlite provides a SQLite-backed implementation of [memory.MessageStore].
Messages are stored in a table with columns for role, content (JSON), metadata
(JSON), and created_at timestamp. This uses the pure-Go modernc.org/sqlite
driver (no CGO required).

## Usage

```go
import (
    "database/sql"
    _ "modernc.org/sqlite"
    "github.com/lookatitude/beluga-ai/memory/stores/sqlite"
)

db, err := sql.Open("sqlite", ":memory:")
if err != nil {
    log.Fatal(err)
}
store, err := sqlite.New(sqlite.Config{DB: db})
if err != nil {
    log.Fatal(err)
}
err = store.EnsureTable(ctx) // auto-create table if needed
```

## Schema

The auto-created table has the following columns:

- id: INTEGER PRIMARY KEY AUTOINCREMENT
- role: TEXT NOT NULL
- content: TEXT NOT NULL (JSON)
- metadata: TEXT (JSON)
- created_at: TEXT NOT NULL DEFAULT datetime('now')

Use `MessageStore.EnsureTable` to create it automatically.
