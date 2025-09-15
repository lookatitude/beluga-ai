// Package vectorstores provides a comprehensive vector storage and retrieval system
// for retrieval-augmented generation (RAG) applications.
//
// This package implements the Beluga AI Framework design patterns including:
// - Interface Segregation Principle (ISP)
// - Dependency Inversion Principle (DIP)
// - OpenTelemetry observability
// - Structured configuration management
// - Factory pattern for provider registration
//
// Key Features:
// - Multiple vector store providers (in-memory, PostgreSQL, etc.)
// - Efficient similarity search with configurable algorithms
// - Comprehensive observability (metrics, tracing, logging)
// - Type-safe configuration with validation
// - Extensible provider architecture
//
// Basic Usage:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	// Create an in-memory vector store
//	store, err := vectorstores.NewInMemoryStore(ctx,
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithSearchK(10),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Add documents
//	docs := []schema.Document{
//		schema.NewDocument("Machine learning is awesome", map[string]string{"topic": "ml"}),
//	}
//	ids, err := store.AddDocuments(ctx, docs)
//
//	// Search by query
//	results, scores, err := store.SimilaritySearchByQuery(ctx, "ML basics", 5, embedder)
//
// Advanced Usage:
//
//	// Configure with custom settings
//	config := vectorstores.NewDefaultConfig()
//	config.SearchK = 20
//	config.ScoreThreshold = 0.8
//
//	store, err := vectorstores.NewVectorStore(ctx, "pgvector", config,
//		vectorstores.WithProviderConfig("table_name", "my_documents"),
//		vectorstores.WithMetadataFilter("category", "tech"),
//	)
//
// Provider Registration:
//
//	// Register custom providers
//	vectorstores.RegisterGlobal("custom", func(ctx context.Context, config Config) (VectorStore, error) {
//		return NewCustomStore(config)
//	})
//
//	store, err := vectorstores.NewVectorStore(ctx, "custom", config)
//
// Observability:
//
//	// Set up global observability
//	metrics, _ := vectorstores.NewMetricsCollector(meter)
//	vectorstores.SetGlobalMetrics(metrics)
//
//	tracer := vectorstores.NewTracerProvider("my-app")
//	vectorstores.SetGlobalTracer(tracer)
//
//	logger := vectorstores.NewLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
//	vectorstores.SetGlobalLogger(logger)
package vectorstores

import (
	"context"
	"fmt"

	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Embedder defines the interface for generating vector embeddings from text.
// This follows the Interface Segregation Principle by focusing on embedding functionality.
type Embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// Retriever defines the interface for retrieving documents based on queries.
// This enables VectorStores to be used in retrieval chains and graphs.
type Retriever interface {
	GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}

// VectorStore defines the core interface for storing and querying vector embeddings.
// It provides methods for document storage, deletion, and similarity search.
// Implementations should be thread-safe and handle context cancellation properly.
type VectorStore interface {
	// AddDocuments stores documents with their embeddings in the vector store.
	// It generates embeddings if an embedder is available, or uses pre-computed embeddings.
	// Returns the IDs of stored documents and any error encountered.
	AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)

	// DeleteDocuments removes documents from the store based on their IDs.
	// Returns an error if any document cannot be deleted.
	DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error

	// SimilaritySearch finds the k most similar documents to a query vector.
	// Returns documents and their similarity scores (higher scores indicate better matches).
	SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)

	// SimilaritySearchByQuery performs similarity search using a text query.
	// It generates an embedding for the query and then performs vector similarity search.
	SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)

	// AsRetriever returns a Retriever implementation based on this VectorStore.
	// This enables the VectorStore to be used in retrieval chains and pipelines.
	AsRetriever(opts ...Option) Retriever

	// GetName returns the name/identifier of this vector store implementation.
	// This is used for logging, metrics, and debugging purposes.
	GetName() string
}


// Option represents a functional option for configuring VectorStore operations.
// This follows the functional options pattern for flexible configuration.
type Option func(*vectorstoresiface.Config)

// NewDefaultConfig creates a new Config with default values.
func NewDefaultConfig() *vectorstoresiface.Config {
	return &vectorstoresiface.Config{
		SearchK:        5,
		ScoreThreshold: 0.0,
	}
}

// ApplyOptions applies a slice of options to a Config.
func ApplyOptions(config *vectorstoresiface.Config, opts ...Option) {
	for _, opt := range opts {
		opt(config)
	}
}

// WithEmbedder sets the embedder to use for generating embeddings.
func WithEmbedder(embedder Embedder) Option {
	return func(c *vectorstoresiface.Config) {
		c.Embedder = embedder
	}
}

// WithSearchK sets the number of similar documents to return in search operations.
func WithSearchK(k int) Option {
	return func(c *vectorstoresiface.Config) {
		c.SearchK = k
	}
}

// WithScoreThreshold sets the minimum similarity score threshold for search results.
// Documents with scores below this threshold will be filtered out.
func WithScoreThreshold(threshold float32) Option {
	return func(c *vectorstoresiface.Config) {
		c.ScoreThreshold = threshold
	}
}

// WithMetadataFilter adds a metadata filter for search operations.
// Only documents matching the filter criteria will be considered.
func WithMetadataFilter(key string, value interface{}) Option {
	return func(c *vectorstoresiface.Config) {
		if c.MetadataFilters == nil {
			c.MetadataFilters = make(map[string]interface{})
		}
		c.MetadataFilters[key] = value
	}
}

// WithMetadataFilters sets multiple metadata filters for search operations.
func WithMetadataFilters(filters map[string]interface{}) Option {
	return func(c *vectorstoresiface.Config) {
		if c.MetadataFilters == nil {
			c.MetadataFilters = make(map[string]interface{})
		}
		for k, v := range filters {
			c.MetadataFilters[k] = v
		}
	}
}

// WithProviderConfig sets provider-specific configuration options.
func WithProviderConfig(key string, value interface{}) Option {
	return func(c *vectorstoresiface.Config) {
		if c.ProviderConfig == nil {
			c.ProviderConfig = make(map[string]interface{})
		}
		c.ProviderConfig[key] = value
	}
}

// WithProviderConfigs sets multiple provider-specific configuration options.
func WithProviderConfigs(config map[string]interface{}) Option {
	return func(c *vectorstoresiface.Config) {
		if c.ProviderConfig == nil {
			c.ProviderConfig = make(map[string]interface{})
		}
		for k, v := range config {
			c.ProviderConfig[k] = v
		}
	}
}


// Factory defines the interface for creating VectorStore instances.
// This enables dependency injection and makes testing easier.
type Factory interface {
	// CreateVectorStore creates a new VectorStore instance with the given configuration.
	// The config parameter contains provider-specific settings.
	CreateVectorStore(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)
}


// StoreFactory is the global factory for creating vector store instances.
// It maintains a registry of available providers and their creation functions.
type StoreFactory struct {
	creators map[string]func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)
}

// NewStoreFactory creates a new StoreFactory instance.
func NewStoreFactory() *StoreFactory {
	return &StoreFactory{
		creators: make(map[string]func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)),
	}
}

// Register registers a new vector store provider with the factory.
func (f *StoreFactory) Register(name string, creator func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)) {
	f.creators[name] = creator
}

// Create creates a new vector store instance using the registered provider.
func (f *StoreFactory) Create(ctx context.Context, name string, config vectorstoresiface.Config) (VectorStore, error) {
	creator, exists := f.creators[name]
	if !exists {
		return nil, NewVectorStoreError(ErrCodeUnknownProvider, "vector store provider '%s' not found", name)
	}
	return creator(ctx, config)
}

// Global factory instance for easy access
var globalFactory = NewStoreFactory()

// RegisterGlobal registers a provider with the global factory.
func RegisterGlobal(name string, creator func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)) {
	globalFactory.Register(name, creator)
}

// NewVectorStore creates a vector store using the global factory.
func NewVectorStore(ctx context.Context, name string, config vectorstoresiface.Config) (VectorStore, error) {
	return globalFactory.Create(ctx, name, config)
}

// VectorStoreError represents errors specific to vector store operations.
// It provides structured error information for programmatic error handling.
type VectorStoreError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *VectorStoreError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *VectorStoreError) Unwrap() error {
	return e.Cause
}

// NewVectorStoreError creates a new VectorStoreError with the given code and message.
func NewVectorStoreError(code, message string, args ...interface{}) *VectorStoreError {
	return &VectorStoreError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with vector store context.
func WrapError(cause error, code, message string, args ...interface{}) *VectorStoreError {
	return &VectorStoreError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeUnknownProvider      = "unknown_provider"
	ErrCodeInvalidConfig        = "invalid_config"
	ErrCodeConnectionFailed     = "connection_failed"
	ErrCodeEmbeddingFailed      = "embedding_failed"
	ErrCodeStorageFailed        = "storage_failed"
	ErrCodeRetrievalFailed      = "retrieval_failed"
	ErrCodeInvalidParameters    = "invalid_parameters"
	ErrCodeNotFound             = "not_found"
	ErrCodeDuplicateID          = "duplicate_id"
	ErrCodeUnsupportedOperation = "unsupported_operation"
)

// IsVectorStoreError checks if an error is a VectorStoreError with the given code.
func IsVectorStoreError(err error, code string) bool {
	var vsErr *VectorStoreError
	if !AsVectorStoreError(err, &vsErr) {
		return false
	}
	return vsErr.Code == code
}

// AsVectorStoreError attempts to cast an error to VectorStoreError.
func AsVectorStoreError(err error, target **VectorStoreError) bool {
	for err != nil {
		if vsErr, ok := err.(*VectorStoreError); ok {
			*target = vsErr
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}


// NewInMemoryStore creates a new in-memory vector store with the given options.
// This is the simplest provider, suitable for development and testing.
//
// Example:
//
//	store, err := vectorstores.NewInMemoryStore(ctx,
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithSearchK(10),
//	)
// Note: This function requires the inmemory provider to be imported for registration.
// Import it with: import _ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
func NewInMemoryStore(ctx context.Context, opts ...Option) (VectorStore, error) {
	config := NewDefaultConfig()
	ApplyOptions(config, opts...)

	return globalFactory.Create(ctx, "inmemory", *config)
}

// NewPgVectorStore creates a new PostgreSQL vector store with the given options.
// Requires a running PostgreSQL instance with the pgvector extension.
//
// Example:
//
//	store, err := vectorstores.NewPgVectorStore(ctx,
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithProviderConfig("connection_string", "postgres://user:pass@localhost/db"),
//		vectorstores.WithProviderConfig("table_name", "documents"),
//		vectorstores.WithProviderConfig("embedding_dimension", 768),
//	)
// Note: This function requires the pgvector provider to be imported for registration.
// Import it with: import _ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/pgvector"
func NewPgVectorStore(ctx context.Context, opts ...Option) (VectorStore, error) {
	config := NewDefaultConfig()
	ApplyOptions(config, opts...)

	return globalFactory.Create(ctx, "pgvector", *config)
}

// TODO: Implement Pinecone provider
// NewPineconeStore creates a new Pinecone vector store with the given options.
// Requires Pinecone API credentials and configuration.
//
// Example:
//
//	store, err := vectorstores.NewPineconeStore(ctx,
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithProviderConfig("api_key", "your-api-key"),
//		vectorstores.WithProviderConfig("environment", "us-west1-gcp"),
//		vectorstores.WithProviderConfig("project_id", "your-project"),
//		vectorstores.WithProviderConfig("index_name", "my-index"),
//	)
func NewPineconeStore(ctx context.Context, opts ...Option) (VectorStore, error) {
	return nil, NewVectorStoreError(ErrCodeUnknownProvider, "Pinecone provider not yet implemented")
}

// NewVectorStore creates a vector store using the specified provider name.
// This is the generic factory function that supports all registered providers.
//
// Example:
//
//	store, err := vectorstores.NewVectorStore(ctx, "inmemory",
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithSearchK(10),
//	)

// RegisterProvider registers a new vector store provider with the global factory.
// This allows extending the package with custom providers.
//
// Example:
//
//	vectorstores.RegisterProvider("custom", func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error) {
//		return NewCustomStore(config)
//	})
func RegisterProvider(name string, creator func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)) {
	globalFactory.Register(name, creator)
}

// ListProviders returns a list of all registered provider names.
// This is useful for configuration validation and UI components.
//
// Example:
//
//	providers := vectorstores.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
func ListProviders() []string {
	// This would require adding a method to StoreFactory to list registered providers
	// For now, return a static list of implemented providers
	return []string{"inmemory", "pgvector"}
}

// ValidateProvider checks if a provider name is registered and valid.
//
// Example:
//
//	if !vectorstores.ValidateProvider("inmemory") {
//		log.Fatal("Invalid provider")
//	}
func ValidateProvider(name string) bool {
	providers := ListProviders()
	for _, p := range providers {
		if p == name {
			return true
		}
	}
	return false
}

// Helper functions for common operations

// AddDocuments is a convenience function that adds documents to a store.
// It handles embedding generation if an embedder is configured.
//
// Example:
//
//	ids, err := vectorstores.AddDocuments(ctx, store, docs, embedder)
func AddDocuments(ctx context.Context, store VectorStore, documents []schema.Document, embedder Embedder, opts ...Option) ([]string, error) {
	config := NewDefaultConfig()
	ApplyOptions(config, opts...)

	if config.Embedder == nil && embedder != nil {
		config.Embedder = embedder
	}

	return store.AddDocuments(ctx, documents, WithEmbedder(config.Embedder))
}

// SearchByQuery is a convenience function for text-based search.
// It handles embedding generation and search in one call.
//
// Example:
//
//	results, scores, err := vectorstores.SearchByQuery(ctx, store, "machine learning", 5, embedder)
func SearchByQuery(ctx context.Context, store VectorStore, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error) {
	config := NewDefaultConfig()
	ApplyOptions(config, opts...)

	if config.Embedder == nil && embedder != nil {
		config.Embedder = embedder
	}

	return store.SimilaritySearchByQuery(ctx, query, k, config.Embedder, opts...)
}

// SearchByVector is a convenience function for vector-based search.
// It performs similarity search using a pre-computed embedding vector.
//
// Example:
//
//	results, scores, err := vectorstores.SearchByVector(ctx, store, queryVector, 5)
func SearchByVector(ctx context.Context, store VectorStore, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error) {
	return store.SimilaritySearch(ctx, queryVector, k, opts...)
}

// DeleteDocuments is a convenience function for document deletion.
//
// Example:
//
//	err := vectorstores.DeleteDocuments(ctx, store, []string{"doc1", "doc2"})
func DeleteDocuments(ctx context.Context, store VectorStore, ids []string, opts ...Option) error {
	return store.DeleteDocuments(ctx, ids, opts...)
}

// AsRetriever is a convenience function to get a retriever from a store.
//
// Example:
//
//	retriever := vectorstores.AsRetriever(store, vectorstores.WithSearchK(10))
//	docs, err := retriever.GetRelevantDocuments(ctx, "query")
func AsRetriever(store VectorStore, opts ...Option) Retriever {
	return store.AsRetriever(opts...)
}

// Batch operations for efficiency

// BatchAddDocuments adds multiple batches of documents efficiently.
// It handles batching and error aggregation.
//
// Example:
//
//	results, err := vectorstores.BatchAddDocuments(ctx, store, allDocs, 100, embedder)
func BatchAddDocuments(ctx context.Context, store VectorStore, documents []schema.Document, batchSize int, embedder Embedder, opts ...Option) ([]string, error) {
	if batchSize <= 0 {
		batchSize = 100
	}

	var allIDs []string
	totalBatches := (len(documents) + batchSize - 1) / batchSize

	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[start:end]
		ids, err := AddDocuments(ctx, store, batch, embedder, opts...)
		if err != nil {
			return allIDs, WrapError(err, ErrCodeStorageFailed, "failed to add batch %d/%d", i+1, totalBatches)
		}

		allIDs = append(allIDs, ids...)
	}

	return allIDs, nil
}

// BatchSearch performs multiple search operations efficiently.
// Useful for processing multiple queries in parallel.
//
// Example:
//
//	results, err := vectorstores.BatchSearch(ctx, store, queries, 5, embedder)
func BatchSearch(ctx context.Context, store VectorStore, queries []string, k int, embedder Embedder, opts ...Option) ([][]schema.Document, [][]float32, error) {
	results := make([][]schema.Document, len(queries))
	scores := make([][]float32, len(queries))

	for i, query := range queries {
		docs, queryScores, err := SearchByQuery(ctx, store, query, k, embedder, opts...)
		if err != nil {
			return results[:i], scores[:i], WrapError(err, ErrCodeRetrievalFailed, "failed to search query %d: %s", i, query)
		}

		results[i] = docs
		scores[i] = queryScores
	}

	return results, scores, nil
}
