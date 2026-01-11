// Package qdrant provides a Qdrant vector store implementation.
// Qdrant is an open-source vector database that provides efficient
// similarity search and vector storage capabilities.
//
// Features:
// - High-performance vector search
// - Metadata filtering
// - Payload storage
// - Collection management
// - REST and gRPC APIs
//
// Requirements:
// - Qdrant server URL
// - Collection name
// - Optional API key for Qdrant Cloud
//
// Example:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	store, err := vectorstores.NewVectorStore(ctx, "qdrant",
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithProviderConfig("url", "http://localhost:6333"),
//		vectorstores.WithProviderConfig("collection_name", "my-collection"),
//		vectorstores.WithProviderConfig("api_key", "your-api-key"), // Optional
//	)
package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// QdrantStore implements the VectorStore interface using Qdrant API.
type QdrantStore struct {
	url            string
	apiKey         string
	collectionName string
	embeddingDim   int
	name           string
	httpClient     *http.Client
	mu             sync.RWMutex
}

// QdrantConfig holds configuration specific to QdrantStore.
type QdrantConfig struct {
	URL            string
	APIKey         string
	CollectionName string
	EmbeddingDim   int
}

// Qdrant API request/response structures.
type qdrantUpsertRequest struct {
	Points []qdrantPoint `json:"points"`
}

type qdrantPoint struct {
	ID      interface{}          `json:"id"`      // Can be string or int
	Vector  []float32             `json:"vector"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type qdrantSearchRequest struct {
	Vector      []float32           `json:"vector"`
	Limit       int                 `json:"limit"`
	WithPayload bool                `json:"with_payload"`
	WithVector  bool                `json:"with_vector"`
	Filter      *qdrantFilter       `json:"filter,omitempty"`
	ScoreThreshold float32          `json:"score_threshold,omitempty"`
}

type qdrantFilter struct {
	Must   []qdrantCondition `json:"must,omitempty"`
	Should []qdrantCondition `json:"should,omitempty"`
	MustNot []qdrantCondition `json:"must_not,omitempty"`
}

type qdrantCondition struct {
	Key   string      `json:"key"`
	Match interface{} `json:"match"` // Can be string, int, or map
}

type qdrantSearchResponse struct {
	Result []qdrantScoredPoint `json:"result"`
}

type qdrantScoredPoint struct {
	ID      interface{}          `json:"id"`
	Score   float32              `json:"score"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Vector  []float32            `json:"vector,omitempty"`
}

type qdrantDeleteRequest struct {
	Points []interface{} `json:"points"` // Can be string IDs or filter
}

type qdrantCollectionInfo struct {
	Config qdrantCollectionConfig `json:"config"`
}

type qdrantCollectionConfig struct {
	Params qdrantCollectionParams `json:"params"`
}

type qdrantCollectionParams struct {
	Vectors qdrantVectorsConfig `json:"vectors"`
}

type qdrantVectorsConfig struct {
	Size int `json:"size"`
}

// NewQdrantStoreFromConfig creates a new QdrantStore from configuration.
func NewQdrantStoreFromConfig(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
	// Extract qdrant-specific configuration
	url, _ := config.ProviderConfig["url"].(string)
	if url == "" {
		url = "http://localhost:6333" // Default Qdrant URL
	}

	apiKey, _ := config.ProviderConfig["api_key"].(string)

	collectionName, _ := config.ProviderConfig["collection_name"].(string)
	if collectionName == "" {
		return nil, vectorstores.NewVectorStoreErrorWithMessage("NewQdrantStoreFromConfig", vectorstores.ErrCodeInvalidConfig,
			"collection_name is required in qdrant provider config", nil)
	}

	embeddingDim, _ := config.ProviderConfig["embedding_dimension"].(int)
	if embeddingDim == 0 {
		embeddingDim = 1536 // Default OpenAI embedding dimension
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	store := &QdrantStore{
		url:            url,
		apiKey:         apiKey,
		collectionName: collectionName,
		embeddingDim:   embeddingDim,
		name:           "qdrant",
		httpClient:     httpClient,
	}

	// Verify collection exists or create it
	if err := store.ensureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return store, nil
}

// ensureCollection checks if the collection exists and creates it if needed.
func (s *QdrantStore) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/collections/%s", s.url, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Collection exists, verify dimension
		var info qdrantCollectionInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
			if size := info.Config.Params.Vectors.Size; size > 0 && size != s.embeddingDim {
				return fmt.Errorf("collection dimension mismatch: expected %d, got %d", s.embeddingDim, size)
			}
		}
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Collection doesn't exist, create it
		return s.createCollection(ctx)
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("unexpected status checking collection: %d - %s", resp.StatusCode, string(body))
}

// createCollection creates a new Qdrant collection.
func (s *QdrantStore) createCollection(ctx context.Context) error {
	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     s.embeddingDim,
			"distance": "Cosine",
		},
	}

	bodyBytes, err := json.Marshal(createReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s", s.url, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: status %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// AddDocuments stores documents with their embeddings in Qdrant.
func (s *QdrantStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply options
	config := vectorstores.NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Generate embeddings if embedder is provided
	var embeddings [][]float32
	var err error
	if config.Embedder != nil {
		texts := make([]string, len(documents))
		for i, doc := range documents {
			texts[i] = doc.GetContent()
		}
		embeddings, err = config.Embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			return nil, vectorstores.NewVectorStoreErrorWithMessage("AddDocuments", vectorstores.ErrCodeEmbeddingFailed,
				"failed to generate embeddings for documents", err)
		}
	} else {
		// Check if documents already have embeddings
		embeddings = make([][]float32, len(documents))
		for i, doc := range documents {
			if len(doc.Embedding) > 0 {
				embeddings[i] = doc.Embedding
			} else {
				return nil, vectorstores.NewVectorStoreErrorWithMessage("AddDocuments", vectorstores.ErrCodeEmbeddingFailed,
					fmt.Sprintf("no embedder provided and document %d has no embedding", i), nil)
			}
		}
	}

	// Prepare points for Qdrant upsert
	points := make([]qdrantPoint, len(documents))
	ids := make([]string, len(documents))

	for i, doc := range documents {
		// Generate ID if not present
		id := doc.ID
		if id == "" {
			id = fmt.Sprintf("doc-%d-%d", time.Now().UnixNano(), i)
		}
		ids[i] = id

		// Prepare payload (metadata + content)
		payload := make(map[string]interface{})
		for k, v := range doc.Metadata {
			payload[k] = v
		}
		payload["content"] = doc.GetContent()

		points[i] = qdrantPoint{
			ID:      id,
			Vector:  embeddings[i],
			Payload: payload,
		}
	}

	// Upsert to Qdrant
	upsertReq := qdrantUpsertRequest{
		Points: points,
	}

	bodyBytes, err := json.Marshal(upsertReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal upsert request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points", s.url, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to upsert points: status %d - %s", resp.StatusCode, string(body))
	}

	return ids, nil
}

// DeleteDocuments removes documents from Qdrant.
func (s *QdrantStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	// Convert string IDs to interface slice
	pointIDs := make([]interface{}, len(ids))
	for i, id := range ids {
		pointIDs[i] = id
	}

	deleteReq := qdrantDeleteRequest{
		Points: pointIDs,
	}

	bodyBytes, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", s.url, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete points: status %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// SimilaritySearch finds the k most similar documents to a query vector.
func (s *QdrantStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Apply options
	config := vectorstores.NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Use k from options if provided, otherwise use parameter
	searchK := k
	if config.SearchK > 0 {
		searchK = config.SearchK
	}

	// Build filter from metadata filters if provided
	var filter *qdrantFilter
	if len(config.MetadataFilters) > 0 {
		filter = &qdrantFilter{
			Must: make([]qdrantCondition, 0, len(config.MetadataFilters)),
		}
		for k, v := range config.MetadataFilters {
			filter.Must = append(filter.Must, qdrantCondition{
				Key:   k,
				Match: v,
			})
		}
	}

	searchReq := qdrantSearchRequest{
		Vector:      queryVector,
		Limit:       searchK,
		WithPayload: true,
		WithVector:  false,
		Filter:      filter,
	}

	if config.ScoreThreshold > 0 {
		searchReq.ScoreThreshold = config.ScoreThreshold
	}

	bodyBytes, err := json.Marshal(searchReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", s.url, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("failed to search points: status %d - %s", resp.StatusCode, string(body))
	}

	var searchResp qdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Convert results to schema.Document
	documents := make([]schema.Document, len(searchResp.Result))
	scores := make([]float32, len(searchResp.Result))

	for i, point := range searchResp.Result {
		scores[i] = point.Score

		// Extract content and metadata from payload
		content := ""
		metadata := make(map[string]string)

		if payload := point.Payload; payload != nil {
			if contentVal, ok := payload["content"].(string); ok {
				content = contentVal
			}

			for k, v := range payload {
				if k != "content" {
					if strVal, ok := v.(string); ok {
						metadata[k] = strVal
					}
				}
			}
		}

		// Convert ID to string
		var id string
		switch v := point.ID.(type) {
		case string:
			id = v
		case float64:
			id = fmt.Sprintf("%.0f", v)
		default:
			id = fmt.Sprintf("%v", v)
		}

		documents[i] = schema.Document{
			ID:          id,
			PageContent: content,
			Metadata:    metadata,
			Score:       point.Score,
		}
	}

	return documents, scores, nil
}

// SimilaritySearchByQuery performs similarity search using a text query.
func (s *QdrantStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	// Generate embedding for query
	if embedder == nil {
		// Try to get embedder from options
		config := vectorstores.NewDefaultConfig()
		for _, opt := range opts {
			opt(config)
		}
		embedder = config.Embedder
	}

	if embedder == nil {
		return nil, nil, vectorstores.NewVectorStoreErrorWithMessage("SimilaritySearchByQuery", vectorstores.ErrCodeEmbeddingFailed,
			"embedder is required for SimilaritySearchByQuery", nil)
	}

	queryVector, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, vectorstores.NewVectorStoreErrorWithMessage("SimilaritySearchByQuery", vectorstores.ErrCodeEmbeddingFailed,
			"failed to generate embedding for query", err)
	}

	return s.SimilaritySearch(ctx, queryVector, k, opts...)
}

// AsRetriever returns a Retriever implementation based on this VectorStore.
func (s *QdrantStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	return &QdrantRetriever{
		store: s,
		opts:  opts,
	}
}

// GetName returns the name of the vector store.
func (s *QdrantStore) GetName() string {
	return s.name
}

// QdrantRetriever implements the Retriever interface for QdrantStore.
type QdrantRetriever struct {
	store *QdrantStore
	opts  []vectorstores.Option
}

// GetRelevantDocuments retrieves relevant documents for a query.
func (r *QdrantRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Apply default search options
	opts := append(r.opts, vectorstores.WithSearchK(5))

	// Get embedder from options
	config := vectorstores.NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	docs, _, err := r.store.SimilaritySearchByQuery(ctx, query, 5, config.Embedder, opts...)
	return docs, err
}

// Ensure QdrantStore implements the VectorStore interface.
var _ vectorstores.VectorStore = (*QdrantStore)(nil)

// Ensure QdrantRetriever implements the Retriever interface.
var _ vectorstores.Retriever = (*QdrantRetriever)(nil)
