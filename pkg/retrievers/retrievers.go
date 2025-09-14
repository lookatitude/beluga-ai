// Package retrievers provides implementations of retrieval components for RAG (Retrieval Augmented Generation) pipelines.
//
// This package offers:
//   - VectorStoreRetriever: Retrieves documents from vector stores using similarity search
//   - Configuration management with validation
//   - OpenTelemetry integration for metrics and tracing
//   - Custom error types for better error handling
//   - Functional options pattern for flexible configuration
//
// Basic usage:
//
//	import "github.com/lookatitude/beluga-ai/pkg/retrievers"
//
//	// Create a vector store retriever
//	retriever := retrievers.NewVectorStoreRetriever(vectorStore,
//	    retrievers.WithDefaultK(5),
//	    retrievers.WithScoreThreshold(0.7),
//	)
//
//	// Retrieve documents
//	docs, err := retriever.GetRelevantDocuments(ctx, "What is machine learning?")
//
// Extensibility:
//
// The package is designed to be easily extensible:
//   - Add new retriever implementations in the providers/ directory
//   - Implement custom retrievers by implementing the core.Retriever interface
//   - Add new configuration options using the functional options pattern
//   - Extend metrics collection for new retriever types
package retrievers

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Retriever represents a component that can retrieve relevant documents for a given query.
type Retriever interface {
	core.Retriever
}

// Option is a functional option for configuring retrievers.
type Option func(*RetrieverOptions)

// RetrieverOptions holds configuration options for retrievers.
type RetrieverOptions struct {
	DefaultK       int
	ScoreThreshold float32
	MaxRetries     int
	Timeout        time.Duration
	EnableTracing  bool
	EnableMetrics  bool
	Logger         *slog.Logger
	Tracer         trace.Tracer
	Meter          metric.Meter
	Metrics        *Metrics
}

// WithDefaultK sets the default number of documents to retrieve.
func WithDefaultK(k int) Option {
	return func(opts *RetrieverOptions) {
		opts.DefaultK = k
	}
}

// WithMaxRetries sets the maximum number of retries for failed operations.
func WithMaxRetries(retries int) Option {
	return func(opts *RetrieverOptions) {
		opts.MaxRetries = retries
	}
}

// WithTimeout sets the timeout for operations.
func WithTimeout(timeout time.Duration) Option {
	return func(opts *RetrieverOptions) {
		opts.Timeout = timeout
	}
}

// WithTracing enables or disables tracing.
func WithTracing(enabled bool) Option {
	return func(opts *RetrieverOptions) {
		opts.EnableTracing = enabled
	}
}

// WithMetrics enables or disables metrics collection.
func WithMetrics(enabled bool) Option {
	return func(opts *RetrieverOptions) {
		opts.EnableMetrics = enabled
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(opts *RetrieverOptions) {
		opts.Logger = logger
	}
}

// WithTracer sets a custom tracer.
func WithTracer(tracer trace.Tracer) Option {
	return func(opts *RetrieverOptions) {
		opts.Tracer = tracer
	}
}

// WithMeter sets a custom metrics meter.
func WithMeter(meter metric.Meter) Option {
	return func(opts *RetrieverOptions) {
		opts.Meter = meter
	}
}

// NewVectorStoreRetriever creates a new VectorStoreRetriever with the given vector store and options.
//
// Example:
//
//	retriever := retrievers.NewVectorStoreRetriever(vectorStore,
//	    retrievers.WithDefaultK(5),
//	    retrievers.WithScoreThreshold(0.7),
//	    retrievers.WithTimeout(30*time.Second),
//	)
func NewVectorStoreRetriever(vectorStore vectorstores.VectorStore, options ...Option) (*VectorStoreRetriever, error) {
	opts := &RetrieverOptions{
		DefaultK:       4,
		ScoreThreshold: 0.0,
		MaxRetries:     3,
		Timeout:        30 * time.Second,
		EnableTracing:  true,
		EnableMetrics:  true,
	}

	for _, option := range options {
		option(opts)
	}

	// Validate configuration
	if opts.DefaultK < 1 || opts.DefaultK > 100 {
		return nil, NewRetrieverErrorWithMessage("NewVectorStoreRetriever", nil, ErrCodeInvalidConfig,
			fmt.Sprintf("DefaultK must be between 1 and 100, got %d", opts.DefaultK))
	}

	// Create metrics if enabled
	if opts.EnableMetrics && opts.Meter != nil {
		var err error
		opts.Metrics, err = NewMetrics(opts.Meter)
		if err != nil {
			return nil, NewRetrieverError("NewVectorStoreRetriever", err, ErrCodeInvalidConfig)
		}
	}

	// Set up default logger if not provided
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return newVectorStoreRetrieverInternal(vectorStore, opts), nil
}

// NewVectorStoreRetrieverFromConfig creates a VectorStoreRetriever from a configuration struct.
//
// Example:
//
//	config := retrievers.VectorStoreRetrieverConfig{
//	    K: 5,
//	    ScoreThreshold: 0.7,
//	    Timeout: 30 * time.Second,
//	}
//	retriever, err := retrievers.NewVectorStoreRetrieverFromConfig(vectorStore, config)
func NewVectorStoreRetrieverFromConfig(vectorStore vectorstores.VectorStore, config VectorStoreRetrieverConfig) (*VectorStoreRetriever, error) {
	// Apply defaults to unset fields
	config.ApplyDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewRetrieverError("NewVectorStoreRetrieverFromConfig", err, ErrCodeInvalidConfig)
	}

	opts := &RetrieverOptions{
		DefaultK:       config.K,
		ScoreThreshold: config.ScoreThreshold,
		MaxRetries:     3, // Default value
		Timeout:        config.Timeout,
		EnableTracing:  true,
		EnableMetrics:  true,
		Logger:         slog.Default(),
	}

	return newVectorStoreRetrieverInternal(vectorStore, opts), nil
}

// GetRetrieverTypes returns a list of available retriever types.
func GetRetrieverTypes() []string {
	return []string{
		"vector_store",
		// Add more retriever types as they are implemented
	}
}

// ValidateRetrieverConfig validates a retriever configuration.
func ValidateRetrieverConfig(config Config) error {
	return config.Validate()
}
