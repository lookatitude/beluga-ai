---
title: Enterprise Rag Knowledge Base
sidebar_position: 1
---

# Use Case 1: Enterprise RAG Knowledge Base System

## Overview & Objectives

### Business Problem

Enterprises struggle with managing vast amounts of internal knowledge scattered across documents, wikis, codebases, and databases. Employees waste significant time searching for information, leading to reduced productivity and inconsistent decision-making. Traditional keyword search fails to understand context and semantic relationships.

### Solution Approach

This use case implements a comprehensive Retrieval-Augmented Generation (RAG) system that:
- Ingests documents from multiple sources
- Creates semantic embeddings and stores them in vector databases
- Provides intelligent Q&A capabilities with context-aware responses
- Maintains conversation history for follow-up questions
- Offers REST API for integration with existing systems

### Key Benefits

- **Semantic Understanding**: Finds relevant information based on meaning, not just keywords
- **Context-Aware Responses**: LLM generates answers using retrieved context
- **Scalable Architecture**: Handles millions of documents efficiently
- **Production-Ready**: Full observability, error handling, and monitoring
- **Multi-Provider Support**: Flexible vector store and LLM provider selection

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Applications                       │
│              (Web UI, Mobile App, API Consumers)                 │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              │ HTTP/REST
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    REST Server (pkg/server)                      │
│  - Request routing                                              │
│  - Authentication/Authorization                                 │
│  - Rate limiting                                                 │
│  - Request/Response logging                                      │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              RAG Orchestration Chain (pkg/orchestration)          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  1. Query    │→ │  2. Retrieve │→ │  3. Generate  │         │
│  │  Processing  │  │  Documents    │  │  Response     │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Embeddings │    │  Retrievers   │    │     LLMs      │
│  (pkg/       │    │  (pkg/       │    │  (pkg/llms)   │
│  embeddings) │    │  retrievers) │    │               │
└──────┬───────┘    └──────┬───────┘    └──────┬───────┘
       │                   │                     │
       │                   │                     │
       └───────────────────┼─────────────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │   Vector Stores        │
              │  (pkg/vectorstores)   │
              │  - PgVector           │
              │  - Pinecone           │
              │  - InMemory           │
              └────────────────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │   PostgreSQL /        │
              │   Pinecone Cloud      │
              └────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    Observability Layer                           │
│  - OpenTelemetry Metrics (pkg/monitoring)                       │
│  - Distributed Tracing                                           │
│  - Structured Logging                                            │
│  - Health Checks                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Component Usage

### Beluga AI Packages Used

1. **pkg/vectorstores**
   - Store document embeddings
   - Perform similarity search
   - Support multiple providers (PgVector, Pinecone)

2. **pkg/embeddings**
   - Generate embeddings for queries and documents
   - Support OpenAI and Ollama providers

3. **pkg/retrievers**
   - Retrieve relevant documents based on queries
   - Configurable K and score thresholds

4. **pkg/llms**
   - Generate context-aware responses
   - Support multiple providers (OpenAI, Anthropic, etc.)

5. **pkg/memory**
   - Maintain conversation history
   - Support BufferMemory for context retention

6. **pkg/prompts**
   - Template-based prompt generation
   - Context injection for RAG responses

7. **pkg/orchestration**
   - Chain orchestration for RAG pipeline
   - Sequential execution of query → retrieve → generate

8. **pkg/monitoring**
   - OpenTelemetry metrics and tracing
   - Performance monitoring

9. **pkg/config**
   - Configuration management
   - Environment variable support

10. **pkg/server**
    - REST API server
    - Request handling and routing

## Implementation Guide

### Step 1: Project Setup

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/memory"
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "github.com/lookatitude/beluga-ai/pkg/retrievers"
    "github.com/lookatitude/beluga-ai/pkg/server"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)
```

### Step 2: Initialize Components

```go
func setupRAGSystem(ctx context.Context, cfg *config.Config) (*RAGSystem, error) {
    // Initialize embedding provider
    embedder, err := embeddings.NewEmbedder(ctx, "openai",
        embeddings.WithAPIKey(cfg.GetString("embeddings.openai.api_key")),
        embeddings.WithModel("text-embedding-ada-002"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create embedder: %w", err)
    }

    // Initialize vector store
    vectorStore, err := vectorstores.NewVectorStore(ctx, "pgvector",
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithProviderConfig("connection_string", 
            cfg.GetString("vectorstore.postgres.connection_string")),
        vectorstores.WithProviderConfig("table_name", "knowledge_base"),
        vectorstores.WithSearchK(10),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create vector store: %w", err)
    }

    // Initialize retriever
    retriever, err := retrievers.NewVectorStoreRetriever(vectorStore,
        retrievers.WithDefaultK(5),
        retrievers.WithScoreThreshold(0.7),
        retrievers.WithTimeout(30*time.Second),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create retriever: %w", err)
    }

    // Initialize LLM
    llm, err := llms.NewChatModel(ctx, "openai",
        llms.WithAPIKey(cfg.GetString("llm.openai.api_key")),
        llms.WithModel("gpt-4"),
        llms.WithTemperature(0.7),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create LLM: %w", err)
    }

    // Initialize memory
    mem := memory.NewBufferMemory()

    // Initialize prompt template
    promptTemplate := prompts.NewPromptTemplate(
        "You are a helpful assistant answering questions based on the following context:\n\n{{.context}}\n\nQuestion: {{.question}}\n\nAnswer:",
    )

    return &RAGSystem{
        embedder:      embedder,
        vectorStore:   vectorStore,
        retriever:     retriever,
        llm:           llm,
        memory:        mem,
        promptTemplate: promptTemplate,
    }, nil
}
```

### Step 3: Create RAG Chain

```go
func createRAGChain(rag *RAGSystem) (orchestration.Chain, error) {
    // Step 1: Process query and retrieve context
    retrieveStep := &RetrieveStep{
        retriever: rag.retriever,
    }

    // Step 2: Format prompt with context
    formatStep := &FormatPromptStep{
        template: rag.promptTemplate,
    }

    // Step 3: Generate response
    generateStep := &GenerateStep{
        llm: rag.llm,
    }

    // Create chain
    chain, err := orchestration.NewChain(
        []core.Runnable{retrieveStep, formatStep, generateStep},
        orchestration.WithChainName("rag-pipeline"),
        orchestration.WithChainTimeout(60*time.Second),
        orchestration.WithChainMemory(rag.memory),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create chain: %w", err)
    }

    return chain, nil
}

// RetrieveStep implements core.Runnable
type RetrieveStep struct {
    retriever retrievers.Retriever
}

func (r *RetrieveStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    query, ok := input.(map[string]any)["query"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid input: query not found")
    }

    docs, err := r.retriever.GetRelevantDocuments(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("retrieval failed: %w", err)
    }

    // Combine document content
    context := ""
    for _, doc := range docs {
        context += doc.GetContent() + "\n\n"
    }

    return map[string]any{
        "query":   query,
        "context": context,
    }, nil
}

// FormatPromptStep implements core.Runnable
type FormatPromptStep struct {
    template *prompts.PromptTemplate
}

func (f *FormatPromptStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    inputMap := input.(map[string]any)
    query := inputMap["query"].(string)
    context := inputMap["context"].(string)

    prompt, err := f.template.Format(map[string]any{
        "question": query,
        "context":  context,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to format prompt: %w", err)
    }

    return map[string]any{
        "prompt": prompt,
    }, nil
}

// GenerateStep implements core.Runnable
type GenerateStep struct {
    llm llms.ChatModel
}

func (g *GenerateStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    inputMap := input.(map[string]any)
    prompt := inputMap["prompt"].(string)

    messages := []schema.Message{
        schema.NewHumanMessage(prompt),
    }

    response, err := g.llm.Generate(ctx, messages)
    if err != nil {
        return nil, fmt.Errorf("generation failed: %w", err)
    }

    return map[string]any{
        "answer": response.GetContent(),
    }, nil
}
```

### Step 4: Set Up REST Server

```go
func setupServer(chain orchestration.Chain, cfg *config.Config) error {
    restProvider, err := server.NewRESTServer(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: cfg.GetString("server.host"),
                Port: cfg.GetInt("server.port"),
            },
            APIBasePath: "/api/v1",
        }),
    )
    if err != nil {
        return fmt.Errorf("failed to create REST server: %w", err)
    }

    // Register RAG handler
    ragHandler := &RAGHandler{chain: chain}
    restProvider.RegisterHandler("POST", "/api/v1/query", ragHandler.HandleQuery)

    // Start server
    ctx := context.Background()
    return restProvider.Start(ctx)
}

type RAGHandler struct {
    chain orchestration.Chain
}

func (h *RAGHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query string `json:"query"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    input := map[string]any{"query": req.Query}
    result, err := h.chain.Invoke(r.Context(), input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]any{
        "answer": result.(map[string]any)["answer"],
    }
    json.NewEncoder(w).Encode(response)
}
```

## Workflow & Data Flow

### End-to-End Process Flow

1. **Document Ingestion**
   ```
   Document → Embedding → Vector Store
   ```

2. **Query Processing**
   ```
   User Query → Embedding → Similarity Search → Retrieve Top K Documents
   ```

3. **Response Generation**
   ```
   Retrieved Context + Query → Prompt Template → LLM → Generated Answer
   ```

4. **Memory Management**
   ```
   Query + Answer → Save to Memory → Available for Follow-up Questions
   ```

### Component Interactions

- **Retriever ↔ VectorStore**: Retrieves documents based on query embeddings
- **LLM ↔ Memory**: Loads conversation history for context
- **Chain ↔ All Components**: Orchestrates the entire RAG pipeline
- **Server ↔ Chain**: Handles HTTP requests and executes chain

### Error Handling Strategies

```go
func (r *RAGSystem) Query(ctx context.Context, query string) (string, error) {
    // Set timeout
    ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()

    // Retry logic for transient errors
    var result any
    var err error
    for i := 0; i < 3; i++ {
        result, err = r.chain.Invoke(ctx, map[string]any{"query": query})
        if err == nil {
            break
        }
        if !isRetryable(err) {
            return "", err
        }
        time.Sleep(time.Duration(i+1) * time.Second)
    }

    if err != nil {
        return "", fmt.Errorf("query failed after retries: %w", err)
    }

    return result.(map[string]any)["answer"].(string), nil
}

func isRetryable(err error) bool {
    // Check for timeout, rate limit, or network errors
    return errors.Is(err, context.DeadlineExceeded) ||
           strings.Contains(err.Error(), "rate limit") ||
           strings.Contains(err.Error(), "network")
}
```

## Observability Setup

### OpenTelemetry Configuration

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
)

func setupObservability(ctx context.Context) (*trace.TracerProvider, error) {
    // Create OTLP exporter
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    // Create resource
    res, err := resource.New(ctx,
        resource.WithAttributes(
            attribute.String("service.name", "rag-knowledge-base"),
            attribute.String("service.version", "1.0.0"),
        ),
    )
    if err != nil {
        return nil, err
    }

    // Create tracer provider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
    )
    otel.SetTracerProvider(tp)

    return tp, nil
}
```

### Metrics to Monitor

- `rag_queries_total`: Total number of queries processed
- `rag_query_duration_seconds`: Query processing latency
- `rag_retrieval_documents_count`: Number of documents retrieved per query
- `rag_llm_tokens_used`: Token usage for LLM calls
- `rag_errors_total`: Error count by type
- `vector_store_search_duration_seconds`: Vector search latency

### Tracing Setup

```go
func (r *RAGSystem) Query(ctx context.Context, query string) (string, error) {
    tracer := otel.Tracer("rag-system")
    ctx, span := tracer.Start(ctx, "rag.query")
    defer span.End()

    span.SetAttributes(
        attribute.String("query", query),
    )

    // Add child spans for each step
    ctx, retrieveSpan := tracer.Start(ctx, "rag.retrieve")
    docs, err := r.retriever.GetRelevantDocuments(ctx, query)
    retrieveSpan.End()
    if err != nil {
        span.RecordError(err)
        return "", err
    }

    // ... rest of the pipeline
    return answer, nil
}
```

### Logging Configuration

```go
import (
    "log/slog"
    "os"
)

func setupLogger() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        AddSource: true,
    }))
}

// Usage
logger.Info("query processed",
    "query", query,
    "documents_retrieved", len(docs),
    "answer_length", len(answer),
    "duration", duration,
)
```

## Configuration Examples

### Complete YAML Configuration

```yaml
# config.yaml
app:
  name: "rag-knowledge-base"
  version: "1.0.0"

server:
  host: "0.0.0.0"
  port: 8080
  api_base_path: "/api/v1"

embeddings:
  provider: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-ada-002"
    timeout: 30s

vectorstore:
  provider: "pgvector"
  postgres:
    connection_string: "${POSTGRES_CONNECTION_STRING}"
    table_name: "knowledge_base"
    search_k: 10
    score_threshold: 0.7

retrievers:
  default_k: 5
  score_threshold: 0.7
  timeout: 30s
  max_retries: 3

llm:
  provider: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
    temperature: 0.7
    max_tokens: 1000
    timeout: 60s

memory:
  type: "buffer"
  buffer:
    return_messages: true

prompts:
  template: |
    You are a helpful assistant answering questions based on the following context:
    
    {{.context}}
    
    Question: {{.question}}
    
    Answer:

orchestration:
  chain:
    timeout: 60s
    max_retries: 3
    enable_tracing: true

monitoring:
  otel:
    endpoint: "localhost:4317"
    insecure: true
  metrics:
    enabled: true
    prefix: "rag"
  tracing:
    enabled: true
    sample_rate: 1.0
  logging:
    level: "info"
    format: "json"
```

### Environment Variables

```bash
# .env
OPENAI_API_KEY=your-openai-api-key-here
POSTGRES_CONNECTION_STRING=postgres://user:pass@localhost/knowledge_db
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
LOG_LEVEL=info
```

## Deployment Considerations

### Production Requirements

- **Vector Store**: PostgreSQL with pgvector extension or Pinecone cloud
- **LLM Provider**: OpenAI, Anthropic, or self-hosted (Ollama)
- **Compute**: Minimum 4 CPU cores, 8GB RAM
- **Storage**: SSD recommended for vector operations
- **Network**: Low latency to LLM API endpoints

### Scaling Strategies

1. **Horizontal Scaling**: Deploy multiple RAG service instances behind load balancer
2. **Vector Store Scaling**: Use read replicas for vector search
3. **Caching**: Cache frequent queries and embeddings
4. **Batch Processing**: Process document ingestion in batches

### Performance Optimization

```go
// Batch document ingestion
func (r *RAGSystem) IngestDocuments(ctx context.Context, docs []schema.Document) error {
    batchSize := 100
    for i := 0; i < len(docs); i += batchSize {
        end := i + batchSize
        if end > len(docs) {
            end = len(docs)
        }
        batch := docs[i:end]
        if err := r.vectorStore.AddDocuments(ctx, batch); err != nil {
            return fmt.Errorf("batch %d failed: %w", i/batchSize, err)
        }
    }
    return nil
}

// Connection pooling for vector store
func setupVectorStorePool(cfg *config.Config) (*pgxpool.Pool, error) {
    return pgxpool.New(context.Background(), cfg.GetString("vectorstore.postgres.connection_string"))
}
```

### Security Considerations

- **API Key Management**: Use environment variables or secret management systems
- **Rate Limiting**: Implement rate limits on REST endpoints
- **Input Validation**: Validate and sanitize user queries
- **Access Control**: Implement authentication and authorization
- **Data Encryption**: Encrypt data at rest and in transit

## Testing Strategy

### Unit Test Examples

```go
func TestRetrieveStep(t *testing.T) {
    mockRetriever := &MockRetriever{
        documents: []schema.Document{
            schema.NewDocument("Test content", nil),
        },
    }
    step := &RetrieveStep{retriever: mockRetriever}

    input := map[string]any{"query": "test"}
    result, err := step.Invoke(context.Background(), input)
    assert.NoError(t, err)
    assert.Contains(t, result.(map[string]any)["context"].(string), "Test content")
}

func TestRAGChain(t *testing.T) {
    // Setup mocks
    mockRetriever := &MockRetriever{}
    mockLLM := &MockLLM{}
    
    // Create chain
    chain := createTestChain(mockRetriever, mockLLM)
    
    // Execute
    input := map[string]any{"query": "What is AI?"}
    result, err := chain.Invoke(context.Background(), input)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.(map[string]any)["answer"])
}
```

### Integration Test Scenarios

```go
func TestRAGPipelineIntegration(t *testing.T) {
    // Setup real components (with test config)
    rag, err := setupRAGSystem(testCtx, testConfig)
    require.NoError(t, err)

    // Ingest test documents
    docs := []schema.Document{
        schema.NewDocument("Beluga AI is a Go framework for AI applications", nil),
        schema.NewDocument("RAG stands for Retrieval-Augmented Generation", nil),
    }
    err = rag.IngestDocuments(testCtx, docs)
    require.NoError(t, err)

    // Query
    answer, err := rag.Query(testCtx, "What is Beluga AI?")
    require.NoError(t, err)
    assert.Contains(t, answer, "Go framework")
}
```

### Performance Benchmarks

```go
func BenchmarkRAGQuery(b *testing.B) {
    rag := setupBenchmarkRAG(b)
    query := "What is machine learning?"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := rag.Query(context.Background(), query)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Troubleshooting Guide

### Common Issues

1. **Slow Query Performance**
   - **Cause**: Large vector store or inefficient similarity search
   - **Solution**: 
     - Increase `search_k` for better recall
     - Use HNSW index for faster search
     - Consider vector store read replicas

2. **Low Quality Retrievals**
   - **Cause**: Poor embeddings or incorrect K value
   - **Solution**:
     - Use better embedding models
     - Adjust `score_threshold`
     - Increase `default_k` for more context

3. **LLM Timeout Errors**
   - **Cause**: Long prompts or slow LLM responses
   - **Solution**:
     - Reduce retrieved document count
     - Increase timeout settings
     - Use faster LLM models

4. **Memory Issues**
   - **Cause**: Large conversation history
   - **Solution**:
     - Use WindowMemory instead of BufferMemory
     - Implement conversation summarization
     - Set memory size limits

### Debugging Tips

```go
// Enable debug logging
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Add detailed tracing
ctx, span := tracer.Start(ctx, "rag.query",
    trace.WithAttributes(
        attribute.String("query", query),
        attribute.Int("k", 5),
    ),
)
defer span.End()

// Monitor metrics
metrics.Counter(ctx, "rag_queries_total", 1, map[string]string{
    "status": "success",
})
```

### Performance Tuning

1. **Optimize Embedding Generation**
   - Batch embedding requests
   - Cache common query embeddings
   - Use faster embedding models for non-critical queries

2. **Vector Store Optimization**
   - Create appropriate indexes
   - Use connection pooling
   - Monitor query performance

3. **LLM Optimization**
   - Use streaming for better UX
   - Cache responses for common queries
   - Implement response compression

## Conclusion

This Enterprise RAG Knowledge Base System demonstrates Beluga AI's capabilities in building production-ready RAG applications. The system showcases:

- **Comprehensive Component Integration**: Multiple packages working together seamlessly
- **Production-Grade Observability**: Full OpenTelemetry integration
- **Flexible Architecture**: Support for multiple providers and configurations
- **Scalable Design**: Ready for enterprise deployment

The system can be extended with additional features like:
- Multi-tenant support
- Advanced retrieval strategies (hybrid search)
- Document versioning
- User feedback and learning
- Advanced prompt engineering

