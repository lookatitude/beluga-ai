package sentencetransformers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/httpclient"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
)

func init() {
	embedding.Register("sentence_transformers", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const (
	defaultModel      = "sentence-transformers/all-MiniLM-L6-v2"
	defaultDimensions = 384
	// defaultBaseURL is the HuggingFace Inference Providers router for the
	// hf-inference provider. The legacy api-inference.huggingface.co host is
	// deprecated.
	defaultBaseURL = "https://router.huggingface.co/hf-inference"
)

// modelDimensions maps known Sentence Transformers models to their dimensions.
var modelDimensions = map[string]int{
	"sentence-transformers/all-MiniLM-L6-v2":        384,
	"sentence-transformers/all-MiniLM-L12-v2":       384,
	"sentence-transformers/all-mpnet-base-v2":       768,
	"sentence-transformers/paraphrase-MiniLM-L6-v2": 384,
	"BAAI/bge-small-en-v1.5":                        384,
	"BAAI/bge-base-en-v1.5":                         768,
	"BAAI/bge-large-en-v1.5":                        1024,
}

// Embedder implements embedding.Embedder using the HuggingFace Inference API.
type Embedder struct {
	client *httpclient.Client
	model  string
	dims   int
	// path is the request path relative to the base URL, built once from
	// the escaped model ID.
	path string
}

// Compile-time interface check.
var _ embedding.Embedder = (*Embedder)(nil)

// New creates a new SentenceTransformers Embedder from a ProviderConfig.
func New(cfg config.ProviderConfig) (*Embedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("sentence_transformers embedding: api_key is required")
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

	path, err := featureExtractionPath(model)
	if err != nil {
		return nil, err
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
		path:   path,
	}, nil
}

// featureExtractionPath builds "models/<model>/pipeline/feature-extraction".
// HuggingFace model IDs are "<org>/<name>", so each "/"-separated segment is
// escaped individually; empty, "." and ".." segments are rejected so a
// config-supplied model cannot traverse out of the models/ prefix.
func featureExtractionPath(model string) (string, error) {
	segs := strings.Split(model, "/")
	for i, seg := range segs {
		if seg == "" || seg == "." || seg == ".." {
			return "", core.Errorf(core.ErrInvalidInput,
				"sentence_transformers embedding: invalid model %q", model)
		}
		segs[i] = url.PathEscape(seg)
	}
	return "models/" + strings.Join(segs, "/") + "/pipeline/feature-extraction", nil
}

// embedRequest is the request body for the HuggingFace inference API.
type embedRequest struct {
	Inputs []string `json:"inputs"`
}

// Embed produces embeddings for a batch of texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body := embedRequest{
		Inputs: texts,
	}

	// HuggingFace feature-extraction returns [][]float32 directly.
	resp, err := httpclient.DoJSON[[][]float32](ctx, e.client, http.MethodPost, e.path, body)
	if err != nil {
		return nil, fmt.Errorf("sentence_transformers embedding: %w", err)
	}

	if len(resp) != len(texts) {
		return nil, core.Errorf(core.ErrProviderDown,
			"sentence_transformers embedding: got %d embeddings for %d inputs", len(resp), len(texts))
	}
	for i, v := range resp {
		if v == nil {
			return nil, core.Errorf(core.ErrProviderDown, "sentence_transformers embedding: missing embedding at position %d", i)
		}
	}

	return resp, nil
}

// EmbedSingle produces an embedding for a single text.
func (e *Embedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("sentence_transformers embedding: empty response")
	}
	return vecs[0], nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}
