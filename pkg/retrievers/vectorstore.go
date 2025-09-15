// Package retrievers provides implementations of the core.Retriever interface.
package retrievers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// VectorStoreRetriever implements the core.Retriever interface using an underlying vectorstores.VectorStore.
// It provides configurable retrieval with support for tracing, metrics, and structured logging.
type VectorStoreRetriever struct {
	vectorStore    vectorstores.VectorStore // The vector store to retrieve from
	defaultK       int                      // Default number of documents to retrieve
	scoreThreshold float32                  // Minimum similarity score threshold
	maxRetries     int                      // Maximum number of retries for failed operations
	timeout        time.Duration            // Timeout for operations
	enableTracing  bool                     // Whether to enable tracing
	enableMetrics  bool                     // Whether to enable metrics collection
	logger         *slog.Logger             // Structured logger
	tracer         trace.Tracer             // OpenTelemetry tracer
	metrics        *Metrics                 // Metrics collector
}

// newVectorStoreRetrieverInternal creates a new VectorStoreRetriever with internal configuration.
// This is used by the public factory functions in retrievers.go
func newVectorStoreRetrieverInternal(vectorStore vectorstores.VectorStore, config *RetrieverOptions) *VectorStoreRetriever {
	return &VectorStoreRetriever{
		vectorStore:    vectorStore,
		defaultK:       config.DefaultK,
		scoreThreshold: config.ScoreThreshold,
		maxRetries:     config.MaxRetries,
		timeout:        config.Timeout,
		enableTracing:  config.EnableTracing,
		enableMetrics:  config.EnableMetrics,
		logger:         config.Logger,
		tracer:         config.Tracer,
		metrics:        config.Metrics,
	}
}

// getEmbedder extracts an embedder from the options or returns a default.
func (r *VectorStoreRetriever) getEmbedder(options ...core.Option) iface.Embedder {
	// TODO: Implement proper embedder extraction from options
	// For now, return nil - the vector store should handle this
	return nil
}

// getCombinedOptions processes call-specific options.
// Since we now use struct fields for defaults, this mainly handles call-specific overrides.
func (r *VectorStoreRetriever) getCombinedOptions(callOptions ...core.Option) map[string]any {
	combined := make(map[string]any)
	// Set defaults from struct fields
	combined["k"] = r.defaultK
	combined["score_threshold"] = r.scoreThreshold

	// Apply call-specific options, potentially overriding defaults
	for _, opt := range callOptions {
		opt.Apply(&combined)
	}
	return combined
}

// GetRelevantDocuments retrieves documents from the vector store based on the query.
// This method adheres to the core.Retriever interface and uses the retriever's default configuration.
// For call-specific options, use the Invoke method.
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Use the retriever's default configuration
	return r.getRelevantDocumentsWithOptions(ctx, query)
}

// getRelevantDocumentsWithOptions is an internal helper that accepts options.
func (r *VectorStoreRetriever) getRelevantDocumentsWithOptions(ctx context.Context, query string, options ...core.Option) ([]schema.Document, error) {
	startTime := time.Now()

	// Extract options
	combinedOptionsMap := r.getCombinedOptions(options...)
	k := r.defaultK
	if kOpt, ok := combinedOptionsMap["k"].(int); ok && kOpt > 0 {
		k = kOpt
	}

	// Get embedder from options
	embedder := r.getEmbedder(options...)

	// Create tracing span
	var span trace.Span
	if r.enableTracing && r.tracer != nil {
		ctx, span = r.tracer.Start(ctx, "vector_store_retriever.retrieve",
			trace.WithAttributes(
				attribute.String("retriever.type", "vector_store"),
				attribute.String("query", query),
				attribute.Int("k", k),
				attribute.Float64("score_threshold", float64(r.scoreThreshold)),
			))
		defer span.End()
	}

	// Log the retrieval operation
	if r.logger != nil {
		r.logger.Info("retrieving documents",
			"query", query,
			"k", k,
			"score_threshold", r.scoreThreshold,
		)
	}

	// Use SimilaritySearchByQuery since we have a string query
	docs, scores, err := r.vectorStore.SimilaritySearchByQuery(ctx, query, k, embedder)
	duration := time.Since(startTime)

	// Record metrics
	if r.enableMetrics && r.metrics != nil {
		avgScore := 0.0
		if len(scores) > 0 {
			sum := 0.0
			for _, score := range scores {
				sum += float64(score)
			}
			avgScore = sum / float64(len(scores))
		}

		r.metrics.RecordRetrieval(ctx, "vector_store", duration, len(docs), avgScore, err)
	}

	// Handle errors
	if err != nil {
		if r.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		if r.logger != nil {
			r.logger.Error("retrieval failed",
				"error", err,
				"query", query,
				"duration", duration,
			)
		}
		return nil, NewRetrieverError("getRelevantDocumentsWithOptions", err, ErrCodeRetrievalFailed)
	}

	// Apply score threshold filtering
	if r.scoreThreshold > 0 && len(scores) == len(docs) {
		filteredDocs := make([]schema.Document, 0, len(docs))
		for i, score := range scores {
			if score >= r.scoreThreshold {
				filteredDocs = append(filteredDocs, docs[i])
			}
		}
		docs = filteredDocs
	}

	// Log successful retrieval
	if r.logger != nil {
		r.logger.Info("retrieval completed",
			"documents_returned", len(docs),
			"duration", duration,
		)
	}

	if r.enableTracing && span != nil {
		span.SetAttributes(
			attribute.Int("documents_returned", len(docs)),
			attribute.Float64("duration_seconds", duration.Seconds()),
		)
		span.SetStatus(codes.Ok, "retrieval completed")
	}

	return docs, nil
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (r *VectorStoreRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	query, ok := input.(string)
	if !ok {
		return nil, NewRetrieverErrorWithMessage("Invoke", nil, ErrCodeInvalidInput,
			fmt.Sprintf("invalid input type for VectorStoreRetriever: expected string, got %T", input))
	}
	// Invoke uses the internal helper that can take call-specific options
	return r.getRelevantDocumentsWithOptions(ctx, query, options...)
}

// Batch implements the core.Runnable interface.
func (r *VectorStoreRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		output, err := r.Invoke(ctx, input, options...)
		if err != nil && firstErr == nil {
			firstErr = NewRetrieverErrorWithMessage("Batch", err, ErrCodeRetrievalFailed,
				fmt.Sprintf("error processing batch item %d", i))
		}
		results[i] = output
	}
	return results, firstErr
}

// Stream implements the core.Runnable interface.
// Streaming is not typically applicable to retrievers, so it returns an error.
func (r *VectorStoreRetriever) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	return nil, NewRetrieverErrorWithMessage("Stream", nil, ErrCodeInvalidInput,
		"streaming is not supported by VectorStoreRetriever")
}

// CheckHealth implements the core.HealthChecker interface.
// It performs a basic health check by validating the retriever's configuration and dependencies.
func (r *VectorStoreRetriever) CheckHealth(ctx context.Context) error {
	// Create tracing span for health check
	var span trace.Span
	if r.enableTracing && r.tracer != nil {
		ctx, span = r.tracer.Start(ctx, "vector_store_retriever.health_check")
		defer span.End()
	}

	// Log health check operation
	if r.logger != nil {
		r.logger.Debug("performing health check")
	}

	// Validate configuration
	if r.defaultK < 1 || r.defaultK > 100 {
		err := NewRetrieverErrorWithMessage("CheckHealth", nil, ErrCodeInvalidConfig,
			"invalid defaultK configuration")
		if r.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	if r.scoreThreshold < 0 || r.scoreThreshold > 1 {
		err := NewRetrieverErrorWithMessage("CheckHealth", nil, ErrCodeInvalidConfig,
			"invalid scoreThreshold configuration")
		if r.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	// Check if vector store is available (basic check)
	if r.vectorStore == nil {
		err := NewRetrieverErrorWithMessage("CheckHealth", nil, ErrCodeInvalidConfig,
			"vector store is not configured")
		if r.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	// Log successful health check
	if r.logger != nil {
		r.logger.Debug("health check passed")
	}

	if r.enableTracing && span != nil {
		span.SetStatus(codes.Ok, "health check passed")
	}

	return nil
}

// Compile-time check to ensure VectorStoreRetriever implements interfaces.
var _ core.Retriever = (*VectorStoreRetriever)(nil)
var _ core.Runnable = (*VectorStoreRetriever)(nil)
var _ core.HealthChecker = (*VectorStoreRetriever)(nil)
