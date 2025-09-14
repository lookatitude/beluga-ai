package ollama

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
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
	client *api.Client
	config *Config
	tracer trace.Tracer
}

// NewOllamaEmbedder creates a new OllamaEmbedder with the given configuration.
func NewOllamaEmbedder(config *Config, tracer trace.Tracer) (*OllamaEmbedder, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Model == "" {
		return nil, fmt.Errorf("Ollama model name is required")
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	if client == nil {
		return nil, errors.New("failed to create Ollama client (nil client returned)")
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
				firstErr = fmt.Errorf("Ollama Embeddings failed for document %d: %w", i, err)
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
		err := fmt.Errorf("query cannot be empty")
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

		return nil, fmt.Errorf("Ollama Embeddings for query failed: %w", err)
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

// Invoke implements the core.Runnable interface.
// Input can be a string (for single query) or []string (for batch documents).
// Output is []float32 for single query or [][]float32 for batch.
func (e *OllamaEmbedder) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	switch v := input.(type) {
	case string:
		return e.EmbedQuery(ctx, v)
	case []string:
		return e.EmbedDocuments(ctx, v)
	default:
		return nil, fmt.Errorf("unsupported input type: %T, expected string or []string", input)
	}
}

// Batch implements the core.Runnable interface.
// Each input can be a string or []string, returns corresponding embeddings.
func (e *OllamaEmbedder) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := e.Invoke(ctx, input, options...)
		if err != nil {
			return nil, fmt.Errorf("failed to process input %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream implements the core.Runnable interface.
// For embeddings, streaming is not typically meaningful, so we return the result immediately.
func (e *OllamaEmbedder) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	resultCh := make(chan any, 1)

	go func() {
		defer close(resultCh)
		result, err := e.Invoke(ctx, input, options...)
		if err != nil {
			resultCh <- err
			return
		}
		resultCh <- result
	}()

	return resultCh, nil
}

// Ensure OllamaEmbedder implements the interfaces.
var _ iface.Embedder = (*OllamaEmbedder)(nil)
var _ core.Runnable = (*OllamaEmbedder)(nil)
var _ HealthChecker = (*OllamaEmbedder)(nil)
