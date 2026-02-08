// Package chroma provides a VectorStore backed by ChromaDB. It communicates
// with ChromaDB via its HTTP REST API.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"
//
//	store, err := vectorstore.New("chroma", config.ProviderConfig{
//	    BaseURL: "http://localhost:8000",
//	    Options: map[string]any{
//	        "collection": "my_collection",
//	        "tenant":     "default_tenant",
//	        "database":   "default_database",
//	    },
//	})
package chroma

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
	vectorstore.Register("chroma", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is a VectorStore backed by ChromaDB.
type Store struct {
	client       HTTPClient
	baseURL      string
	collection   string
	collectionID string // Resolved after EnsureCollection
	tenant       string
	database     string
}

// Option configures a Store.
type Option func(*Store)

// WithCollection sets the collection name.
func WithCollection(name string) Option {
	return func(s *Store) { s.collection = name }
}

// WithCollectionID sets the collection ID directly (skips resolution).
func WithCollectionID(id string) Option {
	return func(s *Store) { s.collectionID = id }
}

// WithTenant sets the tenant name.
func WithTenant(t string) Option {
	return func(s *Store) { s.tenant = t }
}

// WithDatabase sets the database name.
func WithDatabase(db string) Option {
	return func(s *Store) { s.database = db }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new ChromaDB Store.
func New(baseURL string, opts ...Option) *Store {
	s := &Store{
		client:   http.DefaultClient,
		baseURL:  baseURL,
		tenant:   "default_tenant",
		database: "default_database",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("chroma: base_url is required")
	}
	var opts []Option
	if col, ok := config.GetOption[string](cfg, "collection"); ok {
		opts = append(opts, WithCollection(col))
	}
	if tenant, ok := config.GetOption[string](cfg, "tenant"); ok {
		opts = append(opts, WithTenant(tenant))
	}
	if db, ok := config.GetOption[string](cfg, "database"); ok {
		opts = append(opts, WithDatabase(db))
	}
	return New(cfg.BaseURL, opts...), nil
}

// EnsureCollection creates the collection if it does not exist and resolves
// the collection ID.
func (s *Store) EnsureCollection(ctx context.Context) error {
	body := map[string]any{
		"name":            s.collection,
		"get_or_create":   true,
	}

	resp, err := s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/tenants/%s/databases/%s/collections", s.tenant, s.database), body)
	if err != nil {
		return fmt.Errorf("chroma: ensure collection: %w", err)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("chroma: unmarshal collection response: %w", err)
	}
	s.collectionID = result.ID
	return nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("chroma: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	colID, err := s.resolveCollectionID(ctx)
	if err != nil {
		return err
	}

	ids := make([]string, len(docs))
	documents := make([]string, len(docs))
	metadatas := make([]map[string]any, len(docs))
	embeds := make([][]float64, len(docs))

	for i, doc := range docs {
		ids[i] = doc.ID
		documents[i] = doc.Content
		meta := make(map[string]any)
		for k, v := range doc.Metadata {
			meta[k] = v
		}
		metadatas[i] = meta
		embeds[i] = float32SliceToFloat64(embeddings[i])
	}

	body := map[string]any{
		"ids":        ids,
		"documents":  documents,
		"metadatas":  metadatas,
		"embeddings": embeds,
	}

	_, err = s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/collections/%s/upsert", colID), body)
	return err
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	colID, err := s.resolveCollectionID(ctx)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"query_embeddings": [][]float64{float32SliceToFloat64(query)},
		"n_results":        k,
		"include":          []string{"documents", "metadatas", "distances"},
	}

	// Build where clause for metadata filters.
	if len(cfg.Filter) > 0 {
		where := make(map[string]any)
		for key, val := range cfg.Filter {
			where[key] = map[string]any{"$eq": val}
		}
		body["where"] = where
	}

	resp, err := s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/collections/%s/query", colID), body)
	if err != nil {
		return nil, err
	}

	var result struct {
		IDs       [][]string           `json:"ids"`
		Documents [][]string           `json:"documents"`
		Metadatas [][]map[string]any   `json:"metadatas"`
		Distances [][]float64          `json:"distances"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("chroma: unmarshal query response: %w", err)
	}

	if len(result.IDs) == 0 || len(result.IDs[0]) == 0 {
		return nil, nil
	}

	ids := result.IDs[0]
	docTexts := result.Documents[0]
	metas := result.Metadatas[0]
	dists := result.Distances[0]

	type scored struct {
		doc   schema.Document
		score float64
	}

	var candidates []scored
	for i := range ids {
		// ChromaDB returns distances (lower = more similar).
		// Convert to similarity score: 1 / (1 + distance).
		score := 1.0 / (1.0 + dists[i])

		if cfg.Threshold > 0 && score < cfg.Threshold {
			continue
		}

		doc := schema.Document{
			ID:      ids[i],
			Content: docTexts[i],
			Score:   score,
		}
		if len(metas[i]) > 0 {
			doc.Metadata = metas[i]
		}
		candidates = append(candidates, scored{doc: doc, score: score})
	}

	// Sort by score descending.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	docs := make([]schema.Document, len(candidates))
	for i, c := range candidates {
		docs[i] = c.doc
	}
	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	colID, err := s.resolveCollectionID(ctx)
	if err != nil {
		return err
	}

	body := map[string]any{
		"ids": ids,
	}

	_, err = s.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/collections/%s/delete", colID), body)
	return err
}

// resolveCollectionID returns the collection ID, ensuring the collection
// exists if needed.
func (s *Store) resolveCollectionID(ctx context.Context) (string, error) {
	if s.collectionID != "" {
		return s.collectionID, nil
	}

	if s.collection == "" {
		return "", fmt.Errorf("chroma: collection name or ID is required")
	}

	if err := s.EnsureCollection(ctx); err != nil {
		return "", err
	}
	return s.collectionID, nil
}

// doJSON performs an HTTP request with a JSON body and returns the response body.
func (s *Store) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("chroma: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("chroma: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chroma: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chroma: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("chroma: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
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
