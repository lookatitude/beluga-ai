---
title: Rag
sidebar_position: 1
---

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

Load documents from files, directories, or other sources using document loaders:

```go
import (
    "os"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

// Load from directory
fsys := os.DirFS("./data")
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithExtensions(".txt", ".md"),
    documentloaders.WithMaxDepth(2),
)
documents, _ := loader.Load(ctx)

// Or load single file
textLoader, _ := documentloaders.NewTextLoader("./document.txt")
documents, _ := textLoader.Load(ctx)
```

**See:** [Document Loading Concepts](./document-loading.md) for detailed information.

### 2. Text Splitting

Split large documents into chunks that fit embedding model context windows:

```go
import "github.com/lookatitude/beluga-ai/pkg/textsplitters"

// Recursive character splitting (general purpose)
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
chunks, _ := splitter.SplitDocuments(ctx, documents)

// Markdown-aware splitting
markdownSplitter, _ := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithHeadersToSplitOn("#", "##"),
)
chunks, _ := markdownSplitter.SplitDocuments(ctx, documents)
```

**See:** [Text Splitting Concepts](./text-splitting.md) for detailed information.

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

### Recursive Character Splitting

Uses a hierarchy of separators (paragraphs → lines → words → characters):

```go
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
    textsplitters.WithSeparators("\n\n", "\n", " ", ""),
)
chunks, _ := splitter.SplitText(ctx, text)
```

### Markdown-Aware Splitting

Respects markdown structure, splitting at headers and preserving code blocks:

```go
splitter, _ := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
chunks, _ := splitter.SplitText(ctx, markdownText)
```

### Token-Based Splitting

Use custom length functions for token-aware chunking:

```go
tokenizer := func(text string) int {
    // Implement token counting
    return len(strings.Fields(text))
}

splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveLengthFunction(tokenizer),
    textsplitters.WithRecursiveChunkSize(100), // 100 tokens
)
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

// 2. Load documents
loader, _ := documentloaders.NewDirectoryLoader(os.DirFS("./data"))
documents, _ := loader.Load(ctx)

// 3. Split into chunks
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
chunks, _ := splitter.SplitDocuments(ctx, documents)

// 4. Add to vector store
store.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))

// 5. Query
query := "user question"
docs, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder)

// 6. Build context
context := buildContext(docs)

// 7. Generate
messages := []schema.Message{
    schema.NewSystemMessage("Answer using: " + context),
    schema.NewHumanMessage(query),
}
response, _ := llm.Generate(ctx, messages)
```

## Related Concepts

- [Document Loading Concepts](./document-loading.md) - Loading documents from files and directories
- [Text Splitting Concepts](./text-splitting.md) - Splitting documents into chunks
- [LLM Concepts](./llms) - LLM integration
- [Memory Concepts](./memory) - Vector store memory
- [Provider Documentation](../../providers/) - Provider guides

---

**Next:** Learn about [Orchestration Concepts](./orchestration) or [Getting Started Tutorial](../../getting-started/)

