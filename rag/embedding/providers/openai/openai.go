// Package openai provides an OpenAI embeddings provider for the Beluga AI framework.
// It implements the embedding.Embedder interface using the openai-go SDK.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
//
//	emb, err := embedding.New("openai", config.ProviderConfig{
//	    APIKey: "sk-...",
//	})
package openai

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func init() {
	embedding.Register("openai", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "text-embedding-3-small"
	defaultDimensions = 1536
	defaultBaseURL    = "https://api.openai.com/v1"
)

// modelDimensions maps known OpenAI embedding models to their default dimensions.
var modelDimensions = map[string]int{
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,
}

// Embedder implements embedding.Embedder using the OpenAI embeddings API.
type Embedder struct {
	client openai.Client
	model  string
	dims   int
}

// New creates a new OpenAI Embedder from a ProviderConfig.
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

	opts := []option.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	if cfg.Timeout > 0 {
		opts = append(opts, option.WithRequestTimeout(cfg.Timeout))
	}

	client := openai.NewClient(opts...)

	return &Embedder{
		client: client,
		model:  model,
		dims:   dims,
	}, nil
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	inputs := make([]openai.EmbeddingNewParamsInputUnion, len(texts))
	for i, t := range texts {
		inputs[i] = openai.EmbeddingNewParamsInputUnion{OfString: openai.String(t)}
	}

	resp, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model:          openai.EmbeddingModel(e.model),
		Input:          openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: texts},
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embedding: %w", err)
	}

	result := make([][]float32, len(texts))
	for _, emb := range resp.Data {
		if emb.Index >= int64(len(texts)) {
			continue
		}
		vec := make([]float32, len(emb.Embedding))
		for j, v := range emb.Embedding {
			vec[j] = float32(v)
		}
		result[emb.Index] = vec
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
		return nil, fmt.Errorf("openai embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
