// Package cohere provides a Cohere embeddings provider for the Beluga AI framework.
// It implements the embedding.Embedder interface using the Cohere Embed API
// via the internal httpclient.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
//
//	emb, err := embedding.New("cohere", config.ProviderConfig{
//	    APIKey: "...",
//	})
package cohere

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("cohere", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "embed-english-v3.0"
	defaultDimensions = 1024
	defaultBaseURL    = "https://api.cohere.com/v2"
)

// modelDimensions maps known Cohere embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"embed-english-v3.0":      1024,
	"embed-multilingual-v3.0": 1024,
	"embed-english-light-v3.0": 384,
	"embed-multilingual-light-v3.0": 384,
	"embed-english-v2.0":      4096,
}

// Embedder implements embedding.Embedder using the Cohere Embed API.
type Embedder struct {
	client    *httpclient.Client
	model     string
	dims      int
	inputType string
}

// New creates a new Cohere Embedder from a ProviderConfig.
func New(cfg config.ProviderConfig) (*Embedder, error) {
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	dims := defaultDimensions
	if d, ok := modelDimensions[model]; ok {
		dims = d
	}
	if d, ok := config.GetOption[float64](cfg, "dimensions"); ok && d > 0 {
		dims = int(d)
	}

	inputType := "search_document"
	if it, ok := config.GetOption[string](cfg, "input_type"); ok && it != "" {
		inputType = it
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	opts := []httpclient.Option{
		httpclient.WithBaseURL(baseURL),
		httpclient.WithBearerToken(cfg.APIKey),
	}
	if cfg.Timeout > 0 {
		opts = append(opts, httpclient.WithTimeout(cfg.Timeout))
	}

	client := httpclient.New(opts...)

	return &Embedder{
		client:    client,
		model:     model,
		dims:      dims,
		inputType: inputType,
	}, nil
}

// embedRequest is the request body for the Cohere embed API.
type embedRequest struct {
	Texts         []string `json:"texts"`
	Model         string   `json:"model"`
	InputType     string   `json:"input_type"`
	EmbeddingTypes []string `json:"embedding_types"`
}

// embedResponse is the response from the Cohere embed API.
type embedResponse struct {
	ID         string `json:"id"`
	Embeddings struct {
		Float [][]float32 `json:"float"`
	} `json:"embeddings"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body := embedRequest{
		Texts:         texts,
		Model:         e.model,
		InputType:     e.inputType,
		EmbeddingTypes: []string{"float"},
	}

	resp, err := httpclient.DoJSON[embedResponse](ctx, e.client, http.MethodPost, "embed", body)
	if err != nil {
		return nil, fmt.Errorf("cohere embedding: %w", err)
	}

	return resp.Embeddings.Float, nil
}

// EmbedSingle produces an embedding for a single text.
func (e *Embedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("cohere embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
