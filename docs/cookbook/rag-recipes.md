# RAG Recipes

Common RAG (Retrieval-Augmented Generation) patterns and recipes.

## Basic RAG Pipeline

```go
// Setup
embedder := setupEmbedder(ctx)
store := setupVectorStore(ctx, embedder)
llm := setupLLM(ctx)

// Load and split documents
loader, _ := documentloaders.NewDirectoryLoader(os.DirFS("./data"))
docs, _ := loader.Load(ctx)

splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
chunks, _ := splitter.SplitDocuments(ctx, docs)

// Ingest documents
store.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))

// Query
docs, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder)
context := buildContext(docs)

// Generate
messages := []schema.Message{
    schema.NewSystemMessage("Answer using: " + context),
    schema.NewHumanMessage(query),
}
response, _ := llm.Generate(ctx, messages)
```

## RAG with Memory

```go
mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)
// Use memory to maintain conversation context
```

## RAG with Metadata Filtering

```go
docs, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder,
    vectorstores.WithMetadataFilter("category", "tech"),
)
```

## Batch RAG Processing

```go
queries := []string{"query1", "query2", "query3"}
for _, query := range queries {
    docs, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder)
    // Process each query
}
```

## RAG with Document Loaders

Use document loaders for real-world document ingestion:

```go
// Load from directory
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("./docs"),
    documentloaders.WithExtensions(".txt", ".md"),
    documentloaders.WithMaxDepth(2),
)
docs, _ := loader.Load(ctx)

// Split into chunks
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
chunks, _ := splitter.SplitDocuments(ctx, docs)

// Add to vector store
store.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))
```

**See:** [Document Ingestion Recipes](./document-ingestion-recipes.md) for more patterns.

---

**More Recipes:** [Document Ingestion Recipes](./document-ingestion-recipes.md) | [Agent Recipes](./agent-recipes.md) | [Tool Recipes](./tool-recipes.md)

