// Package jina provides a Jina AI embeddings provider for the Beluga AI framework.
// It implements the embedding.Embedder interface using the Jina Embeddings API
// via the internal httpclient.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"
//
//	emb, err := embedding.New("jina", config.ProviderConfig{
//	    APIKey: "...",
//	})
package jina

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("jina", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "jina-embeddings-v2-base-en"
	defaultDimensions = 768
	defaultBaseURL    = "https://api.jina.ai/v1"
)

// modelDimensions maps known Jina embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"jina-embeddings-v2-base-en":  768,
	"jina-embeddings-v2-small-en": 512,
	"jina-embeddings-v2-base-de":  768,
	"jina-embeddings-v2-base-zh":  768,
	"jina-embeddings-v3":          1024,
}

// Embedder implements embedding.Embedder using the Jina AI Embeddings API.
type Embedder struct {
	client *httpclient.Client
	model  string
	dims   int
}

// Compile-time interface check.
var _ embedding.Embedder = (*Embedder)(nil)

// New creates a new Jina AI Embedder from a ProviderConfig.
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
		client: client,
		model:  model,
		dims:   dims,
	}, nil
}

// embedRequest is the request body for the Jina embeddings API.
type embedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// embedResponse is the response from the Jina embeddings API.
type embedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens  int `json:"total_tokens"`
		PromptTokens int `json:"prompt_tokens"`
	} `json:"usage"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body := embedRequest{
		Input: texts,
		Model: e.model,
	}

	resp, err := httpclient.DoJSON[embedResponse](ctx, e.client, http.MethodPost, "embeddings", body)
	if err != nil {
		return nil, fmt.Errorf("jina embedding: %w", err)
	}

	result := make([][]float32, len(texts))
	for _, d := range resp.Data {
		if d.Index < len(result) {
			result[d.Index] = d.Embedding
		}
	}

	return result, nil
}

// EmbedSingle produces an embedding for a single text.
func (e *Embedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("jina embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
