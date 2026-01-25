// Package documentloaders provides advanced test utilities and comprehensive mocks for testing document loader implementations.
//
// Test Coverage Exclusions:
//
// The following code paths are intentionally excluded from 100% coverage requirements:
//
// 1. Panic Recovery Paths:
//   - Panic handlers in concurrent test runners
//   - These paths are difficult to test without causing actual panics in test code
//
// 2. Context Cancellation Edge Cases:
//   - Some context cancellation paths in LazyLoad operations are difficult to reliably test
//   - Race conditions between context cancellation and channel operations
//
// 3. Error Paths Requiring System Conditions:
//   - File system errors that require specific OS conditions
//   - Permission errors that require specific file system states
//   - Network errors for remote file loading (if implemented)
//
// 4. Provider-Specific Untestable Paths:
//   - Provider implementations in pkg/documentloaders/providers/* require external file systems
//   - These are tested through integration tests rather than unit tests
//   - File system operations that require actual file system state
//
// 5. Test Utility Functions:
//   - Helper functions in test_utils.go that are used by tests but not directly tested
//   - These are validated through their usage in actual test cases
//
// 6. Initialization Code:
//   - Package init() functions and global variable initialization
//   - Registry registration code that executes automatically
//
// 7. OTEL Context Logging:
//   - logWithOTELContext function has paths that require valid OTEL context
//   - Some edge cases in trace/span ID extraction are difficult to test in isolation
//
// All exclusions are documented here to maintain transparency about coverage goals.
// The target is 100% coverage of testable code paths, excluding the above categories.
package documentloaders

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AdvancedMockLoader provides a comprehensive mock implementation for testing.
type AdvancedMockLoader struct {
	errorToReturn error
	documents     []schema.Document
	callCount     int
	lazyLoadCount int
	mu            sync.RWMutex
	shouldError   bool
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

// WithErrorCode configures the mock to return a specific error code.
func WithErrorCode(code string) MockOption {
	return func(m *AdvancedMockLoader) {
		m.shouldError = true
		m.errorToReturn = NewLoaderError("Load", code, "", "mock error", nil)
	}
}

// WithIOError configures the mock to return an IO error.
func WithIOError() MockOption {
	return WithErrorCode(ErrCodeIOError)
}

// WithNotFoundError configures the mock to return a not found error.
func WithNotFoundError() MockOption {
	return WithErrorCode(ErrCodeNotFound)
}

// WithInvalidConfigError configures the mock to return an invalid config error.
func WithInvalidConfigError() MockOption {
	return WithErrorCode(ErrCodeInvalidConfig)
}

// WithFileTooLargeError configures the mock to return a file too large error.
func WithFileTooLargeError() MockOption {
	return WithErrorCode(ErrCodeFileTooLarge)
}

// WithCancelledError configures the mock to return a canceled error.
func WithCancelledError() MockOption {
	return WithErrorCode(ErrCodeCancelled)
}

// WithCycleDetectedError configures the mock to return a cycle detected error.
func WithCycleDetectedError() MockOption {
	return WithErrorCode(ErrCodeCycleDetected)
}

// WithBinaryFileError configures the mock to return a binary file error.
func WithBinaryFileError() MockOption {
	return WithErrorCode(ErrCodeBinaryFile)
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

// Ensure AdvancedMockLoader implements iface.DocumentLoader.
var _ iface.DocumentLoader = (*AdvancedMockLoader)(nil)
