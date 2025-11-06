# RAG Concepts

This document explains Retrieval-Augmented Generation (RAG) in Beluga AI, including embeddings, vector stores, retrievers, and document processing.

## What is RAG?

RAG (Retrieval-Augmented Generation) combines:
1. **Retrieval**: Finding relevant documents
2. **Augmentation**: Adding context to prompts
3. **Generation**: Creating responses with context

## Embeddings

Embeddings are vector representations of text.

### Generating Embeddings

```go
embedder, _ := embeddings.NewEmbedderFactory(config)
embedderInstance, _ := embedder.NewEmbedder("openai")

// Embed documents
texts := []string{"Document 1", "Document 2"}
embeddings, _ := embedderInstance.EmbedDocuments(ctx, texts)

// Embed query
query := "search query"
queryEmbedding, _ := embedderInstance.EmbedQuery(ctx, query)
```

### Embedding Models

- **OpenAI**: `text-embedding-ada-002`, `text-embedding-3-small`
- **Ollama**: Local embedding models
- **Custom**: Your own embedding models

## Vector Stores

Vector stores enable similarity search.

### Adding Documents

```go
documents := []schema.Document{
    schema.NewDocument("Content 1", map[string]string{"source": "doc1"}),
    schema.NewDocument("Content 2", map[string]string{"source": "doc2"}),
}

ids, err := store.AddDocuments(ctx, documents,
    vectorstores.WithEmbedder(embedder),
)
```

### Similarity Search

```go
query := "search query"
docs, scores, err := store.SimilaritySearchByQuery(ctx, query, 5, embedder)
```

### Vector Store Providers

- **InMemory**: Fast, in-memory storage
- **PgVector**: PostgreSQL with pgvector extension
- **Pinecone**: Managed vector database

## Retrievers

Retrievers provide document retrieval strategies.

### Vector Store Retriever

```go
retriever, err := retrievers.NewVectorStoreRetriever(
    vectorStore,
    retrievers.WithDefaultK(5),
    retrievers.WithScoreThreshold(0.7),
)
```

### Retrieval Methods

- **Similarity search**: Cosine similarity
- **MMR**: Maximum marginal relevance
- **Metadata filtering**: Filter by metadata

## Document Processing Pipeline

### 1. Document Loading

```go
// Load documents from various sources
documents := loadDocuments(source)
```

### 2. Text Splitting

```go
// Split large documents into chunks
chunks := splitText(documents, chunkSize, overlap)
```

### 3. Embedding Generation

```go
embeddings, _ := embedder.EmbedDocuments(ctx, chunks)
```

### 4. Vector Storage

```go
store.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))
```

### 5. Retrieval

```go
relevantDocs, _ := retriever.GetRelevantDocuments(ctx, query)
```

### 6. Generation

```go
context := buildContext(relevantDocs)
response, _ := llm.Generate(ctx, messagesWithContext)
```

## Chunking Strategies

### Fixed Size Chunking

```go
chunks := fixedSizeChunk(text, chunkSize, overlap)
```

### Semantic Chunking

```go
chunks := semanticChunk(text, minSize, maxSize)
```

### Recursive Chunking

```go
chunks := recursiveChunk(text, separators)
```

## RAG Best Practices

1. **Chunk size**: Balance between context and granularity
2. **Overlap**: Add overlap between chunks
3. **Metadata**: Include metadata for filtering
4. **Retrieval count**: Retrieve enough documents for context
5. **Context building**: Build clear context from retrieved docs

## Complete RAG Pipeline

```go
// 1. Setup components
embedder := setupEmbedder(ctx)
store := setupVectorStore(ctx, embedder)
llm := setupLLM(ctx)

// 2. Add documents
store.AddDocuments(ctx, documents, vectorstores.WithEmbedder(embedder))

// 3. Query
query := "user question"
docs, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder)

// 4. Build context
context := buildContext(docs)

// 5. Generate
messages := []schema.Message{
    schema.NewSystemMessage("Answer using: " + context),
    schema.NewHumanMessage(query),
}
response, _ := llm.Generate(ctx, messages)
```

## Related Concepts

- [LLM Concepts](./llms.md) - LLM integration
- [Memory Concepts](./memory.md) - Vector store memory
- [Provider Documentation](../providers/) - Provider guides

---

**Next:** Learn about [Orchestration Concepts](./orchestration.md) or [Getting Started Tutorial](../getting-started/)

