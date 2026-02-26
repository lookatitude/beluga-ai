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

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

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

	body := buildSearchBody(s.collection, query, k, cfg)

	respBody, err := s.doJSON(ctx, http.MethodPost, "/v2/vectordb/entities/search", body)
	if err != nil {
		return nil, err
	}

	return parseSearchResponse(respBody, cfg.Threshold)
}

// buildSearchBody constructs the JSON body for a Milvus search request.
func buildSearchBody(collection string, query []float32, k int, cfg *vectorstore.SearchConfig) map[string]any {
	body := map[string]any{
		"collectionName": collection,
		"data":           [][]float64{float32SliceToFloat64(query)},
		"limit":          k,
		"outputFields":   []string{"id", "content", "*"},
	}

	if len(cfg.Filter) > 0 {
		body["filter"] = buildMetadataFilter(cfg.Filter)
	}
	return body
}

// buildMetadataFilter constructs a Milvus filter expression from metadata key-value pairs.
func buildMetadataFilter(filter map[string]any) string {
	expr := ""
	for key, val := range filter {
		if expr != "" {
			expr += " and "
		}
		expr += fmt.Sprintf(`%s == "%v"`, key, val)
	}
	return expr
}

// parseSearchResponse parses a Milvus search response body into documents.
func parseSearchResponse(respBody []byte, threshold float64) ([]schema.Document, error) {
	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("milvus: unmarshal response: %w", err)
	}

	dataRaw, ok := raw["data"].([]any)
	if !ok {
		return nil, nil
	}

	var docs []schema.Document
	for _, itemRaw := range dataRaw {
		items := flattenResultItem(itemRaw)
		for _, obj := range items {
			doc := resultObjectToDocument(obj)
			if threshold > 0 && doc.Score < threshold {
				continue
			}
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

// flattenResultItem normalizes a Milvus result item (which may be nested) into a flat slice of objects.
func flattenResultItem(itemRaw any) []map[string]any {
	if items, ok := itemRaw.([]any); ok {
		var result []map[string]any
		for _, sub := range items {
			if obj, ok := sub.(map[string]any); ok {
				result = append(result, obj)
			}
		}
		return result
	}
	if obj, ok := itemRaw.(map[string]any); ok {
		return []map[string]any{obj}
	}
	return nil
}

// resultObjectToDocument converts a single Milvus result object to a schema.Document.
func resultObjectToDocument(obj map[string]any) schema.Document {
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
	return doc
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
