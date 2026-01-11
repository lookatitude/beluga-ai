// Package textsplitters provides advanced test utilities and comprehensive mocks for testing text splitter implementations.
package textsplitters

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
)

// AdvancedMockSplitter provides a comprehensive mock implementation for testing.
type AdvancedMockSplitter struct {
	chunks         []string
	documents      []schema.Document
	errorToReturn  error
	shouldError    bool
	callCount      int
	splitTextCount int
	mu             sync.RWMutex
}

// NewAdvancedMockSplitter creates a new advanced mock with configurable behavior.
func NewAdvancedMockSplitter(chunks []string, opts ...MockOption) *AdvancedMockSplitter {
	m := &AdvancedMockSplitter{
		chunks: chunks,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockSplitter.
type MockOption func(*AdvancedMockSplitter)

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockSplitter) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithChunks sets the chunks to return from SplitText.
func WithChunks(chunks []string) MockOption {
	return func(m *AdvancedMockSplitter) {
		m.chunks = chunks
	}
}

// WithDocuments sets the documents to return from SplitDocuments.
func WithDocuments(docs []schema.Document) MockOption {
	return func(m *AdvancedMockSplitter) {
		m.documents = docs
	}
}

// SplitText implements the TextSplitter interface.
func (m *AdvancedMockSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
	m.mu.Lock()
	m.splitTextCount++
	m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, NewSplitterError("SplitText", ErrCodeInvalidConfig, "mock error", nil)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.chunks))
	copy(result, m.chunks)
	return result, nil
}

// SplitDocuments implements the TextSplitter interface.
func (m *AdvancedMockSplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, NewSplitterError("SplitDocuments", ErrCodeInvalidConfig, "mock error", nil)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]schema.Document, len(m.documents))
	copy(result, m.documents)
	return result, nil
}

// CreateDocuments implements the TextSplitter interface.
func (m *AdvancedMockSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, NewSplitterError("CreateDocuments", ErrCodeInvalidConfig, "mock error", nil)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]schema.Document, len(m.documents))
	copy(result, m.documents)
	return result, nil
}

// GetCallCount returns the number of times SplitDocuments was called.
func (m *AdvancedMockSplitter) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetSplitTextCount returns the number of times SplitText was called.
func (m *AdvancedMockSplitter) GetSplitTextCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.splitTextCount
}

// Reset resets the mock state.
func (m *AdvancedMockSplitter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.splitTextCount = 0
	m.shouldError = false
	m.errorToReturn = nil
}

// Ensure AdvancedMockSplitter implements iface.TextSplitter
var _ iface.TextSplitter = (*AdvancedMockSplitter)(nil)
