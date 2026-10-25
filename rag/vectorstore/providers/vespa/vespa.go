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
	"time"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore/internal/filterutil"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const (
	// maxResponseBytes bounds every response body read from Vespa.
	maxResponseBytes = 32 << 20
	// defaultTimeout is the timeout of the default HTTP client.
	defaultTimeout = 30 * time.Second
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
		client:    &http.Client{Timeout: defaultTimeout},
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
	return fmt.Sprintf("/document/v1/%s/%s/docid/%s",
		url.PathEscape(s.namespace), url.PathEscape(s.docType), url.PathEscape(id))
}

// validate rejects a namespace or document type that is not a plain
// identifier; both are interpolated into YQL and URL paths.
func (s *Store) validate(op string) error {
	if !filterutil.ValidKey(s.namespace) {
		return core.NewError(op, core.ErrInvalidInput, fmt.Sprintf("invalid namespace %q", s.namespace), nil)
	}
	if !filterutil.ValidKey(s.docType) {
		return core.NewError(op, core.ErrInvalidInput, fmt.Sprintf("invalid document type %q", s.docType), nil)
	}
	return nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("vespa: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}
	if err := s.validate("vespa.add"); err != nil {
		return err
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

	if err := s.validate("vespa.search"); err != nil {
		return nil, err
	}
	filterClause, err := buildFilterClause(cfg.Filter)
	if err != nil {
		return nil, err
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
	yql := fmt.Sprintf("select * from %s where {targetHits:%d}nearestNeighbor(embedding, q)%s", s.docType, k, filterClause)

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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
		docs = append(docs, hitToDocument(hit.ID, hit.Relevance, hit.Fields))
	}

	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.validate("vespa.delete"); err != nil {
		return err
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
		_ = resp.Body.Close()

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
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		return fmt.Errorf("returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// buildFilterClause renders metadata filters as YQL " and ..." predicates.
// Keys must be safe identifiers. String values use "contains" (YQL's string
// match) with an escaped string literal; bools and numbers use "=". A value
// can therefore never terminate its literal and rewrite the query.
func buildFilterClause(filter map[string]any) (string, error) {
	if len(filter) == 0 {
		return "", nil
	}
	keys, err := filterutil.Validate("vespa.search", filter)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, key := range keys {
		switch v := filter[key].(type) {
		case string:
			fmt.Fprintf(&b, " and %s contains %s", key, quoteYQLString(v))
		case bool:
			fmt.Fprintf(&b, " and %s = %t", key, v)
		default:
			fmt.Fprintf(&b, " and %s = %v", key, v)
		}
	}
	return b.String(), nil
}

// quoteYQLString returns s as a double-quoted YQL string literal with
// backslashes, quotes and control characters escaped.
func quoteYQLString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '\\' || r == '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// hitToDocument converts a search hit to a schema.Document, extracting
// content and building metadata from the remaining fields.
func hitToDocument(id string, score float64, fields map[string]any) schema.Document {
	doc := schema.Document{ID: id, Score: score}
	if content, ok := fields["content"].(string); ok {
		doc.Content = content
	}
	meta := make(map[string]any)
	for k, v := range fields {
		if k != "content" && k != "embedding" {
			meta[k] = v
		}
	}
	if len(meta) > 0 {
		doc.Metadata = meta
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

// formatVectorParam formats a float32 slice as a Vespa tensor parameter.
func formatVectorParam(v []float32) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, f := range v {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "%g", f)
	}
	sb.WriteString("]")
	return sb.String()
}
