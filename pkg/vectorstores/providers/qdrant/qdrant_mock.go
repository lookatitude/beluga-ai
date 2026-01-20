package qdrant

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockQdrantStore provides a comprehensive mock implementation for testing Qdrant provider.
type AdvancedMockQdrantStore struct {
	mock.Mock
	mu                sync.RWMutex
	callCount         int
	shouldError       bool
	errorToReturn     error
	documents         []schema.Document
	embeddings        [][]float32
	documentIDs       []string
	simulateDelay     time.Duration
	simulateRateLimit bool
	rateLimitCount    int
	url               string
	collectionName    string
	name              string
}

// NewAdvancedMockQdrantStore creates a new advanced mock with configurable behavior.
func NewAdvancedMockQdrantStore(url, collectionName string) *AdvancedMockQdrantStore {
	mock := &AdvancedMockQdrantStore{
		url:            url,
		collectionName: collectionName,
		name:           "qdrant-mock",
		documents:      make([]schema.Document, 0),
		embeddings:     make([][]float32, 0),
		documentIDs:    make([]string, 0),
	}
	return mock
}

// AddDocuments implements the VectorStore interface.
func (m *AdvancedMockQdrantStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, errors.New("mock add documents error")
	}

	m.mu.Lock()
	ids := make([]string, len(documents))
	nextID := len(m.documentIDs) + 1
	for i, doc := range documents {
		id := fmt.Sprintf("qdrant_doc_%d", nextID)
		nextID++
		ids[i] = id
		m.documentIDs = append(m.documentIDs, id)
		m.documents = append(m.documents, doc)
		embedding := make([]float32, 128)
		for j := range embedding {
			embedding[j] = float32(i+j) / 128.0
		}
		m.embeddings = append(m.embeddings, embedding)
	}
	m.mu.Unlock()

	return ids, nil
}

// DeleteDocuments implements the VectorStore interface.
func (m *AdvancedMockQdrantStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return errors.New("mock delete documents error")
	}

	m.mu.Lock()
	for _, id := range ids {
		for i, docID := range m.documentIDs {
			if docID == id {
				m.documents = append(m.documents[:i], m.documents[i+1:]...)
				m.embeddings = append(m.embeddings[:i], m.embeddings[i+1:]...)
				m.documentIDs = append(m.documentIDs[:i], m.documentIDs[i+1:]...)
				break
			}
		}
	}
	m.mu.Unlock()

	return nil
}

// SimilaritySearch implements the VectorStore interface.
func (m *AdvancedMockQdrantStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	m.mu.RLock()
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.RUnlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, nil, errorToReturn
		}
		return nil, nil, errors.New("mock similarity search error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if k > len(m.documents) {
		k = len(m.documents)
	}
	if k < 0 {
		k = 0
	}

	docs := make([]schema.Document, k)
	scores := make([]float32, k)
	for i := 0; i < k && i < len(m.documents); i++ {
		docs[i] = m.documents[i]
		scores[i] = 0.9 - float32(i)*0.1
	}

	return docs, scores, nil
}

// SimilaritySearchByQuery implements the VectorStore interface.
func (m *AdvancedMockQdrantStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if embedder == nil {
		return nil, nil, errors.New("embedder is required")
	}

	queryVector, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	return m.SimilaritySearch(ctx, queryVector, k, opts...)
}

// AsRetriever implements the VectorStore interface.
func (m *AdvancedMockQdrantStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	return &simpleMockRetriever{
		store: m,
		k:     5,
	}
}

// GetName implements the VectorStore interface.
func (m *AdvancedMockQdrantStore) GetName() string {
	return m.name
}

// SetError configures the mock to return an error.
func (m *AdvancedMockQdrantStore) SetError(shouldError bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
}

// SetDelay configures the mock to simulate delay.
func (m *AdvancedMockQdrantStore) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
}

// GetCallCount returns the number of times methods have been called.
func (m *AdvancedMockQdrantStore) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// Reset resets the mock state.
func (m *AdvancedMockQdrantStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.shouldError = false
	m.errorToReturn = nil
	m.documents = m.documents[:0]
	m.embeddings = m.embeddings[:0]
	m.documentIDs = m.documentIDs[:0]
	m.rateLimitCount = 0
	m.simulateRateLimit = false
	m.simulateDelay = 0
}

type simpleMockRetriever struct {
	store *AdvancedMockQdrantStore
	k     int
}

func (r *simpleMockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	return []schema.Document{}, nil
}

var (
	_ vectorstores.VectorStore = (*AdvancedMockQdrantStore)(nil)
)
