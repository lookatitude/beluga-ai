# Extensibility Guide

> **Learn how to extend Beluga AI with custom providers, tools, and components using the framework's registry and factory patterns.**

## Introduction

Beluga AI is designed from the ground up to be extensible. Whether you need to integrate a new LLM provider, add a custom vector store, implement specialized tools, or create new memory backends, the framework provides consistent patterns that make extension straightforward.

In this guide, you'll learn:

- The core extensibility patterns used throughout Beluga AI
- How to extend each major component type
- Best practices for creating maintainable extensions
- How to test your custom components

By the end, you'll be confident in extending any part of the framework to meet your specific needs.

## Core Extensibility Patterns

Before diving into specific components, let's understand the patterns that make Beluga AI extensible.

### The Registry Pattern

At the heart of Beluga AI's extensibility is the **Registry Pattern**. Each extensible component type has a global registry where providers are registered and retrieved:

```
┌────────────────────────────────────────────────────────────────┐
│                    Component Registry                           │
├────────────────────────────────────────────────────────────────┤
│  "provider-name" → Factory Function                            │
│                                                                 │
│  Register("name", factory)  ──▶  Factory is stored             │
│  GetProvider("name", config) ──▶  Factory creates instance     │
└────────────────────────────────────────────────────────────────┘
```

**Why this pattern works:**

1. **Decoupling**: Application code doesn't need to know implementation details
2. **Configuration-driven**: Switch providers by changing config, not code
3. **Testing**: Easily swap in mocks for testing
4. **Lazy initialization**: Instances are created only when needed

### The Interface Pattern

Every extensible component is defined by an interface. Implementing the interface is all you need to create a compatible component:

```go
// Example: The ChatModel interface for LLM providers
type ChatModel interface {
    Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
    StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)
    BindTools(toolsToBind []tools.Tool) ChatModel
    GetModelName() string
    CheckHealth() map[string]any
}
```

### The Factory Pattern

Instead of direct constructors, we use factory functions. This allows the registry to create instances with proper configuration:

```go
// Factory function signature
func NewMyProviderFactory() func(*Config) (Interface, error) {
    return func(config *Config) (Interface, error) {
        return NewMyProvider(config)
    }
}
```

### The Functional Options Pattern

Configuration is handled through functional options, providing a clean, extensible API:

```go
provider, err := NewProvider(
    WithAPIKey("key"),
    WithModel("model-name"),
    WithTimeout(30 * time.Second),
)
```

## LLM Provider Extension

LLM providers are the most common extension point. See the [LLM Providers Guide](./llm-providers.md) for a complete tutorial.

### Quick Reference

```go
// 1. Implement the ChatModel interface
type MyLLMProvider struct {
    config *llms.Config
    // ...
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
    // Your implementation
}

// 2. Create a factory function
func NewMyLLMProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
    return func(config *llms.Config) (iface.ChatModel, error) {
        return NewMyLLMProvider(config)
    }
}

// 3. Register with the global registry
func init() {
    llms.GetRegistry().Register("my-provider", NewMyLLMProviderFactory())
}
```

### Key Interface Methods

| Method | Purpose |
|--------|---------|
| `Generate` | Single request/response generation |
| `StreamChat` | Streaming response chunks |
| `BindTools` | Attach callable tools |
| `GetModelName` | Return model identifier |
| `CheckHealth` | Health check for monitoring |

## Vector Store Extension

Vector stores enable RAG applications by storing and retrieving embeddings. Here's how to add a custom vector store.

### The VectorStore Interface

```go
type VectorStore interface {
    // AddDocuments stores documents with their embeddings
    AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)
    
    // DeleteDocuments removes documents by ID
    DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error
    
    // SimilaritySearch finds similar documents by vector
    SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)
    
    // SimilaritySearchByQuery searches using text (embeds the query internally)
    SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)
    
    // AsRetriever returns a Retriever for use in chains
    AsRetriever(opts ...Option) Retriever
    
    // GetName returns the store identifier
    GetName() string
}
```

### Step-by-Step Implementation

#### Step 1: Create Your Store Struct

```go
// pkg/vectorstores/providers/mystore/mystore.go
package mystore

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

const ProviderName = "mystore"

type MyVectorStore struct {
    config   iface.Config
    embedder vectorstores.Embedder
    // Your storage client
    // client *mydb.Client
}

func NewMyVectorStore(ctx context.Context, config iface.Config) (*MyVectorStore, error) {
    // Validate config
    // Initialize your storage client
    
    return &MyVectorStore{
        config:   config,
        embedder: config.Embedder,
    }, nil
}
```

#### Step 2: Implement Core Methods

```go
func (s *MyVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
    // Apply options
    config := s.config
    vectorstores.ApplyOptions(&config, opts...)
    
    embedder := config.Embedder
    if embedder == nil {
        embedder = s.embedder
    }
    
    // Generate embeddings for documents
    texts := make([]string, len(documents))
    for i, doc := range documents {
        texts[i] = doc.PageContent
    }
    
    embeddings, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        return nil, err
    }
    
    // Store in your backend
    ids := make([]string, len(documents))
    for i, doc := range documents {
        // id := s.client.Store(doc, embeddings[i])
        ids[i] = generateID() // Your ID generation
    }
    
    return ids, nil
}

func (s *MyVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
    // Apply options
    config := s.config
    vectorstores.ApplyOptions(&config, opts...)
    
    // Query your backend
    // results := s.client.Search(queryVector, k)
    
    // Apply score threshold filtering
    var docs []schema.Document
    var scores []float32
    
    // for _, result := range results {
    //     if result.Score >= config.ScoreThreshold {
    //         docs = append(docs, result.Document)
    //         scores = append(scores, result.Score)
    //     }
    // }
    
    return docs, scores, nil
}

func (s *MyVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
    // Embed the query
    queryVector, err := embedder.EmbedQuery(ctx, query)
    if err != nil {
        return nil, nil, err
    }
    
    // Delegate to vector search
    return s.SimilaritySearch(ctx, queryVector, k, opts...)
}

func (s *MyVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
    // Delete from your backend
    // return s.client.Delete(ids)
    return nil
}

func (s *MyVectorStore) GetName() string {
    return ProviderName
}
```

#### Step 3: Implement Retriever

```go
func (s *MyVectorStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
    return &myRetriever{
        store: s,
        opts:  opts,
    }
}

type myRetriever struct {
    store *MyVectorStore
    opts  []vectorstores.Option
}

func (r *myRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    embedder := r.store.embedder
    docs, _, err := r.store.SimilaritySearchByQuery(ctx, query, r.store.config.SearchK, embedder, r.opts...)
    return docs, err
}
```

#### Step 4: Register the Provider

```go
// pkg/vectorstores/providers/mystore/init.go
package mystore

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
    vectorstores.GetRegistry().Register(ProviderName, 
        func(ctx context.Context, config iface.Config) (vectorstores.VectorStore, error) {
            return NewMyVectorStore(ctx, config)
        },
    )
}
```

#### Step 5: Use Your Store

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    _ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/mystore" // Register
)

func main() {
    ctx := context.Background()
    
    store, err := vectorstores.NewVectorStore(ctx, "mystore",
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithSearchK(10),
        vectorstores.WithProviderConfig("connection_string", "..."),
    )
    
    // Add documents
    ids, _ := store.AddDocuments(ctx, documents)
    
    // Search
    docs, scores, _ := store.SimilaritySearchByQuery(ctx, "query", 5, embedder)
}
```

### Adding OTEL Instrumentation

For production vector stores, add observability:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func (s *MyVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
    tracer := otel.Tracer("mystore")
    ctx, span := tracer.Start(ctx, "mystore.SimilaritySearch",
        trace.WithAttributes(
            attribute.String("store", ProviderName),
            attribute.Int("k", k),
            attribute.Int("vector_dim", len(queryVector)),
        ),
    )
    defer span.End()
    
    // ... implementation
    
    span.SetAttributes(attribute.Int("results_count", len(docs)))
    return docs, scores, nil
}
```

## Embedding Provider Extension

Embedding providers convert text to vectors for similarity search.

### The Embedder Interface

```go
type Embedder interface {
    EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
```

### Implementation Example

```go
type MyEmbedder struct {
    client    *myapi.Client
    modelName string
    dimension int
}

func NewMyEmbedder(apiKey, modelName string) (*MyEmbedder, error) {
    client := myapi.NewClient(apiKey)
    return &MyEmbedder{
        client:    client,
        modelName: modelName,
        dimension: 1536, // Your model's dimension
    }, nil
}

func (e *MyEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
    // Batch embedding API call
    return e.client.Embed(ctx, texts, e.modelName)
}

func (e *MyEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
    embeddings, err := e.EmbedDocuments(ctx, []string{text})
    if err != nil {
        return nil, err
    }
    return embeddings[0], nil
}
```

## Memory Backend Extension

Memory backends store conversation history for agents.

### The Memory Interface

```go
type Memory interface {
    // Add messages to memory
    AddMessage(ctx context.Context, message schema.Message) error
    
    // Get conversation history
    GetHistory(ctx context.Context) ([]schema.Message, error)
    
    // Clear all messages
    Clear(ctx context.Context) error
    
    // Get memory variables for prompts
    LoadMemoryVariables(ctx context.Context) (map[string]any, error)
}
```

### Implementation Patterns

```go
type MyMemory struct {
    store   *mydb.Store
    maxSize int
}

func (m *MyMemory) AddMessage(ctx context.Context, message schema.Message) error {
    // Store in your backend
    return m.store.Append(message)
}

func (m *MyMemory) GetHistory(ctx context.Context) ([]schema.Message, error) {
    messages, err := m.store.GetAll()
    if err != nil {
        return nil, err
    }
    
    // Apply window if needed
    if len(messages) > m.maxSize {
        messages = messages[len(messages)-m.maxSize:]
    }
    
    return messages, nil
}
```

## Tool Extension

Tools allow agents to interact with external systems.

### The Tool Interface

```go
type Tool interface {
    Definition() Definition
    Execute(ctx context.Context, input string) (string, error)
}

type Definition struct {
    Name        string
    Description string
    InputSchema any  // JSON schema
}
```

### Implementation Example

```go
type WebSearchTool struct {
    client *searchapi.Client
    name   string
}

func NewWebSearchTool(apiKey string) *WebSearchTool {
    return &WebSearchTool{
        client: searchapi.NewClient(apiKey),
        name:   "web_search",
    }
}

func (t *WebSearchTool) Definition() tools.Definition {
    return tools.Definition{
        Name:        t.name,
        Description: "Search the web for current information",
        InputSchema: `{
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "The search query"
                }
            },
            "required": ["query"]
        }`,
    }
}

func (t *WebSearchTool) Execute(ctx context.Context, input string) (string, error) {
    // Parse input JSON
    var params struct {
        Query string `json:"query"`
    }
    if err := json.Unmarshal([]byte(input), &params); err != nil {
        return "", err
    }
    
    // Execute search
    results, err := t.client.Search(ctx, params.Query)
    if err != nil {
        return "", err
    }
    
    // Format results
    return formatResults(results), nil
}
```

## Best Practices

### 1. Follow SOLID Principles

```go
// ✅ Good: Single responsibility
type MyProvider struct {
    generator   Generator
    streamer    Streamer
    healthCheck HealthChecker
}

// ❌ Bad: God object with many responsibilities
type MyProvider struct {
    // Everything in one place
}
```

### 2. Use Composition Over Inheritance

```go
// ✅ Good: Embed for composition
type MyProvider struct {
    *BaseProvider  // Provides common functionality
    client *myapi.Client
}

// ❌ Bad: Deep inheritance hierarchy (not possible in Go anyway)
```

### 3. Handle Errors Properly

```go
// ✅ Good: Wrap errors with context
if err != nil {
    return nil, fmt.Errorf("mystore: failed to search: %w", err)
}

// ❌ Bad: Swallow errors
if err != nil {
    return nil, nil  // Caller has no idea what happened
}
```

### 4. Respect Context

```go
// ✅ Good: Check context cancellation
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
    // Continue processing
}

// ❌ Bad: Ignore context
// Long operation without checking ctx.Done()
```

### 5. Thread Safety

```go
// ✅ Good: Immutable returns for BindTools
func (p *Provider) BindTools(tools []Tool) ChatModel {
    newProvider := *p
    newProvider.tools = make([]Tool, len(tools))
    copy(newProvider.tools, tools)
    return &newProvider
}

// ❌ Bad: Mutate shared state
func (p *Provider) BindTools(tools []Tool) ChatModel {
    p.tools = tools  // Race condition!
    return p
}
```

## Testing Extensions

### Interface Compliance

```go
func TestMyProviderImplementsInterface(t *testing.T) {
    provider := setupTestProvider(t)
    var _ iface.ChatModel = provider  // Compile-time check
}
```

### Table-Driven Tests

```go
func TestSimilaritySearch(t *testing.T) {
    tests := []struct {
        name        string
        queryVector []float32
        k           int
        wantCount   int
        wantErr     bool
    }{
        {"basic search", []float32{1, 0, 0}, 5, 5, false},
        {"empty vector", []float32{}, 5, 0, true},
        {"k=0", []float32{1, 0, 0}, 0, 0, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            store := setupTestStore(t)
            docs, _, err := store.SimilaritySearch(ctx, tt.queryVector, tt.k)
            
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Len(t, docs, tt.wantCount)
            }
        })
    }
}
```

### Concurrency Tests

```go
func TestConcurrentAccess(t *testing.T) {
    store := setupTestStore(t)
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _, err := store.SimilaritySearch(ctx, vector, 5)
            assert.NoError(t, err)
        }()
    }
    wg.Wait()
}
```

## Related Resources

- **[LLM Providers Guide](./llm-providers.md)**: Complete LLM provider integration tutorial
- **[Voice Providers Guide](./voice-providers.md)**: STT, TTS, and S2S provider integration
- **[Custom LLM Provider Example](/examples/llms/custom_provider/)**: Working code example
- **[Observability Tracing Guide](./observability-tracing.md)**: Adding OTEL to your extensions
- **[Error Handling Cookbook](../cookbook/llm-error-handling.md)**: Error handling patterns
