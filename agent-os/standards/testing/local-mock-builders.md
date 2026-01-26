# Local Mock Builders

Define test-specific mocks with fluent interface to avoid circular imports.

```go
// In test file, not in shared package
type integrationMockVectorStore struct {
    searchByQueryErr  error
    addDocumentsErr   error
    similarityResults []schema.Document
    similarityScores  []float32
    mu                sync.RWMutex
}

func newIntegrationMockVectorStore() *integrationMockVectorStore {
    return &integrationMockVectorStore{}
}

// Fluent builder methods return self for chaining
func (m *integrationMockVectorStore) WithSimilarityResults(
    docs []schema.Document,
    scores []float32,
) *integrationMockVectorStore {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.similarityResults = docs
    m.similarityScores = scores
    return m  // Return self for chaining
}

func (m *integrationMockVectorStore) WithSearchByQueryError(
    err error,
) *integrationMockVectorStore {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.searchByQueryErr = err
    return m
}
```

## Usage
```go
store := newIntegrationMockVectorStore().
    WithSimilarityResults(testDocs, testScores).
    WithSearchByQueryError(nil)
```

## Why Local Mocks
- **Avoids circular imports**: Test file imports package, not vice versa
- **Type-safe**: Implements exact interface needed
- **Tailored**: Only methods needed for specific test scenario

## When to Use
- Integration tests between packages
- When shared mocks would create import cycles
- When test needs custom behavior not in standard mocks
