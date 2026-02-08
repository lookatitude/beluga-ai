// Package voyage provides a Voyage AI embeddings provider for the Beluga AI framework.
// It implements the embedding.Embedder interface using the Voyage Embed API
// via the internal httpclient.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"
//
//	emb, err := embedding.New("voyage", config.ProviderConfig{
//	    APIKey: "...",
//	})
package voyage

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("voyage", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "voyage-2"
	defaultDimensions = 1024
	defaultBaseURL    = "https://api.voyageai.com/v1"
)

// modelDimensions maps known Voyage AI embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"voyage-2":         1024,
	"voyage-large-2":   1536,
	"voyage-code-2":    1536,
	"voyage-lite-02-instruct": 1024,
	"voyage-3":         1024,
	"voyage-3-lite":    512,
}

// Embedder implements embedding.Embedder using the Voyage AI Embed API.
type Embedder struct {
	client    *httpclient.Client
	model     string
	dims      int
	inputType string
}

// New creates a new Voyage AI Embedder from a ProviderConfig.
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

	inputType := "document"
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

// embedRequest is the request body for the Voyage embed API.
type embedRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type,omitempty"`
}

// embedResponse is the response from the Voyage embed API.
type embedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body := embedRequest{
		Input:     texts,
		Model:     e.model,
		InputType: e.inputType,
	}

	resp, err := httpclient.DoJSON[embedResponse](ctx, e.client, http.MethodPost, "embeddings", body)
	if err != nil {
		return nil, fmt.Errorf("voyage embedding: %w", err)
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
		return nil, fmt.Errorf("voyage embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
