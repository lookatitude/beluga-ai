// Package ollama provides an Ollama embeddings provider for the Beluga AI framework.
// It implements the embedding.Embedder interface using the Ollama REST API
// via the internal httpclient.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
//
//	emb, err := embedding.New("ollama", config.ProviderConfig{
//	    BaseURL: "http://localhost:11434",
//	})
package ollama

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("ollama", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "nomic-embed-text"
	defaultDimensions = 768
	defaultBaseURL    = "http://localhost:11434"
)

// modelDimensions maps known Ollama embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"nomic-embed-text":    768,
	"mxbai-embed-large":   1024,
	"all-minilm":          384,
	"snowflake-arctic-embed": 1024,
}

// Embedder implements embedding.Embedder using the Ollama embedding API.
type Embedder struct {
	client *httpclient.Client
	model  string
	dims   int
}

// New creates a new Ollama Embedder from a ProviderConfig.
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

// embedRequest is the request body for the Ollama embed API.
type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// embedResponse is the response from the Ollama embed API.
type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	result := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.embedOne(ctx, text)
		if err != nil {
			return nil, err
		}
		result[i] = vec
	}

	return result, nil
}

func (e *Embedder) embedOne(ctx context.Context, text string) ([]float32, error) {
	body := embedRequest{
		Model: e.model,
		Input: text,
	}

	resp, err := httpclient.DoJSON[embedResponse](ctx, e.client, http.MethodPost, "api/embed", body)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embedding: empty response")
	}

	return resp.Embeddings[0], nil
}

// EmbedSingle produces an embedding for a single text.
func (e *Embedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	return e.embedOne(ctx, text)
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
