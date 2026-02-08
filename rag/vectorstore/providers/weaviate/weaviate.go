// Package weaviate provides a VectorStore backed by the Weaviate vector database.
// It communicates with Weaviate via its REST API using the internal httpclient.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"
//
//	store, err := vectorstore.New("weaviate", config.ProviderConfig{
//	    BaseURL: "http://localhost:8080",
//	    APIKey:  "optional-api-key",
//	    Options: map[string]any{
//	        "class": "Document",
//	    },
//	})
package weaviate

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
	vectorstore.Register("weaviate", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by the Weaviate vector database.
type Store struct {
	client  HTTPClient
	baseURL string
	apiKey  string
	class   string
}

// Option configures a Store.
type Option func(*Store)

// WithClass sets the Weaviate class name.
func WithClass(name string) Option {
	return func(s *Store) { s.class = name }
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(s *Store) { s.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new Weaviate Store.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:  http.DefaultClient,
		baseURL: baseURL,
		class:   "Document",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("weaviate: base_url is required")
	}
	var opts []Option
	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if cls, ok := config.GetOption[string](cfg, "class"); ok {
		opts = append(opts, WithClass(cls))
	}
	return New(cfg.BaseURL, opts...), nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("weaviate: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	objects := make([]map[string]any, len(docs))
	for i, doc := range docs {
		props := map[string]any{
			"content":  doc.Content,
			"_beluga_id": doc.ID,
		}
		for k, v := range doc.Metadata {
			props[k] = v
		}

		objects[i] = map[string]any{
			"class":      s.class,
			"id":         uuidFromID(doc.ID),
			"properties": props,
			"vector":     float32SliceToFloat64(embeddings[i]),
		}
	}

	body := map[string]any{
		"objects": objects,
	}
	_, err := s.doJSON(ctx, http.MethodPost, "/v1/batch/objects", body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build GraphQL nearVector query.
	whereClause := ""
	if len(cfg.Filter) > 0 {
		operands := make([]string, 0, len(cfg.Filter))
		for key, val := range cfg.Filter {
			operands = append(operands, fmt.Sprintf(
				`{path:["%s"],operator:Equal,valueText:"%v"}`, key, val))
		}
		if len(operands) == 1 {
			whereClause = fmt.Sprintf(`, where: %s`, operands[0])
		} else {
			whereClause = fmt.Sprintf(`, where: {operator:And,operands:[%s]}`,
				joinStrings(operands, ","))
		}
	}

	distanceClause := ""
	if cfg.Threshold > 0 {
		// Weaviate distance = 1 - similarity for cosine.
		distanceClause = fmt.Sprintf(`, distance: %f`, 1.0-cfg.Threshold)
	}

	graphQL := fmt.Sprintf(`{
		Get {
			%s(
				limit: %d,
				nearVector: {vector: %s%s}
				%s
			) {
				content
				_beluga_id
				_additional {
					id
					distance
				}
			}
		}
	}`, s.class, k, float32SliceToJSONArray(query), distanceClause, whereClause)

	body := map[string]any{
		"query": graphQL,
	}

	respBody, err := s.doJSON(ctx, http.MethodPost, "/v1/graphql", body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Get map[string][]struct {
				Content   string `json:"content"`
				BelugaID  string `json:"_beluga_id"`
				Additional struct {
					ID       string  `json:"id"`
					Distance float64 `json:"distance"`
				} `json:"_additional"`
			} `json:""`
		} `json:"data"`
	}

	// Parse response manually since the class name is dynamic.
	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("weaviate: unmarshal response: %w", err)
	}

	dataMap, ok := raw["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("weaviate: missing data in response")
	}
	getMap, ok := dataMap["Get"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("weaviate: missing Get in response")
	}
	classResults, ok := getMap[s.class].([]any)
	if !ok {
		return nil, nil
	}

	docs := make([]schema.Document, 0, len(classResults))
	for _, item := range classResults {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}

		doc := schema.Document{
			Metadata: make(map[string]any),
		}

		if id, ok := obj["_beluga_id"].(string); ok {
			doc.ID = id
		}
		if content, ok := obj["content"].(string); ok {
			doc.Content = content
		}

		// Extract additional fields.
		if additional, ok := obj["_additional"].(map[string]any); ok {
			if dist, ok := additional["distance"].(float64); ok {
				doc.Score = 1.0 - dist // Convert distance to similarity.
			}
			if doc.ID == "" {
				if id, ok := additional["id"].(string); ok {
					doc.ID = id
				}
			}
		}

		// Copy remaining properties to metadata.
		for k, v := range obj {
			if k != "content" && k != "_beluga_id" && k != "_additional" {
				doc.Metadata[k] = v
			}
		}

		if cfg.Threshold > 0 && doc.Score < cfg.Threshold {
			continue
		}

		docs = append(docs, doc)
	}

	_ = result // suppress unused warning
	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		_, err := s.doJSON(ctx, http.MethodDelete,
			fmt.Sprintf("/v1/objects/%s/%s", s.class, uuidFromID(id)), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("weaviate: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("weaviate: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weaviate: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("weaviate: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("weaviate: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// uuidFromID generates a deterministic UUID v5 from a document ID string.
// This provides a stable mapping from arbitrary IDs to Weaviate's UUID format.
func uuidFromID(id string) string {
	// Use a simple deterministic UUID format based on the ID.
	// Pad or hash to fill UUID format: 8-4-4-4-12 hex digits.
	h := fmt.Sprintf("%032x", []byte(id))
	if len(h) < 32 {
		h = h + "00000000000000000000000000000000"
		h = h[:32]
	}
	h = h[:32]
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

// float32SliceToFloat64 converts []float32 to []float64 for JSON serialization.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}

// float32SliceToJSONArray converts a float32 slice to a JSON array string.
func float32SliceToJSONArray(v []float32) string {
	data, _ := json.Marshal(float32SliceToFloat64(v))
	return string(data)
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
