# Mock Factory Caching

Reuse mock instances within a test helper for consistency.

```go
type IntegrationTestHelper struct {
    mockLLMs      map[string]llmsiface.ChatModel
    mockMemories  map[string]memoryiface.Memory
    mockEmbedders map[string]embeddingsiface.Embedder
    mu            sync.RWMutex
}

func (h *IntegrationTestHelper) CreateMockLLM(name string) llmsiface.ChatModel {
    h.mu.Lock()
    defer h.mu.Unlock()

    if existing, exists := h.mockLLMs[name]; exists {
        return existing  // Reuse same mock
    }

    mockLLM := llms.NewAdvancedMockChatModel(name,
        llms.WithResponses(
            "Mock response 1",
            "Mock response 2",
        ),
    )

    h.mockLLMs[name] = mockLLM
    return mockLLM
}
```

## Why Cache Mocks
- **Consistency**: Same mock instance across operations
- **State tracking**: Mock can accumulate call counts
- **Performance**: Avoid repeated mock construction

## Reset Pattern
```go
func (h *IntegrationTestHelper) Reset() {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.mockLLMs = make(map[string]llmsiface.ChatModel)
    h.mockMemories = make(map[string]memoryiface.Memory)
    // ... clear all maps
}
```

## When to Use
- Integration test helpers shared across test functions
- Tests that need consistent mock behavior
- Multi-step scenarios within a single test
