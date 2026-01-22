# MongoDB Context Persistence

Welcome, colleague! In this integration guide, we're going to integrate MongoDB for persisting conversation context with Beluga AI's memory package. MongoDB provides scalable, persistent storage for conversation history and context.

## What you will build

You will configure Beluga AI to use MongoDB for persisting conversation context, enabling long-term memory storage, multi-session conversations, and scalable context management.

## Learning Objectives

- ✅ Configure MongoDB with Beluga AI memory
- ✅ Persist conversation context in MongoDB
- ✅ Retrieve conversation history
- ✅ Understand MongoDB schema design

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- MongoDB instance (local or cloud)
- MongoDB Go driver

## Step 1: Setup and Installation

Install MongoDB Go driver:
bash
```bash
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/mongo/options
```

Start MongoDB (local):
mongod
```

Or use MongoDB Atlas (cloud).

## Step 2: Create MongoDB Memory Store

Create a MongoDB-backed memory implementation:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/memory/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBMemory struct {
    client     *mongo.Client
    database   *mongo.Database
    collection *mongo.Collection
    sessionID  string
}

type ConversationDocument struct {
    SessionID  string          `bson:"session_id"`
    Messages   []schema.Message `bson:"messages"`
    CreatedAt  time.Time       `bson:"created_at"`
    UpdatedAt  time.Time       `bson:"updated_at"`
    Metadata   map[string]any  `bson:"metadata,omitempty"`
}

func NewMongoDBMemory(uri, databaseName, sessionID string) (*MongoDBMemory, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    
    db := client.Database(databaseName)
    collection := db.Collection("conversations")
    
    // Create index
    indexModel := mongo.IndexModel{
        Keys: bson.D{{Key: "session_id", Value: 1}},
        Options: options.Index().SetUnique(true),
    }
    collection.Indexes().CreateOne(ctx, indexModel)
    
    return &MongoDBMemory{
        client:     client,
        database:   db,
        collection: collection,
        sessionID:  sessionID,
    }, nil
}
```

## Step 3: Implement Memory Interface

Implement Beluga AI memory interface:
```go
func (m *MongoDBMemory) LoadMemoryVariables(ctx context.Context, _ map[string]any) (map[string]any, error) {
    filter := bson.M{"session_id": m.sessionID}
    
    var doc ConversationDocument
    err := m.collection.FindOne(ctx, filter).Decode(&doc)
    if err == mongo.ErrNoDocuments {
        return map[string]any{"history": []schema.Message{}}, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to load: %w", err)
    }
    
    return map[string]any{"history": doc.Messages}, nil
}

func (m *MongoDBMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
    // Get existing messages
    existing, err := m.LoadMemoryVariables(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to load existing: %w", err)
    }
    
    messages, _ := existing["history"].([]schema.Message)
    
    // Add new messages
    if input, ok := inputs["input"].(string); ok {
        messages = append(messages, schema.NewHumanMessage(input))
    }
    if output, ok := outputs["output"].(string); ok {
        messages = append(messages, schema.NewAIMessage(output))
    }
    
    // Upsert document
    filter := bson.M{"session_id": m.sessionID}
    update := bson.M{
        "$set": bson.M{
            "messages":  messages,
            "updated_at": time.Now(),
        },
        "$setOnInsert": bson.M{
            "created_at": time.Now(),
        },
    }
    
    opts := options.Update().SetUpsert(true)
    _, err = m.collection.UpdateOne(ctx, filter, update, opts)
    if err != nil {
        return fmt.Errorf("failed to save: %w", err)
    }
    
    return nil
}

func (m *MongoDBMemory) Clear(ctx context.Context) error {
    filter := bson.M{"session_id": m.sessionID}
    _, err := m.collection.DeleteOne(ctx, filter)
    return err
}

func (m *MongoDBMemory) MemoryVariables() []string {
    return []string{"history"}
}
```

## Step 4: Use with Beluga AI

Use MongoDB memory in your application:
```go
func main() {
    ctx := context.Background()
    
    // Create MongoDB memory
    memory, err := NewMongoDBMemory(
        "mongodb://localhost:27017",
        "beluga_ai",
        "session-123",
    )
    if err != nil {
        log.Fatalf("Failed to create memory: %v", err)
    }
    defer memory.client.Disconnect(ctx)
    
    // Load existing context
    vars, err := memory.LoadMemoryVariables(ctx, nil)
    if err != nil {
        log.Fatalf("Failed to load: %v", err)
    }

    

    fmt.Printf("Loaded %d messages\n", len(vars["history"].([]schema.Message)))
    
    // Save new context
    err = memory.SaveContext(ctx,
        map[string]any{"input": "Hello"},
        map[string]any{"output": "Hi there!"},
    )
    if err != nil {
        log.Fatalf("Failed to save: %v", err)
    }
    
    fmt.Println("Context saved successfully")
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/memory/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionMongoDBMemory struct {
    client     *mongo.Client
    collection *mongo.Collection
    sessionID  string
    tracer     trace.Tracer
}

func NewProductionMongoDBMemory(uri, databaseName, sessionID string) (*ProductionMongoDBMemory, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    
    db := client.Database(databaseName)
    collection := db.Collection("conversations")
    
    // Create indexes
    indexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "session_id", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{{Key: "updated_at", Value: -1}},
        },
    }
    collection.Indexes().CreateMany(ctx, indexes)
    
    return &ProductionMongoDBMemory{
        client:     client,
        collection: collection,
        sessionID:  sessionID,
        tracer:     otel.Tracer("beluga.memory.mongodb"),
    }, nil
}

func (m *ProductionMongoDBMemory) LoadMemoryVariables(ctx context.Context, _ map[string]any) (map[string]any, error) {
    ctx, span := m.tracer.Start(ctx, "mongodb.load",
        trace.WithAttributes(attribute.String("session_id", m.sessionID)),
    )
    defer span.End()
    
    filter := bson.M{"session_id": m.sessionID}
    var doc ConversationDocument
    
    err := m.collection.FindOne(ctx, filter).Decode(&doc)
    if err == mongo.ErrNoDocuments {
        span.SetAttributes(attribute.Bool("found", false))
        return map[string]any{"history": []schema.Message{}}, nil
    }
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to load: %w", err)
    }
    
    span.SetAttributes(
        attribute.Bool("found", true),
        attribute.Int("message_count", len(doc.Messages)),
    )
    
    return map[string]any{"history": doc.Messages}, nil
}

func (m *ProductionMongoDBMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
    ctx, span := m.tracer.Start(ctx, "mongodb.save",
        trace.WithAttributes(attribute.String("session_id", m.sessionID)),
    )
    defer span.End()
    
    // Load and append
    existing, err := m.LoadMemoryVariables(ctx, nil)
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    messages, _ := existing["history"].([]schema.Message)
    
    if input, ok := inputs["input"].(string); ok {
        messages = append(messages, schema.NewHumanMessage(input))
    }
    if output, ok := outputs["output"].(string); ok {
        messages = append(messages, schema.NewAIMessage(output))
    }
    
    filter := bson.M{"session_id": m.sessionID}
    update := bson.M{
        "$set": bson.M{
            "messages":  messages,
            "updated_at": time.Now(),
        },
        "$setOnInsert": bson.M{
            "created_at": time.Now(),
        },
    }
    
    opts := options.Update().SetUpsert(true)
    result, err := m.collection.UpdateOne(ctx, filter, update, opts)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to save: %w", err)
    }
    
    span.SetAttributes(
        attribute.Bool("upserted", result.UpsertedID != nil),
        attribute.Int("message_count", len(messages)),
    )
    
    return nil
}

func (m *ProductionMongoDBMemory) Clear(ctx context.Context) error {
    ctx, span := m.tracer.Start(ctx, "mongodb.clear")
    defer span.End()
    
    filter := bson.M{"session_id": m.sessionID}
    _, err := m.collection.DeleteOne(ctx, filter)
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}

func (m *ProductionMongoDBMemory) MemoryVariables() []string {
    return []string{"history"}
}

func main() {
    ctx := context.Background()
    
    memory, err := NewProductionMongoDBMemory(
        "mongodb://localhost:27017",
        "beluga_ai",
        "session-123",
    )
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    defer memory.client.Disconnect(ctx)
    
    // Use memory
    err = memory.SaveContext(ctx,
        map[string]any{"input": "Hello"},
        map[string]any{"output": "Hi!"},
    )
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    vars, _ := memory.LoadMemoryVariables(ctx, nil)
    fmt.Printf("Messages: %d\n", len(vars["history"].([]schema.Message)))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `URI` | MongoDB connection URI | `mongodb://localhost:27017` | No |
| `Database` | Database name | `beluga_ai` | No |
| `Collection` | Collection name | `conversations` | No |
| `SessionID` | Session identifier | - | Yes |
| `TTL` | Document TTL | - | No |

## Common Issues

### "Connection refused"

**Problem**: MongoDB not running or wrong URI.

**Solution**: Verify MongoDB connection:mongosh "mongodb://localhost:27017"
```

### "Index creation failed"

**Problem**: Index already exists or permissions issue.

**Solution**: Drop and recreate indexes or check permissions.

## Production Considerations

When using MongoDB in production:

- **Connection pooling**: Configure connection pool size
- **Indexes**: Create appropriate indexes for queries
- **TTL**: Set TTL for old conversations
- **Replication**: Use replica sets for high availability
- **Sharding**: Consider sharding for large scale

## Next Steps

Congratulations! You've integrated MongoDB with Beluga AI memory. Next, learn how to:

- **[Redis Distributed Locking](./redis-distributed-locking.md)** - Distributed locking
- **[Memory Package Documentation](../../api/packages/memory.md)** - Deep dive into memory package
- **[Memory Tutorial](../../getting-started/05-memory-management.md)** - Memory patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
