// Package qdrant provides a VectorStore backed by the Qdrant vector database.
// It communicates with Qdrant via its HTTP REST API to avoid heavy gRPC
// dependencies, and supports cosine, dot-product, and Euclidean distance.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"
//
//	store, err := vectorstore.New("qdrant", config.ProviderConfig{
//	    BaseURL: "http://localhost:6333",
//	    APIKey:  "optional-api-key",
//	    Options: map[string]any{
//	        "collection": "my_collection",
//	        "dimension":  float64(1536),
//	    },
//	})
package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	vectorstore.Register("qdrant", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by the Qdrant vector database.
type Store struct {
	client     HTTPClient
	baseURL    string
	apiKey     string
	collection string
	dimension  int
}

// Option configures a Store.
type Option func(*Store)

// WithCollection sets the collection name.
func WithCollection(name string) Option {
	return func(s *Store) { s.collection = name }
}

// WithDimension sets the vector dimension.
func WithDimension(dim int) Option {
	return func(s *Store) { s.dimension = dim }
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(s *Store) { s.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new Qdrant Store.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:     http.DefaultClient,
		baseURL:    baseURL,
		collection: "documents",
		dimension:  1536,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("qdrant: base_url is required")
	}
	var opts []Option
	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if col, ok := config.GetOption[string](cfg, "collection"); ok {
		opts = append(opts, WithCollection(col))
	}
	if dim, ok := config.GetOption[float64](cfg, "dimension"); ok {
		opts = append(opts, WithDimension(int(dim)))
	}
	return New(cfg.BaseURL, opts...), nil
}

// EnsureCollection creates the collection if it does not exist.
func (s *Store) EnsureCollection(ctx context.Context) error {
	distName := "Cosine"
	body := map[string]any{
		"vectors": map[string]any{
			"size":     s.dimension,
			"distance": distName,
		},
	}
	_, err := s.doJSON(ctx, http.MethodPut, fmt.Sprintf("/collections/%s", s.collection), body)
	return err
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("qdrant: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	points := make([]map[string]any, len(docs))
	for i, doc := range docs {
		payload := map[string]any{
			"content": doc.Content,
		}
		for k, v := range doc.Metadata {
			payload[k] = v
		}

		points[i] = map[string]any{
			"id":      doc.ID,
			"vector":  float32SliceToFloat64(embeddings[i]),
			"payload": payload,
		}
	}

	body := map[string]any{
		"points": points,
	}
	_, err := s.doJSON(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/points", s.collection), body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	body := map[string]any{
		"vector":       float32SliceToFloat64(query),
		"limit":        k,
		"with_payload": true,
	}

	// Build filter from metadata.
	if len(cfg.Filter) > 0 {
		must := make([]map[string]any, 0, len(cfg.Filter))
		for key, val := range cfg.Filter {
			must = append(must, map[string]any{
				"key":   key,
				"match": map[string]any{"value": val},
			})
		}
		body["filter"] = map[string]any{
			"must": must,
		}
	}

	// Set score threshold.
	if cfg.Threshold > 0 {
		body["score_threshold"] = cfg.Threshold
	}

	resp, err := s.doJSON(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/search", s.collection), body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("qdrant: unmarshal search response: %w", err)
	}

	docs := make([]schema.Document, 0, len(result.Result))
	for _, r := range result.Result {
		doc := schema.Document{
			ID:    fmt.Sprintf("%v", r.ID),
			Score: r.Score,
		}

		// Extract content from payload.
		if content, ok := r.Payload["content"].(string); ok {
			doc.Content = content
		}

		// Build metadata from remaining payload fields.
		meta := make(map[string]any)
		for k, v := range r.Payload {
			if k != "content" {
				meta[k] = v
			}
		}
		if len(meta) > 0 {
			doc.Metadata = meta
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	body := map[string]any{
		"points": ids,
	}
	_, err := s.doJSON(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/delete", s.collection), body)
	return err
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("qdrant: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("qdrant: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("qdrant: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// float32SliceToFloat64 converts []float32 to []float64 for JSON serialization.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}
