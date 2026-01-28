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
//	:= retrievers.NewVectorStoreRetriever(vectorStore,
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
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
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
	Tracer         trace.Tracer
	Meter          metric.Meter
	Logger         *slog.Logger
	Metrics        *Metrics
	DefaultK       int
	MaxRetries     int
	Timeout        time.Duration
	ScoreThreshold float32
	EnableTracing  bool
	EnableMetrics  bool
}

// WithDefaultK sets the default number of documents to retrieve.
// This is used when GetRelevantDocuments is called without specifying k.
//
// Parameters:
//   - k: Number of documents to retrieve by default
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	retriever, _ := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithDefaultK(10),
//	)
//
// Example usage can be found in examples/rag/simple/main.go.
func WithDefaultK(k int) Option {
	return func(opts *RetrieverOptions) {
		opts.DefaultK = k
	}
}

// WithMaxRetries sets the maximum number of retries for failed operations.
// Retries are attempted with exponential backoff.
//
// Parameters:
//   - retries: Maximum number of retry attempts
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	retriever, _ := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithMaxRetries(3),
//	)
//
// Example usage can be found in examples/rag/simple/main.go.
func WithMaxRetries(retries int) Option {
	return func(opts *RetrieverOptions) {
		opts.MaxRetries = retries
	}
}

// WithTimeout sets the timeout for operations.
// Operations that exceed this timeout will be canceled.
//
// Parameters:
//   - timeout: Maximum duration for operations
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	retriever, _ := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithTimeout(30*time.Second),
//	)
//
// Example usage can be found in examples/rag/simple/main.go.
func WithTimeout(timeout time.Duration) Option {
	return func(opts *RetrieverOptions) {
		opts.Timeout = timeout
	}
}

// WithTracing enables or disables tracing.
// When enabled, operations are traced using OpenTelemetry.
//
// Parameters:
//   - enabled: Whether to enable tracing
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	retriever, _ := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithTracing(true),
//	)
//
// Example usage can be found in examples/rag/simple/main.go.
func WithTracing(enabled bool) Option {
	return func(opts *RetrieverOptions) {
		opts.EnableTracing = enabled
	}
}

// WithMetrics enables or disables metrics collection.
// When enabled, operation metrics are recorded using OpenTelemetry.
//
// Parameters:
//   - enabled: Whether to enable metrics
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	retriever, _ := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithMetrics(true),
//	)
//
// Example usage can be found in examples/rag/simple/main.go.
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
// The retriever uses the vector store to perform similarity search and retrieve relevant documents.
//
// Parameters:
//   - vectorStore: Vector store instance to use for retrieval
//   - options: Optional configuration functions (WithDefaultK, WithScoreThreshold, etc.)
//
// Returns:
//   - *VectorStoreRetriever: A new retriever instance
//   - error: Configuration validation errors
//
// Example:
//
//	store, _ := vectorstores.NewInMemoryStore(ctx)
//	retriever, err := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithDefaultK(5),
//	    retrievers.WithScoreThreshold(0.7),
//	    retrievers.WithTimeout(30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	docs, err := retriever.GetRelevantDocuments(ctx, "What is AI?")
//
// Example usage can be found in examples/rag/simple/main.go.
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
		opts.Metrics, err = NewMetrics(opts.Meter, opts.Tracer)
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

// NewMultiQueryRetriever creates a new MultiQueryRetriever with the given retriever and LLM.
// The retriever generates multiple query variations using the LLM to improve retrieval results.
//
// Parameters:
//   - retriever: Underlying retriever to use for document retrieval
//   - llm: Language model for generating query variations
//   - options: Optional configuration functions
//
// Returns:
//   - *MultiQueryRetriever: A new multi-query retriever instance
//   - error: Configuration validation errors
func NewMultiQueryRetriever(retriever core.Retriever, llm llmsiface.ChatModel, options ...Option) (*MultiQueryRetriever, error) {
	opts := &RetrieverOptions{
		DefaultK:      3, // Default number of query variations
		EnableTracing: true,
		EnableMetrics: true,
	}

	for _, option := range options {
		option(opts)
	}

	// Set up default logger if not provided
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	if retriever == nil {
		return nil, NewRetrieverErrorWithMessage("NewMultiQueryRetriever", nil, ErrCodeInvalidConfig,
			"retriever is required")
	}

	if llm == nil {
		return nil, NewRetrieverErrorWithMessage("NewMultiQueryRetriever", nil, ErrCodeInvalidConfig,
			"LLM is required")
	}

	return newMultiQueryRetrieverInternal(retriever, llm, opts), nil
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
func NewVectorStoreRetrieverFromConfig(vs vectorstores.VectorStore, config VectorStoreRetrieverConfig) (*VectorStoreRetriever, error) {
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

	return newVectorStoreRetrieverInternal(vs, opts), nil
}

// GetRetrieverTypes returns a list of available retriever types.
// Deprecated: Use ListProviders() instead for a dynamic list of registered providers.
func GetRetrieverTypes() []string {
	return []string{
		"vector_store",
		"multi_query",
		"mock",
	}
}

// ValidateRetrieverConfig validates a retriever configuration.
func ValidateRetrieverConfig(config Config) error {
	return config.Validate()
}

// WithScoreThreshold sets the minimum similarity score threshold.
// Documents with scores below this threshold will be filtered out.
//
// Parameters:
//   - threshold: Minimum score (0.0 to 1.0) for returned documents
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	retriever, _ := retrievers.NewVectorStoreRetriever(store,
//	    retrievers.WithScoreThreshold(0.7),
//	)
func WithScoreThreshold(threshold float32) Option {
	return func(opts *RetrieverOptions) {
		opts.ScoreThreshold = threshold
	}
}
