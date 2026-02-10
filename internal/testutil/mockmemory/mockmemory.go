package mockmemory

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/schema"
)

// MockMemory is an in-memory mock of the memory.Memory interface.
// It stores Save'd messages and returns them from Load. It tracks call
// counts for each method.
type MockMemory struct {
	mu sync.Mutex

	messages   []schema.Message
	documents  []schema.Document
	err        error
	searchErr  error

	saveCalls   int
	loadCalls   int
	searchCalls int
	clearCalls  int
}

// Option configures a MockMemory.
type Option func(*MockMemory)

// New creates a MockMemory with the given options.
func New(opts ...Option) *MockMemory {
	m := &MockMemory{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithMessages pre-loads the mock with the given messages, which will be
// returned by Load.
func WithMessages(msgs []schema.Message) Option {
	return func(m *MockMemory) {
		m.messages = make([]schema.Message, len(msgs))
		copy(m.messages, msgs)
	}
}

// WithDocuments configures the mock to return the given documents from Search.
func WithDocuments(docs []schema.Document) Option {
	return func(m *MockMemory) {
		m.documents = make([]schema.Document, len(docs))
		copy(m.documents, docs)
	}
}

// WithError configures the mock to return the given error from Save and Load.
func WithError(err error) Option {
	return func(m *MockMemory) {
		m.err = err
	}
}

// WithSearchError configures the mock to return the given error from Search.
func WithSearchError(err error) Option {
	return func(m *MockMemory) {
		m.searchErr = err
	}
}

// Save persists an input/output message pair. It appends both to the
// internal message store.
func (m *MockMemory) Save(ctx context.Context, input schema.Message, output schema.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveCalls++

	if m.err != nil {
		return m.err
	}

	m.messages = append(m.messages, input, output)
	return nil
}

// Load returns all stored messages. The query parameter is recorded but
// not used for filtering in this mock.
func (m *MockMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loadCalls++

	if m.err != nil {
		return nil, m.err
	}

	result := make([]schema.Message, len(m.messages))
	copy(result, m.messages)
	return result, nil
}

// Search returns the configured documents, limited to k results.
func (m *MockMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.searchCalls++

	if m.searchErr != nil {
		return nil, m.searchErr
	}

	docs := m.documents
	if k < len(docs) {
		docs = docs[:k]
	}

	result := make([]schema.Document, len(docs))
	copy(result, docs)
	return result, nil
}

// Clear removes all stored messages.
func (m *MockMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clearCalls++

	if m.err != nil {
		return m.err
	}

	m.messages = nil
	m.documents = nil
	return nil
}

// SaveCalls returns the number of times Save has been called.
func (m *MockMemory) SaveCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveCalls
}

// LoadCalls returns the number of times Load has been called.
func (m *MockMemory) LoadCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadCalls
}

// SearchCalls returns the number of times Search has been called.
func (m *MockMemory) SearchCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.searchCalls
}

// ClearCalls returns the number of times Clear has been called.
func (m *MockMemory) ClearCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clearCalls
}

// Messages returns a copy of all currently stored messages.
func (m *MockMemory) Messages() []schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]schema.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// Reset clears all stored data and call counters.
func (m *MockMemory) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
	m.documents = nil
	m.err = nil
	m.searchErr = nil
	m.saveCalls = 0
	m.loadCalls = 0
	m.searchCalls = 0
	m.clearCalls = 0
}
