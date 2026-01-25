// Package weaviate provides a Weaviate vector store implementation.
// Weaviate is an open-source vector database that provides semantic search,
// classification, and knowledge graph capabilities.
//
// Features:
// - High-performance vector search
// - GraphQL and REST APIs
// - Automatic schema management
// - Multi-tenancy support
// - Hybrid search (vector + keyword)
//
// Requirements:
// - Weaviate server URL
// - Class name (collection)
// - Optional API key for Weaviate Cloud
//
// Example:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	store, err := vectorstores.NewVectorStore(ctx, "weaviate",
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithProviderConfig("url", "http://localhost:8080"),
//		vectorstores.WithProviderConfig("class_name", "Document"),
//		vectorstores.WithProviderConfig("api_key", "your-api-key"), // Optional
//	)
package weaviate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// WeaviateStore implements the VectorStore interface using Weaviate API.
type WeaviateStore struct {
	httpClient   *http.Client
	url          string
	apiKey       string
	className    string
	name         string
	embeddingDim int
	mu           sync.RWMutex
}

// Weaviate API request/response structures.
type weaviateObject struct {
	ID         string         `json:"id,omitempty"`
	Class      string         `json:"class,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Vector     []float32      `json:"vector,omitempty"`
}

type weaviateBatchRequest struct {
	Objects []weaviateObject `json:"objects"`
}

type weaviateGraphQLQuery struct {
	Query string `json:"query"`
}

type weaviateGraphQLResponse struct {
	Data   map[string]any  `json:"data"`
	Errors []weaviateError `json:"errors,omitempty"`
}

type weaviateError struct {
	Message string `json:"message"`
}

type weaviateGetResponse struct {
	Objects []weaviateObject `json:"objects"`
}

type weaviateSearchRequest struct {
	NearVector *weaviateNearVector `json:"nearVector,omitempty"`
	Where      map[string]any      `json:"where,omitempty"`
	Query      string              `json:"query,omitempty"`
	Fields     string              `json:"fields,omitempty"`
	Limit      int                 `json:"limit"`
}

type weaviateNearVector struct {
	Vector    []float32 `json:"vector"`
	Certainty float32   `json:"certainty,omitempty"`
	Distance  float32   `json:"distance,omitempty"`
}

type weaviateSearchResponse struct {
	Data weaviateSearchData `json:"data"`
}

type weaviateSearchData struct {
	Get map[string][]weaviateObject `json:"Get"`
}

// NewWeaviateStoreFromConfig creates a new WeaviateStore from configuration.
// NewWeaviateStoreFromConfig creates a new WeaviateStore from configuration.
// This is used by the factory pattern for creating stores via the registry.
// Weaviate is an open-source vector database with semantic search and knowledge graph capabilities.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - config: Configuration containing embedder, URL, class name, API key (optional), and embedding dimension
//
// Returns:
//   - VectorStore: A new Weaviate vector store instance
//   - error: Configuration errors or connection failures
//
// Example:
//
//	config := vectorstoresiface.Config{
//	    Embedder: embedder,
//	    ProviderConfig: map[string]any{
//	        "url":                "http://localhost:8080",
//	        "class_name":         "Document",
//	        "api_key":            "your-api-key", // Optional for Weaviate Cloud
//	        "embedding_dimension": 768,
//	    },
//	}
//	store, err := weaviate.NewWeaviateStoreFromConfig(ctx, config)
//
// Example usage can be found in examples/rag/simple/main.go.
func NewWeaviateStoreFromConfig(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
	// Extract weaviate-specific configuration
	url, _ := config.ProviderConfig["url"].(string)
	if url == "" {
		url = "http://localhost:8080" // Default Weaviate URL
	}

	apiKey, _ := config.ProviderConfig["api_key"].(string)

	className, _ := config.ProviderConfig["class_name"].(string)
	if className == "" {
		return nil, vectorstores.NewVectorStoreErrorWithMessage("NewWeaviateStoreFromConfig", vectorstores.ErrCodeInvalidConfig,
			"class_name is required in weaviate provider config", nil)
	}

	embeddingDim, _ := config.ProviderConfig["embedding_dimension"].(int)
	if embeddingDim == 0 {
		embeddingDim = 1536 // Default OpenAI embedding dimension
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	store := &WeaviateStore{
		url:          url,
		apiKey:       apiKey,
		className:    className,
		embeddingDim: embeddingDim,
		name:         "weaviate",
		httpClient:   httpClient,
	}

	// Verify class exists or create it
	if err := store.ensureClass(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure class: %w", err)
	}

	return store, nil
}

// ensureClass checks if the class exists and creates it if needed.
func (s *WeaviateStore) ensureClass(ctx context.Context) error {
	// Check if class exists
	url := fmt.Sprintf("%s/v1/schema/%s", s.url, s.className)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check class: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Class exists
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Class doesn't exist, create it
		return s.createClass(ctx)
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("unexpected status checking class: %d - %s", resp.StatusCode, string(body))
}

// createClass creates a new Weaviate class.
func (s *WeaviateStore) createClass(ctx context.Context) error {
	classDef := map[string]any{
		"class":      s.className,
		"vectorizer": "none", // We provide vectors ourselves
		"properties": []map[string]any{
			{
				"name":     "content",
				"dataType": []string{"text"},
			},
		},
	}

	bodyBytes, err := json.Marshal(classDef)
	if err != nil {
		return err
	}

	url := s.url + "/v1/schema"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create class: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create class: status %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// AddDocuments stores documents with their embeddings in Weaviate.
func (s *WeaviateStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
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

	// Prepare objects for Weaviate batch insert
	objects := make([]weaviateObject, len(documents))
	ids := make([]string, len(documents))

	for i, doc := range documents {
		// Generate ID if not present
		id := doc.ID
		if id == "" {
			id = fmt.Sprintf("%s-%d-%d", s.className, time.Now().UnixNano(), i)
		}
		ids[i] = id

		// Prepare properties (metadata + content)
		properties := make(map[string]any)
		properties["content"] = doc.GetContent()
		for k, v := range doc.Metadata {
			properties[k] = v
		}

		objects[i] = weaviateObject{
			ID:         id,
			Class:      s.className,
			Properties: properties,
			Vector:     embeddings[i],
		}
	}

	// Batch insert to Weaviate
	batchReq := weaviateBatchRequest{
		Objects: objects,
	}

	bodyBytes, err := json.Marshal(batchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	url := s.url + "/v1/batch/objects"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to batch insert objects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to batch insert objects: status %d - %s", resp.StatusCode, string(body))
	}

	return ids, nil
}

// DeleteDocuments removes documents from Weaviate.
func (s *WeaviateStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	// Delete objects one by one (Weaviate doesn't have batch delete in REST API)
	for _, id := range ids {
		url := fmt.Sprintf("%s/v1/objects/%s/%s", s.url, s.className, id)
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
		if err != nil {
			return err
		}

		if s.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.apiKey)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %w", id, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to delete object %s: status %d - %s", id, resp.StatusCode, string(body))
		}
	}

	return nil
}

// SimilaritySearch finds the k most similar documents to a query vector.
func (s *WeaviateStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
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

	// Build GraphQL query for vector search
	query := fmt.Sprintf(`{
		Get {
			%s(
				limit: %d
				nearVector: {
					vector: %s
				}
			) {
				_id
				content
				_additional {
					distance
				}
			}
		}
	}`, s.className, searchK, vectorToGraphQLArray(queryVector))

	// Add where clause for metadata filters if provided
	if len(config.MetadataFilters) > 0 {
		whereClause := buildWhereClause(config.MetadataFilters)
		query = fmt.Sprintf(`{
			Get {
				%s(
					limit: %d
					nearVector: {
						vector: %s
					}
					where: %s
				) {
					_id
					content
					_additional {
						distance
					}
				}
			}
		}`, s.className, searchK, vectorToGraphQLArray(queryVector), whereClause)
	}

	graphQLReq := weaviateGraphQLQuery{
		Query: query,
	}

	bodyBytes, err := json.Marshal(graphQLReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	url := s.url + "/v1/graphql"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("failed to execute GraphQL query: status %d - %s", resp.StatusCode, string(body))
	}

	var graphQLResp weaviateGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphQLResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(graphQLResp.Errors) > 0 {
		return nil, nil, fmt.Errorf("GraphQL errors: %v", graphQLResp.Errors)
	}

	// Extract results from GraphQL response
	getData, ok := graphQLResp.Data["Get"].(map[string]any)
	if !ok {
		return nil, nil, errors.New("unexpected GraphQL response format")
	}

	classData, ok := getData[s.className].([]any)
	if !ok {
		return []schema.Document{}, []float32{}, nil
	}

	// Convert results to schema.Document
	documents := make([]schema.Document, 0, len(classData))
	scores := make([]float32, 0, len(classData))

	for _, item := range classData {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		id, _ := itemMap["_id"].(string)
		content, _ := itemMap["content"].(string)

		additional, _ := itemMap["_additional"].(map[string]any)
		distance, _ := additional["distance"].(float64)
		score := 1.0 - float32(distance) // Convert distance to similarity score

		// Apply score threshold if configured
		if config.ScoreThreshold > 0 && score < config.ScoreThreshold {
			continue
		}

		// Extract metadata (all fields except content and _id, _additional)
		metadata := make(map[string]string)
		for k, v := range itemMap {
			if k != "content" && k != "_id" && k != "_additional" {
				if strVal, ok := v.(string); ok {
					metadata[k] = strVal
				}
			}
		}

		documents = append(documents, schema.Document{
			ID:          id,
			PageContent: content,
			Metadata:    metadata,
			Score:       score,
		})
		scores = append(scores, score)
	}

	return documents, scores, nil
}

// SimilaritySearchByQuery performs similarity search using a text query.
func (s *WeaviateStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
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
func (s *WeaviateStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	return &WeaviateRetriever{
		store: s,
		opts:  opts,
	}
}

// GetName returns the name of the vector store.
func (s *WeaviateStore) GetName() string {
	return s.name
}

// WeaviateRetriever implements the Retriever interface for WeaviateStore.
type WeaviateRetriever struct {
	store *WeaviateStore
	opts  []vectorstores.Option
}

// GetRelevantDocuments retrieves relevant documents for a query.
func (r *WeaviateRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
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

// Helper functions

// vectorToGraphQLArray converts a float32 slice to a GraphQL array string.
func vectorToGraphQLArray(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	buf := bytes.Buffer{}
	buf.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%.6f", v))
	}
	buf.WriteString("]")
	return buf.String()
}

// buildWhereClause builds a Weaviate where clause from metadata filters.
func buildWhereClause(filters map[string]any) string {
	if len(filters) == 0 {
		return "{}"
	}

	// Simple where clause - can be extended for complex filters
	conditions := make([]string, 0, len(filters))
	for k, v := range filters {
		conditions = append(conditions, fmt.Sprintf(`{path: ["%s"], operator: Equal, valueText: "%v"}`, k, v))
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	// Multiple conditions - use And operator
	return fmt.Sprintf(`{operator: And, operands: [%s]}`, joinStrings(conditions, ","))
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	buf := bytes.Buffer{}
	for i, s := range strs {
		if i > 0 {
			buf.WriteString(sep)
		}
		buf.WriteString(s)
	}
	return buf.String()
}

// Ensure WeaviateStore implements the VectorStore interface.
var _ vectorstores.VectorStore = (*WeaviateStore)(nil)

// Ensure WeaviateRetriever implements the Retriever interface.
var _ vectorstores.Retriever = (*WeaviateRetriever)(nil)
