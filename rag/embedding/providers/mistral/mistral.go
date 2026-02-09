// Package mistral provides an Embedder backed by the Mistral AI embeddings API.
// It implements the embedding.Embedder interface using Mistral's embed endpoint
// via the internal httpclient.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"
//
//	emb, err := embedding.New("mistral", config.ProviderConfig{
//	    APIKey: "...",
//	    Model:  "mistral-embed",
//	})
package mistral

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("mistral", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "mistral-embed"
	defaultDimensions = 1024
	defaultBaseURL    = "https://api.mistral.ai/v1"
)

// modelDimensions maps known Mistral embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"mistral-embed": 1024,
}

// Embedder implements embedding.Embedder using the Mistral AI embeddings API.
type Embedder struct {
	client *httpclient.Client
	model  string
	dims   int
}

// Compile-time interface check.
var _ embedding.Embedder = (*Embedder)(nil)

// New creates a new Mistral Embedder from a ProviderConfig.
func New(cfg config.ProviderConfig) (*Embedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("mistral embedding: api_key is required")
	}

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

// embedRequest is the request body for the Mistral embed API.
type embedRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format"`
}

// embedResponse is the response from the Mistral embed API.
type embedResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body := embedRequest{
		Model:          e.model,
		Input:          texts,
		EncodingFormat: "float",
	}

	resp, err := httpclient.DoJSON[embedResponse](ctx, e.client, http.MethodPost, "embeddings", body)
	if err != nil {
		return nil, fmt.Errorf("mistral embedding: %w", err)
	}

	// Ensure results are in order.
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
		return nil, fmt.Errorf("mistral embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
