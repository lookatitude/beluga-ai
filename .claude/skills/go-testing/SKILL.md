---
name: go-testing
description: Go testing patterns for AI framework components. Use when writing tests, creating mocks, or benchmarking Beluga AI v2 code. Covers table-driven tests, mock patterns, stream testing, and integration test setup.
---

# Go Testing Patterns for Beluga AI v2

## Table-Driven Tests

```go
func TestGenerate(t *testing.T) {
    tests := []struct {
        name    string
        msgs    []schema.Message
        opts    []llm.GenerateOption
        want    *schema.AIMessage
        wantErr error
    }{
        {
            name: "simple completion",
            msgs: []schema.Message{schema.HumanMessage("hello")},
            want: &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart("hi")}},
        },
        {
            name:    "empty messages returns error",
            msgs:    nil,
            wantErr: &core.Error{Code: core.ErrInvalidInput},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := mockllm.New(mockllm.WithResponse(tt.want))
            got, err := model.Generate(context.Background(), tt.msgs, tt.opts...)
            if tt.wantErr != nil {
                require.Error(t, err)
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Stream Testing

```go
func TestStream(t *testing.T) {
    chunks := []schema.StreamChunk{
        {Type: schema.EventText, Text: "Hello"},
        {Type: schema.EventText, Text: " World"},
    }
    model := mockllm.New(mockllm.WithStreamChunks(chunks))

    var collected []string
    for chunk, err := range model.Stream(context.Background(), msgs) {
        require.NoError(t, err)
        collected = append(collected, chunk.Text)
    }
    assert.Equal(t, []string{"Hello", " World"}, collected)
}

func TestStreamCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    model := mockllm.New(mockllm.WithSlowStream(100 * time.Millisecond))

    count := 0
    for _, err := range model.Stream(ctx, msgs) {
        count++
        if count == 2 {
            cancel()
        }
        if err != nil {
            assert.ErrorIs(t, err, context.Canceled)
            break
        }
    }
    assert.LessOrEqual(t, count, 3)
}
```

## Mock Pattern

```go
// internal/testutil/mockllm/mockllm.go
type MockChatModel struct {
    GenerateFunc func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error)
    StreamFunc   func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindToolsFunc func(tools []tool.Tool) llm.ChatModel
    ModelIDFunc  func() string

    generateCalls int
    mu            sync.Mutex
}

func (m *MockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    m.mu.Lock()
    m.generateCalls++
    m.mu.Unlock()
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, msgs, opts...)
    }
    return &schema.AIMessage{}, nil
}

func (m *MockChatModel) CallCount() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.generateCalls
}
```

## Integration Tests

```go
//go:build integration

func TestOpenAIIntegration(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: apiKey,
        Model:  "gpt-4o-mini",
    })
    require.NoError(t, err)

    resp, err := model.Generate(context.Background(), []schema.Message{
        schema.HumanMessage("Say 'test' and nothing else"),
    })
    require.NoError(t, err)
    assert.Contains(t, strings.ToLower(resp.Text()), "test")
}
```

## Benchmarks

```go
func BenchmarkStreamProcessing(b *testing.B) {
    model := mockllm.New(mockllm.WithStreamChunks(generateChunks(100)))
    msgs := []schema.Message{schema.HumanMessage("test")}

    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        for _, err := range model.Stream(context.Background(), msgs) {
            if err != nil {
                b.Fatal(err)
            }
        }
    }
}
```

## Key Rules
- Every exported function gets a test
- Test error paths, not just happy path
- Always test context cancellation for streams
- Use `require` for fatal checks, `assert` for non-fatal
- `var _ Interface = (*Impl)(nil)` for compile-time checks
- Integration tests behind `//go:build integration`
