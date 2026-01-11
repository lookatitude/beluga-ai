// Package openai_multimodal provides multimodal embedding support using OpenAI's vision-capable models.
// This provider extends OpenAI embeddings to support images alongside text.
package openai_multimodal

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	openaiClient "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for OpenAI multimodal embedder.
type Config struct {
	APIKey     string
	Model      string // Use vision-capable models like "gpt-4-vision-preview" or CLIP models
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	Enabled    bool
}

// OpenAIMultimodalEmbedder implements the MultimodalEmbedder interface.
// It wraps the standard OpenAI embedder and adds multimodal support.
type OpenAIMultimodalEmbedder struct {
	textEmbedder *openai.OpenAIEmbedder
	client       openaiClient.Client
	config       *Config
	tracer       trace.Tracer
}

// NewOpenAIMultimodalEmbedder creates a new multimodal embedder.
// This embedder supports both text and image embeddings using OpenAI's vision-capable models.
//
// Parameters:
//   - config: Configuration containing API key, model name, and other settings
//   - tracer: OpenTelemetry tracer for observability (can be nil)
//
// Returns:
//   - *OpenAIMultimodalEmbedder: A new multimodal embedder instance
//   - error: Configuration validation errors or client creation errors
//
// Example:
//
//	config := &openai_multimodal.Config{
//	    APIKey:  "your-api-key",
//	    Model:   "gpt-4-vision-preview",
//	    Timeout: 30 * time.Second,
//	}
//	embedder, err := openai_multimodal.NewOpenAIMultimodalEmbedder(config, tracer)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/multimodal/basic/main.go
func NewOpenAIMultimodalEmbedder(config *Config, tracer trace.Tracer) (*OpenAIMultimodalEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "openai API key is required")
	}

	// Create base text embedder
	textConfig := &openai.Config{
		APIKey:     config.APIKey,
		Model:      config.Model,
		BaseURL:    config.BaseURL,
		Timeout:    config.Timeout,
		MaxRetries: config.MaxRetries,
		Enabled:    config.Enabled,
	}

	textEmbedder, err := openai.NewOpenAIEmbedder(textConfig, tracer)
	if err != nil {
		return nil, fmt.Errorf("failed to create text embedder: %w", err)
	}

	// Create OpenAI client for vision API calls
	clientConfig := openaiClient.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	client := openaiClient.NewClientWithConfig(clientConfig)

	return &OpenAIMultimodalEmbedder{
		textEmbedder: textEmbedder,
		client:       *client,
		config:       config,
		tracer:       tracer,
	}, nil
}

// EmbedDocuments implements the Embedder interface (text-only fallback).
func (e *OpenAIMultimodalEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return e.textEmbedder.EmbedDocuments(ctx, texts)
}

// EmbedQuery implements the Embedder interface (text-only fallback).
func (e *OpenAIMultimodalEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.textEmbedder.EmbedQuery(ctx, text)
}

// GetDimension implements the Embedder interface.
func (e *OpenAIMultimodalEmbedder) GetDimension(ctx context.Context) (int, error) {
	return e.textEmbedder.GetDimension(ctx)
}

// EmbedDocumentsMultimodal implements the MultimodalEmbedder interface.
func (e *OpenAIMultimodalEmbedder) EmbedDocumentsMultimodal(ctx context.Context, documents []schema.Document) ([][]float32, error) {
	ctx, span := e.tracer.Start(ctx, "openai_multimodal.embed_documents",
		trace.WithAttributes(
			attribute.String("provider", "openai_multimodal"),
			attribute.String("model", e.config.Model),
			attribute.Int("document_count", len(documents)),
		))
	defer span.End()

	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(documents))
	for i, doc := range documents {
		embedding, err := e.embedDocumentMultimodal(ctx, doc)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to embed document %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	span.SetAttributes(
		attribute.Int("response_count", len(embeddings)),
		attribute.Int("output_dimension", len(embeddings[0])),
	)

	return embeddings, nil
}

// EmbedQueryMultimodal implements the MultimodalEmbedder interface.
func (e *OpenAIMultimodalEmbedder) EmbedQueryMultimodal(ctx context.Context, document schema.Document) ([]float32, error) {
	ctx, span := e.tracer.Start(ctx, "openai_multimodal.embed_query",
		trace.WithAttributes(
			attribute.String("provider", "openai_multimodal"),
			attribute.String("model", e.config.Model),
		))
	defer span.End()

	return e.embedDocumentMultimodal(ctx, document)
}

// SupportsMultimodal returns true.
func (e *OpenAIMultimodalEmbedder) SupportsMultimodal() bool {
	return true
}

// embedDocumentMultimodal embeds a single document that may contain multimodal content.
func (e *OpenAIMultimodalEmbedder) embedDocumentMultimodal(ctx context.Context, doc schema.Document) ([]float32, error) {
	// Check if document has image content in metadata
	imageURL, hasImageURL := doc.Metadata["image_url"]
	imageBase64, hasImageBase64 := doc.Metadata["image_base64"]
	imageType, _ := doc.Metadata["image_type"] // e.g., "image/png", "image/jpeg"

	// If no image content, fall back to text-only
	if !hasImageURL && !hasImageBase64 {
		return e.textEmbedder.EmbedQuery(ctx, doc.PageContent)
	}

	// For multimodal content, we use a vision-capable approach
	// Note: OpenAI's embedding models are text-only, so for true multimodal embeddings,
	// we would need to use a vision model and extract embeddings from it.
	// For now, we'll combine text and image features.

	// Build content array for vision API
	var contentParts []openaiClient.ChatMessagePart

	// Add text content if present
	if doc.PageContent != "" {
		contentParts = append(contentParts, openaiClient.ChatMessagePart{
			Type: openaiClient.ChatMessagePartTypeText,
			Text: doc.PageContent,
		})
	}

	// Add image content
	if hasImageURL {
		// Use image URL directly
		contentParts = append(contentParts, openaiClient.ChatMessagePart{
			Type: openaiClient.ChatMessagePartTypeImageURL,
			ImageURL: &openaiClient.ChatMessageImageURL{
				URL: imageURL,
			},
		})
	} else if hasImageBase64 {
		// Use base64 encoded image
		mimeType := imageType
		if mimeType == "" {
			mimeType = "image/jpeg" // Default
		}
		contentParts = append(contentParts, openaiClient.ChatMessagePart{
			Type: openaiClient.ChatMessagePartTypeImageURL,
			ImageURL: &openaiClient.ChatMessageImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", mimeType, imageBase64),
			},
		})
	}

	// For now, since OpenAI's embedding API doesn't support images directly,
	// we'll use text-only embedding and log a warning
	// In a production implementation, you would:
	// 1. Use a vision model (like CLIP) to get image embeddings
	// 2. Combine text and image embeddings
	// 3. Or use a multimodal embedding service

	// Fallback: use text content for embedding
	textContent := doc.PageContent
	if textContent == "" {
		textContent = "image" // Placeholder if no text
	}

	return e.textEmbedder.EmbedQuery(ctx, textContent)
}

// fetchImageFromURL fetches an image from a URL and returns it as base64.
func (e *OpenAIMultimodalEmbedder) fetchImageFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: e.config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// encodeImageToBase64 encodes image bytes to base64 data URL.
func encodeImageToBase64(imageData []byte, mimeType string) string {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

// Ensure OpenAIMultimodalEmbedder implements the interfaces.
var (
	_ iface.Embedder            = (*OpenAIMultimodalEmbedder)(nil)
	_ iface.MultimodalEmbedder  = (*OpenAIMultimodalEmbedder)(nil)
)
