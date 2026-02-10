package inmemory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	vectorstore.Register("inmemory", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return New(), nil
	})
}

// entry stores a document alongside its embedding vector.
type entry struct {
	doc       schema.Document
	embedding []float32
}

// Store is a thread-safe in-memory vector store using linear scan.
type Store struct {
	mu      sync.RWMutex
	entries map[string]entry
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// New creates a new empty in-memory Store.
func New() *Store {
	return &Store{
		entries: make(map[string]entry),
	}
}

// Add inserts documents with their embeddings. The docs and embeddings slices
// must have the same length. Documents are keyed by ID; adding a document with
// an existing ID overwrites the previous entry.
func (s *Store) Add(_ context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("inmemory: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, doc := range docs {
		s.entries[doc.ID] = entry{
			doc:       doc,
			embedding: embeddings[i],
		}
	}
	return nil
}

// Search finds the k most similar documents to the query vector using the
// configured distance strategy (defaults to cosine similarity). Results are
// returned in descending order of similarity with their Score field populated.
func (s *Store) Search(_ context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		doc   schema.Document
		score float64
	}
	var candidates []scored

	for _, e := range s.entries {
		if !matchesFilter(e.doc, cfg.Filter) {
			continue
		}

		score := similarity(query, e.embedding, cfg.Strategy)
		if cfg.Threshold > 0 && score < cfg.Threshold {
			continue
		}

		candidates = append(candidates, scored{doc: e.doc, score: score})
	}

	// Sort by score descending.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if k > len(candidates) {
		k = len(candidates)
	}

	results := make([]schema.Document, k)
	for i := 0; i < k; i++ {
		doc := candidates[i].doc
		doc.Score = candidates[i].score
		results[i] = doc
	}
	return results, nil
}

// Delete removes documents with the given IDs from the store.
// IDs that do not exist are silently ignored.
func (s *Store) Delete(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.entries, id)
	}
	return nil
}

// matchesFilter returns true if the document's metadata contains all
// key-value pairs in the filter. A nil filter matches everything.
func matchesFilter(doc schema.Document, filter map[string]any) bool {
	if filter == nil {
		return true
	}
	for k, v := range filter {
		if doc.Metadata == nil {
			return false
		}
		mv, ok := doc.Metadata[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", mv) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

// similarity computes the similarity between two vectors using the given
// strategy. For Cosine and DotProduct, higher is more similar. For Euclidean,
// the score is negated so higher still means more similar.
func similarity(a, b []float32, strategy vectorstore.SearchStrategy) float64 {
	switch strategy {
	case vectorstore.DotProduct:
		return dotProduct(a, b)
	case vectorstore.Euclidean:
		return -euclideanDistance(a, b)
	default: // Cosine
		return cosineSimilarity(a, b)
	}
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

// dotProduct computes the dot product between two vectors.
func dotProduct(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

// euclideanDistance computes the Euclidean (L2) distance between two vectors.
func euclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum float64
	for i := range a {
		d := float64(a[i]) - float64(b[i])
		sum += d * d
	}
	return math.Sqrt(sum)
}
