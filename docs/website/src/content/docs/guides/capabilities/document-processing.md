---
title: Document Loading & Processing
description: "Load, parse, and chunk documents from files, PDFs, HTML, and cloud storage for RAG pipelines — with lazy loading, semantic splitting, and metadata preservation in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, document loading, text splitting, PDF, chunking, RAG pipeline, semantic splitting"
---

Before documents can be embedded and searched in a RAG pipeline, they need to be loaded from their source format, cleaned, and split into chunks that embedding models can process. This ingestion stage is the foundation of any knowledge-powered AI system — the quality of your chunking strategy directly affects retrieval precision and, consequently, the quality of generated answers.

Beluga AI provides document loaders for multiple formats (text, PDF, HTML, CSV, cloud storage) and intelligent text splitters that preserve semantic structure. The entire pipeline is designed for production scale: lazy loading for memory efficiency, batch processing for throughput, and comprehensive error handling for reliability.

## What You'll Learn

This guide covers:
- Loading documents from multiple sources (filesystem, PDF, HTML, cloud storage)
- Using lazy loading for large datasets that exceed available memory
- Text splitting strategies and when to use each (recursive, semantic, markdown-aware, code-aware)
- Preserving document structure and metadata through the pipeline
- Batch processing with error handling and progress tracking
- Optimizing chunk size and overlap for retrieval quality

## When Document Processing Matters

Document processing is the first step in building any knowledge-powered AI feature:
- **RAG systems** that need searchable knowledge bases built from internal documents
- **Document intelligence** extracting structured data from files (invoices, contracts, reports)
- **Content migration** moving legacy document stores to AI-powered semantic search
- **Compliance** indexing regulated documents with metadata for auditing and access control
- **Knowledge management** making organizational knowledge discoverable through natural language

## Prerequisites

Before starting this guide:
- Complete [RAG Pipeline](/guides/rag-pipeline) to understand where document processing fits in the pipeline
- Understand vector embeddings and why documents need to be chunked
- Familiarity with file I/O in Go

## Document Loaders

Document loaders abstract the details of reading and parsing different file formats into a uniform `[]schema.Document` output. Each loader handles format-specific concerns — PDF text extraction, HTML tag stripping, CSV row parsing — so that downstream pipeline stages (splitting, embedding, storing) work with clean, structured documents regardless of the original format.

### Directory Loader

The directory loader recursively walks a filesystem tree, loading all matching files. It supports extension filtering, path exclusions, depth limits, and parallel loading. This is the typical starting point for building a knowledge base from a collection of local files.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func LoadDirectory(dirPath string) ([]schema.Document, error) {
    ctx := context.Background()

    // Create filesystem
    fsys := os.DirFS(dirPath)

    // Configure directory loader
    loader, err := documentloaders.NewDirectoryLoader(fsys,
        documentloaders.WithMaxDepth(10),
        documentloaders.WithExtensions(".md", ".txt", ".pdf", ".html"),
        documentloaders.WithExclusions("**/node_modules/**", "**/.git/**", "**/.DS_Store"),
        documentloaders.WithConcurrency(4), // Parallel loading
    )
    if err != nil {
        return nil, fmt.Errorf("create loader: %w", err)
    }

    // Load all documents
    docs, err := loader.Load(ctx)
    if err != nil {
        return nil, fmt.Errorf("load documents: %w", err)
    }

    fmt.Printf("Loaded %d documents\n", len(docs))
    return docs, nil
}

func main() {
    docs, err := LoadDirectory("./knowledge_base")
    if err != nil {
        log.Fatal(err)
    }

    for _, doc := range docs {
        fmt.Printf("Document: %s (%d bytes)\n",
            doc.Metadata["source"],
            len(doc.PageContent),
        )
    }
}
```

### PDF Loader

PDF is one of the most common document formats in enterprise environments, but extracting clean text from PDFs is notoriously difficult due to complex layouts, embedded fonts, and mixed content. The PDF loader handles text extraction with options for formatting preservation and page separation.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/pdf"
)

func LoadPDF(filePath string) ([]schema.Document, error) {
    ctx := context.Background()

    loader := pdf.NewLoader(filePath,
        pdf.WithExtractImages(false), // Skip images
        pdf.WithPreserveFormatting(true),
        pdf.WithPageSeparator("\n---\n"),
    )

    docs, err := loader.Load(ctx)
    if err != nil {
        return nil, fmt.Errorf("load PDF: %w", err)
    }

    return docs, nil
}
```

### HTML Loader

Web pages contain significant noise — scripts, stylesheets, navigation elements — that degrades embedding quality. The HTML loader strips non-content elements and extracts clean text, optionally preserving page metadata (title, description, Open Graph tags) that can be used for filtering during retrieval.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/html"
)

func LoadHTML(url string) ([]schema.Document, error) {
    ctx := context.Background()

    loader := html.NewLoader(url,
        html.WithRemoveScripts(true),
        html.WithRemoveStyles(true),
        html.WithExtractMetadata(true), // Extract title, description, etc.
        html.WithFollowLinks(false),
    )

    docs, err := loader.Load(ctx)
    if err != nil {
        return nil, fmt.Errorf("load HTML: %w", err)
    }

    return docs, nil
}
```

### S3 Loader

For cloud-native applications, documents often reside in object storage rather than local filesystems. The S3 loader handles bucket traversal, prefix filtering, and file type selection, integrating with the AWS SDK for authentication and access control.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/s3"
    "github.com/aws/aws-sdk-go/aws/session"
)

func LoadFromS3(bucket, prefix string) ([]schema.Document, error) {
    ctx := context.Background()

    sess, err := session.NewSession()
    if err != nil {
        return nil, err
    }

    loader := s3.NewLoader(sess, bucket,
        s3.WithPrefix(prefix),
        s3.WithRecursive(true),
        s3.WithIncludeExtensions(".txt", ".md", ".pdf"),
    )

    docs, err := loader.Load(ctx)
    if err != nil {
        return nil, fmt.Errorf("load from S3: %w", err)
    }

    return docs, nil
}
```

## Lazy Loading for Large Datasets

Loading millions of documents into memory at once is not feasible — it would exhaust available RAM and potentially crash the process. Lazy loading solves this by returning an iterator that yields documents one at a time, processing each before loading the next. This keeps memory usage constant regardless of dataset size, making it suitable for ingesting entire document repositories.

```go
func ProcessLargeDataset(dirPath string) error {
    ctx := context.Background()

    fsys := os.DirFS(dirPath)
    loader, err := documentloaders.NewDirectoryLoader(fsys)
    if err != nil {
        return err
    }

    // LazyLoad returns an iterator
    docIterator, err := loader.LazyLoad(ctx)
    if err != nil {
        return err
    }

    processed := 0
    for doc := range docIterator {
        if doc.Error != nil {
            log.Printf("Error loading document: %v", doc.Error)
            continue
        }

        // Process one document at a time
        if err := processDocument(ctx, doc.Document); err != nil {
            log.Printf("Error processing %s: %v", doc.Document.Metadata["source"], err)
            continue
        }

        processed++
        if processed%100 == 0 {
            fmt.Printf("Processed %d documents\n", processed)
        }
    }

    fmt.Printf("Total processed: %d documents\n", processed)
    return nil
}

func processDocument(ctx context.Context, doc schema.Document) error {
    // Chunk, embed, and store
    chunks := splitDocument(doc)
    embeddings := embedChunks(ctx, chunks)
    return storeInVectorDB(ctx, chunks, embeddings)
}
```

## Text Splitting Strategies

Choosing the right splitting strategy is one of the most impactful decisions in a RAG pipeline. Different content types have different structure — prose paragraphs, markdown headings, code functions, semantic topics — and a splitter that respects that structure produces chunks that are more semantically coherent and easier to retrieve accurately.

### Recursive Character Splitter

The recursive character splitter is the recommended default for general-purpose text. It tries a hierarchy of separators (paragraph breaks, line breaks, spaces, then characters) and recursively splits at the highest-level separator that produces chunks within the size limit. The overlap parameter ensures that sentences at chunk boundaries are not lost.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

func SplitRecursive(doc schema.Document) ([]schema.Document, error) {
    splitter := textsplitters.NewRecursiveCharacterSplitter(
        textsplitters.WithChunkSize(1000),
        textsplitters.WithChunkOverlap(200), // Preserve context between chunks
        textsplitters.WithSeparators([]string{"\n\n", "\n", " ", ""}),
    )

    chunks, err := splitter.SplitDocuments([]schema.Document{doc})
    if err != nil {
        return nil, fmt.Errorf("split document: %w", err)
    }

    return chunks, nil
}
```

### Markdown-Aware Splitter

Markdown documents have explicit structure through headings. The markdown splitter leverages this structure by splitting at heading boundaries, ensuring that each chunk corresponds to a coherent section. When `IncludeHeaders` is enabled, each chunk includes its parent headings as context, so a chunk from a deeply nested section retains the full path of headings that scope its content.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/markdown"
)

func SplitMarkdown(doc schema.Document) ([]schema.Document, error) {
    splitter := markdown.NewSplitter(
        markdown.WithHeadersToSplitOn([]string{"#", "##", "###"}),
        markdown.WithIncludeHeaders(true), // Keep headers in chunks
        markdown.WithMaxChunkSize(1500),
    )

    chunks, err := splitter.SplitDocuments([]schema.Document{doc})
    if err != nil {
        return nil, fmt.Errorf("split markdown: %w", err)
    }

    return chunks, nil
}
```

### Code-Aware Splitter

Splitting code at arbitrary character boundaries breaks function definitions, class bodies, and control flow structures, producing chunks that are syntactically invalid and semantically incoherent. The code-aware splitter parses language-specific syntax and splits at function or class boundaries, keeping logical units intact. This produces chunks that are meaningful for code search and explanation tasks.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/code"
)

func SplitCode(doc schema.Document, language string) ([]schema.Document, error) {
    splitter := code.NewSplitter(language,
        code.WithChunkSize(1000),
        code.WithOverlap(100),
        code.WithPreserveFunctions(true), // Keep functions together
        code.WithPreserveClasses(true),   // Keep classes together
    )

    chunks, err := splitter.SplitDocuments([]schema.Document{doc})
    if err != nil {
        return nil, fmt.Errorf("split code: %w", err)
    }

    return chunks, nil
}
```

### Semantic Splitter

Character-based splitting ignores the actual meaning of the text. Two adjacent paragraphs about completely different topics get merged into one chunk, diluting the embedding. The semantic splitter uses an embedding model to detect topic boundaries by measuring similarity between consecutive text segments. When similarity drops below the threshold, a new chunk begins. This produces chunks that are topically coherent, improving retrieval precision at the cost of an extra embedding pass during indexing.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/semantic"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func SplitSemantic(ctx context.Context, doc schema.Document, embedder embeddings.Embedder) ([]schema.Document, error) {
    splitter := semantic.NewSplitter(embedder,
        semantic.WithBreakpointThreshold(0.75), // Similarity threshold
        semantic.WithMinChunkSize(200),
        semantic.WithMaxChunkSize(1500),
    )

    chunks, err := splitter.SplitDocuments(ctx, []schema.Document{doc})
    if err != nil {
        return nil, fmt.Errorf("split semantically: %w", err)
    }

    return chunks, nil
}
```

## Metadata Enrichment

Metadata attached to documents flows through the entire pipeline — from loading through splitting to storage. Rich metadata enables filtered searches (e.g., "find documents from the engineering team written in 2024"), debugging (tracing a chunk back to its source file), and analytics (monitoring which document types are most frequently retrieved). Investing in metadata enrichment early in the pipeline pays dividends in retrieval quality and operational visibility.

```go
func EnrichMetadata(doc schema.Document) schema.Document {
    // Add file metadata
    if source, ok := doc.Metadata["source"].(string); ok {
        doc.Metadata["file_extension"] = filepath.Ext(source)
        doc.Metadata["file_name"] = filepath.Base(source)
        doc.Metadata["directory"] = filepath.Dir(source)
    }

    // Add content metadata
    doc.Metadata["word_count"] = len(strings.Fields(doc.PageContent))
    doc.Metadata["char_count"] = len(doc.PageContent)
    doc.Metadata["line_count"] = strings.Count(doc.PageContent, "\n")

    // Add processing metadata
    doc.Metadata["processed_at"] = time.Now().Format(time.RFC3339)
    doc.Metadata["processor_version"] = "1.0"

    // Extract language (simple heuristic)
    doc.Metadata["language"] = detectLanguage(doc.PageContent)

    return doc
}

func detectLanguage(text string) string {
    // Simple detection based on keywords
    text = strings.ToLower(text)

    if strings.Contains(text, "func ") && strings.Contains(text, "package ") {
        return "go"
    }
    if strings.Contains(text, "def ") && strings.Contains(text, "import ") {
        return "python"
    }
    if strings.Contains(text, "function ") && strings.Contains(text, "const ") {
        return "javascript"
    }

    return "unknown"
}
```

## Batch Processing Pipeline

Production document ingestion requires batching for efficiency, error handling for resilience, and progress tracking for observability. Processing documents one at a time wastes API call overhead (embedding models accept batches), while loading everything at once risks memory exhaustion. The batch processing pattern below processes documents in configurable groups, accumulates statistics, and continues processing when individual documents fail.

```go
type DocumentProcessor struct {
    loader    documentloaders.Loader
    splitter  textsplitters.TextSplitter
    embedder  embeddings.Embedder
    vectorDB  vectorstore.VectorStore
    batchSize int
}

func NewDocumentProcessor(
    loader documentloaders.Loader,
    splitter textsplitters.TextSplitter,
    embedder embeddings.Embedder,
    vectorDB vectorstore.VectorStore,
) *DocumentProcessor {
    return &DocumentProcessor{
        loader:    loader,
        splitter:  splitter,
        embedder:  embedder,
        vectorDB:  vectorDB,
        batchSize: 10,
    }
}

type ProcessingStats struct {
    DocsLoaded    int
    DocsProcessed int
    ChunksCreated int
    Errors        []error
    StartTime     time.Time
    EndTime       time.Time
}

func (dp *DocumentProcessor) Process(ctx context.Context) (*ProcessingStats, error) {
    stats := &ProcessingStats{
        StartTime: time.Now(),
    }

    // Load documents
    docIterator, err := dp.loader.LazyLoad(ctx)
    if err != nil {
        return stats, fmt.Errorf("lazy load: %w", err)
    }

    var batch []schema.Document
    for docResult := range docIterator {
        if docResult.Error != nil {
            stats.Errors = append(stats.Errors, docResult.Error)
            continue
        }

        stats.DocsLoaded++
        batch = append(batch, docResult.Document)

        // Process in batches
        if len(batch) >= dp.batchSize {
            if err := dp.processBatch(ctx, batch, stats); err != nil {
                stats.Errors = append(stats.Errors, err)
            }
            batch = nil
        }
    }

    // Process remaining documents
    if len(batch) > 0 {
        if err := dp.processBatch(ctx, batch, stats); err != nil {
            stats.Errors = append(stats.Errors, err)
        }
    }

    stats.EndTime = time.Now()
    return stats, nil
}

func (dp *DocumentProcessor) processBatch(
    ctx context.Context,
    docs []schema.Document,
    stats *ProcessingStats,
) error {
    // Split documents
    var allChunks []schema.Document
    for _, doc := range docs {
        chunks, err := dp.splitter.SplitDocuments([]schema.Document{doc})
        if err != nil {
            return fmt.Errorf("split document %s: %w", doc.Metadata["source"], err)
        }
        allChunks = append(allChunks, chunks...)
    }

    stats.ChunksCreated += len(allChunks)

    // Embed chunks
    var embeddings [][]float32
    for _, chunk := range allChunks {
        embedding, err := dp.embedder.EmbedText(ctx, chunk.PageContent)
        if err != nil {
            return fmt.Errorf("embed chunk: %w", err)
        }
        embeddings = append(embeddings, embedding)
    }

    // Store in vector database
    if err := dp.vectorDB.AddDocuments(ctx, allChunks, embeddings); err != nil {
        return fmt.Errorf("store in vector DB: %w", err)
    }

    stats.DocsProcessed += len(docs)
    fmt.Printf("Processed batch: %d docs, %d chunks\n", len(docs), len(allChunks))

    return nil
}

func (stats *ProcessingStats) Report() string {
    duration := stats.EndTime.Sub(stats.StartTime)
    rate := float64(stats.DocsProcessed) / duration.Seconds()

    return fmt.Sprintf(`Processing Report:
- Documents Loaded: %d
- Documents Processed: %d
- Chunks Created: %d
- Errors: %d
- Duration: %s
- Rate: %.2f docs/sec`,
        stats.DocsLoaded,
        stats.DocsProcessed,
        stats.ChunksCreated,
        len(stats.Errors),
        duration,
        rate,
    )
}
```

## Custom Loaders

When built-in loaders do not cover your data source, you can implement custom loaders by satisfying the `Load(ctx context.Context) ([]schema.Document, error)` contract. Custom loaders integrate seamlessly with the rest of the pipeline — splitters, embedders, and vector stores do not care where the documents came from. The following example loads a CSV file, using one column as document content and the remaining columns as metadata.

```go
type CSVLoader struct {
    filePath   string
    columnName string // Column to use as document content
}

func NewCSVLoader(filePath, columnName string) *CSVLoader {
    return &CSVLoader{
        filePath:   filePath,
        columnName: columnName,
    }
}

func (l *CSVLoader) Load(ctx context.Context) ([]schema.Document, error) {
    file, err := os.Open(l.filePath)
    if err != nil {
        return nil, fmt.Errorf("open file: %w", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("read CSV: %w", err)
    }

    if len(records) < 1 {
        return nil, fmt.Errorf("empty CSV file")
    }

    // Find column index
    header := records[0]
    columnIdx := -1
    for i, col := range header {
        if col == l.columnName {
            columnIdx = i
            break
        }
    }

    if columnIdx == -1 {
        return nil, fmt.Errorf("column not found: %s", l.columnName)
    }

    // Create documents
    var docs []schema.Document
    for i, record := range records[1:] {
        if columnIdx >= len(record) {
            continue
        }

        // Create metadata from other columns
        metadata := map[string]interface{}{
            "source":     l.filePath,
            "row_number": i + 1,
        }

        for j, col := range header {
            if j != columnIdx {
                metadata[col] = record[j]
            }
        }

        docs = append(docs, schema.Document{
            PageContent: record[columnIdx],
            Metadata:    metadata,
        })
    }

    return docs, nil
}
```

## Optimizing for RAG Pipelines

The most impactful tuning parameters in a RAG pipeline are chunk size and overlap. Chunks that are too small lack context for the embedding model to capture meaning. Chunks that are too large dilute the embedding with multiple topics, reducing precision. The optimal size depends on your typical query length and content density.

### Chunk Size Optimization

A useful heuristic is to size chunks at 2-3x the average query length. This ensures that the chunk's embedding captures enough context to match the query's intent without introducing unrelated content.

```go
func OptimalChunkSize(avgQueryLength int) int {
    // Rule of thumb: chunks should be 2-3x query length
    baseSize := avgQueryLength * 2

    // Clamp to reasonable bounds
    if baseSize < 200 {
        return 200
    }
    if baseSize > 2000 {
        return 2000
    }

    return baseSize
}
```

### Overlap Strategy

Overlap ensures that sentences spanning chunk boundaries are not lost. A 10-20% overlap is typically sufficient to preserve context without significantly increasing storage requirements.

```go
func CalculateOverlap(chunkSize int) int {
    // 10-20% overlap preserves context
    overlap := chunkSize / 10

    if overlap < 50 {
        return 50 // Minimum overlap
    }
    if overlap > 200 {
        return 200 // Maximum overlap
    }

    return overlap
}
```

### Metadata Filtering

Adding hierarchical metadata derived from file paths enables scoped retrieval. For example, a knowledge base organized as `engineering/backend/auth.md` can be searched with metadata filters that restrict results to the `engineering` category or the `backend` subcategory, reducing noise from unrelated document categories.

```go
// Add hierarchical metadata for filtering
func AddHierarchicalMetadata(doc schema.Document) schema.Document {
    // Extract hierarchy from file path
    source := doc.Metadata["source"].(string)
    parts := strings.Split(source, string(filepath.Separator))

    if len(parts) > 0 {
        doc.Metadata["category"] = parts[0]
    }
    if len(parts) > 1 {
        doc.Metadata["subcategory"] = parts[1]
    }
    if len(parts) > 2 {
        doc.Metadata["topic"] = parts[2]
    }

    return doc
}
```

## Error Handling

Production document pipelines process thousands of files, and individual failures should not halt the entire pipeline. The error handling pattern below categorizes errors by pipeline phase (load, split, embed, store), distinguishes transient errors that can be retried from permanent failures, and logs enough context to diagnose problems after the fact.

```go
type DocumentError struct {
    Source string
    Err    error
    Phase  string // "load", "split", "embed", "store"
}

func (e *DocumentError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Phase, e.Source, e.Err)
}

func ProcessWithErrorHandling(ctx context.Context, doc schema.Document) error {
    // Try to process
    if err := processDocument(ctx, doc); err != nil {
        // Log error
        docErr := &DocumentError{
            Source: doc.Metadata["source"].(string),
            Err:    err,
            Phase:  identifyPhase(err),
        }

        log.Printf("Error processing document: %v", docErr)

        // Decide whether to retry or skip
        if isRetryable(err) {
            time.Sleep(time.Second)
            return processDocument(ctx, doc)
        }

        return docErr
    }

    return nil
}

func isRetryable(err error) bool {
    // Check for transient errors
    return strings.Contains(err.Error(), "timeout") ||
        strings.Contains(err.Error(), "rate limit") ||
        strings.Contains(err.Error(), "connection refused")
}
```

## Production Best Practices

The following checklist summarizes the key considerations for deploying document processing pipelines in production environments:

1. **Use lazy loading** for datasets larger than available memory
2. **Process in batches** to optimize API calls and database writes
3. **Add comprehensive metadata** for filtering and debugging
4. **Choose appropriate chunk sizes** based on query patterns
5. **Implement retry logic** for transient failures
6. **Monitor processing metrics** (throughput, error rate, latency)
7. **Version your processing pipeline** to track changes
8. **Test with edge cases** (empty files, malformed content, large files)
9. **Implement checkpointing** to resume after failures
10. **Use parallel processing** where possible

## Next Steps

Now that you understand document processing:
- Learn about [RAG Pipeline](/guides/rag-pipeline) for retrieval integration
- Explore [Multimodal AI](/guides/multimodal) for image and audio processing
- Read [RAG Recipes](/cookbook/rag-recipes) for advanced patterns
- Check out [Document Loader Integrations](/integrations/document-loaders) for more sources
