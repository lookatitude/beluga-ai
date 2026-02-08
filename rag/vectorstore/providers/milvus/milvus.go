// Package milvus provides a VectorStore backed by the Milvus vector database.
// It communicates with Milvus via its REST API for broad compatibility.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"
//
//	store, err := vectorstore.New("milvus", config.ProviderConfig{
//	    BaseURL: "http://localhost:19530",
//	    Options: map[string]any{
//	        "collection": "documents",
//	        "dimension":  float64(1536),
//	    },
//	})
package milvus

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
	vectorstore.Register("milvus", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by the Milvus vector database.
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

// New creates a new Milvus Store.
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
		return nil, fmt.Errorf("milvus: base_url is required")
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

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("milvus: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	data := make([]map[string]any, len(docs))
	for i, doc := range docs {
		entry := map[string]any{
			"id":        doc.ID,
			"content":   doc.Content,
			"embedding": float32SliceToFloat64(embeddings[i]),
		}
		for k, v := range doc.Metadata {
			entry[k] = v
		}
		data[i] = entry
	}

	body := map[string]any{
		"collectionName": s.collection,
		"data":           data,
	}

	_, err := s.doJSON(ctx, http.MethodPost, "/v2/vectordb/entities/insert", body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	body := map[string]any{
		"collectionName": s.collection,
		"data":           [][]float64{float32SliceToFloat64(query)},
		"limit":          k,
		"outputFields":   []string{"id", "content", "*"},
	}

	if len(cfg.Filter) > 0 {
		filter := ""
		for key, val := range cfg.Filter {
			if filter != "" {
				filter += " and "
			}
			filter += fmt.Sprintf(`%s == "%v"`, key, val)
		}
		body["filter"] = filter
	}

	respBody, err := s.doJSON(ctx, http.MethodPost, "/v2/vectordb/entities/search", body)
	if err != nil {
		return nil, err
	}

	// Parse manually for flexible field handling.
	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("milvus: unmarshal response: %w", err)
	}

	dataRaw, ok := raw["data"].([]any)
	if !ok {
		return nil, nil
	}

	docs := make([]schema.Document, 0, len(dataRaw))
	for _, itemRaw := range dataRaw {
		// Milvus returns search results as nested arrays.
		items, ok := itemRaw.([]any)
		if !ok {
			// If not nested, treat as single item.
			if obj, ok := itemRaw.(map[string]any); ok {
				items = []any{obj}
			} else {
				continue
			}
		}
		for _, subItem := range items {
			obj, ok := subItem.(map[string]any)
			if !ok {
				continue
			}

			doc := schema.Document{
				Metadata: make(map[string]any),
			}

			if id, ok := obj["id"].(string); ok {
				doc.ID = id
			}
			if content, ok := obj["content"].(string); ok {
				doc.Content = content
			}
			if dist, ok := obj["distance"].(float64); ok {
				doc.Score = 1.0 - dist // Convert distance to similarity.
			}

			for k, v := range obj {
				if k != "id" && k != "content" && k != "distance" && k != "embedding" {
					doc.Metadata[k] = v
				}
			}

			if cfg.Threshold > 0 && doc.Score < cfg.Threshold {
				continue
			}

			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	body := map[string]any{
		"collectionName": s.collection,
		"filter":         buildIDFilter(ids),
	}
	_, err := s.doJSON(ctx, http.MethodPost, "/v2/vectordb/entities/delete", body)
	return err
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("milvus: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("milvus: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("milvus: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("milvus: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("milvus: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// buildIDFilter creates a Milvus filter expression for the given IDs.
func buildIDFilter(ids []string) string {
	filter := `id in [`
	for i, id := range ids {
		if i > 0 {
			filter += ","
		}
		filter += fmt.Sprintf(`"%s"`, id)
	}
	filter += "]"
	return filter
}

// float32SliceToFloat64 converts []float32 to []float64 for JSON serialization.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}
