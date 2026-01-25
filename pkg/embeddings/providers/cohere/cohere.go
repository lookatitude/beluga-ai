package cohere

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for Cohere embedder.
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	Enabled    bool
}

// HealthChecker interface for health checks.
type HealthChecker interface {
	Check(ctx context.Context) error
}

// CohereEmbedder implements the iface.Embedder interface using the Cohere API.
type CohereEmbedder struct {
	client Client
	config *Config
	tracer trace.Tracer
}

// Client defines the interface for Cohere API client.
// This allows for dependency injection and testing.
type Client interface {
	Embed(ctx context.Context, texts []string, model string) (*CohereEmbedResponse, error)
}

// CohereEmbedRequest represents a request to Cohere embed API.
type CohereEmbedRequest struct {
	Model     string   `json:"model"`
	InputType string   `json:"input_type,omitempty"`
	Truncate  string   `json:"truncate,omitempty"`
	Texts     []string `json:"texts"`
}

// CohereEmbedResponse represents a response from Cohere embed API.
type CohereEmbedResponse struct {
	ID         string      `json:"id"`
	Meta       CohereMeta  `json:"meta"`
	Embeddings [][]float32 `json:"embeddings"`
}

// CohereMeta represents metadata in Cohere response.
type CohereMeta struct {
	APIVersion CohereAPIVersion `json:"api_version"`
}

// CohereAPIVersion represents API version information.
type CohereAPIVersion struct {
	Version string `json:"version"`
}

// NewCohereEmbedder creates a new CohereEmbedder with the given configuration.
// This embedder uses the Cohere API for generating text embeddings.
//
// Parameters:
//   - config: Configuration containing API key, model name, base URL, and other settings
//   - tracer: OpenTelemetry tracer for observability (can be nil)
//
// Returns:
//   - *CohereEmbedder: A new Cohere embedder instance
//   - error: Configuration validation errors or client creation errors
//
// Example:
//
//	config := &cohere.Config{
//	    APIKey:  "your-api-key",
//	    Model:   "embed-english-v3.0",
//	    Timeout: 30 * time.Second,
//	}
//	embedder, err := cohere.NewCohereEmbedder(config, tracer)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	embeddings, err := embedder.EmbedDocuments(ctx, texts)
//
// Example usage can be found in examples/rag/simple/main.go.
func NewCohereEmbedder(config *Config, tracer trace.Tracer) (*CohereEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "cohere API key is required")
	}

	// TODO: Initialize Cohere client
	// This will involve:
	// 1. Creating a Cohere client with API key
	// 2. Setting up the base URL (https://api.cohere.ai/v1)
	// 3. Configuring retry and timeout settings
	var client Client
	// client = cohere.NewClient(config.APIKey) // Placeholder

	return &CohereEmbedder{
		client: client,
		config: config,
		tracer: tracer,
	}, nil
}

// NewCohereEmbedderWithClient creates a new CohereEmbedder with a provided client.
// This is primarily used for testing with mocked clients or when you need
// to inject a custom client implementation.
//
// Parameters:
//   - config: Configuration containing API key, model name, and other settings
//   - tracer: OpenTelemetry tracer for observability (can be nil)
//   - client: Cohere client implementation (must not be nil)
//
// Returns:
//   - *CohereEmbedder: A new Cohere embedder instance
//   - error: Configuration validation errors or if client is nil
//
// Example:
//
//	mockClient := &MockCohereClient{}
//	embedder, err := cohere.NewCohereEmbedderWithClient(config, tracer, mockClient)
//
// Example usage can be found in examples/rag/simple/main.go.
func NewCohereEmbedderWithClient(config *Config, tracer trace.Tracer, client Client) (*CohereEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "cohere API key is required")
	}

	if client == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeConnectionFailed, "client cannot be nil")
	}

	return &CohereEmbedder{
		client: client,
		config: config,
		tracer: tracer,
	}, nil
}

// EmbedDocuments creates embeddings for a batch of document texts.
func (e *CohereEmbedder) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	ctx, span := e.tracer.Start(ctx, "cohere.embed_documents",
		trace.WithAttributes(
			attribute.String("provider", "cohere"),
			attribute.String("model", e.config.Model),
			attribute.Int("document_count", len(documents)),
		))
	defer span.End()

	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	// Check that client is initialized (not yet implemented)
	if e.client == nil {
		err := iface.NewEmbeddingError(iface.ErrCodeConnectionFailed,
			"cohere provider is not yet implemented: client initialization is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Prepare request
	model := e.config.Model
	if model == "" {
		model = "embed-english-v3.0" // Default Cohere model
	}

	// Call Cohere API
	resp, err := e.client.Embed(ctx, documents, model)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, iface.WrapError(err, iface.ErrCodeEmbeddingFailed, "cohere embed documents failed")
	}

	if len(resp.Embeddings) != len(documents) {
		err := iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed,
			"mismatch between input documents and output embeddings")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("embedding_count", len(resp.Embeddings)))
	if len(resp.Embeddings) > 0 {
		span.SetAttributes(attribute.Int("embedding_dimension", len(resp.Embeddings[0])))
	}

	return resp.Embeddings, nil
}

// EmbedQuery creates an embedding for a single query text.
func (e *CohereEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	ctx, span := e.tracer.Start(ctx, "cohere.embed_query",
		trace.WithAttributes(
			attribute.String("provider", "cohere"),
			attribute.String("model", e.config.Model),
		))
	defer span.End()

	if text == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidParameters, "text cannot be empty")
	}

	// Check that client is initialized (not yet implemented)
	if e.client == nil {
		err := iface.NewEmbeddingError(iface.ErrCodeConnectionFailed,
			"cohere provider is not yet implemented: client initialization is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Prepare request
	model := e.config.Model
	if model == "" {
		model = "embed-english-v3.0" // Default Cohere model
	}

	// Call Cohere API with input_type="search_query"
	resp, err := e.client.Embed(ctx, []string{text}, model)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, iface.WrapError(err, iface.ErrCodeEmbeddingFailed, "cohere embed query failed")
	}

	if len(resp.Embeddings) == 0 {
		err := iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "no embedding returned")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("embedding_dimension", len(resp.Embeddings[0])))

	return resp.Embeddings[0], nil
}

// GetDimension returns the dimension of embeddings produced by this embedder.
// Cohere models have different dimensions depending on the model.
func (e *CohereEmbedder) GetDimension(ctx context.Context) (int, error) {
	// Cohere model dimensions:
	// - embed-english-v3.0: 1024
	// - embed-multilingual-v3.0: 1024
	// - embed-english-light-v3.0: 384
	// - embed-multilingual-light-v3.0: 384
	model := e.config.Model
	if model == "" {
		model = "embed-english-v3.0"
	}

	// Determine dimension based on model
	switch model {
	case "embed-english-v3.0", "embed-multilingual-v3.0":
		return 1024, nil
	case "embed-english-light-v3.0", "embed-multilingual-light-v3.0":
		return 384, nil
	default:
		// Default to 1024 for unknown models
		return 1024, nil
	}
}

// Check implements the HealthChecker interface.
func (e *CohereEmbedder) Check(ctx context.Context) error {
	// Perform a simple health check by getting dimension
	_, err := e.GetDimension(ctx)
	return err
}
