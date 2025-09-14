package openai

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for OpenAI embedder
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	APIVersion string
	Timeout    time.Duration
	MaxRetries int
	Enabled    bool
}

// HealthChecker interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) error
}

// OpenAIEmbedder implements the iface.Embedder interface using the OpenAI API.
type OpenAIEmbedder struct {
	client *openai.Client
	config *Config
	tracer trace.Tracer
}

// NewOpenAIEmbedder creates a new OpenAIEmbedder with the given configuration.
func NewOpenAIEmbedder(config *Config, tracer trace.Tracer) (*OpenAIEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "openai API key is required")
	}

	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	if config.APIVersion != "" {
		// Note: OpenAI client doesn't directly expose API version in config
		// This would need to be handled differently if needed
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIEmbedder{
		client: client,
		config: config,
		tracer: tracer,
	}, nil
}

// EmbedDocuments creates embeddings for a batch of document texts.
func (e *OpenAIEmbedder) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	ctx, span := e.tracer.Start(ctx, "openai.embed_documents",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", e.config.Model),
			attribute.Int("document_count", len(documents)),
		))
	defer span.End()

	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	req := openai.EmbeddingRequest{
		Input: documents,
		Model: openai.EmbeddingModel(e.config.Model),
		User:  "", // Could be added to config if needed
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Error recorded in span
		return nil, iface.WrapError(err, iface.ErrCodeEmbeddingFailed, "openai create embeddings failed")
	}

	if len(resp.Data) != len(documents) {
		err := iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "openai returned %d embeddings for %d documents", len(resp.Data), len(documents))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		if data.Index != i {
			err := iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "openai embedding index mismatch: expected %d, got %d", i, data.Index)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, err
		}
		embeddings[i] = data.Embedding
	}

	span.SetAttributes(
		attribute.Int("response_count", len(embeddings)),
		attribute.Int("output_dimension", len(embeddings[0])),
	)

	// Metrics are handled at the factory level

	return embeddings, nil
}

// EmbedQuery creates an embedding for a single query string.
func (e *OpenAIEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	ctx, span := e.tracer.Start(ctx, "openai.embed_query",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", e.config.Model),
			attribute.Int("query_length", len(query)),
		))
	defer span.End()

	// Request tracking handled at factory level

	if query == "" {
		err := iface.NewEmbeddingError(iface.ErrCodeInvalidParameters, "query cannot be empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	req := openai.EmbeddingRequest{
		Input: []string{query},
		Model: openai.EmbeddingModel(e.config.Model),
		User:  "", // Could be added to config if needed
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Error recorded in span
		return nil, iface.WrapError(err, iface.ErrCodeEmbeddingFailed, "openai create embeddings for query failed")
	}

	if len(resp.Data) != 1 {
		err := iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "openai returned %d embeddings for 1 query", len(resp.Data))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	span.SetAttributes(
		attribute.Int("output_dimension", len(resp.Data[0].Embedding)),
	)

	return resp.Data[0].Embedding, nil
}

// GetDimension returns the dimensionality of embeddings.
func (e *OpenAIEmbedder) GetDimension(ctx context.Context) (int, error) {
	_, span := e.tracer.Start(ctx, "openai.get_dimension",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", e.config.Model),
		))
	defer span.End()

	switch e.config.Model {
	case "text-embedding-ada-002":
		return 1536, nil
	case "text-embedding-3-small":
		return 1536, nil
	case "text-embedding-3-large":
		return 3072, nil
	default:
		span.SetAttributes(attribute.String("warning", "unknown_model_defaulting"))
		return 1536, nil // Default to ada-002 dimensions
	}
}

// Check performs a health check on the OpenAI embedder
func (e *OpenAIEmbedder) Check(ctx context.Context) error {
	_, span := e.tracer.Start(ctx, "openai.health_check")
	defer span.End()

	// Perform a lightweight embedding request for health check
	_, err := e.EmbedQuery(ctx, "health check")
	return err
}


// Ensure OpenAIEmbedder implements the interfaces.
var _ iface.Embedder = (*OpenAIEmbedder)(nil)
var _ HealthChecker = (*OpenAIEmbedder)(nil)
