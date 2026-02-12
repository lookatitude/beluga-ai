---
title: Intelligent Document Processing Pipeline
description: "Automate document classification, entity extraction, and semantic search with LLM-powered pipelines and vector indexing."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "intelligent document processing, IDP, document classification, entity extraction, semantic search, Beluga AI, Go, RAG pipeline"
---

Organizations process thousands of documents daily — contracts, invoices, reports, emails, and regulatory filings. Manual processing creates bottlenecks: a single analyst can classify and extract data from 20-30 documents per hour, accuracy varies with fatigue, and documents sit in queues for days before reaching the right department. Searching historical documents means knowing which folder to look in, making institutional knowledge fragile and person-dependent.

An intelligent document processing (IDP) pipeline automates extraction, classification, and indexing, making documents instantly searchable by meaning rather than filename. The pipeline turns unstructured documents into structured, queryable data within seconds of ingestion.

## Solution Architecture

The pipeline chains four stages: load documents from any source, classify them using an LLM, extract structured entities using structured output, and index them in a vector store for semantic retrieval. Each stage is a `core.Runnable` step composed into an orchestration chain. This staged design allows each component to be tested independently and replaced without affecting the rest of the pipeline — for example, swapping pgvector for Qdrant changes only the vector store provider, not the classification or extraction logic.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Document    │───▶│  Text        │───▶│  LLM         │───▶│  Structured  │
│  Loaders     │    │  Splitters   │    │  Classifier  │    │  Extraction  │
│  (PDF, HTML, │    │  (Recursive, │    │  (Category,  │    │  (Entities,  │
│   Markdown)  │    │   Semantic)  │    │   Tags)      │    │   Fields)    │
└──────────────┘    └──────────────┘    └──────────────┘    └──────┬───────┘
                                                                   │
                                                                   ▼
                                                            ┌──────────────┐
                                                            │  VectorStore │
                                                            │  (Index for  │
                                                            │   Search)    │
                                                            └──────────────┘
```

## Document Loading

Beluga AI's loader package supports multiple document formats through its registry pattern (`Register()` + `New()` + `List()`). Each format is a provider that registers via `init()`, so adding PDF, HTML, or CSV support is a matter of importing the provider package — no factory configuration or manual wiring required.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/pdf"
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/html"
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/csv"
)

func loadDocuments(ctx context.Context, source string) ([]schema.Document, error) {
    // List available loaders
    available := loader.List() // ["pdf", "html", "csv", ...]

    pdfLoader, err := loader.New("pdf", nil)
    if err != nil {
        return nil, fmt.Errorf("create pdf loader: %w", err)
    }

    docs, err := pdfLoader.Load(ctx, source)
    if err != nil {
        return nil, fmt.Errorf("load documents: %w", err)
    }

    log.Printf("Loaded %d documents from %s", len(docs), source)
    return docs, nil
}
```

## LLM-Powered Classification

Rule-based classifiers require maintaining pattern lists for every document type and break when formats change. LLM-based classification adapts to new document types without rule updates. Using `llm.NewStructured[T]` guarantees the response matches the `Classification` struct schema — the LLM returns typed Go data, not raw text that needs parsing and validation:

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

type Classification struct {
    Category string   `json:"category" jsonschema:"enum=invoice,contract,report,email,other"`
    Tags     []string `json:"tags" jsonschema:"description=Relevant tags for the document"`
    Language string   `json:"language" jsonschema:"description=ISO 639-1 language code"`
}

func classifyDocument(ctx context.Context, model llm.ChatModel, content string) (Classification, error) {
    structured := llm.NewStructured[Classification](model)

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Classify this document. Determine its category, " +
                "extract relevant tags, and identify the language."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: content[:min(len(content), 2000)]},
        }},
    }

    result, err := structured.Generate(ctx, msgs)
    if err != nil {
        return Classification{}, fmt.Errorf("classify: %w", err)
    }

    return result, nil
}
```

## Entity Extraction

Extract structured entities from documents — amounts, dates, parties, and custom fields — using structured output with a schema that matches your domain. The struct's JSON tags define the extraction schema, so the LLM knows exactly what fields to populate and in what format:

```go
type InvoiceEntities struct {
    InvoiceNumber string  `json:"invoice_number"`
    Vendor        string  `json:"vendor"`
    Amount        float64 `json:"amount"`
    Currency      string  `json:"currency"`
    DueDate       string  `json:"due_date"`
    LineItems     []LineItem `json:"line_items"`
}

type LineItem struct {
    Description string  `json:"description"`
    Quantity    int     `json:"quantity"`
    UnitPrice   float64 `json:"unit_price"`
}

func extractInvoiceEntities(ctx context.Context, model llm.ChatModel, content string) (InvoiceEntities, error) {
    structured := llm.NewStructured[InvoiceEntities](model)

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Extract invoice details from this document. " +
                "Be precise with amounts and dates."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: content},
        }},
    }

    return structured.Generate(ctx, msgs)
}
```

## Complete Processing Pipeline

Combine loading, splitting, classification, extraction, and indexing into a single pipeline:

```go
import (
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/splitter/providers/recursive"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

type DocumentPipeline struct {
    loader    loader.DocumentLoader
    splitter  splitter.TextSplitter
    embedder  embedding.Embedder
    store     vectorstore.VectorStore
    model     llm.ChatModel
}

func (p *DocumentPipeline) Process(ctx context.Context, source string) error {
    // 1. Load
    docs, err := p.loader.Load(ctx, source)
    if err != nil {
        return fmt.Errorf("load: %w", err)
    }

    // 2. Split
    chunks, err := p.splitter.SplitDocuments(ctx, docs)
    if err != nil {
        return fmt.Errorf("split: %w", err)
    }

    // 3. Classify and enrich metadata
    for i, chunk := range chunks {
        classification, err := classifyDocument(ctx, p.model, chunk.Content)
        if err != nil {
            log.Printf("classify chunk %d: %v", i, err)
            continue
        }
        chunks[i].Metadata["category"] = classification.Category
        chunks[i].Metadata["language"] = classification.Language
        for _, tag := range classification.Tags {
            chunks[i].Metadata["tag_"+tag] = "true"
        }
    }

    // 4. Embed
    texts := make([]string, len(chunks))
    for i, chunk := range chunks {
        texts[i] = chunk.Content
    }

    embeddings, err := p.embedder.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("embed: %w", err)
    }

    // 5. Store
    if err := p.store.Add(ctx, chunks, embeddings); err != nil {
        return fmt.Errorf("store: %w", err)
    }

    log.Printf("Processed %d chunks from %s", len(chunks), source)
    return nil
}
```

## Filtered Semantic Search

Once documents are indexed, search by content meaning and filter by classification metadata:

```go
func (p *DocumentPipeline) Search(ctx context.Context, query string, category string) ([]schema.Document, error) {
    queryEmbedding, err := p.embedder.EmbedSingle(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }

    results, err := p.store.Search(ctx, queryEmbedding, 10,
        vectorstore.WithFilter(map[string]any{"category": category}),
        vectorstore.WithThreshold(0.7),
    )
    if err != nil {
        return nil, fmt.Errorf("search: %w", err)
    }

    return results, nil
}
```

## Batch Processing with Concurrency

Process large document collections with bounded concurrency:

```go
func (p *DocumentPipeline) ProcessBatch(ctx context.Context, sources []string) error {
    sem := make(chan struct{}, 10) // Limit concurrent processing
    var mu sync.Mutex
    var errs []error

    var wg sync.WaitGroup
    for _, source := range sources {
        wg.Add(1)
        go func(src string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            if err := p.Process(ctx, src); err != nil {
                mu.Lock()
                errs = append(errs, fmt.Errorf("%s: %w", src, err))
                mu.Unlock()
            }
        }(source)
    }
    wg.Wait()

    if len(errs) > 0 {
        return fmt.Errorf("batch processing had %d errors: %w", len(errs), errs[0])
    }
    return nil
}
```

## Production Considerations

### Observability

Track document processing throughput, classification accuracy, and embedding latency:

```go
tracer := otel.Tracer("document-processor")
ctx, span := tracer.Start(ctx, "document.process")
defer span.End()

span.SetAttributes(
    attribute.String("document.source", source),
    attribute.Int("document.chunk_count", len(chunks)),
    attribute.String("document.category", classification.Category),
)
```

### Error Handling

- **Loader failures**: Skip individual documents and continue batch processing. Log failures for retry.
- **Classification failures**: Fall back to "uncategorized" rather than blocking the pipeline.
- **Embedding API rate limits**: Use Beluga AI's `resilience.Retry` with exponential backoff.
- **Storage failures**: Use transactions where supported (pgvector). Buffer and retry on transient errors.

### Scaling

- Process documents in parallel batches with bounded concurrency
- Use connection pooling for the vector store (pgxpool for PostgreSQL)
- Cache embeddings for duplicate content using content hashes
- For very large collections (millions of documents), partition the vector store by category or date range
- Consider async processing with a message queue for non-blocking ingestion

### Security

- Validate document sources before loading (prevent path traversal)
- Sanitize extracted text before passing to the LLM
- Apply document-level access control through vector store metadata filters
- Use the guard pipeline to prevent sensitive data from reaching the LLM

## Related Resources

- [Enterprise RAG Knowledge Base](/use-cases/enterprise-rag/) for the query side of the pipeline
- [RAG Pipeline Guide](/guides/rag-pipeline/) for detailed RAG configuration
- [LLM Recipes](/cookbook/llm-recipes/) for extraction patterns
