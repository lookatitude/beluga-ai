// Package documentloaders provides advanced test utilities and comprehensive mocks for testing document loader implementations.
package documentloaders

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AdvancedMockLoader provides a comprehensive mock implementation for testing.
type AdvancedMockLoader struct {
	documents     []schema.Document
	errorToReturn error
	shouldError   bool
	callCount     int
	lazyLoadCount int
	mu            sync.RWMutex
}

// NewAdvancedMockLoader creates a new advanced mock with configurable behavior.
func NewAdvancedMockLoader(documents []schema.Document, opts ...MockOption) *AdvancedMockLoader {
	m := &AdvancedMockLoader{
		documents: documents,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockLoader.
type MockOption func(*AdvancedMockLoader)

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockLoader) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithDocuments sets the documents to return.
func WithDocuments(docs []schema.Document) MockOption {
	return func(m *AdvancedMockLoader) {
		m.documents = docs
	}
}

// Load implements the DocumentLoader interface.
func (m *AdvancedMockLoader) Load(ctx context.Context) ([]schema.Document, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, NewLoaderError("Load", ErrCodeIOError, "", "mock error", nil)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]schema.Document, len(m.documents))
	copy(result, m.documents)
	return result, nil
}

// LazyLoad implements the DocumentLoader interface.
func (m *AdvancedMockLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
	m.mu.Lock()
	m.lazyLoadCount++
	m.mu.Unlock()

	ch := make(chan any, 10)

	go func() {
		defer close(ch)

		if m.shouldError {
			if m.errorToReturn != nil {
				ch <- m.errorToReturn
				return
			}
			ch <- NewLoaderError("LazyLoad", ErrCodeIOError, "", "mock error", nil)
			return
		}

		m.mu.RLock()
		docs := make([]schema.Document, len(m.documents))
		copy(docs, m.documents)
		m.mu.RUnlock()

		for _, doc := range docs {
			select {
			case ch <- doc:
			case <-ctx.Done():
				ch <- ctx.Err()
				return
			}
		}
	}()

	return ch, nil
}

// GetCallCount returns the number of times Load was called.
func (m *AdvancedMockLoader) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLazyLoadCount returns the number of times LazyLoad was called.
func (m *AdvancedMockLoader) GetLazyLoadCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lazyLoadCount
}

// Reset resets the mock state.
func (m *AdvancedMockLoader) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.lazyLoadCount = 0
	m.shouldError = false
	m.errorToReturn = nil
}

// Ensure AdvancedMockLoader implements iface.DocumentLoader
var _ iface.DocumentLoader = (*AdvancedMockLoader)(nil)
