package ollama

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/ollama/ollama/api"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for Ollama embedder
type Config struct {
	ServerURL  string
	Model      string
	Timeout    time.Duration
	MaxRetries int
	KeepAlive  string
	Enabled    bool
}

// HealthChecker interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) error
}

// OllamaEmbedder implements the iface.Embedder interface using a local Ollama instance.
type OllamaEmbedder struct {
	client Client
	config *Config
	tracer trace.Tracer
}

// NewOllamaEmbedder creates a new OllamaEmbedder with the given configuration.
// It creates an Ollama client from environment variables.
func NewOllamaEmbedder(config *Config, tracer trace.Tracer) (*OllamaEmbedder, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, iface.WrapError(err, iface.ErrCodeConnectionFailed, "failed to create ollama client")
	}

	return NewOllamaEmbedderWithClient(config, tracer, client)
}

// NewOllamaEmbedderWithClient creates a new OllamaEmbedder with a provided client.
// This is primarily used for testing with mocked clients.
func NewOllamaEmbedderWithClient(config *Config, tracer trace.Tracer, client Client) (*OllamaEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.Model == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "ollama model name is required")
	}

	if client == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeConnectionFailed, "client cannot be nil")
	}

	return &OllamaEmbedder{
		client: client,
		config: config,
		tracer: tracer,
	}, nil
}

// EmbedDocuments creates embeddings for a batch of document texts.
// Ollama API currently processes one document at a time for embeddings.
func (e *OllamaEmbedder) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	ctx, span := e.tracer.Start(ctx, "ollama.embed_documents",
		trace.WithAttributes(
			attribute.String("provider", "ollama"),
			attribute.String("model", e.config.Model),
			attribute.Int("document_count", len(documents)),
		))
	defer span.End()

	defer func() {

	}()

	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(documents))
	var firstErr error

	for i, doc := range documents {
		docCtx, docSpan := e.tracer.Start(ctx, "ollama.embed_single_document",
			trace.WithAttributes(
				attribute.Int("document_index", i),
				attribute.Int("document_length", len(doc)),
			))

		req := &api.EmbeddingRequest{
			Model:  e.config.Model,
			Prompt: doc,
			// TODO: Add Options map for keep_alive, etc. if needed
		}

		resp, err := e.client.Embeddings(docCtx, req)
		docSpan.End()

		if err != nil {
			if firstErr == nil {
				firstErr = iface.WrapError(err, iface.ErrCodeEmbeddingFailed, "ollama embeddings failed for document %d", i)
			}

			embeddings[i] = nil
			continue
		}

		// Convert []float64 to []float32
		embeddingF32 := make([]float32, len(resp.Embedding))
		for j, val := range resp.Embedding {
			embeddingF32[j] = float32(val)
		}
		embeddings[i] = embeddingF32
	}

	// If any errors occurred, return the first one encountered with partial results
	if firstErr != nil {
		span.RecordError(firstErr)
		span.SetStatus(codes.Error, firstErr.Error())
		return embeddings, firstErr
	}

	if len(embeddings) > 0 && len(embeddings[0]) > 0 {
		span.SetAttributes(
			attribute.Int("output_dimension", len(embeddings[0])),
		)

	}

	return embeddings, nil
}

// EmbedQuery creates an embedding for a single query string.
func (e *OllamaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	ctx, span := e.tracer.Start(ctx, "ollama.embed_query",
		trace.WithAttributes(
			attribute.String("provider", "ollama"),
			attribute.String("model", e.config.Model),
			attribute.Int("query_length", len(query)),
		))
	defer span.End()

	defer func() {

	}()

	if query == "" {
		err := iface.NewEmbeddingError(iface.ErrCodeInvalidParameters, "query cannot be empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	req := &api.EmbeddingRequest{
		Model:  e.config.Model,
		Prompt: query,
		// TODO: Add Options map for keep_alive, etc. if needed
	}

	resp, err := e.client.Embeddings(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, iface.WrapError(err, iface.ErrCodeEmbeddingFailed, "ollama embeddings for query failed")
	}

	// Convert []float64 to []float32
	embeddingF32 := make([]float32, len(resp.Embedding))
	for j, val := range resp.Embedding {
		embeddingF32[j] = float32(val)
	}

	span.SetAttributes(
		attribute.Int("output_dimension", len(embeddingF32)),
	)

	return embeddingF32, nil
}

// GetDimension returns the dimensionality of embeddings.
// For Ollama, this might vary by model, so we return 0 as unknown.
func (e *OllamaEmbedder) GetDimension(ctx context.Context) (int, error) {
	_, span := e.tracer.Start(ctx, "ollama.get_dimension",
		trace.WithAttributes(
			attribute.String("provider", "ollama"),
			attribute.String("model", e.config.Model),
		))
	defer span.End()

	// TODO: This could be improved by querying the model info from Ollama
	// For now, return 0 indicating unknown dimension
	span.SetAttributes(attribute.String("dimension_status", "unknown"))
	return 0, nil
}

// Check performs a health check on the Ollama embedder
func (e *OllamaEmbedder) Check(ctx context.Context) error {
	_, span := e.tracer.Start(ctx, "ollama.health_check")
	defer span.End()

	// Perform a lightweight embedding request for health check
	_, err := e.EmbedQuery(ctx, "health check")
	return err
}

// Ensure OllamaEmbedder implements the interfaces.
var _ iface.Embedder = (*OllamaEmbedder)(nil)
var _ HealthChecker = (*OllamaEmbedder)(nil)
