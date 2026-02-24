---
title: Enterprise RAG Knowledge Base
description: "Build a production RAG pipeline with hybrid search, multi-source ingestion, and semantic retrieval. Grounded answers with source citations."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "enterprise RAG, knowledge base AI, hybrid search pipeline, semantic retrieval, document ingestion, Beluga AI, Go, RAG use case"
---

Organizations accumulate vast amounts of knowledge across documents, wikis, codebases, and databases. When employees search for information, keyword-based systems return documents that contain the search terms but may not actually answer the question. A search for "how to handle customer refunds" returns every document mentioning "refund" — policy documents, meeting notes, email threads — without understanding which ones actually explain the refund process. This semantic gap between intent and keyword matching leads to lost productivity and inconsistent decision-making.

An enterprise RAG (Retrieval-Augmented Generation) system brings semantic understanding to internal knowledge, enabling employees to ask natural language questions and receive accurate, context-grounded answers. RAG is chosen over fine-tuning the LLM on internal documents because it keeps the knowledge current (new documents are indexed immediately), transparent (answers cite their sources), and controllable (access policies can be enforced at retrieval time).

## Solution Architecture

Beluga AI provides a complete RAG pipeline: document loaders ingest content from multiple sources, text splitters chunk it into semantically meaningful units, embedders convert text to vectors, vector stores index them for fast similarity search, and retriever strategies combine multiple signals for high-quality results. The LLM generates grounded answers from retrieved context.

Beluga AI defaults to hybrid search (Vector + BM25 + RRF fusion) because neither vector similarity nor keyword matching alone is sufficient. Vector search captures semantic meaning but can miss exact terms (product names, error codes, policy numbers). Keyword search catches exact terms but misses semantic relationships. Reciprocal Rank Fusion (RRF) combines both rankings without requiring score normalization, producing results that satisfy both intent-based and term-based queries.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Documents  │───▶│  Splitter    │───▶│  Embedder    │
│  (PDF, HTML, │    │  (Recursive, │    │  (OpenAI,    │
│   Markdown)  │    │   Semantic)  │    │   Ollama)    │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Response   │◀───│  LLM         │◀───│ VectorStore  │
│   (Grounded  │    │  (Generate   │    │ (pgvector,   │
│    Answer)   │    │   Answer)    │    │  Pinecone)   │
└──────────────┘    └──────────────┘    └──────────────┘
                          ▲
                          │
                    ┌─────┴────────┐
                    │  Retriever   │
                    │  (Hybrid,    │
                    │   CRAG, HyDE)│
                    └──────────────┘
```

## Document Ingestion Pipeline

The ingestion pipeline loads documents from various sources, splits them into chunks, generates embeddings, and stores them in a vector database.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/pdf"
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/html"
    _ "github.com/lookatitude/beluga-ai/rag/splitter/providers/recursive"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

func ingestDocuments(ctx context.Context) error {
    // Load documents from PDF files
    pdfLoader, err := loader.New("pdf", nil)
    if err != nil {
        return fmt.Errorf("create loader: %w", err)
    }

    docs, err := pdfLoader.Load(ctx, "/data/knowledge-base/")
    if err != nil {
        return fmt.Errorf("load documents: %w", err)
    }

    // Split into chunks for embedding
    textSplitter, err := splitter.New("recursive", nil)
    if err != nil {
        return fmt.Errorf("create splitter: %w", err)
    }

    chunks, err := textSplitter.SplitDocuments(ctx, docs)
    if err != nil {
        return fmt.Errorf("split documents: %w", err)
    }

    // Generate embeddings
    embedder, err := embedding.New("openai", nil)
    if err != nil {
        return fmt.Errorf("create embedder: %w", err)
    }

    texts := make([]string, len(chunks))
    for i, chunk := range chunks {
        texts[i] = chunk.Content
    }

    embeddings, err := embedder.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("embed documents: %w", err)
    }

    // Store in vector database
    store, err := vectorstore.New("pgvector", nil)
    if err != nil {
        return fmt.Errorf("create vector store: %w", err)
    }

    if err := store.Add(ctx, chunks, embeddings); err != nil {
        return fmt.Errorf("store documents: %w", err)
    }

    log.Printf("Ingested %d chunks from %d documents", len(chunks), len(docs))
    return nil
}
```

## Hybrid Retrieval with RRF Fusion

Beluga AI's default retrieval strategy combines vector similarity search with BM25 keyword matching, using Reciprocal Rank Fusion to merge results. This hybrid approach outperforms either method alone.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/retriever"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

// RAGSystem combines retrieval and generation.
type RAGSystem struct {
    retriever retriever.Retriever
    model     llm.ChatModel
}

func NewRAGSystem(ctx context.Context) (*RAGSystem, error) {
    embedder, err := embedding.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", nil)
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    // Hybrid retriever: vector + BM25 + RRF fusion
    ret, err := retriever.New("hybrid", nil)
    if err != nil {
        return nil, fmt.Errorf("create retriever: %w", err)
    }

    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    return &RAGSystem{
        retriever: ret,
        model:     model,
    }, nil
}

func (r *RAGSystem) Query(ctx context.Context, question string) (string, error) {
    // Retrieve relevant documents
    docs, err := r.retriever.Retrieve(ctx, question,
        retriever.WithTopK(5),
        retriever.WithThreshold(0.7),
    )
    if err != nil {
        return "", fmt.Errorf("retrieve: %w", err)
    }

    // Build context from retrieved documents
    var context string
    for _, doc := range docs {
        context += doc.Content + "\n\n"
    }

    // Generate answer grounded in retrieved context
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Answer the question using only the provided context. " +
                "If the context doesn't contain the answer, say so."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context, question)},
        }},
    }

    resp, err := r.model.Generate(ctx, msgs)
    if err != nil {
        return "", fmt.Errorf("generate: %w", err)
    }

    return resp.Parts[0].(schema.TextPart).Text, nil
}
```

## Advanced Retrieval Strategies

Beyond hybrid search, Beluga AI supports advanced retrieval strategies for higher-quality results.

### CRAG (Corrective RAG)

CRAG evaluates the relevance of retrieved documents and falls back to alternative sources when relevance is low:

```go
// CRAG retriever evaluates retrieval quality and self-corrects
cragRetriever, err := retriever.New("crag", nil)
if err != nil {
    return fmt.Errorf("create crag retriever: %w", err)
}

docs, err := cragRetriever.Retrieve(ctx, "What is our refund policy?",
    retriever.WithTopK(10),
    retriever.WithThreshold(0.6),
)
```

### HyDE (Hypothetical Document Embeddings)

HyDE generates a hypothetical answer to the query first, then uses that hypothetical answer's embedding for retrieval. This bridges the gap between short queries and long documents:

```go
// HyDE generates a hypothetical document for better matching
hydeRetriever, err := retriever.New("hyde", nil)
if err != nil {
    return fmt.Errorf("create hyde retriever: %w", err)
}

docs, err := hydeRetriever.Retrieve(ctx, "quarterly revenue trends",
    retriever.WithTopK(5),
)
```

## Streaming Responses

For a better user experience, stream responses as they are generated:

```go
func (r *RAGSystem) StreamQuery(ctx context.Context, question string) iter.Seq2[schema.StreamChunk, error] {
    docs, err := r.retriever.Retrieve(ctx, question, retriever.WithTopK(5))
    if err != nil {
        return func(yield func(schema.StreamChunk, error) bool) {
            yield(schema.StreamChunk{}, err)
        }
    }

    var context string
    for _, doc := range docs {
        context += doc.Content + "\n\n"
    }

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Answer using the provided context."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context, question)},
        }},
    }

    return r.model.Stream(ctx, msgs)
}
```

## Production Considerations

### Observability

Instrument the RAG pipeline with OpenTelemetry to track latency, retrieval quality, and token usage:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (r *RAGSystem) QueryWithTracing(ctx context.Context, question string) (string, error) {
    tracer := otel.Tracer("rag-system")
    ctx, span := tracer.Start(ctx, "rag.query")
    defer span.End()

    span.SetAttributes(attribute.String("gen_ai.prompt", question))

    docs, err := r.retriever.Retrieve(ctx, question, retriever.WithTopK(5))
    if err != nil {
        span.RecordError(err)
        return "", err
    }

    span.SetAttributes(attribute.Int("rag.documents_retrieved", len(docs)))

    // ... generate response ...
    return answer, nil
}
```

### Resilience

Wrap LLM calls with retry and circuit breaker for production reliability:

```go
import "github.com/lookatitude/beluga-ai/resilience"

policy := resilience.RetryPolicy{
    MaxAttempts:    3,
    InitialBackoff: 500 * time.Millisecond,
    MaxBackoff:     5 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         true,
}

answer, err := resilience.Retry(ctx, policy, func(ctx context.Context) (string, error) {
    return r.Query(ctx, question)
})
```

### Batch Ingestion

Process large document collections efficiently with batched embedding and storage:

```go
func (r *RAGSystem) IngestBatch(ctx context.Context, docs []schema.Document) error {
    batchSize := 100
    for i := 0; i < len(docs); i += batchSize {
        end := min(i+batchSize, len(docs))
        batch := docs[i:end]

        texts := make([]string, len(batch))
        for j, doc := range batch {
            texts[j] = doc.Content
        }

        embeddings, err := r.embedder.Embed(ctx, texts)
        if err != nil {
            return fmt.Errorf("embed batch %d: %w", i/batchSize, err)
        }

        if err := r.store.Add(ctx, batch, embeddings); err != nil {
            return fmt.Errorf("store batch %d: %w", i/batchSize, err)
        }
    }
    return nil
}
```

### Scaling

- **Horizontal scaling**: Deploy multiple RAG service instances behind a load balancer. Each instance is stateless.
- **Vector store scaling**: Use read replicas for pgvector or managed services like Pinecone for automatic scaling.
- **Embedding caching**: Cache embeddings for frequently queried terms to reduce API calls and latency.
- **Chunk size tuning**: Experiment with chunk sizes (256-1024 tokens) and overlap (10-20%) for your document types.

### Security

- Validate and sanitize user queries before passing to the retriever
- Implement document-level access control by filtering on metadata during retrieval
- Use Beluga AI's `guard/` pipeline to screen LLM inputs and outputs for PII leakage
- Store API keys in environment variables or a secrets manager, never in code

## Related Resources

- [RAG Pipeline Guide](/docs/guides/rag-pipeline/) for step-by-step setup
- [RAG Recipes](/docs/cookbook/rag-recipes/) for advanced retrieval patterns
- [Vector Store Integration](/docs/integrations/vector-stores/) for provider-specific configuration
