---
title: MongoDB Context Persistence
description: "Use MongoDB as a persistent store for Beluga AI memory, enabling long-term conversation history and multi-session context in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "MongoDB, Beluga AI, context persistence, conversation history, memory store, MongoDB Atlas, Go AI framework"
---

In-memory conversation history is lost when a process restarts, which means agents lose context between deployments, scaling events, or crashes. MongoDB persistence solves this by storing conversation messages durably, enabling agents to maintain long-term context across sessions and process lifecycles.

Choose MongoDB for memory persistence when your organization already operates MongoDB infrastructure, when you need flexible schema for varied message metadata, or when you want managed cloud persistence via MongoDB Atlas.

## Overview

The `mongodb` store implements the `memory.MessageStore` interface, storing each message as an individual BSON document with a monotonically increasing sequence number. This design supports:

- Persistent conversation history across process restarts
- Multi-session context management via separate collections or key prefixes
- Scalable storage for high-volume applications
- Integration with MongoDB Atlas for managed cloud deployments

## Prerequisites

- Go 1.23 or later
- A Beluga AI project initialized with `go mod init`
- A MongoDB instance (local, Docker, or MongoDB Atlas)
- The MongoDB Go driver v2

## Installation

Install the MongoDB memory store:

```bash
go get github.com/lookatitude/beluga-ai/memory/stores/mongodb
```

Start a local MongoDB instance if you do not already have one:

```bash
# Using Docker
docker run -d --name mongodb -p 27017:27017 mongo:7

# Or install and run directly
mongod --dbpath /data/db
```

## Configuration

### Basic Setup

Connect to MongoDB and create a message store:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/memory/stores/mongodb"
	"github.com/lookatitude/beluga-ai/schema"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	ctx := context.Background()

	// Connect to MongoDB.
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Disconnect error: %v", err)
		}
	}()

	// Create the message store backed by a MongoDB collection.
	store, err := mongodb.New(mongodb.Config{
		Collection: client.Database("beluga_ai").Collection("messages"),
	})
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Append messages to the store.
	err = store.Append(ctx, schema.NewHumanMessage("Hello, how are you?"))
	if err != nil {
		log.Fatalf("Append failed: %v", err)
	}

	err = store.Append(ctx, schema.NewAIMessage("I'm doing well, thank you!"))
	if err != nil {
		log.Fatalf("Append failed: %v", err)
	}

	// Retrieve recent messages.
	msgs, err := store.Last(ctx, 10)
	if err != nil {
		log.Fatalf("Last failed: %v", err)
	}

	fmt.Printf("Retrieved %d messages\n", len(msgs))
}
```

### Using with CompositeMemory

The MongoDB store plugs into Beluga's `CompositeMemory` as the recall tier for searchable conversation history:

```go
import (
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/memory/stores/mongodb"
)

// Create a MongoDB-backed recall store.
recallStore, err := mongodb.New(mongodb.Config{
	Collection: client.Database("beluga_ai").Collection("recall"),
})
if err != nil {
	log.Fatalf("Failed to create recall store: %v", err)
}

// Build a CompositeMemory with MongoDB as the recall tier.
mem := memory.NewComposite(memory.CompositeConfig{
	RecallStore: recallStore,
})

// Save a conversation turn.
err = mem.Save(ctx,
	schema.NewHumanMessage("What is the weather today?"),
	schema.NewAIMessage("I don't have access to weather data."),
)
if err != nil {
	log.Fatalf("Save failed: %v", err)
}

// Load relevant context.
msgs, err := mem.Load(ctx, "weather")
if err != nil {
	log.Fatalf("Load failed: %v", err)
}
```

## Usage

### Searching Messages

The MongoDB store supports text-based search across stored messages:

```go
results, err := store.Search(ctx, "weather", 5)
if err != nil {
	log.Fatalf("Search failed: %v", err)
}

for _, msg := range results {
	fmt.Printf("[%s] %s\n", msg.GetRole(), extractText(msg))
}
```

### Clearing History

Remove all messages from the store:

```go
err = store.Clear(ctx)
if err != nil {
	log.Fatalf("Clear failed: %v", err)
}
```

### Session Isolation

Use separate MongoDB collections for different conversation sessions:

```go
func storeForSession(client *mongo.Client, sessionID string) (*mongodb.MessageStore, error) {
	coll := client.Database("beluga_ai").Collection("session_" + sessionID)
	return mongodb.New(mongodb.Config{
		Collection: coll,
	})
}
```

## Advanced Topics

### Indexing

Create indexes on the MongoDB collection to optimize query performance:

```go
import "go.mongodb.org/mongo-driver/v2/bson"

coll := client.Database("beluga_ai").Collection("messages")

// Index on sequence number for chronological ordering.
_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
	Keys: bson.D{{Key: "seq", Value: 1}},
})
if err != nil {
	log.Printf("Index creation warning: %v", err)
}

// Index on timestamp for time-range queries.
_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
	Keys: bson.D{{Key: "timestamp", Value: -1}},
})
if err != nil {
	log.Printf("Index creation warning: %v", err)
}
```

### TTL for Automatic Cleanup

Configure MongoDB TTL indexes to automatically remove old messages:

```go
_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
	Keys: bson.D{{Key: "timestamp", Value: 1}},
	Options: options.Index().SetExpireAfterSeconds(86400 * 30), // 30 days
})
```

### Observability

Add OpenTelemetry tracing to monitor MongoDB operations:

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("beluga.memory.mongodb")

ctx, span := tracer.Start(ctx, "mongodb.append",
	trace.WithAttributes(attribute.String("session_id", sessionID)),
)
defer span.End()

err := store.Append(ctx, msg)
if err != nil {
	span.RecordError(err)
	return err
}
```

### Connection Pooling

Configure the MongoDB client for production connection pooling:

```go
clientOpts := options.Client().
	ApplyURI("mongodb://localhost:27017").
	SetMaxPoolSize(50).
	SetMinPoolSize(5).
	SetMaxConnIdleTime(30 * time.Second)

client, err := mongo.Connect(clientOpts)
```

## Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Collection` | MongoDB collection to store messages in | -- | Yes |

Connection-level settings (URI, pool size, timeouts) are configured on the `mongo.Client` before passing the collection to the store.

## Troubleshooting

### "mongodb: collection is required"

The `Config.Collection` field is `nil`. Ensure you pass a valid `*mongo.Collection` (or any type implementing the `mongodb.Collection` interface) when creating the store.

### "connection refused"

MongoDB is not running or the URI is incorrect. Verify:

```bash
# Test connectivity
mongosh "mongodb://localhost:27017"
```

### Slow Queries

If search operations are slow on large collections, ensure the appropriate indexes exist. The store uses sequence-ordered inserts, so a compound index on `(seq, role)` can improve filtered retrieval.

## Related Resources

- [Redis Distributed Locking](/docs/integrations/redis-locking) -- Distributed coordination for memory operations
- [Memory System Guide](/docs/guides/memory) -- Full memory architecture documentation
- [LLM Providers Overview](/docs/integrations/llm-providers) -- Connecting LLM models
