---
title: "Re-indexing Status Tracking"
package: "vectorstores"
category: "operations"
complexity: "intermediate"
---

# Re-indexing Status Tracking

## Problem

You need to track the status of re-indexing operations in your vector store (progress, errors, completion) so users can monitor long-running reindexing jobs and handle failures gracefully.

## Solution

Implement a reindexing tracker that monitors reindexing operations, stores status in a persistent store, and provides status queries. This works because reindexing is often a long-running operation that needs visibility and recovery mechanisms.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.vectorstores.reindexing")

// ReindexStatus represents the status of a reindexing operation
type ReindexStatus string

const (
    ReindexStatusPending   ReindexStatus = "pending"
    ReindexStatusRunning   ReindexStatus = "running"
    ReindexStatusCompleted ReindexStatus = "completed"
    ReindexStatusFailed    ReindexStatus = "failed"
    ReindexStatusCancelled ReindexStatus = "cancelled"
)

// ReindexJob represents a reindexing job
type ReindexJob struct {
    ID          string
    Status      ReindexStatus
    Progress    float64 // 0.0 to 1.0
    TotalDocs   int
    ProcessedDocs int
    StartedAt   time.Time
    CompletedAt *time.Time
    Error       error
    Metadata    map[string]interface{}
}

// ReindexTracker tracks reindexing operations
type ReindexTracker struct {
    jobs    map[string]*ReindexJob
    mu      sync.RWMutex
    store   vectorstores.VectorStore
}

// NewReindexTracker creates a new reindex tracker
func NewReindexTracker(store vectorstores.VectorStore) *ReindexTracker {
    return &ReindexTracker{
        jobs:  make(map[string]*ReindexJob),
        store: store,
    }
}

// StartReindex starts a new reindexing operation
func (rt *ReindexTracker) StartReindex(ctx context.Context, jobID string, documents []schema.Document) error {
    ctx, span := tracer.Start(ctx, "reindex_tracker.start")
    defer span.End()
    
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    job := &ReindexJob{
        ID:          jobID,
        Status:      ReindexStatusPending,
        Progress:    0.0,
        TotalDocs:   len(documents),
        ProcessedDocs: 0,
        StartedAt:   time.Now(),
        Metadata:    make(map[string]interface{}),
    }
    
    rt.jobs[jobID] = job
    
    span.SetAttributes(
        attribute.String("job_id", jobID),
        attribute.Int("document_count", len(documents)),
    )
    
    // Start reindexing in background
    go rt.executeReindex(ctx, jobID, documents)
    
    span.SetStatus(trace.StatusOK, "reindex started")
    return nil
}

// executeReindex performs the actual reindexing
func (rt *ReindexTracker) executeReindex(ctx context.Context, jobID string, documents []schema.Document) {
    ctx, span := tracer.Start(ctx, "reindex_tracker.execute")
    defer span.End()
    
    rt.mu.Lock()
    job := rt.jobs[jobID]
    job.Status = ReindexStatusRunning
    rt.mu.Unlock()
    
    batchSize := 100
    for i := 0; i < len(documents); i += batchSize {
        end := i + batchSize
        if end > len(documents) {
            end = len(documents)
        }
        
        batch := documents[i:end]
        
        // Add documents to store
        _, err := rt.store.AddDocuments(ctx, batch)
        if err != nil {
            rt.mu.Lock()
            job.Status = ReindexStatusFailed
            job.Error = err
            completedAt := time.Now()
            job.CompletedAt = &completedAt
            rt.mu.Unlock()
            
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return
        }
        
        // Update progress
        rt.mu.Lock()
        job.ProcessedDocs = end
        job.Progress = float64(end) / float64(len(documents))
        rt.mu.Unlock()
        
        span.SetAttributes(
            attribute.Float64("progress", job.Progress),
            attribute.Int("processed", end),
        )
    }
    
    // Mark as completed
    rt.mu.Lock()
    job.Status = ReindexStatusCompleted
    job.Progress = 1.0
    completedAt := time.Now()
    job.CompletedAt = &completedAt
    rt.mu.Unlock()
    
    span.SetStatus(trace.StatusOK, "reindex completed")
}

// GetStatus returns the status of a reindexing job
func (rt *ReindexTracker) GetStatus(jobID string) (*ReindexJob, error) {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    
    job, exists := rt.jobs[jobID]
    if !exists {
        return nil, fmt.Errorf("job %s not found", jobID)
    }
    
    return job, nil
}

// CancelReindex cancels a running reindexing job
func (rt *ReindexTracker) CancelReindex(ctx context.Context, jobID string) error {
    ctx, span := tracer.Start(ctx, "reindex_tracker.cancel")
    defer span.End()
    
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    job, exists := rt.jobs[jobID]
    if !exists {
        return fmt.Errorf("job %s not found", jobID)
    }
    
    if job.Status != ReindexStatusRunning {
        return fmt.Errorf("job %s is not running", jobID)
    }
    
    job.Status = ReindexStatusCancelled
    completedAt := time.Now()
    job.CompletedAt = &completedAt
    
    span.SetAttributes(attribute.String("job_id", jobID))
    span.SetStatus(trace.StatusOK, "reindex cancelled")
    
    return nil
}

// ListJobs lists all reindexing jobs
func (rt *ReindexTracker) ListJobs(status *ReindexStatus) []*ReindexJob {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    
    jobs := []*ReindexJob{}
    for _, job := range rt.jobs {
        if status == nil || job.Status == *status {
            jobs = append(jobs, job)
        }
    }
    
    return jobs
}

func main() {
    ctx := context.Background()

    // Create tracker
    // store := yourVectorStore
    tracker := NewReindexTracker(store)
    
    // Start reindexing
    documents := []schema.Document{
        // ... documents to reindex
    }
    
    jobID := "reindex-123"
    if err := tracker.StartReindex(ctx, jobID, documents); err != nil {
        log.Fatalf("Failed to start reindex: %v", err)
    }
    
    // Monitor progress
    for {
        job, _ := tracker.GetStatus(jobID)
        fmt.Printf("Progress: %.2f%%\n", job.Progress*100)
        if job.Status == ReindexStatusCompleted || job.Status == ReindexStatusFailed \{
            break
        }
        time.Sleep(1 * time.Second)
    }
```
    
    fmt.Println("Reindexing completed")
}

## Explanation

Let's break down what's happening:

1. **Status tracking** - Notice how we track multiple status states (pending, running, completed, failed, cancelled). This gives users clear visibility into what's happening with their reindexing jobs.

2. **Progress updates** - We update progress as documents are processed. This allows users to see how far along the operation is, which is important for long-running jobs.

3. **Error handling** - If reindexing fails, we capture the error and mark the job as failed. This allows users to see what went wrong and potentially retry.

```go
**Key insight:** Always provide progress updates for long-running operations. Users need to know the system is working and how long it will take.

## Testing

```
Here's how to test this solution:
```go
func TestReindexTracker_TracksProgress(t *testing.T) {
    mockStore := &MockVectorStore{}
    tracker := NewReindexTracker(mockStore)
    
    documents := make([]schema.Document, 100)
    for i := 0; i < 100; i++ {
        documents[i] = schema.NewDocument(fmt.Sprintf("doc %d", i), nil)
    }
    
    err := tracker.StartReindex(context.Background(), "test-job", documents)
    require.NoError(t, err)
    
    // Wait for completion
    time.Sleep(2 * time.Second)
    
    job, err := tracker.GetStatus("test-job")
    require.NoError(t, err)
    require.Equal(t, ReindexStatusCompleted, job.Status)
    require.Equal(t, 1.0, job.Progress)
}

## Variations

### Persistent Storage

Store job status in a database:
type PersistentReindexTracker struct {
    db *sql.DB
}
```

### Job Retry

Automatically retry failed jobs:
```go
func (rt *ReindexTracker) RetryJob(ctx context.Context, jobID string) error {
    // Retry logic
}
```

## Related Recipes

- **[Vectorstores Advanced Meta-filtering](./vectorstores-advanced-meta-filtering.md)** - Advanced filtering patterns
- **[Documentloaders Parallel File Walkers](./documentloaders-parallel-file-walkers.md)** - Parallel document loading
- **[Vectorstores Package Guide](../package_design_patterns.md)** - For a deeper understanding of vectorstores
