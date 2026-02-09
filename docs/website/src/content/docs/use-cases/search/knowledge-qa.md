---
title: Enterprise Knowledge Q&A System
description: Build scalable knowledge search with semantic retrieval across millions of documents using Beluga AI's vector store package.
---

Enterprises accumulate millions of documents across wikis, knowledge bases, and documentation systems. Traditional keyword search delivers low accuracy and cannot answer complex questions requiring context from multiple sources. A semantic knowledge Q&A system uses vector similarity search to find relevant context and generates accurate answers grounded in retrieved documents.

## Solution Architecture

Beluga AI provides a complete RAG pipeline: document loaders ingest content, text splitters chunk it semantically, embedders convert text to vectors, vector stores index them for similarity search, and LLMs generate answers from retrieved context. The system scales to millions of documents with consistent sub-second response times.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Documents  │───▶│   Splitter   │───▶│   Embedder   │
│ (Millions)   │    │              │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Answer     │◀───│     LLM      │◀───│ Vector Store │
│  + Sources   │    │  (Generate)  │    │ (Similarity) │
└──────────────┘    └──────────────┘    └──────┬───────┘
                          ▲                     │
                          │                     │
                    ┌─────┴────────┐            │
                    │   Retrieved  │◀───────────┘
                    │   Context    │
                    └──────────────┘
```

## Document Ingestion

Load, chunk, embed, and store documents for search:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

type KnowledgeQASystem struct {
    embedder    embedding.Embedder
    vectorStore vectorstore.VectorStore
    model       llm.ChatModel
}

func NewKnowledgeQASystem(ctx context.Context) (*KnowledgeQASystem, error) {
    embedder, err := embedding.New("openai", &embedding.Config{
        Model:  "text-embedding-3-large",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    // Use distributed vector store for scale
    store, err := vectorstore.New("pgvector", &vectorstore.Config{
        ConnectionString: os.Getenv("DATABASE_URL"),
        CollectionName:   "knowledge_base",
        Dimensions:       3072, // text-embedding-3-large
    })
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    return &KnowledgeQASystem{
        embedder:    embedder,
        vectorStore: store,
        model:       model,
    }, nil
}

func (k *KnowledgeQASystem) IndexDocuments(ctx context.Context, docPaths []string) error {
    for _, path := range docPaths {
        if err := k.indexDocument(ctx, path); err != nil {
            // Log error but continue with other documents
            continue
        }
    }
    return nil
}

func (k *KnowledgeQASystem) indexDocument(ctx context.Context, docPath string) error {
    // Load document
    docLoader, err := loader.New("directory", nil)
    if err != nil {
        return fmt.Errorf("create loader: %w", err)
    }

    docs, err := docLoader.Load(ctx, docPath)
    if err != nil {
        return fmt.Errorf("load documents: %w", err)
    }

    // Split into chunks
    textSplitter, err := splitter.New("recursive", &splitter.Config{
        ChunkSize:    1000,
        ChunkOverlap: 200,
    })
    if err != nil {
        return fmt.Errorf("create splitter: %w", err)
    }

    chunks, err := textSplitter.SplitDocuments(ctx, docs)
    if err != nil {
        return fmt.Errorf("split documents: %w", err)
    }

    // Generate embeddings
    texts := make([]string, len(chunks))
    for i, chunk := range chunks {
        texts[i] = chunk.Content
    }

    embeddings, err := k.embedder.EmbedBatch(ctx, texts)
    if err != nil {
        return fmt.Errorf("embed documents: %w", err)
    }

    // Store in vector database
    for i := range chunks {
        chunks[i].Embedding = embeddings[i]
    }

    if err := k.vectorStore.Add(ctx, chunks); err != nil {
        return fmt.Errorf("store documents: %w", err)
    }

    return nil
}
```

## Question Answering

Retrieve relevant context and generate grounded answers:

```go
package main

import (
    "context"
    "fmt"
    "strings"
)

type Answer struct {
    Question string
    Answer   string
    Sources  []Source
}

type Source struct {
    Title   string
    Content string
    URL     string
}

func (k *KnowledgeQASystem) AnswerQuestion(ctx context.Context, question string) (*Answer, error) {
    // Generate query embedding
    queryEmbedding, err := k.embedder.Embed(ctx, []string{question})
    if err != nil {
        return nil, fmt.Errorf("embed question: %w", err)
    }

    // Retrieve relevant documents
    results, err := k.vectorStore.SimilaritySearch(ctx, queryEmbedding[0],
        vectorstore.WithTopK(5),
        vectorstore.WithThreshold(0.7),
    )
    if err != nil {
        return nil, fmt.Errorf("similarity search: %w", err)
    }

    if len(results) == 0 {
        return &Answer{
            Question: question,
            Answer:   "I couldn't find relevant information to answer this question.",
        }, nil
    }

    // Build context from retrieved documents
    var context strings.Builder
    sources := make([]Source, len(results))

    for i, result := range results {
        context.WriteString(result.Content)
        context.WriteString("\n\n")

        sources[i] = Source{
            Title:   result.Metadata["title"].(string),
            Content: result.Content,
            URL:     result.Metadata["url"].(string),
        }
    }

    // Generate answer using LLM
    systemMsg := &schema.SystemMessage{
        Parts: []schema.ContentPart{
            schema.TextPart{Text: "You are a helpful assistant that answers questions based on provided context. " +
                "If the context doesn't contain enough information, say so clearly."},
        },
    }

    humanMsg := &schema.HumanMessage{
        Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf(
                "Context:\n%s\n\nQuestion: %s\n\nProvide a clear, accurate answer based only on the context above.",
                context.String(),
                question,
            )},
        },
    }

    resp, err := k.model.Generate(ctx, []schema.Message{systemMsg, humanMsg})
    if err != nil {
        return nil, fmt.Errorf("generate answer: %w", err)
    }

    return &Answer{
        Question: question,
        Answer:   resp.Parts[0].(schema.TextPart).Text,
        Sources:  sources,
    }, nil
}
```

## Streaming Responses

Stream answers as they are generated for better UX:

```go
package main

import (
    "context"
    "iter"
)

func (k *KnowledgeQASystem) StreamAnswer(ctx context.Context, question string) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        // Retrieve context (same as non-streaming)
        queryEmbedding, err := k.embedder.Embed(ctx, []string{question})
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }

        results, err := k.vectorStore.SimilaritySearch(ctx, queryEmbedding[0],
            vectorstore.WithTopK(5),
        )
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }

        // Build context
        var context strings.Builder
        for _, result := range results {
            context.WriteString(result.Content)
            context.WriteString("\n\n")
        }

        // Generate streaming answer
        systemMsg := &schema.SystemMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Answer based on the provided context."},
            },
        }

        humanMsg := &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context.String(), question)},
            },
        }

        // Stream from LLM
        for chunk, err := range k.model.Stream(ctx, []schema.Message{systemMsg, humanMsg}) {
            if !yield(chunk, err) {
                return
            }
        }
    }
}
```

## Hybrid Search

Combine vector similarity with keyword matching for better results:

```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/rag/retriever"

    _ "github.com/lookatitude/beluga-ai/rag/retriever/providers/hybrid"
)

type HybridKnowledgeQA struct {
    KnowledgeQASystem
    retriever retriever.Retriever
}

func NewHybridKnowledgeQA(ctx context.Context) (*HybridKnowledgeQA, error) {
    base, err := NewKnowledgeQASystem(ctx)
    if err != nil {
        return nil, err
    }

    // Use hybrid retriever (vector + BM25 + RRF fusion)
    ret, err := retriever.New("hybrid", &retriever.Config{
        VectorStore: base.vectorStore,
        Embedder:    base.embedder,
        TopK:        10,
        Threshold:   0.6,
    })
    if err != nil {
        return nil, fmt.Errorf("create retriever: %w", err)
    }

    return &HybridKnowledgeQA{
        KnowledgeQASystem: *base,
        retriever:         ret,
    }, nil
}

func (h *HybridKnowledgeQA) AnswerQuestion(ctx context.Context, question string) (*Answer, error) {
    // Retrieve using hybrid search
    docs, err := h.retriever.Retrieve(ctx, question,
        retriever.WithTopK(5),
        retriever.WithThreshold(0.7),
    )
    if err != nil {
        return nil, fmt.Errorf("retrieve: %w", err)
    }

    // Build context and generate answer (same as before)
    var context strings.Builder
    sources := make([]Source, len(docs))

    for i, doc := range docs {
        context.WriteString(doc.Content)
        context.WriteString("\n\n")

        sources[i] = Source{
            Title:   doc.Metadata["title"].(string),
            Content: doc.Content,
            URL:     doc.Metadata["url"].(string),
        }
    }

    systemMsg := &schema.SystemMessage{
        Parts: []schema.ContentPart{
            schema.TextPart{Text: "Answer based on the provided context."},
        },
    }

    humanMsg := &schema.HumanMessage{
        Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context.String(), question)},
        },
    }

    resp, err := h.model.Generate(ctx, []schema.Message{systemMsg, humanMsg})
    if err != nil {
        return nil, fmt.Errorf("generate: %w", err)
    }

    return &Answer{
        Question: question,
        Answer:   resp.Parts[0].(schema.TextPart).Text,
        Sources:  sources,
    }, nil
}
```

## Production Considerations

### Observability

Track query latency and retrieval quality:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (k *KnowledgeQASystem) AnswerWithTracing(ctx context.Context, question string) (*Answer, error) {
    tracer := otel.Tracer("knowledge-qa")
    ctx, span := tracer.Start(ctx, "qa.answer")
    defer span.End()

    span.SetAttributes(
        attribute.String("gen_ai.prompt", question),
    )

    start := time.Now()
    answer, err := k.AnswerQuestion(ctx, question)
    duration := time.Since(start)

    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("qa.sources_count", len(answer.Sources)),
        attribute.Float64("qa.duration_ms", float64(duration.Milliseconds())),
    )

    return answer, nil
}
```

### Batch Ingestion

Process large document collections efficiently:

```go
func (k *KnowledgeQASystem) IndexBatch(ctx context.Context, docPaths []string) error {
    batchSize := 100

    for i := 0; i < len(docPaths); i += batchSize {
        end := min(i+batchSize, len(docPaths))
        batch := docPaths[i:end]

        // Process batch in parallel
        var wg sync.WaitGroup
        for _, path := range batch {
            wg.Add(1)
            go func(p string) {
                defer wg.Done()
                if err := k.indexDocument(ctx, p); err != nil {
                    // Log error but continue
                }
            }(path)
        }
        wg.Wait()
    }

    return nil
}
```

### Resilience

Add retry logic for transient failures:

```go
import "github.com/lookatitude/beluga-ai/resilience"

policy := resilience.RetryPolicy{
    MaxAttempts:    3,
    InitialBackoff: 500 * time.Millisecond,
    MaxBackoff:     5 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         true,
}

answer, err := resilience.Retry(ctx, policy, func(ctx context.Context) (*Answer, error) {
    return k.AnswerQuestion(ctx, question)
})
```

### Caching

Cache frequent queries to reduce costs:

```go
import "github.com/lookatitude/beluga-ai/cache"

type CachedKnowledgeQA struct {
    KnowledgeQASystem
    cache cache.Cache
}

func (c *CachedKnowledgeQA) AnswerQuestion(ctx context.Context, question string) (*Answer, error) {
    cacheKey := fmt.Sprintf("qa:%s", question)

    // Check cache first
    if cached, ok := c.cache.Get(ctx, cacheKey); ok {
        return cached.(*Answer), nil
    }

    // Generate answer
    answer, err := c.KnowledgeQASystem.AnswerQuestion(ctx, question)
    if err != nil {
        return nil, err
    }

    // Cache for 10 minutes
    c.cache.Set(ctx, cacheKey, answer, 10*time.Minute)

    return answer, nil
}
```

### Access Control

Filter results based on user permissions:

```go
func (k *KnowledgeQASystem) AnswerWithAccessControl(ctx context.Context, userID, question string) (*Answer, error) {
    // Retrieve more results than needed
    queryEmbedding, err := k.embedder.Embed(ctx, []string{question})
    if err != nil {
        return nil, err
    }

    results, err := k.vectorStore.SimilaritySearch(ctx, queryEmbedding[0],
        vectorstore.WithTopK(20), // Get more to filter
    )
    if err != nil {
        return nil, err
    }

    // Filter based on user permissions
    filtered := make([]schema.Document, 0)
    for _, result := range results {
        if canAccess(userID, result.Metadata["acl"]) {
            filtered = append(filtered, result)
            if len(filtered) >= 5 {
                break
            }
        }
    }

    // Generate answer from filtered results
    // ... (same as before)
    return nil, nil
}
```

## Related Resources

- [Enterprise RAG Guide](/use-cases/enterprise-rag/) for complete RAG pipeline
- [Retriever Guide](/guides/retriever-patterns/) for advanced retrieval
- [Vector Store Guide](/guides/vector-stores/) for scaling strategies
- [Recommendation Engine](/use-cases/recommendation-engine/) for similarity patterns
