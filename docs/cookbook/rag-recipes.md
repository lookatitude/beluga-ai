# RAG Recipes

Common RAG (Retrieval-Augmented Generation) patterns and recipes.

## Basic RAG Pipeline

```go
// Setup
embedder := setupEmbedder(ctx)
store := setupVectorStore(ctx, embedder)
llm := setupLLM(ctx)

// Ingest documents
store.AddDocuments(ctx, documents, vectorstores.WithEmbedder(embedder))

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

---

**More Recipes:** [Agent Recipes](./agent-recipes.md) | [Tool Recipes](./tool-recipes.md)

