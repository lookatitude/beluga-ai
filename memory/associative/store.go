package associative

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const (
	opAdd    = "associative.store.add"
	opGet    = "associative.store.get"
	opUpdate = "associative.store.update"
	opDelete = "associative.store.delete"

	errNoteNotFound = "note %q not found"
)

// NoteStore is the persistence interface for notes in associative memory.
// All methods must be safe for concurrent use.
type NoteStore interface {
	// Add stores a new note. Returns an error if a note with the same ID exists.
	Add(ctx context.Context, note *schema.Note) error

	// Get retrieves a note by ID. Returns a core.ErrNotFound error if the note
	// does not exist.
	Get(ctx context.Context, id string) (*schema.Note, error)

	// Update replaces a note in the store. Returns a core.ErrNotFound error if
	// the note does not exist.
	Update(ctx context.Context, note *schema.Note) error

	// Delete removes a note by ID. Returns a core.ErrNotFound error if the note
	// does not exist.
	Delete(ctx context.Context, id string) error

	// Search finds notes whose embeddings are most similar to the query vector,
	// returning at most k results ordered by descending similarity.
	Search(ctx context.Context, queryVec []float32, k int) ([]*schema.Note, error)

	// List returns all notes in the store ordered by creation time (oldest first).
	List(ctx context.Context) ([]*schema.Note, error)
}

// Compile-time check.
var _ NoteStore = (*InMemoryNoteStore)(nil)

// InMemoryNoteStore is a thread-safe in-memory implementation of NoteStore.
// It uses brute-force cosine similarity for search. Suitable for development,
// testing, and small-scale use cases.
type InMemoryNoteStore struct {
	mu    sync.RWMutex
	notes map[string]*schema.Note
}

// NewInMemoryNoteStore creates a new empty InMemoryNoteStore.
func NewInMemoryNoteStore() *InMemoryNoteStore {
	return &InMemoryNoteStore{
		notes: make(map[string]*schema.Note),
	}
}

// Add stores a new note. Returns an error if a note with the same ID exists.
func (s *InMemoryNoteStore) Add(_ context.Context, note *schema.Note) error {
	if note == nil {
		return core.NewError(opAdd, core.ErrInvalidInput, "note is nil", nil)
	}
	if note.ID == "" {
		return core.NewError(opAdd, core.ErrInvalidInput, "note ID is empty", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.notes[note.ID]; exists {
		return core.NewError(opAdd, core.ErrInvalidInput,
			fmt.Sprintf("note %q already exists", note.ID), nil)
	}

	s.notes[note.ID] = copyNote(note)
	return nil
}

// Get retrieves a note by ID.
func (s *InMemoryNoteStore) Get(_ context.Context, id string) (*schema.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	note, ok := s.notes[id]
	if !ok {
		return nil, core.NewError(opGet, core.ErrNotFound,
			fmt.Sprintf(errNoteNotFound, id), nil)
	}
	return copyNote(note), nil
}

// Update replaces a note in the store.
func (s *InMemoryNoteStore) Update(_ context.Context, note *schema.Note) error {
	if note == nil {
		return core.NewError(opUpdate, core.ErrInvalidInput, "note is nil", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[note.ID]; !ok {
		return core.NewError(opUpdate, core.ErrNotFound,
			fmt.Sprintf(errNoteNotFound, note.ID), nil)
	}

	s.notes[note.ID] = copyNote(note)
	return nil
}

// Delete removes a note by ID.
func (s *InMemoryNoteStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[id]; !ok {
		return core.NewError(opDelete, core.ErrNotFound,
			fmt.Sprintf(errNoteNotFound, id), nil)
	}

	delete(s.notes, id)
	return nil
}

// Search finds notes most similar to queryVec using brute-force cosine similarity.
func (s *InMemoryNoteStore) Search(_ context.Context, queryVec []float32, k int) ([]*schema.Note, error) {
	if k <= 0 {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		note  *schema.Note
		score float64
	}
	var results []scored

	for _, note := range s.notes {
		if len(note.Embedding) == 0 {
			continue
		}
		sim := cosineSimilarity(queryVec, note.Embedding)
		results = append(results, scored{note: note, score: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if k > len(results) {
		k = len(results)
	}

	out := make([]*schema.Note, k)
	for i := 0; i < k; i++ {
		out[i] = copyNote(results[i].note)
	}
	return out, nil
}

// List returns all notes ordered by creation time (oldest first).
func (s *InMemoryNoteStore) List(_ context.Context) ([]*schema.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*schema.Note, 0, len(s.notes))
	for _, note := range s.notes {
		out = append(out, copyNote(note))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})

	return out, nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude.
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

// copyNote creates a deep copy of a Note to prevent mutation of stored data.
func copyNote(n *schema.Note) *schema.Note {
	cp := *n
	if n.Keywords != nil {
		cp.Keywords = make([]string, len(n.Keywords))
		copy(cp.Keywords, n.Keywords)
	}
	if n.Tags != nil {
		cp.Tags = make([]string, len(n.Tags))
		copy(cp.Tags, n.Tags)
	}
	if n.Embedding != nil {
		cp.Embedding = make([]float32, len(n.Embedding))
		copy(cp.Embedding, n.Embedding)
	}
	if n.Links != nil {
		cp.Links = make([]string, len(n.Links))
		copy(cp.Links, n.Links)
	}
	if n.Metadata != nil {
		cp.Metadata = make(map[string]any, len(n.Metadata))
		for k, v := range n.Metadata {
			cp.Metadata[k] = v
		}
	}
	cp.CreatedAt = n.CreatedAt.Truncate(time.Nanosecond)
	cp.UpdatedAt = n.UpdatedAt.Truncate(time.Nanosecond)
	return &cp
}
