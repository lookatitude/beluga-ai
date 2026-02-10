package elasticsearch

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
	vectorstore.Register("elasticsearch", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by Elasticsearch.
type Store struct {
	client    HTTPClient
	baseURL   string
	apiKey    string
	index     string
	dimension int
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithIndex sets the Elasticsearch index name.
func WithIndex(name string) Option {
	return func(s *Store) { s.index = name }
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

// New creates a new Elasticsearch Store.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:    http.DefaultClient,
		baseURL:   baseURL,
		index:     "documents",
		dimension: 1536,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("elasticsearch: base_url is required")
	}
	var opts []Option
	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if idx, ok := config.GetOption[string](cfg, "index"); ok {
		opts = append(opts, WithIndex(idx))
	}
	if dim, ok := config.GetOption[float64](cfg, "dimension"); ok {
		opts = append(opts, WithDimension(int(dim)))
	}
	return New(cfg.BaseURL, opts...), nil
}

// EnsureIndex creates the Elasticsearch index with vector mapping if it does not exist.
func (s *Store) EnsureIndex(ctx context.Context) error {
	body := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"content": map[string]any{
					"type": "text",
				},
				"embedding": map[string]any{
					"type":       "dense_vector",
					"dims":       s.dimension,
					"index":      true,
					"similarity": "cosine",
				},
			},
		},
	}
	_, err := s.doJSON(ctx, http.MethodPut, fmt.Sprintf("/%s", s.index), body)
	return err
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("elasticsearch: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	// Use bulk API for efficiency.
	var buf bytes.Buffer
	for i, doc := range docs {
		// Action line.
		action := map[string]any{
			"index": map[string]any{
				"_index": s.index,
				"_id":    doc.ID,
			},
		}
		actionBytes, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("elasticsearch: marshal action: %w", err)
		}
		buf.Write(actionBytes)
		buf.WriteByte('\n')

		// Document body.
		body := map[string]any{
			"content":   doc.Content,
			"embedding": float32SliceToFloat64(embeddings[i]),
		}
		for k, v := range doc.Metadata {
			body[k] = v
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("elasticsearch: marshal doc: %w", err)
		}
		buf.Write(bodyBytes)
		buf.WriteByte('\n')
	}

	_, err := s.doJSONRaw(ctx, http.MethodPost, "/_bulk", buf.Bytes(), "application/x-ndjson")
	return err
}

// Search finds the k most similar documents to the query vector using kNN.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	knnQuery := map[string]any{
		"field":         "embedding",
		"query_vector":  float32SliceToFloat64(query),
		"k":             k,
		"num_candidates": k * 2,
	}

	if len(cfg.Filter) > 0 {
		musts := make([]map[string]any, 0, len(cfg.Filter))
		for key, val := range cfg.Filter {
			musts = append(musts, map[string]any{
				"term": map[string]any{key: val},
			})
		}
		knnQuery["filter"] = map[string]any{
			"bool": map[string]any{"must": musts},
		}
	}

	if cfg.Threshold > 0 {
		knnQuery["similarity"] = cfg.Threshold
	}

	body := map[string]any{
		"knn":     knnQuery,
		"size":    k,
		"_source": map[string]any{"excludes": []string{"embedding"}},
	}

	respBody, err := s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/%s/_search", s.index), body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string         `json:"_id"`
				Score  float64        `json:"_score"`
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("elasticsearch: unmarshal search response: %w", err)
	}

	docs := make([]schema.Document, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		doc := schema.Document{
			ID:       hit.ID,
			Score:    hit.Score,
			Metadata: make(map[string]any),
		}

		if content, ok := hit.Source["content"].(string); ok {
			doc.Content = content
		}

		for k, v := range hit.Source {
			if k != "content" && k != "embedding" {
				doc.Metadata[k] = v
			}
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

	// Use bulk delete API.
	var buf bytes.Buffer
	for _, id := range ids {
		action := map[string]any{
			"delete": map[string]any{
				"_index": s.index,
				"_id":    id,
			},
		}
		actionBytes, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("elasticsearch: marshal delete action: %w", err)
		}
		buf.Write(actionBytes)
		buf.WriteByte('\n')
	}

	_, err := s.doJSONRaw(ctx, http.MethodPost, "/_bulk", buf.Bytes(), "application/x-ndjson")
	return err
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch: marshal request: %w", err)
		}
	}
	return s.doJSONRaw(ctx, method, path, data, "application/json")
}

// doJSONRaw performs an HTTP request with raw bytes and a given content type.
func (s *Store) doJSONRaw(ctx context.Context, method, path string, data []byte, contentType string) ([]byte, error) {
	var bodyReader io.Reader
	if data != nil {
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	if s.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
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
