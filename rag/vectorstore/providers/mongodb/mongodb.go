// Package mongodb provides a VectorStore backed by MongoDB Atlas Vector Search.
// It communicates with MongoDB via its HTTP Data API to avoid requiring the
// full MongoDB Go driver as a dependency, and supports cosine, dot-product,
// and Euclidean distance strategies.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"
//
//	store, err := vectorstore.New("mongodb", config.ProviderConfig{
//	    BaseURL: "https://data.mongodb-api.com/app/<app-id>/endpoint/data/v1",
//	    APIKey:  "your-api-key",
//	    Options: map[string]any{
//	        "database":   "my_db",
//	        "collection": "documents",
//	        "index":      "vector_index",
//	    },
//	})
package mongodb

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
	vectorstore.Register("mongodb", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by MongoDB Atlas Vector Search.
type Store struct {
	client     HTTPClient
	baseURL    string
	apiKey     string
	database   string
	collection string
	index      string
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithDatabase sets the database name.
func WithDatabase(name string) Option {
	return func(s *Store) { s.database = name }
}

// WithCollection sets the collection name.
func WithCollection(name string) Option {
	return func(s *Store) { s.collection = name }
}

// WithIndex sets the vector search index name.
func WithIndex(name string) Option {
	return func(s *Store) { s.index = name }
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(s *Store) { s.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new MongoDB Atlas VectorStore.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:     http.DefaultClient,
		baseURL:    baseURL,
		database:   "beluga",
		collection: "documents",
		index:      "vector_index",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("mongodb: base_url is required")
	}
	var opts []Option
	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if db, ok := config.GetOption[string](cfg, "database"); ok {
		opts = append(opts, WithDatabase(db))
	}
	if col, ok := config.GetOption[string](cfg, "collection"); ok {
		opts = append(opts, WithCollection(col))
	}
	if idx, ok := config.GetOption[string](cfg, "index"); ok {
		opts = append(opts, WithIndex(idx))
	}
	return New(cfg.BaseURL, opts...), nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("mongodb: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	documents := make([]map[string]any, len(docs))
	for i, doc := range docs {
		d := map[string]any{
			"_id":       doc.ID,
			"content":   doc.Content,
			"embedding": float32SliceToFloat64(embeddings[i]),
		}
		if doc.Metadata != nil {
			d["metadata"] = doc.Metadata
		}
		documents[i] = d
	}

	body := map[string]any{
		"dataSource":  "Cluster0",
		"database":    s.database,
		"collection":  s.collection,
		"documents":   documents,
	}
	_, err := s.doJSON(ctx, http.MethodPost, "/action/insertMany", body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build the $vectorSearch aggregation stage.
	vectorSearch := map[string]any{
		"index":         s.index,
		"path":          "embedding",
		"queryVector":   float32SliceToFloat64(query),
		"numCandidates": k * 10,
		"limit":         k,
	}

	// Build filter for metadata.
	if len(cfg.Filter) > 0 {
		filter := make(map[string]any, len(cfg.Filter))
		for key, val := range cfg.Filter {
			filter["metadata."+key] = val
		}
		vectorSearch["filter"] = filter
	}

	pipeline := []map[string]any{
		{"$vectorSearch": vectorSearch},
		{"$addFields": map[string]any{
			"score": map[string]any{"$meta": "vectorSearchScore"},
		}},
	}

	body := map[string]any{
		"dataSource": "Cluster0",
		"database":   s.database,
		"collection": s.collection,
		"pipeline":   pipeline,
	}

	respBody, err := s.doJSON(ctx, http.MethodPost, "/action/aggregate", body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Documents []struct {
			ID       string         `json:"_id"`
			Content  string         `json:"content"`
			Metadata map[string]any `json:"metadata"`
			Score    float64        `json:"score"`
		} `json:"documents"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("mongodb: unmarshal search response: %w", err)
	}

	docs := make([]schema.Document, 0, len(result.Documents))
	for _, r := range result.Documents {
		if cfg.Threshold > 0 && r.Score < cfg.Threshold {
			continue
		}
		doc := schema.Document{
			ID:       r.ID,
			Content:  r.Content,
			Score:    r.Score,
			Metadata: r.Metadata,
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
		"dataSource": "Cluster0",
		"database":   s.database,
		"collection": s.collection,
		"filter": map[string]any{
			"_id": map[string]any{
				"$in": ids,
			},
		},
	}
	_, err := s.doJSON(ctx, http.MethodPost, "/action/deleteMany", body)
	return err
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("mongodb: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("mongodb: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mongodb: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mongodb: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mongodb: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
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
