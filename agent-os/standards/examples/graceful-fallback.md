# Graceful Fallback Pattern

Examples work with or without optional resources.

```go
// Try real approach first
dataDir := "examples/rag/simple/data"
if _, err := os.Stat(dataDir); err == nil {
    fmt.Println("Loading documents from directory...")
    loader, err := documentloaders.NewDirectoryLoader(os.DirFS(dataDir),
        documentloaders.WithExtensions(".txt", ".md"),
    )
    if err == nil {
        loadedDocs, err := loader.Load(ctx)
        if err == nil && len(loadedDocs) > 0 {
            documents = loadedDocs
            fmt.Printf("Loaded %d documents from directory\n", len(documents))
        }
    }
}

// Fallback to manual documents if directory doesn't exist
if len(documents) == 0 {
    fmt.Println("Using manual documents (create data/ for file-based loading)")
    documents = []schema.Document{
        schema.NewDocument("AI is the simulation of human intelligence...", nil),
        schema.NewDocument("Machine Learning is a subset of AI...", nil),
    }
}
```

## API Key Fallback
```go
func createEmbedder(ctx context.Context) (Embedder, error) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Println("OPENAI_API_KEY not set, using mock embedder")
        return embeddings.NewEmbedder(ctx, "mock", mockConfig)
    }
    return embeddings.NewEmbedder(ctx, "openai", openaiConfig)
}
```

## Guidelines
- Always provide working fallback (mocks, inline data)
- Print clear message explaining what's happening
- Tell user how to enable full functionality
- Never fail silently - always log the fallback path
