// Package vectorstores provides implementations of the rag.VectorStore interface.
package vectorstores

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/rag"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/rag/retrievers"

	// Need a math library for vector operations (e.g., cosine similarity)
	"gonum.org/v1/gonum/floats"
)

// storedDoc holds the document and its embedding.
type storedDoc struct {
	ID        string
	Document  schema.Document
	Embedding []float32
}

// InMemoryVectorStore provides a simple in-memory implementation of rag.VectorStore.
// It requires an Embedder to be provided either during creation or via options during AddDocuments.
type InMemoryVectorStore struct {
	mu       sync.RWMutex
	store    map[string]storedDoc // Map from ID to storedDoc
	Embedder rag.Embedder         // Optional default embedder
	// TODO: Add options for similarity function (cosine, dot product, etc.)
}

// NewInMemoryVectorStore creates a new InMemoryVectorStore.
// An embedder can be optionally provided as the default.
func NewInMemoryVectorStore(embedder rag.Embedder) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		store:    make(map[string]storedDoc),
		Embedder: embedder,
	}
}

// getEmbedder resolves the embedder to use, prioritizing options over the default.
func (s *InMemoryVectorStore) getEmbedder(options ...core.Option) (rag.Embedder, error) {
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}
	if embedderOpt, ok := config["embedder"].(rag.Embedder); ok {
		return embedderOpt, nil
	}
	if s.Embedder != nil {
		return s.Embedder, nil
	}
	return nil, errors.New("embedder must be provided either during InMemoryVectorStore creation or via WithEmbedder option")
}

// AddDocuments embeds and stores documents.
func (s *InMemoryVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, options ...core.Option) ([]string, error) {
	embedder, err := s.getEmbedder(options...)
	if err != nil {
		return nil, err
	}

	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.GetContent()
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}

	if len(embeddings) != len(documents) {
		return nil, fmt.Errorf("number of embeddings (%d) does not match number of documents (%d)", len(embeddings), len(documents))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, len(documents))
	for i, doc := range documents {
		newID := uuid.New().String()
		s.store[newID] = storedDoc{
			ID:        newID,
			Document:  doc,
			Embedding: embeddings[i],
		}
		ids[i] = newID
	}

	return ids, nil
}

// DeleteDocuments removes documents by ID.
func (s *InMemoryVectorStore) DeleteDocuments(ctx context.Context, ids []string, options ...core.Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	deletedCount := 0
	for _, id := range ids {
		if _, exists := s.store[id]; exists {
			delete(s.store, id)
			deletedCount++
		}
	}
	// Optionally return an error if not all IDs were found?
	// fmt.Printf("Deleted %d out of %d requested documents\n", deletedCount, len(ids))
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector lengths do not match: %d vs %d", len(a), len(b))
	}
	// Convert []float32 to []float64 for gonum functions
	a64 := make([]float64, len(a))
	b64 := make([]float64, len(b))
	for i := range a {
		a64[i] = float64(a[i])
		b64[i] = float64(b[i])
	}
	dotProduct := floats.Dot(a64, b64)
	normA := floats.Norm(a64, 2)
	normB := floats.Norm(b64, 2)
	if normA == 0 || normB == 0 {
		return 0, nil // Avoid division by zero
	}
	return float32(dotProduct / (normA * normB)), nil
}

// docWithScore holds a document and its similarity score.
type docWithScore struct {
	Document schema.Document
	Score    float32
}

// similaritySearchInternal performs the core similarity search logic.
func (s *InMemoryVectorStore) similaritySearchInternal(ctx context.Context, queryEmbedding []float32, k int, options ...core.Option) ([]docWithScore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}
	scoreThreshold, _ := config["score_threshold"].(float32) // Defaults to 0 if not set or wrong type
	// TODO: Implement metadata filtering
	// metadataFilter, _ := config["metadata_filter"].(map[string]any)

	results := make([]docWithScore, 0, len(s.store))
	for _, item := range s.store {
		// TODO: Apply metadata filter here if implemented

		score, err := cosineSimilarity(queryEmbedding, item.Embedding)
		if err != nil {
			// Should not happen if embeddings have consistent dimensions
			return nil, fmt.Errorf("error calculating similarity for doc %s: %w", item.ID, err)
		}

		if score >= scoreThreshold {
			results = append(results, docWithScore{Document: item.Document, Score: score})
		}
	}

	// Sort results by score descending
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K results
	if k > 0 && len(results) > k {
		results = results[:k]
	}

	return results, nil
}

// SimilaritySearch performs search using a query string.
func (s *InMemoryVectorStore) SimilaritySearch(ctx context.Context, query string, k int, options ...core.Option) ([]schema.Document, error) {
	embedder, err := s.getEmbedder(options...)
	if err != nil {
		return nil, err
	}

	queryEmbedding, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	resultsWithScore, err := s.similaritySearchInternal(ctx, queryEmbedding, k, options...)
	if err != nil {
		return nil, err
	}

	// Extract documents from results
	docs := make([]schema.Document, len(resultsWithScore))
	for i, res := range resultsWithScore {
		// TODO: Optionally add score to document metadata?
		docs[i] = res.Document
	}
	return docs, nil
}

// SimilaritySearchByVector performs search using a pre-computed vector.
func (s *InMemoryVectorStore) SimilaritySearchByVector(ctx context.Context, embedding []float32, k int, options ...core.Option) ([]schema.Document, error) {
	resultsWithScore, err := s.similaritySearchInternal(ctx, embedding, k, options...)
	if err != nil {
		return nil, err
	}

	// Extract documents from results
	docs := make([]schema.Document, len(resultsWithScore))
	for i, res := range resultsWithScore {
		// TODO: Optionally add score to document metadata?
		docs[i] = res.Document
	}
	return docs, nil
}

// AsRetriever returns a Retriever instance based on this VectorStore.
func (s *InMemoryVectorStore) AsRetriever(options ...core.Option) rag.Retriever {
	// Use the VectorStoreRetriever implementation
	return retrievers.NewVectorStoreRetriever(s, options...)
}

// Ensure InMemoryVectorStore implements the interface.
var _ rag.VectorStore = (*InMemoryVectorStore)(nil)
