---
title: "Reindexing Status Tracking"
description: "Track progress, errors, and completion of vector store reindexing operations for operational visibility and failure recovery."
---

## Problem

You need to track the status of reindexing operations in your vector store (progress, errors, completion) so users can monitor long-running jobs and handle failures gracefully.

Reindexing a vector store is a long-running, stateful operation that may process thousands or millions of documents over minutes to hours. Without progress tracking, operators have no way to distinguish a slow job from a stuck one, no visibility into which documents failed, and no ability to resume after a partial failure. In production, transient errors (network timeouts, rate limits, temporary database unavailability) are expected during long operations, so the tracking system must handle partial failures gracefully.

## Solution

Implement a reindexing tracker that manages job lifecycle (pending, running, completed, failed, cancelled), provides real-time progress updates, and records per-batch failure details. The tracker runs the actual reindexing in a background goroutine, allowing callers to query progress asynchronously. This separation of concerns means the reindexing logic doesn't need to know about progress reporting, and the progress API doesn't need to know about document processing.

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

	"github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.vectorstores.reindexing")

type ReindexStatus string

const (
	ReindexStatusPending   ReindexStatus = "pending"
	ReindexStatusRunning   ReindexStatus = "running"
	ReindexStatusCompleted ReindexStatus = "completed"
	ReindexStatusFailed    ReindexStatus = "failed"
	ReindexStatusCancelled ReindexStatus = "cancelled"
)

// ReindexJob represents a reindexing job.
type ReindexJob struct {
	ID            string
	Status        ReindexStatus
	Progress      float64
	TotalDocs     int
	ProcessedDocs int
	StartedAt     time.Time
	CompletedAt   *time.Time
	Error         error
}

// ReindexTracker tracks reindexing operations.
type ReindexTracker struct {
	jobs map[string]*ReindexJob
	mu   sync.RWMutex
}

func NewReindexTracker() *ReindexTracker {
	return &ReindexTracker{
		jobs: make(map[string]*ReindexJob),
	}
}

// StartReindex starts a new reindexing operation.
func (rt *ReindexTracker) StartReindex(ctx context.Context, jobID string, documents []schema.Document) error {
	ctx, span := tracer.Start(ctx, "reindex_tracker.start")
	defer span.End()

	rt.mu.Lock()
	defer rt.mu.Unlock()

	job := &ReindexJob{
		ID:        jobID,
		Status:    ReindexStatusPending,
		TotalDocs: len(documents),
		StartedAt: time.Now(),
	}
	rt.jobs[jobID] = job

	span.SetAttributes(
		attribute.String("job_id", jobID),
		attribute.Int("document_count", len(documents)),
	)

	go rt.executeReindex(context.Background(), jobID, documents)

	span.SetStatus(trace.StatusOK, "reindex started")
	return nil
}

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

		// In production, call store.AddDocuments(ctx, batch) here.
		// Simulating batch processing.
		time.Sleep(10 * time.Millisecond)

		rt.mu.Lock()
		job.ProcessedDocs = end
		job.Progress = float64(end) / float64(len(documents))
		rt.mu.Unlock()
	}

	rt.mu.Lock()
	job.Status = ReindexStatusCompleted
	job.Progress = 1.0
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	rt.mu.Unlock()

	span.SetStatus(trace.StatusOK, "reindex completed")
}

// GetStatus returns the status of a reindexing job.
func (rt *ReindexTracker) GetStatus(jobID string) (*ReindexJob, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	job, exists := rt.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

// CancelReindex cancels a running reindexing job.
func (rt *ReindexTracker) CancelReindex(ctx context.Context, jobID string) error {
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
	return nil
}

func main() {
	ctx := context.Background()

	tracker := NewReindexTracker()

	documents := make([]schema.Document, 500)
	for i := range documents {
		documents[i] = schema.NewDocument(fmt.Sprintf("doc %d", i), nil)
	}

	if err := tracker.StartReindex(ctx, "reindex-001", documents); err != nil {
		log.Fatalf("Failed to start reindex: %v", err)
	}

	for {
		job, _ := tracker.GetStatus("reindex-001")
		fmt.Printf("Progress: %.0f%%\n", job.Progress*100)
		if job.Status == ReindexStatusCompleted || job.Status == ReindexStatusFailed {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Reindexing completed")
}
```

## Explanation

1. **Status state machine** -- Jobs progress through well-defined states: pending, running, completed, failed, or cancelled. Each state transition is explicit and protected by a mutex. The state machine makes it easy for monitoring systems to categorize jobs (e.g., alert on "failed" status, dashboard for "running" jobs with progress bars).

2. **Progress updates** -- Progress is updated as each batch completes, providing a continuous progress signal. The percentage calculation (`processed / total`) gives operators an accurate estimate of remaining time. Progress is updated under the write lock to ensure consistency between the processed count and the progress percentage.

3. **Cancellation support** -- Running jobs can be cancelled via `CancelReindex`, which sets the status to cancelled and records the completion time. In a production implementation, the execute goroutine would check for cancellation via context and stop processing new batches.

4. **Thread-safe access** -- All job state is protected by `sync.RWMutex`. Status queries use read locks (`RLock`) for concurrent read access, while mutations use write locks (`Lock`). This ensures safe concurrent access from the monitoring goroutine and the processing goroutine.

## Variations

### Persistent Storage

Store job status in a database for durability across process restarts:

```go
type PersistentReindexTracker struct {
	db *sql.DB
}

func (p *PersistentReindexTracker) SaveStatus(ctx context.Context, job *ReindexJob) error {
	_, err := p.db.ExecContext(ctx,
		"INSERT INTO reindex_jobs (id, status, progress) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET status=$2, progress=$3",
		job.ID, job.Status, job.Progress,
	)
	return err
}
```

## Related Recipes

- **[Advanced Metadata Filtering](./meta-filtering)** -- Complex vectorstore filtering
- **[Parallel File Loading](./parallel-file-loading)** -- Parallel document loading for reindexing
