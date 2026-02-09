// Package google provides a Google AI embeddings provider for the Beluga AI framework.
// It implements the embedding.Embedder interface using the internal httpclient
// to call the Google AI Gemini embedding API.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"
//
//	emb, err := embedding.New("google", config.ProviderConfig{
//	    APIKey: "...",
//	})
package google

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("google", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "text-embedding-004"
	defaultDimensions = 768
	defaultBaseURL    = "https://generativelanguage.googleapis.com/v1beta"
)

// modelDimensions maps known Google embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"text-embedding-004":    768,
	"embedding-001":         768,
	"text-multilingual-embedding-002": 768,
}

// Embedder implements embedding.Embedder using the Google AI embeddings API.
type Embedder struct {
	client *httpclient.Client
	model  string
	dims   int
	apiKey string
}

// Compile-time interface check.
var _ embedding.Embedder = (*Embedder)(nil)

// New creates a new Google Embedder from a ProviderConfig.
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
		apiKey: cfg.APIKey,
	}, nil
}

// embedRequest is the request body for the Google embedContent API.
type embedRequest struct {
	Content *contentPart `json:"content"`
}

type contentPart struct {
	Parts []partText `json:"parts"`
}

type partText struct {
	Text string `json:"text"`
}

// batchEmbedRequest is the request body for batchEmbedContents.
type batchEmbedRequest struct {
	Requests []singleEmbedRequest `json:"requests"`
}

type singleEmbedRequest struct {
	Model   string       `json:"model"`
	Content *contentPart `json:"content"`
}

// embedResponse is the response from the Google embedContent API.
type embedResponse struct {
	Embedding *embeddingValue `json:"embedding"`
}

type embeddingValue struct {
	Values []float32 `json:"values"`
}

// batchEmbedResponse is the response from batchEmbedContents.
type batchEmbedResponse struct {
	Embeddings []embeddingValue `json:"embeddings"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	requests := make([]singleEmbedRequest, len(texts))
	for i, text := range texts {
		requests[i] = singleEmbedRequest{
			Model: "models/" + e.model,
			Content: &contentPart{
				Parts: []partText{{Text: text}},
			},
		}
	}

	body := batchEmbedRequest{Requests: requests}
	path := fmt.Sprintf("models/%s:batchEmbedContents?key=%s", e.model, e.apiKey)

	resp, err := httpclient.DoJSON[batchEmbedResponse](ctx, e.client, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("google embedding: %w", err)
	}

	result := make([][]float32, len(texts))
	for i, emb := range resp.Embeddings {
		if i >= len(texts) {
			break
		}
		result[i] = emb.Values
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
		return nil, fmt.Errorf("google embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
