package google

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/httpclient"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
)

func init() {
	embedding.Register("google", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "gemini-embedding-001"
	defaultDimensions = 3072
	defaultBaseURL    = "https://generativelanguage.googleapis.com/v1beta"
)

// modelDimensions maps known Google embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"gemini-embedding-001":            3072,
	"text-embedding-004":              768,
	"embedding-001":                   768,
	"text-multilingual-embedding-002": 768,
}

// Embedder implements embedding.Embedder using the Google AI embeddings API.
type Embedder struct {
	client *httpclient.Client
	model  string
	dims   int
	// sendDims is true when the caller configured "dimensions"; the value is
	// then sent as outputDimensionality so Dimensions() matches the vectors.
	sendDims bool
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
	sendDims := false
	if d, ok := config.GetOption[float64](cfg, "dimensions"); ok && d > 0 {
		dims = int(d)
		sendDims = true
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	// The API key is sent in the x-goog-api-key header, never as a ?key=
	// query parameter: Go's *url.Error embeds the full request URL, so a key
	// in the URL would leak into returned errors and logs.
	opts := []httpclient.Option{
		httpclient.WithBaseURL(baseURL),
	}
	if cfg.APIKey != "" {
		opts = append(opts, httpclient.WithHeader("x-goog-api-key", cfg.APIKey))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, httpclient.WithTimeout(cfg.Timeout))
	}

	client := httpclient.New(opts...)

	return &Embedder{
		client:   client,
		model:    model,
		dims:     dims,
		sendDims: sendDims,
	}, nil
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
	Model                string       `json:"model"`
	Content              *contentPart `json:"content"`
	OutputDimensionality int          `json:"outputDimensionality,omitempty"`
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
		if e.sendDims {
			requests[i].OutputDimensionality = e.dims
		}
	}

	body := batchEmbedRequest{Requests: requests}
	// The model is config-derived; escape it so it cannot inject path
	// segments ("../") or a query string ("?").
	path := "models/" + url.PathEscape(e.model) + ":batchEmbedContents"

	resp, err := httpclient.DoJSON[batchEmbedResponse](ctx, e.client, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("google embedding: %w", err)
	}

	if len(resp.Embeddings) != len(texts) {
		return nil, core.Errorf(core.ErrProviderDown,
			"google embedding: got %d embeddings for %d inputs", len(resp.Embeddings), len(texts))
	}

	result := make([][]float32, len(texts))
	for i, emb := range resp.Embeddings {
		if emb.Values == nil {
			return nil, core.Errorf(core.ErrProviderDown, "google embedding: missing embedding at position %d", i)
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
