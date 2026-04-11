package colbert

import (
	"context"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// IndexResult represents a document match from a ColBERT index search,
// carrying the document ID and its MaxSim relevance score.
type IndexResult struct {
	// ID is the unique identifier of the matched document.
	ID string
	// Score is the MaxSim score for this document against the query.
	Score float64
}

// ColBERTIndex stores pre-computed per-token document embeddings and supports
// search by MaxSim scoring. Implementations must be safe for concurrent use.
type ColBERTIndex interface {
	// Add stores per-token embeddings for a document. If a document with the
	// same ID already exists, its embeddings are replaced.
	Add(ctx context.Context, id string, tokenVecs [][]float32) error

	// Search finds the top-k documents most similar to the query token
	// vectors, scored by MaxSim. Results are ordered by decreasing score.
	Search(ctx context.Context, queryVecs [][]float32, k int) ([]IndexResult, error)
}

// indexEntry holds a document's pre-computed token embeddings.
type indexEntry struct {
	id        string
	tokenVecs [][]float32
}

// InMemoryIndex is a brute-force ColBERTIndex that stores all document token
// embeddings in memory and computes MaxSim over every document at search time.
// It is thread-safe and intended for testing and small-scale usage.
type InMemoryIndex struct {
	mu      sync.RWMutex
	entries map[string]*indexEntry
}

// Compile-time check that InMemoryIndex implements ColBERTIndex.
var _ ColBERTIndex = (*InMemoryIndex)(nil)

// NewInMemoryIndex creates a new empty InMemoryIndex.
func NewInMemoryIndex() *InMemoryIndex {
	return &InMemoryIndex{
		entries: make(map[string]*indexEntry),
	}
}

// Add stores per-token embeddings for a document. If a document with the same
// ID already exists, its embeddings are replaced. Returns an error if id is
// empty or tokenVecs is nil.
func (idx *InMemoryIndex) Add(ctx context.Context, id string, tokenVecs [][]float32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return core.Errorf(core.ErrInvalidInput, "colbert: index: document ID must not be empty")
	}
	if tokenVecs == nil {
		return core.Errorf(core.ErrInvalidInput, "colbert: index: tokenVecs must not be nil for document %q", id)
	}

	// Deep copy to avoid caller mutations.
	copied := make([][]float32, len(tokenVecs))
	for i, tv := range tokenVecs {
		c := make([]float32, len(tv))
		copy(c, tv)
		copied[i] = c
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries[id] = &indexEntry{id: id, tokenVecs: copied}
	return nil
}

// Search finds the top-k documents most similar to queryVecs by MaxSim score.
// Results are ordered by decreasing score. Returns at most k results.
func (idx *InMemoryIndex) Search(ctx context.Context, queryVecs [][]float32, k int) ([]IndexResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if k <= 0 {
		return nil, nil
	}

	idx.mu.RLock()
	// Snapshot entries under lock.
	snapshot := make([]*indexEntry, 0, len(idx.entries))
	for _, e := range idx.entries {
		snapshot = append(snapshot, e)
	}
	idx.mu.RUnlock()

	results := make([]IndexResult, 0, len(snapshot))
	for _, e := range snapshot {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		score := MaxSim(queryVecs, e.tokenVecs)
		results = append(results, IndexResult{ID: e.id, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if k < len(results) {
		results = results[:k]
	}
	return results, nil
}

// Len returns the number of documents in the index.
func (idx *InMemoryIndex) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.entries)
}
