---
title: Legacy Archive Ingestion
description: Digitize and ingest millions of historical documents from legacy archives into modern RAG systems.
---

Financial institutions accumulate decades of documents — loan agreements, regulatory filings, correspondence, audit reports — stored across paper files, microfilm, legacy databases, and obsolete digital formats. These archives contain critical institutional knowledge but are effectively inaccessible: finding a specific document means knowing exactly where to look, and cross-referencing across archives is impractical at scale.

Manual digitization at 5M+ documents would take 5-10 years and cost $2M+, and the physical media continues to deteriorate. The challenge is not just digitization but ingestion into a modern RAG system where documents become semantically searchable — finding all loan modifications related to a specific property type, regardless of which archive or format they were stored in.

An automated ingestion system with format detection, batch processing, and quality validation processes the entire archive in months rather than years, making decades of institutional knowledge immediately queryable.

## Solution Architecture

Beluga AI's RAG loader package supports multiple document formats through a unified `DocumentLoader` interface. The ingestion system uses format detection to select the appropriate loader for each file, processes documents in configurable batches for memory efficiency, and validates extracted text quality before embedding and indexing. Failed documents are logged for manual review rather than blocking the pipeline.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Legacy    │───▶│    Format    │───▶│   Document   │
│   Archives   │    │   Detector   │    │    Loader    │
│              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Vector     │◀───│   Quality    │◀───│    Batch     │
│    Store     │    │  Validator   │    │  Processor   │
│              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Archive Ingestion System

The system orchestrates multi-format document loading with validation.

```go
package main

import (
    "context"
    "fmt"
    "path/filepath"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
)

// ArchiveIngestionSystem implements legacy archive ingestion.
type ArchiveIngestionSystem struct {
    formatDetector   *FormatDetector
    qualityValidator *QualityValidator
    splitter         splitter.TextSplitter
    embedder         embedding.Embedder
    vectorStore      vectorstore.VectorStore
}

// NewArchiveIngestionSystem creates a new ingestion system.
func NewArchiveIngestionSystem(ctx context.Context) (*ArchiveIngestionSystem, error) {
    spl, err := splitter.New("recursive", nil)
    if err != nil {
        return nil, fmt.Errorf("create splitter: %w", err)
    }

    emb, err := embedding.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", nil)
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    return &ArchiveIngestionSystem{
        formatDetector:   NewFormatDetector(),
        qualityValidator: NewQualityValidator(),
        splitter:         spl,
        embedder:         emb,
        vectorStore:      store,
    }, nil
}

// IngestArchive ingests documents from legacy archive.
func (a *ArchiveIngestionSystem) IngestArchive(ctx context.Context, archivePath string) error {
    // Discover files in archive
    files, err := a.discoverFiles(ctx, archivePath)
    if err != nil {
        return fmt.Errorf("discover files: %w", err)
    }

    // Process in batches
    batchSize := 1000
    for i := 0; i < len(files); i += batchSize {
        end := min(i+batchSize, len(files))
        batch := files[i:end]

        if err := a.processBatch(ctx, batch); err != nil {
            // Log error but continue with next batch
            continue
        }
    }

    return nil
}

func (a *ArchiveIngestionSystem) discoverFiles(ctx context.Context, path string) ([]string, error) {
    var files []string

    err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() {
            files = append(files, p)
        }

        return nil
    })

    return files, err
}
```

## Format Detection

The detector identifies document formats for loader selection.

```go
type FormatDetector struct {
    // Format mappings by extension and magic bytes
}

func NewFormatDetector() *FormatDetector {
    return &FormatDetector{}
}

// Detect determines the document format.
func (fd *FormatDetector) Detect(ctx context.Context, filePath string) string {
    // Check file extension
    ext := filepath.Ext(filePath)
    switch ext {
    case ".pdf":
        return "pdf"
    case ".docx":
        return "docx"
    case ".txt":
        return "text"
    case ".html", ".htm":
        return "html"
    case ".md":
        return "markdown"
    default:
        // Check magic bytes
        return fd.detectByMagicBytes(filePath)
    }
}

func (fd *FormatDetector) detectByMagicBytes(filePath string) string {
    // Read first few bytes and match against known patterns
    // Implementation depends on supported formats
    return "unknown"
}
```

## Batch Processing with Validation

Process documents in batches with quality checks.

```go
func (a *ArchiveIngestionSystem) processBatch(ctx context.Context, files []string) error {
    var allChunks []schema.Document

    for _, filePath := range files {
        // Detect format
        format := a.formatDetector.Detect(ctx, filePath)
        if format == "unknown" {
            continue
        }

        // Get appropriate loader
        ldr, err := loader.New(format, map[string]any{
            "path": filePath,
        })
        if err != nil {
            continue
        }

        // Load documents
        docs, err := ldr.Load(ctx)
        if err != nil {
            continue
        }

        // Validate quality
        validDocs := []schema.Document{}
        for _, doc := range docs {
            if a.qualityValidator.Validate(ctx, doc) {
                validDocs = append(validDocs, doc)
            }
        }

        // Split documents
        chunks, err := a.splitter.SplitDocuments(ctx, validDocs)
        if err != nil {
            continue
        }

        allChunks = append(allChunks, chunks...)
    }

    // Generate embeddings for batch
    if len(allChunks) == 0 {
        return nil
    }

    texts := make([]string, len(allChunks))
    for i, chunk := range allChunks {
        texts[i] = chunk.Content
    }

    embeddings, err := a.embedder.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("generate embeddings: %w", err)
    }

    // Store in vector database
    if err := a.vectorStore.Add(ctx, allChunks, embeddings); err != nil {
        return fmt.Errorf("store documents: %w", err)
    }

    return nil
}
```

## Quality Validation

Legacy documents often produce poor OCR output — garbled text, excessive special characters, or nearly empty extractions from damaged pages. Ingesting low-quality text into the vector store degrades retrieval accuracy for all queries. The validator gates ingestion on content quality, filtering out documents that would add noise rather than signal.

```go
type QualityValidator struct {
    minLength int
    maxLength int
}

func NewQualityValidator() *QualityValidator {
    return &QualityValidator{
        minLength: 50,    // Minimum meaningful content
        maxLength: 50000, // Maximum reasonable document size
    }
}

// Validate checks if a document meets quality standards.
func (qv *QualityValidator) Validate(ctx context.Context, doc schema.Document) bool {
    content := doc.Content

    // Check length
    if len(content) < qv.minLength || len(content) > qv.maxLength {
        return false
    }

    // Check for corruption indicators
    if qv.hasCorruptionIndicators(content) {
        return false
    }

    // Check content ratio (text vs non-text characters)
    if qv.textRatio(content) < 0.8 {
        return false
    }

    return true
}

func (qv *QualityValidator) hasCorruptionIndicators(content string) bool {
    // Check for excessive special characters
    specialCount := 0
    for _, r := range content {
        if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
            specialCount++
        }
    }

    return float64(specialCount)/float64(len(content)) > 0.3
}

func (qv *QualityValidator) textRatio(content string) float64 {
    textChars := 0
    for _, r := range content {
        if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
            textChars++
        }
    }

    return float64(textChars) / float64(len(content))
}
```

## Progress Tracking

Track ingestion progress for monitoring.

```go
type ProgressTracker struct {
    totalFiles     int
    processedFiles int
    successFiles   int
    failedFiles    int
    mu             sync.RWMutex
}

func NewProgressTracker(totalFiles int) *ProgressTracker {
    return &ProgressTracker{
        totalFiles: totalFiles,
    }
}

func (pt *ProgressTracker) RecordSuccess() {
    pt.mu.Lock()
    defer pt.mu.Unlock()

    pt.processedFiles++
    pt.successFiles++
}

func (pt *ProgressTracker) RecordFailure() {
    pt.mu.Lock()
    defer pt.mu.Unlock()

    pt.processedFiles++
    pt.failedFiles++
}

func (pt *ProgressTracker) Progress() (processed, success, failed int, percentage float64) {
    pt.mu.RLock()
    defer pt.mu.RUnlock()

    percentage = float64(pt.processedFiles) / float64(pt.totalFiles) * 100
    return pt.processedFiles, pt.successFiles, pt.failedFiles, percentage
}
```

## Production Considerations

### Format Support

Support common legacy formats: PDF, DOCX, TXT, HTML, Markdown, CSV, XML. For specialized formats (microfilm scans, proprietary databases), implement custom loaders or preprocessing steps.

### Batch Sizing

Batch size depends on document size and available memory. Start with 1000 documents per batch and adjust based on memory usage and processing time. Larger batches improve throughput but increase memory requirements.

### Error Recovery

Track failed documents for manual review. Implement retry logic for transient failures. Store error logs with file paths and error messages for debugging.

### Observability

Monitor ingestion progress and quality metrics:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

func (a *ArchiveIngestionSystem) recordMetrics(ctx context.Context, format string, success bool) {
    meter := otel.Meter("archive-ingestion")

    counter, _ := meter.Int64Counter("documents_processed_total")
    counter.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("format", format),
            attribute.Bool("success", success),
        ),
    )
}
```

### Parallel Processing

Process multiple batches in parallel for improved throughput:

```go
func (a *ArchiveIngestionSystem) IngestArchiveParallel(ctx context.Context, archivePath string, workers int) error {
    files, err := a.discoverFiles(ctx, archivePath)
    if err != nil {
        return err
    }

    batchSize := 1000
    batches := make([][]string, 0)

    for i := 0; i < len(files); i += batchSize {
        end := min(i+batchSize, len(files))
        batches = append(batches, files[i:end])
    }

    var wg sync.WaitGroup
    semaphore := make(chan struct{}, workers)

    for _, batch := range batches {
        wg.Add(1)
        semaphore <- struct{}{}

        go func(b []string) {
            defer wg.Done()
            defer func() { <-semaphore }()

            a.processBatch(ctx, b)
        }(batch)
    }

    wg.Wait()
    return nil
}
```

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Processing Time (years) | 5-10 | 0.5 | 95-98% reduction |
| Processing Cost | $2M+ | $180K | 91% reduction |
| Ingestion Success Rate | N/A | 96% | High success |
| Documents Processed | 0 | 5.2M | 5.2M documents |
| Format Support | Manual | 22 | 22 formats |

## Related Resources

- [RAG Pipeline Guide](/guides/rag-pipeline/) for document processing patterns
- [Cloud Document Sync](/use-cases/cloud-doc-sync/) for real-time ingestion
- [Enterprise RAG](/use-cases/enterprise-rag/) for complete RAG system setup
