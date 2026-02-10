package vespa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	vectorstore.Register("vespa", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by the Vespa search engine.
type Store struct {
	client    HTTPClient
	baseURL   string
	namespace string
	docType   string
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithNamespace sets the Vespa namespace.
func WithNamespace(ns string) Option {
	return func(s *Store) { s.namespace = ns }
}

// WithDocType sets the Vespa document type.
func WithDocType(dt string) Option {
	return func(s *Store) { s.docType = dt }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new Vespa Store.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:    http.DefaultClient,
		baseURL:   strings.TrimRight(baseURL, "/"),
		namespace: "default",
		docType:   "document",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("vespa: base_url is required")
	}
	var opts []Option
	if ns, ok := config.GetOption[string](cfg, "namespace"); ok {
		opts = append(opts, WithNamespace(ns))
	}
	if dt, ok := config.GetOption[string](cfg, "doc_type"); ok {
		opts = append(opts, WithDocType(dt))
	}
	return New(cfg.BaseURL, opts...), nil
}

// docPath returns the Vespa document API path for a given document ID.
func (s *Store) docPath(id string) string {
	return fmt.Sprintf("/document/v1/%s/%s/docid/%s", s.namespace, s.docType, url.PathEscape(id))
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("vespa: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	for i, doc := range docs {
		fields := map[string]any{
			"content":   doc.Content,
			"embedding": float32SliceToFloat64(embeddings[i]),
		}
		if doc.Metadata != nil {
			for k, v := range doc.Metadata {
				fields[k] = v
			}
		}

		body := map[string]any{
			"fields": fields,
		}

		if err := s.doPut(ctx, s.docPath(doc.ID), body); err != nil {
			return fmt.Errorf("vespa: add document %q: %w", doc.ID, err)
		}
	}

	return nil
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build the ranking profile based on strategy.
	rankProfile := "closeness(embedding)"
	switch cfg.Strategy {
	case vectorstore.DotProduct:
		rankProfile = "dotProduct(embedding)"
	case vectorstore.Euclidean:
		rankProfile = "euclidean(embedding)"
	}

	// Build YQL query for nearest neighbor search.
	yql := fmt.Sprintf("select * from %s where {targetHits:%d}nearestNeighbor(embedding, q)", s.docType, k)

	// Add metadata filters.
	if len(cfg.Filter) > 0 {
		for key, val := range cfg.Filter {
			yql += fmt.Sprintf(" and %s = \"%v\"", key, val)
		}
	}

	params := url.Values{
		"yql":            {yql},
		"hits":           {fmt.Sprintf("%d", k)},
		"ranking":        {rankProfile},
		"input.query(q)": {formatVectorParam(query)},
	}

	searchURL := s.baseURL + "/search/?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("vespa: create search request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vespa: search: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vespa: read search response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("vespa: search returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Root struct {
			Children []struct {
				ID        string         `json:"id"`
				Relevance float64        `json:"relevance"`
				Fields    map[string]any `json:"fields"`
			} `json:"children"`
		} `json:"root"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("vespa: unmarshal search response: %w", err)
	}

	docs := make([]schema.Document, 0, len(result.Root.Children))
	for _, hit := range result.Root.Children {
		if cfg.Threshold > 0 && hit.Relevance < cfg.Threshold {
			continue
		}

		doc := schema.Document{
			ID:    hit.ID,
			Score: hit.Relevance,
		}

		// Extract content from fields.
		if content, ok := hit.Fields["content"].(string); ok {
			doc.Content = content
		}

		// Build metadata from remaining fields.
		meta := make(map[string]any)
		for k, v := range hit.Fields {
			if k != "content" && k != "embedding" {
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

	for _, id := range ids {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.baseURL+s.docPath(id), nil)
		if err != nil {
			return fmt.Errorf("vespa: create delete request: %w", err)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return fmt.Errorf("vespa: delete %q: %w", id, err)
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("vespa: delete %q returned %d", id, resp.StatusCode)
		}
	}

	return nil
}

// doPut sends a PUT request with a JSON body.
func (s *Store) doPut(ctx context.Context, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// float32SliceToFloat64 converts []float32 to []float64 for JSON serialization.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}

// formatVectorParam formats a float32 slice as a Vespa tensor parameter.
func formatVectorParam(v []float32) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, f := range v {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteString("]")
	return sb.String()
}
