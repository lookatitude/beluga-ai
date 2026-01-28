// Package iface defines the metrics interface for retriever providers.
package iface

import (
	"context"
	"time"
)

// MetricsRecorder defines the interface for recording retriever metrics.
// This allows providers to record metrics without importing the parent package.
type MetricsRecorder interface {
	// RecordRetrieval records metrics for a retrieval operation.
	RecordRetrieval(ctx context.Context, retrieverType string, duration time.Duration, documentCount int, avgScore float64, err error)

	// RecordVectorStoreOperation records metrics for a vector store operation.
	RecordVectorStoreOperation(ctx context.Context, operation string, duration time.Duration, documentCount int, err error)

	// RecordBatchOperation records metrics for batch operations.
	RecordBatchOperation(ctx context.Context, operation string, batchSize int, duration time.Duration)
}
