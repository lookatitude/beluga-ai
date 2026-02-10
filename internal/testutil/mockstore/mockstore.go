package mockstore

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compile-time interface check
var _ vectorstore.VectorStore = (*MockVectorStore)(nil)

// MockVectorStore is a configurable mock for the VectorStore interface.
// It records all Add, Search, and Delete calls and can return preset results
// or errors.
type MockVectorStore struct {
	mu sync.Mutex

	documents   []schema.Document
	err         error
	addFn       func(ctx context.Context, docs []schema.Document, embeddings [][]float32) error
	searchFn    func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error)
	deleteFn    func(ctx context.Context, ids []string) error

	addCalls    int
	searchCalls int
	deleteCalls int
	lastDocs    []schema.Document
	lastQuery   []float32
	lastIDs     []string
}

// Option configures a MockVectorStore.
type Option func(*MockVectorStore)

// New creates a MockVectorStore with the given options.
func New(opts ...Option) *MockVectorStore {
	m := &MockVectorStore{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithDocuments configures the mock to return the given documents from Search.
func WithDocuments(docs []schema.Document) Option {
	return func(m *MockVectorStore) {
		m.documents = make([]schema.Document, len(docs))
		copy(m.documents, docs)
	}
}

// WithError configures the mock to return the given error from all methods.
func WithError(err error) Option {
	return func(m *MockVectorStore) {
		m.err = err
	}
}

// WithAddFunc sets a custom function to call on Add, overriding the canned error.
func WithAddFunc(fn func(ctx context.Context, docs []schema.Document, embeddings [][]float32) error) Option {
	return func(m *MockVectorStore) {
		m.addFn = fn
	}
}

// WithSearchFunc sets a custom function to call on Search, overriding
// the canned documents/error.
func WithSearchFunc(fn func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error)) Option {
	return func(m *MockVectorStore) {
		m.searchFn = fn
	}
}

// WithDeleteFunc sets a custom function to call on Delete, overriding the canned error.
func WithDeleteFunc(fn func(ctx context.Context, ids []string) error) Option {
	return func(m *MockVectorStore) {
		m.deleteFn = fn
	}
}

// Add inserts documents with their embeddings into the mock store.
// It records the call and returns the configured error, if any.
func (m *MockVectorStore) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.addCalls++
	m.lastDocs = make([]schema.Document, len(docs))
	copy(m.lastDocs, docs)

	if m.addFn != nil {
		return m.addFn(ctx, docs, embeddings)
	}

	if m.err != nil {
		return m.err
	}

	// Default: store the documents
	m.documents = append(m.documents, docs...)
	return nil
}

// Search finds the k most similar documents to the query vector.
// It returns the configured documents or error, and records the call.
func (m *MockVectorStore) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.searchCalls++
	m.lastQuery = make([]float32, len(query))
	copy(m.lastQuery, query)

	if m.searchFn != nil {
		return m.searchFn(ctx, query, k, opts...)
	}

	if m.err != nil {
		return nil, m.err
	}

	// Return up to k documents
	docs := m.documents
	if k < len(docs) {
		docs = docs[:k]
	}

	result := make([]schema.Document, len(docs))
	copy(result, docs)
	return result, nil
}

// Delete removes documents with the given IDs from the store.
// It records the call and returns the configured error, if any.
func (m *MockVectorStore) Delete(ctx context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls++
	m.lastIDs = make([]string, len(ids))
	copy(m.lastIDs, ids)

	if m.deleteFn != nil {
		return m.deleteFn(ctx, ids)
	}

	if m.err != nil {
		return m.err
	}

	// Default: remove documents with matching IDs
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	filtered := make([]schema.Document, 0, len(m.documents))
	for _, doc := range m.documents {
		if !idSet[doc.ID] {
			filtered = append(filtered, doc)
		}
	}
	m.documents = filtered
	return nil
}

// AddCalls returns the number of times Add has been called.
func (m *MockVectorStore) AddCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addCalls
}

// SearchCalls returns the number of times Search has been called.
func (m *MockVectorStore) SearchCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.searchCalls
}

// DeleteCalls returns the number of times Delete has been called.
func (m *MockVectorStore) DeleteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deleteCalls
}

// LastDocs returns the documents passed to the most recent Add call.
func (m *MockVectorStore) LastDocs() []schema.Document {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]schema.Document, len(m.lastDocs))
	copy(result, m.lastDocs)
	return result
}

// LastQuery returns the query vector passed to the most recent Search call.
func (m *MockVectorStore) LastQuery() []float32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]float32, len(m.lastQuery))
	copy(result, m.lastQuery)
	return result
}

// LastIDs returns the IDs passed to the most recent Delete call.
func (m *MockVectorStore) LastIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.lastIDs))
	copy(result, m.lastIDs)
	return result
}

// Documents returns a copy of all currently stored documents.
func (m *MockVectorStore) Documents() []schema.Document {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]schema.Document, len(m.documents))
	copy(result, m.documents)
	return result
}

// SetDocuments updates the documents returned by Search.
func (m *MockVectorStore) SetDocuments(docs []schema.Document) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.documents = make([]schema.Document, len(docs))
	copy(m.documents, docs)
	m.err = nil
}

// SetError updates the error for subsequent calls.
func (m *MockVectorStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Reset clears all recorded calls and stored data.
func (m *MockVectorStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.documents = nil
	m.err = nil
	m.addFn = nil
	m.searchFn = nil
	m.deleteFn = nil
	m.addCalls = 0
	m.searchCalls = 0
	m.deleteCalls = 0
	m.lastDocs = nil
	m.lastQuery = nil
	m.lastIDs = nil
}
