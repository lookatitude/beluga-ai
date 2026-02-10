package pinecone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	vectorstore.Register("pinecone", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by Pinecone.
type Store struct {
	client    HTTPClient
	baseURL   string
	apiKey    string
	namespace string
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithNamespace sets the Pinecone namespace.
func WithNamespace(ns string) Option {
	return func(s *Store) { s.namespace = ns }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new Pinecone Store.
func New(baseURL, apiKey string, opts ...Option) *Store {
	s := &Store{
		client:  http.DefaultClient,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("pinecone: base_url (index host) is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("pinecone: api_key is required")
	}
	var opts []Option
	if ns, ok := config.GetOption[string](cfg, "namespace"); ok {
		opts = append(opts, WithNamespace(ns))
	}
	return New(cfg.BaseURL, cfg.APIKey, opts...), nil
}

// pineconeVector is the JSON representation of a vector for upsert.
type pineconeVector struct {
	ID       string         `json:"id"`
	Values   []float64      `json:"values"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("pinecone: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	vectors := make([]pineconeVector, len(docs))
	for i, doc := range docs {
		meta := make(map[string]any)
		meta["content"] = doc.Content
		for k, v := range doc.Metadata {
			meta[k] = v
		}
		vectors[i] = pineconeVector{
			ID:       doc.ID,
			Values:   float32SliceToFloat64(embeddings[i]),
			Metadata: meta,
		}
	}

	body := map[string]any{
		"vectors": vectors,
	}
	if s.namespace != "" {
		body["namespace"] = s.namespace
	}

	_, err := s.doJSON(ctx, http.MethodPost, "/vectors/upsert", body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	body := map[string]any{
		"vector":          float32SliceToFloat64(query),
		"topK":            k,
		"includeMetadata": true,
	}
	if s.namespace != "" {
		body["namespace"] = s.namespace
	}

	// Build filter from metadata.
	if len(cfg.Filter) > 0 {
		filter := make(map[string]any)
		for key, val := range cfg.Filter {
			filter[key] = map[string]any{"$eq": val}
		}
		body["filter"] = filter
	}

	resp, err := s.doJSON(ctx, http.MethodPost, "/query", body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Matches []struct {
			ID       string         `json:"id"`
			Score    float64        `json:"score"`
			Metadata map[string]any `json:"metadata"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("pinecone: unmarshal query response: %w", err)
	}

	// Sort by score descending (Pinecone should already do this, but be safe).
	sort.Slice(result.Matches, func(i, j int) bool {
		return result.Matches[i].Score > result.Matches[j].Score
	})

	docs := make([]schema.Document, 0, len(result.Matches))
	for _, m := range result.Matches {
		if cfg.Threshold > 0 && m.Score < cfg.Threshold {
			continue
		}

		doc := schema.Document{
			ID:    m.ID,
			Score: m.Score,
		}

		// Extract content from metadata.
		if content, ok := m.Metadata["content"].(string); ok {
			doc.Content = content
		}

		// Build metadata from remaining fields.
		meta := make(map[string]any)
		for k, v := range m.Metadata {
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
		"ids": ids,
	}
	if s.namespace != "" {
		body["namespace"] = s.namespace
	}

	_, err := s.doJSON(ctx, http.MethodPost, "/vectors/delete", body)
	return err
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("pinecone: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("pinecone: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pinecone: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pinecone: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("pinecone: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
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
