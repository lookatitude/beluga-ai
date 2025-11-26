// Package inmemory provides an in-memory implementation of the VectorStore interface.
// This provider is suitable for development, testing, and small-scale applications
// where persistence is not required.
//
// Key Features:
// - Fast in-memory operations
// - Cosine similarity search
// - Thread-safe concurrent access
// - Automatic ID generation
//
// Limitations:
// - Data is lost on restart
// - Memory usage scales linearly with document count
// - Not suitable for large datasets (>100K documents)
//
// Example:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	// Create store
//	store, err := vectorstores.NewInMemoryStore(ctx, vectorstores.WithEmbedder(embedder))
//
//	// Add documents
//	docs := []schema.Document{
//		schema.NewDocument("Machine learning is awesome", map[string]string{"topic": "ml"}),
//	}
//	ids, err := store.AddDocuments(ctx, docs)
//
//	// Search
//	results, scores, err := store.SimilaritySearchByQuery(ctx, "ML basics", 5, embedder)
package inmemory

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Interfaces and types for the inmemory vector store
// These are defined locally to avoid import cycles with the main vectorstores package

// Embedder defines the interface for generating vector embeddings from text.
type Embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// Retriever defines the interface for retrieving documents based on queries.
type Retriever interface {
	GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}

// VectorStore defines the interface for vector storage and retrieval.
type VectorStore interface {
	AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)
	DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error
	SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)
	SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)
	AsRetriever(opts ...Option) Retriever
	GetName() string
}

// Option represents a functional option for configuring operations.
type Option func(*Config)

// Config holds configuration options for operations.
type Config struct {
	Embedder       Embedder
	SearchK        int
	ScoreThreshold float32
}

// NewDefaultConfig creates a new Config with default values.
func NewDefaultConfig() *Config {
	return &Config{
		SearchK:        5,
		ScoreThreshold: 0.0,
	}
}

// ApplyOptions applies a slice of options to a Config.
func ApplyOptions(config *Config, opts ...Option) {
	for _, opt := range opts {
		opt(config)
	}
}

// WithSearchK sets the number of similar documents to return.
func WithSearchK(k int) Option {
	return func(c *Config) {
		c.SearchK = k
	}
}

// InMemoryVectorStore is a simple in-memory implementation of the VectorStore interface.
// It stores documents and their embeddings in memory for fast access and similarity search.
type InMemoryVectorStore struct {
	embedder   Embedder
	name       string
	documents  []schema.Document
	embeddings [][]float32
	nextID     int
	mu         sync.RWMutex
}

// docWithScore holds a document and its similarity score for search results.
type docWithScore struct {
	Document schema.Document
	Score    float32
}

// NewInMemoryVectorStore creates a new in-memory vector store.
// The embedder parameter is optional but required for text-based operations.
func NewInMemoryVectorStore(embedder Embedder) *InMemoryVectorStore {
	store := &InMemoryVectorStore{
		documents:  make([]schema.Document, 0),
		embeddings: make([][]float32, 0),
		embedder:   embedder,
		name:       "inmemory",
		nextID:     1,
	}

	return store
}

// NewInMemoryVectorStoreFromConfig creates a new in-memory store from configuration.
// This is used by the factory pattern.
func NewInMemoryVectorStoreFromConfig(ctx context.Context, config Config) (VectorStore, error) {
	store := NewInMemoryVectorStore(config.Embedder)
	return store, nil
}

// Note: Provider registration is handled externally to avoid import cycles.
// Applications should import this package for side effects to register the provider.
// import _ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"

// AddDocuments adds documents to the vector store.
func (s *InMemoryVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
	// Apply options
	config := NewDefaultConfig()
	ApplyOptions(config, opts...)

	// Use embedder from options or stored embedder
	embedder := config.Embedder
	if embedder == nil {
		embedder = s.embedder
	}

	if embedder == nil && len(documents) > 0 && documents[0].Embedding == nil {
		return nil, errors.New("embedder is required if documents do not have pre-computed embeddings")
	}

	// Generate embeddings if needed
	textsToEmbed := make([]string, 0, len(documents))

	for _, doc := range documents {
		if doc.Embedding == nil || len(doc.Embedding) == 0 {
			textsToEmbed = append(textsToEmbed, doc.GetContent())
		}
	}

	// Embed documents that need embedding
	var embeddings [][]float32
	if len(textsToEmbed) > 0 {
		embeds, err := embedder.EmbedDocuments(ctx, textsToEmbed)
		if err != nil {
			return nil, fmt.Errorf("failed to embed documents: %w", err)
		}
		embeddings = embeds
	}

	// Store documents
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, len(documents))
	embedIndex := 0

	for i, doc := range documents {
		var embedding []float32
		if doc.Embedding != nil && len(doc.Embedding) > 0 {
			embedding = doc.Embedding
		} else {
			embedding = embeddings[embedIndex]
			embedIndex++
		}

		// Generate ID
		id := uuid.New().String()
		ids[i] = id

		// Create a copy of the document with the generated ID
		docWithID := schema.NewDocument(doc.GetContent(), doc.Metadata)
		docWithID.ID = id

		s.documents = append(s.documents, docWithID)
		s.embeddings = append(s.embeddings, embedding)
	}

	return ids, nil
}

// SimilaritySearch performs similarity search using a pre-computed query vector.
func (s *InMemoryVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 {
		return []schema.Document{}, []float32{}, nil
	}

	if k <= 0 {
		return nil, nil, errors.New("k must be greater than 0")
	}

	// Apply options
	config := NewDefaultConfig()
	ApplyOptions(config, opts...)
	if k == 0 {
		k = config.SearchK
	}

	// Calculate similarities
	results := make([]docWithScore, 0, len(s.documents))
	for i, emb := range s.embeddings {
		score, err := s.cosineSimilarity(queryVector, emb)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate similarity for document %d: %w", i, err)
		}

		if score >= config.ScoreThreshold {
			results = append(results, docWithScore{
				Document: s.documents[i],
				Score:    score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	numResults := k
	if len(results) < k {
		numResults = len(results)
	}

	resultDocs := make([]schema.Document, numResults)
	resultScores := make([]float32, numResults)
	for i := 0; i < numResults; i++ {
		resultDocs[i] = results[i].Document
		resultScores[i] = results[i].Score
	}

	return resultDocs, resultScores, nil
}

// SimilaritySearchByQuery performs similarity search using a text query.
func (s *InMemoryVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error) {
	// Use embedder from parameter or stored embedder
	currentEmbedder := embedder
	if currentEmbedder == nil {
		currentEmbedder = s.embedder
	}
	if currentEmbedder == nil {
		return nil, nil, errors.New("embedder is required for SimilaritySearchByQuery")
	}

	// Generate query embedding
	queryEmbedding, err := currentEmbedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Perform similarity search
	return s.SimilaritySearch(ctx, queryEmbedding, k, opts...)
}

// DeleteDocuments removes documents from the store based on their IDs.
func (s *InMemoryVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create set of IDs to delete for efficient lookup
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	// Filter documents
	var newDocuments []schema.Document
	var newEmbeddings [][]float32

	for i, doc := range s.documents {
		if !idSet[doc.ID] {
			newDocuments = append(newDocuments, doc)
			newEmbeddings = append(newEmbeddings, s.embeddings[i])
		}
	}

	s.documents = newDocuments
	s.embeddings = newEmbeddings

	return nil
}

// AsRetriever returns a Retriever instance based on this VectorStore.
func (s *InMemoryVectorStore) AsRetriever(opts ...Option) Retriever {
	return &InMemoryRetriever{
		store: s,
		opts:  opts,
	}
}

// GetName returns the name of the vector store.
func (s *InMemoryVectorStore) GetName() string {
	return s.name
}

// cosineSimilarity calculates cosine similarity between two vectors.
func (s *InMemoryVectorStore) cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors have different lengths: %d vs %d", len(a), len(b))
	}
	if len(a) == 0 {
		return 0, errors.New("vectors are empty")
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, errors.New("cannot compute cosine similarity with zero vector")
	}

	return float32(float64(dotProduct) / (math.Sqrt(float64(normA)) * math.Sqrt(float64(normB)))), nil
}

// InMemoryRetriever implements the Retriever interface for InMemoryVectorStore.
type InMemoryRetriever struct {
	store *InMemoryVectorStore
	opts  []Option
}

// GetRelevantDocuments retrieves relevant documents for a query.
func (r *InMemoryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Apply default search options
	opts := append(r.opts, WithSearchK(5))

	docs, _, err := r.store.SimilaritySearchByQuery(ctx, query, 5, nil, opts...)
	return docs, err
}

// Ensure InMemoryVectorStore implements the VectorStore interface.
var _ VectorStore = (*InMemoryVectorStore)(nil)

// Ensure InMemoryRetriever implements the Retriever interface.
var _ Retriever = (*InMemoryRetriever)(nil)
