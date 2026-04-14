---
title: "Document Ingestion Recipes"
description: "Go recipes for RAG document ingestion: load from files, split into chunks, process metadata, and build complete ingestion pipelines with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, document ingestion, Go RAG loader, text splitting, document processing, ingestion pipeline, DocumentLoader recipe"
---

Document ingestion is the first stage of any RAG pipeline: getting documents from their source format into chunks that can be embedded and stored. The quality of ingestion directly affects retrieval quality downstream. Poorly split documents produce poor embeddings; missing metadata means no filtering capability; sequential loading wastes time on large collections.

## Problem

You need to load documents from various sources, split them into chunks, and prepare them for embedding and storage in a RAG pipeline.

## Solution

Use Beluga AI's document loaders and text splitters to build composable ingestion pipelines. The framework provides the `DocumentLoader` and `TextSplitter` interfaces with multiple implementations. Combine loaders, splitters, and a pipeline to handle diverse document sources efficiently.

## Code Example

### Loading a Text File

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/loader"
)

func main() {
    ctx := context.Background()

    l := loader.NewTextLoader()

    docs, err := l.Load(ctx, "./data/architecture.md")
    if err != nil {
        log.Fatalf("load failed: %v", err)
    }

    fmt.Printf("Loaded %d document(s)\n", len(docs))
}
```

### Loading Multiple File Types

Each loader handles a specific format. Use `NewTextLoader` for plain text and Markdown,
`NewCSVLoader` for tabular data, `NewJSONLoader` for JSON, and `NewMarkdownLoader` for
Markdown with header-based section splitting:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/schema"
)

func loadSources(ctx context.Context, sources map[string]loader.DocumentLoader) ([]schema.Document, error) {
    var all []schema.Document
    for path, l := range sources {
        docs, err := l.Load(ctx, path)
        if err != nil {
            return nil, fmt.Errorf("load %s: %w", path, err)
        }
        all = append(all, docs...)
    }
    return all, nil
}

func main() {
    ctx := context.Background()

    sources := map[string]loader.DocumentLoader{
        "./docs/readme.md":     loader.NewMarkdownLoader(),
        "./data/records.csv":   loader.NewCSVLoader(),
        "./config/schema.json": loader.NewJSONLoader(),
    }

    docs, err := loadSources(ctx, sources)
    if err != nil {
        log.Fatalf("load failed: %v", err)
    }

    fmt.Printf("Loaded %d document(s)\n", len(docs))
}
```

### Splitting Documents into Chunks

Use `splitter.NewRecursiveSplitter` to divide documents into chunks. The splitter tries
separators from most significant (paragraph break) to least significant (character-level),
keeping chunks near the target size:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/rag/splitter"
)

func main() {
    ctx := context.Background()

    l := loader.NewTextLoader()
    docs, err := l.Load(ctx, "./data/architecture.md")
    if err != nil {
        log.Fatalf("load failed: %v", err)
    }

    s := splitter.NewRecursiveSplitter(
        splitter.WithChunkSize(1000),
        splitter.WithChunkOverlap(200),
    )

    chunks, err := s.SplitDocuments(ctx, docs)
    if err != nil {
        log.Fatalf("split failed: %v", err)
    }

    fmt.Printf("Split into %d chunk(s)\n", len(chunks))
    for i, c := range chunks {
        fmt.Printf("  Chunk %d: %d chars\n", i, len(c.Content))
    }
}
```

### Markdown-Aware Splitting

`NewMarkdownSplitter` understands Markdown heading structure and splits at heading
boundaries first, preserving each section as a coherent chunk:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/rag/splitter"
)

func main() {
    ctx := context.Background()

    l := loader.NewMarkdownLoader()
    docs, err := l.Load(ctx, "./docs/guide.md")
    if err != nil {
        log.Fatalf("load failed: %v", err)
    }

    s := splitter.NewMarkdownSplitter(
        splitter.WithMarkdownChunkSize(500),
        splitter.WithMarkdownChunkOverlap(50),
    )

    chunks, err := s.SplitDocuments(ctx, docs)
    if err != nil {
        log.Fatalf("split failed: %v", err)
    }

    for _, c := range chunks {
        fmt.Printf("Section: %v — %d chars\n", c.Metadata["heading"], len(c.Content))
    }
}
```

### Complete Ingestion Pipeline

Load, split, embed, and store in a three-stage pipeline:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
)

func ingest(ctx context.Context, paths []string) error {
    // 1. Load
    l := loader.NewTextLoader()
    var all []schema.Document
    for _, p := range paths {
        docs, err := l.Load(ctx, p)
        if err != nil {
            return fmt.Errorf("load %s: %w", p, err)
        }
        all = append(all, docs...)
    }

    // 2. Split
    s := splitter.NewRecursiveSplitter(
        splitter.WithChunkSize(1000),
        splitter.WithChunkOverlap(200),
    )
    chunks, err := s.SplitDocuments(ctx, all)
    if err != nil {
        return fmt.Errorf("split: %w", err)
    }

    // 3. Embed
    emb, err := embedding.New("openai", config.ProviderConfig{
        Provider: "openai",
        APIKey:   os.Getenv("OPENAI_API_KEY"),
        Model:    "text-embedding-3-small",
    })
    if err != nil {
        return fmt.Errorf("embedder: %w", err)
    }

    texts := make([]string, len(chunks))
    for i, c := range chunks {
        texts[i] = c.Content
    }
    embeddings, err := emb.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("embed: %w", err)
    }

    // 4. Store
    store, err := vectorstore.New("pgvector", config.ProviderConfig{
        Provider: "pgvector",
        Options:  map[string]any{"connection_string": os.Getenv("PGVECTOR_DSN")},
    })
    if err != nil {
        return fmt.Errorf("vectorstore: %w", err)
    }

    if err := store.Add(ctx, chunks, embeddings); err != nil {
        return fmt.Errorf("add: %w", err)
    }

    fmt.Printf("Ingested %d chunks from %d document(s)\n", len(chunks), len(all))
    return nil
}

func main() {
    if err := ingest(context.Background(), []string{
        "./docs/readme.md",
        "./docs/reference/architecture/overview.md",
    }); err != nil {
        log.Fatalf("ingestion failed: %v", err)
    }
}
```

## Explanation

1. **Loaders by format** -- Each loader implementation handles a specific source format. `NewTextLoader` reads raw text, `NewMarkdownLoader` parses Markdown structure, `NewCSVLoader` converts rows to documents (one per row), and `NewJSONLoader` extracts JSON fields as document content. Use the loader that matches your source format.

2. **Splitter choice** -- `NewRecursiveSplitter` is the general-purpose choice: it tries paragraph breaks, line breaks, then whitespace, ensuring splits happen at natural boundaries. `NewMarkdownSplitter` understands Markdown heading hierarchy and preserves section structure in chunk metadata. `NewTokenSplitter` counts tokens rather than characters, which is more accurate for embedding models with strict token limits.

3. **Embedder and store** -- Both `embedding.New` and `vectorstore.New` use the registry pattern with `config.ProviderConfig`. Provider credentials come from environment variables, never hardcoded.

4. **Pipeline composition** -- Load, split, embed, and store are independent stages. You can replace any stage (e.g., swap the vector store provider) without changing the others. This composability is the primary benefit of the registry pattern.

## Variations

### Processing Different File Types Separately

```go
var textDocs, mdDocs []schema.Document
for _, doc := range allDocs {
    if src, _ := doc.Metadata["source"].(string); strings.HasSuffix(src, ".md") {
        mdDocs = append(mdDocs, doc)
    } else {
        textDocs = append(textDocs, doc)
    }
}

textChunks, err := splitter.NewRecursiveSplitter().SplitDocuments(ctx, textDocs)
if err != nil {
    log.Fatalf("text split: %v", err)
}
mdChunks, err := splitter.NewMarkdownSplitter().SplitDocuments(ctx, mdDocs)
if err != nil {
    log.Fatalf("markdown split: %v", err)
}
```

### Batch Processing for Memory Control

```go
const batchSize = 100
for i := 0; i < len(docs); i += batchSize {
    end := i + batchSize
    if end > len(docs) {
        end = len(docs)
    }
    batch := docs[i:end]
    chunks, err := s.SplitDocuments(ctx, batch)
    if err != nil {
        log.Printf("batch %d split failed: %v", i/batchSize, err)
        continue
    }
    processBatch(chunks)
}
```

## Related Recipes

- [Parallel File Loading](/docs/recipes/rag/parallel-file-loading) -- Concurrent file loading with bounded parallelism
- [Corrupt Document Handling](/docs/recipes/rag/corrupt-doc-handling) -- Graceful error handling for corrupt documents
- [Sentence-Aware Splitting](/docs/recipes/rag/sentence-splitting) -- Sentence-boundary-aware text splitting
- [Code Splitting](/docs/recipes/rag/code-splitting) -- Language-aware code splitting
