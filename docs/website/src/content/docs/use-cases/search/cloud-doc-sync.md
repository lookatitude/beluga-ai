---
title: Automated Cloud Document Sync for RAG
description: "Maintain real-time RAG knowledge bases by auto-syncing documents from S3, GCS, and Azure Blob with ETag change detection."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "cloud document sync, RAG real-time, S3 ingestion, document pipeline, change detection, Beluga AI, Go, knowledge base sync"
---

A technology company needed to maintain a real-time knowledge base by automatically syncing documents from cloud storage (S3, GCS, Azure Blob) into their RAG system. The core problem is that RAG systems are only as good as their indexed data — stale embeddings from outdated documents produce wrong answers with high confidence, which is more damaging than no answer at all. Manual document ingestion led to 3-5 day delays, 15-20% stale data in RAG indexes, and missed updates from 50+ cloud buckets.

An automated cloud sync system with change detection enables near real-time RAG updates with less than 5 minute latency. The system uses ETag-based change detection rather than re-indexing everything, which means only modified documents are re-processed through the embedding pipeline. This incremental approach keeps costs proportional to change volume, not total document count.

## Solution Architecture

Beluga AI's RAG loader package supports multiple cloud storage providers through a unified interface. The sync system monitors cloud buckets for changes via webhooks and polling, tracks file ETags to detect modifications, and incrementally processes only changed documents through the RAG pipeline.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Cloud     │───▶│    Change    │───▶│     Sync     │
│   Storage    │    │   Detector   │    │ Coordinator  │
│   Buckets    │    │              │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Vector     │◀───│  Embeddings  │◀───│   Document   │
│    Store     │    │              │    │    Loader    │
│              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Cloud Storage Adapters

The system uses provider-specific loaders that implement a unified interface.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/schema"
)

// CloudLoaderConfig configures cloud storage access.
type CloudLoaderConfig struct {
    Provider    string // "s3", "gcs", "azure"
    Bucket      string
    Prefix      string
    Credentials map[string]string
}

// CreateCloudLoader creates a document loader for cloud storage.
func CreateCloudLoader(ctx context.Context, cfg CloudLoaderConfig) (loader.DocumentLoader, error) {
    switch cfg.Provider {
    case "s3":
        return loader.New("s3", map[string]any{
            "bucket":      cfg.Bucket,
            "prefix":      cfg.Prefix,
            "credentials": cfg.Credentials,
        })
    case "gcs":
        return loader.New("gcs", map[string]any{
            "bucket":      cfg.Bucket,
            "prefix":      cfg.Prefix,
            "credentials": cfg.Credentials,
        })
    case "azure":
        return loader.New("azure", map[string]any{
            "container":   cfg.Bucket,
            "prefix":      cfg.Prefix,
            "credentials": cfg.Credentials,
        })
    default:
        return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
    }
}
```

## Change Detection

The change detector monitors buckets for new and modified files using ETags.

```go
import (
    "time"
)

// ChangeEvent represents a file change in cloud storage.
type ChangeEvent struct {
    Path     string
    Bucket   string
    ETag     string
    Action   string // "added", "modified", "deleted"
    Provider string
}

// ChangeDetector monitors cloud storage for changes.
type ChangeDetector struct {
    buckets      []CloudLoaderConfig
    tracker      *ChangeTracker
    pollInterval time.Duration
}

// DetectChanges monitors buckets and emits change events.
func (d *ChangeDetector) DetectChanges(ctx context.Context) iter.Seq2[ChangeEvent, error] {
    return func(yield func(ChangeEvent, error) bool) {
        // Setup webhook listeners for real-time updates
        webhookEvents := make(chan ChangeEvent, 100)
        for _, bucket := range d.buckets {
            go d.listenWebhooks(ctx, bucket, webhookEvents)
        }

        // Poll as fallback
        ticker := time.NewTicker(d.pollInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case event := <-webhookEvents:
                if !yield(event, nil) {
                    return
                }
            case <-ticker.C:
                for _, bucket := range d.buckets {
                    for _, change := range d.pollBucket(ctx, bucket) {
                        if !yield(change, nil) {
                            return
                        }
                    }
                }
            }
        }
    }
}

func (d *ChangeDetector) pollBucket(ctx context.Context, cfg CloudLoaderConfig) []ChangeEvent {
    var changes []ChangeEvent

    // List objects in bucket
    loader, err := CreateCloudLoader(ctx, cfg)
    if err != nil {
        return changes
    }

    docs, err := loader.Load(ctx)
    if err != nil {
        return changes
    }

    for _, doc := range docs {
        path := doc.Metadata["source"].(string)
        etag := doc.Metadata["etag"].(string)

        // Check if file has changed
        lastETag, exists := d.tracker.GetETag(path)
        if !exists || lastETag != etag {
            changes = append(changes, ChangeEvent{
                Path:     path,
                Bucket:   cfg.Bucket,
                ETag:     etag,
                Action:   "modified",
                Provider: cfg.Provider,
            })
        }
    }

    return changes
}
```

## Change Tracker

The tracker stores file ETags to detect modifications.

```go
import (
    "sync"
)

// ChangeTracker stores file ETags for change detection.
type ChangeTracker struct {
    etags map[string]string
    mu    sync.RWMutex
}

func NewChangeTracker() *ChangeTracker {
    return &ChangeTracker{
        etags: make(map[string]string),
    }
}

func (ct *ChangeTracker) GetETag(path string) (string, bool) {
    ct.mu.RLock()
    defer ct.mu.RUnlock()

    etag, exists := ct.etags[path]
    return etag, exists
}

func (ct *ChangeTracker) UpdateETag(path string, etag string) {
    ct.mu.Lock()
    defer ct.mu.Unlock()

    ct.etags[path] = etag
}
```

## Sync Coordinator

The coordinator processes change events through the RAG pipeline.

```go
import (
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
)

// SyncCoordinator orchestrates document sync workflow.
type SyncCoordinator struct {
    loaders     map[string]loader.DocumentLoader
    splitter    splitter.TextSplitter
    embedder    embedding.Embedder
    vectorStore vectorstore.VectorStore
    tracker     *ChangeTracker
}

// ProcessChange handles a single change event.
func (c *SyncCoordinator) ProcessChange(ctx context.Context, event ChangeEvent) error {
    // Get loader for bucket
    ldr, exists := c.loaders[event.Bucket]
    if !exists {
        return fmt.Errorf("loader not found for bucket: %s", event.Bucket)
    }

    // Load document
    docs, err := ldr.LoadPath(ctx, event.Path)
    if err != nil {
        return fmt.Errorf("load document: %w", err)
    }

    // Split documents
    chunks, err := c.splitter.SplitDocuments(ctx, docs)
    if err != nil {
        return fmt.Errorf("split documents: %w", err)
    }

    // Generate embeddings
    texts := make([]string, len(chunks))
    for i, chunk := range chunks {
        texts[i] = chunk.Content
    }

    embeddings, err := c.embedder.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("generate embeddings: %w", err)
    }

    // Upsert to vector store (handles updates)
    if err := c.vectorStore.Upsert(ctx, chunks, embeddings); err != nil {
        return fmt.Errorf("upsert to vector store: %w", err)
    }

    // Update change tracker
    c.tracker.UpdateETag(event.Path, event.ETag)

    return nil
}
```

## Batch Processing

Process multiple changes efficiently with controlled concurrency.

```go
// ProcessBatch handles multiple changes in parallel.
func (c *SyncCoordinator) ProcessBatch(ctx context.Context, events []ChangeEvent) error {
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 10) // Limit concurrency
    errors := make(chan error, len(events))

    for _, event := range events {
        wg.Add(1)
        semaphore <- struct{}{}

        go func(e ChangeEvent) {
            defer wg.Done()
            defer func() { <-semaphore }()

            if err := c.ProcessChange(ctx, e); err != nil {
                errors <- err
            }
        }(event)
    }

    wg.Wait()
    close(errors)

    // Collect errors
    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("batch processing failed: %v", errs)
    }

    return nil
}
```

## Production Considerations

### Webhook Reliability

Webhooks provide real-time updates but can be unreliable. Implement polling as a fallback mechanism to catch missed webhook events. Verify webhook signatures to prevent spoofing.

### Incremental Processing

Track ETags for all files to detect actual changes. This avoids reprocessing unchanged files, reducing API calls and processing time. Store ETags in a persistent database for durability across restarts.

### Rate Limiting

Cloud storage APIs have rate limits. Implement exponential backoff and request queuing to stay within limits. Batch operations where possible to reduce API call count.

### Error Handling

Handle transient failures with retries. Log failed sync operations for manual review. Consider dead letter queues for persistent failures.

### Observability

Track sync metrics with OpenTelemetry:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

func (c *SyncCoordinator) recordMetrics(ctx context.Context, bucket string, duration time.Duration, err error) {
    meter := otel.Meter("cloud-sync")

    if err != nil {
        counter, _ := meter.Int64Counter("sync_errors_total")
        counter.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("bucket", bucket),
            ),
        )
    } else {
        counter, _ := meter.Int64Counter("sync_success_total")
        counter.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("bucket", bucket),
            ),
        )

        histogram, _ := meter.Float64Histogram("sync_duration_seconds")
        histogram.Record(ctx, duration.Seconds(),
            metric.WithAttributes(
                attribute.String("bucket", bucket),
            ),
        )
    }
}
```

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Sync Latency (minutes) | 4320 | 5 | 99.9% reduction |
| Data Freshness | 80% | 99.2% | 24% improvement |
| Manual Interventions/month | 50+ | 0 | 100% reduction |
| Documents Synced/day | 500 | 6200 | 1140% increase |
| Sync Accuracy | 95% | 99.5% | 4.7% improvement |

## Related Resources

- [RAG Pipeline Guide](/guides/rag-pipeline/) for document processing patterns
- [Legacy Archive Ingestion](/use-cases/legacy-archive/) for batch ingestion patterns
- [Enterprise RAG](/use-cases/enterprise-rag/) for complete RAG system setup
