// Package google_multimodal provides multimodal embedding support using Google's Gemini API.
// This provider extends Google embeddings to support images alongside text.
package google_multimodal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for Google multimodal embedder.
type Config struct {
	APIKey     string
	Model      string // Use multimodal models like "text-embedding-004" or "multimodalembedding"
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	Enabled    bool
}

// GoogleMultimodalEmbedder implements the MultimodalEmbedder interface.
type GoogleMultimodalEmbedder struct {
	httpClient *http.Client
	config     *Config
	tracer     trace.Tracer
	baseURL    string
}

// Google embedding API request/response structures.
type googleEmbedRequest struct {
	Model  string              `json:"model,omitempty"`
	Task   string              `json:"task,omitempty"` // "RETRIEVAL_DOCUMENT", "RETRIEVAL_QUERY", "SEMANTIC_SIMILARITY", etc.
	Title  string              `json:"title,omitempty"`
	Text   string              `json:"text,omitempty"`
	Images []googleImageInput  `json:"images,omitempty"`
}

type googleImageInput struct {
	Data     string `json:"data,omitempty"`     // Base64 encoded image
	MimeType string `json:"mime_type,omitempty"`
}

type googleEmbedResponse struct {
	Embedding googleEmbeddingValue `json:"embedding"`
}

type googleEmbeddingValue struct {
	Values []float32 `json:"values"`
}

// NewGoogleMultimodalEmbedder creates a new multimodal embedder.
func NewGoogleMultimodalEmbedder(config *Config, tracer trace.Tracer) (*GoogleMultimodalEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "google API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &GoogleMultimodalEmbedder{
		httpClient: httpClient,
		config:     config,
		tracer:     tracer,
		baseURL:    baseURL,
	}, nil
}

// EmbedDocuments implements the Embedder interface (text-only fallback).
func (e *GoogleMultimodalEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	ctx, span := e.tracer.Start(ctx, "google_multimodal.embed_documents",
		trace.WithAttributes(
			attribute.String("provider", "google_multimodal"),
			attribute.String("model", e.config.Model),
			attribute.Int("document_count", len(texts)),
		))
	defer span.End()

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := e.embedText(ctx, text, "RETRIEVAL_DOCUMENT")
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

// EmbedQuery implements the Embedder interface (text-only fallback).
func (e *GoogleMultimodalEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.embedText(ctx, text, "RETRIEVAL_QUERY")
}

// GetDimension implements the Embedder interface.
func (e *GoogleMultimodalEmbedder) GetDimension(ctx context.Context) (int, error) {
	// Google's text-embedding-004 model produces 768-dimensional embeddings
	// Multimodal embeddings may vary, but default to 768
	if e.config.Model == "text-embedding-004" {
		return 768, nil
	}
	// Default dimension for multimodal models
	return 768, nil
}

// EmbedDocumentsMultimodal implements the MultimodalEmbedder interface.
func (e *GoogleMultimodalEmbedder) EmbedDocumentsMultimodal(ctx context.Context, documents []schema.Document) ([][]float32, error) {
	ctx, span := e.tracer.Start(ctx, "google_multimodal.embed_documents_multimodal",
		trace.WithAttributes(
			attribute.String("provider", "google_multimodal"),
			attribute.String("model", e.config.Model),
			attribute.Int("document_count", len(documents)),
		))
	defer span.End()

	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(documents))
	for i, doc := range documents {
		embedding, err := e.embedDocumentMultimodal(ctx, doc, "RETRIEVAL_DOCUMENT")
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
func (e *GoogleMultimodalEmbedder) EmbedQueryMultimodal(ctx context.Context, document schema.Document) ([]float32, error) {
	return e.embedDocumentMultimodal(ctx, document, "RETRIEVAL_QUERY")
}

// SupportsMultimodal returns true.
func (e *GoogleMultimodalEmbedder) SupportsMultimodal() bool {
	return true
}

// embedText embeds a text string.
func (e *GoogleMultimodalEmbedder) embedText(ctx context.Context, text string, task string) ([]float32, error) {
	req := &googleEmbedRequest{
		Model: e.config.Model,
		Task:  task,
		Text:  text,
	}

	return e.makeEmbeddingRequest(ctx, req)
}

// embedDocumentMultimodal embeds a document that may contain multimodal content.
func (e *GoogleMultimodalEmbedder) embedDocumentMultimodal(ctx context.Context, doc schema.Document, task string) ([]float32, error) {
	// Check if document has image content in metadata
	imageURL, hasImageURL := doc.Metadata["image_url"]
	imageBase64, hasImageBase64 := doc.Metadata["image_base64"]
	imageType, _ := doc.Metadata["image_type"] // e.g., "image/png", "image/jpeg"

	req := &googleEmbedRequest{
		Model: e.config.Model,
		Task:  task,
		Text:  doc.PageContent,
	}

	// Add image content if present
	if hasImageURL {
		// Fetch and encode image from URL
		imageData, err := e.fetchImageFromURL(ctx, imageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch image from URL: %w", err)
		}
		mimeType := imageType
		if mimeType == "" {
			mimeType = "image/jpeg" // Default
		}
		req.Images = []googleImageInput{
			{
				Data:     base64.StdEncoding.EncodeToString(imageData),
				MimeType: mimeType,
			},
		}
	} else if hasImageBase64 {
		// Use base64 encoded image
		mimeType := imageType
		if mimeType == "" {
			mimeType = "image/jpeg" // Default
		}
		req.Images = []googleImageInput{
			{
				Data:     imageBase64,
				MimeType: mimeType,
			},
		}
	}

	return e.makeEmbeddingRequest(ctx, req)
}

// makeEmbeddingRequest makes an HTTP request to the Google embedding API.
func (e *GoogleMultimodalEmbedder) makeEmbeddingRequest(ctx context.Context, req *googleEmbedRequest) ([]float32, error) {
	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", e.baseURL, e.config.Model, e.config.APIKey)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp googleEmbedResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding in response")
	}

	return geminiResp.Embedding.Values, nil
}

// fetchImageFromURL fetches an image from a URL and returns it as bytes.
func (e *GoogleMultimodalEmbedder) fetchImageFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// Ensure GoogleMultimodalEmbedder implements the interfaces.
var (
	_ iface.Embedder            = (*GoogleMultimodalEmbedder)(nil)
	_ iface.MultimodalEmbedder  = (*GoogleMultimodalEmbedder)(nil)
)
