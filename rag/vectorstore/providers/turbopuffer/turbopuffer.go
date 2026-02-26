package turbopuffer

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
	vectorstore.Register("turbopuffer", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

const defaultBaseURL = "https://api.turbopuffer.com/v1"

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by the Turbopuffer vector database.
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

// WithNamespace sets the Turbopuffer namespace.
func WithNamespace(ns string) Option {
	return func(s *Store) { s.namespace = ns }
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(s *Store) { s.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// WithBaseURL sets the base URL.
func WithBaseURL(url string) Option {
	return func(s *Store) { s.baseURL = url }
}

// New creates a new Turbopuffer Store.
func New(opts ...Option) *Store {
	s := &Store{
		client:    http.DefaultClient,
		baseURL:   defaultBaseURL,
		namespace: "documents",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	var opts []Option
	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, WithBaseURL(cfg.BaseURL))
	}
	if ns, ok := config.GetOption[string](cfg, "namespace"); ok {
		opts = append(opts, WithNamespace(ns))
	}
	return New(opts...), nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("turbopuffer: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	ids := make([]string, len(docs))
	vectors := make([][]float64, len(docs))
	attributes := make(map[string][]any)

	// Pre-initialize content attribute.
	contentVals := make([]any, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
		vectors[i] = float32SliceToFloat64(embeddings[i])
		contentVals[i] = doc.Content

		for k, v := range doc.Metadata {
			if attributes[k] == nil {
				attributes[k] = make([]any, len(docs))
			}
			attributes[k][i] = v
		}
	}
	attributes["content"] = contentVals

	body := map[string]any{
		"ids":        ids,
		"vectors":    vectors,
		"attributes": attributes,
	}

	_, err := s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/vectors/%s", s.namespace), body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	body := map[string]any{
		"vector":             float32SliceToFloat64(query),
		"top_k":              k,
		"include_attributes": []string{"content"},
		"include_vectors":    false,
	}

	if len(cfg.Filter) > 0 {
		filters := make([][]any, 0, len(cfg.Filter))
		for key, val := range cfg.Filter {
			filters = append(filters, []any{key, "Eq", val})
		}
		if len(filters) == 1 {
			body["filters"] = filters[0]
		} else {
			body["filters"] = append([]any{"And"}, filtersToAny(filters)...)
		}
	}

	if cfg.Threshold > 0 {
		body["distance_threshold"] = cfg.Threshold
	}

	distMetric := "cosine_distance"
	switch cfg.Strategy {
	case vectorstore.DotProduct:
		distMetric = "dot_product"
	case vectorstore.Euclidean:
		distMetric = "euclidean_squared"
	}
	body["distance_metric"] = distMetric

	respBody, err := s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/vectors/%s/query", s.namespace), body)
	if err != nil {
		return nil, err
	}

	var results []struct {
		ID         string         `json:"id"`
		Dist       float64        `json:"dist"`
		Attributes map[string]any `json:"attributes"`
	}
	if err := json.Unmarshal(respBody, &results); err != nil {
		return nil, fmt.Errorf("turbopuffer: unmarshal search response: %w", err)
	}

	docs := make([]schema.Document, 0, len(results))
	for _, r := range results {
		docs = append(docs, attributesToDocument(r.ID, 1.0-r.Dist, r.Attributes))
	}

	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Turbopuffer deletes by upserting with null vectors.
	vectors := make([]any, len(ids))
	for i := range ids {
		vectors[i] = nil
	}

	body := map[string]any{
		"ids":     ids,
		"vectors": vectors,
	}

	_, err := s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/vectors/%s", s.namespace), body)
	return err
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("turbopuffer: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("turbopuffer: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("turbopuffer: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("turbopuffer: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("turbopuffer: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// attributesToDocument converts a Turbopuffer result to a schema.Document.
func attributesToDocument(id string, score float64, attrs map[string]any) schema.Document {
	doc := schema.Document{
		ID:       id,
		Score:    score,
		Metadata: make(map[string]any),
	}
	if content, ok := attrs["content"].(string); ok {
		doc.Content = content
	}
	for k, v := range attrs {
		if k != "content" {
			doc.Metadata[k] = v
		}
	}
	return doc
}

// float32SliceToFloat64 converts []float32 to []float64 for JSON serialization.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}

// filtersToAny converts a slice of filter arrays to []any for JSON marshaling.
func filtersToAny(filters [][]any) []any {
	result := make([]any, len(filters))
	for i, f := range filters {
		result[i] = f
	}
	return result
}
