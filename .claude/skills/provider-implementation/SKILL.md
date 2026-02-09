---
name: provider-implementation
description: Implementing providers against Beluga AI v2 interfaces. Use when creating new LLM providers, embedding providers, vector store providers, voice providers, or any other provider that registers into a Beluga registry. Covers the init() registration pattern, error mapping, streaming, and testing.
---

# Provider Implementation Guide for Beluga AI v2

## Provider Checklist

Every provider must:
1. Implement the full interface (ChatModel, Embedder, VectorStore, STT, TTS, etc.)
2. Register via `init()` with the parent package's `Register()` function
3. Map provider-specific errors to `core.Error` with correct `ErrorCode`
4. Support context cancellation
5. Include token/usage metrics where applicable
6. Have a `New(cfg Config) (Interface, error)` constructor
7. Have a compile-time interface check: `var _ Interface = (*Impl)(nil)`
8. Have unit tests with mocked HTTP responses

## File Structure

```
llm/providers/openai/
├── openai.go          # Main implementation + New() + init()
├── options.go         # Provider-specific options
├── stream.go          # Streaming implementation
├── tools.go           # Tool calling support
├── errors.go          # Error mapping
├── openai_test.go     # Unit tests
└── testdata/          # Recorded HTTP responses for tests
    ├── chat_completion.json
    └── stream_completion.jsonl
```

## Implementation Template

```go
package openai

import (
    "context"
    "iter"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
)

var _ llm.ChatModel = (*Model)(nil) // compile-time check

type Model struct {
    client  *Client
    model   string
    tools   []tool.Tool
}

func New(cfg llm.ProviderConfig) (*Model, error) {
    if cfg.APIKey == "" {
        return nil, &core.Error{Op: "openai.new", Code: core.ErrAuth, Message: "API key required"}
    }
    return &Model{
        client: newClient(cfg.APIKey, cfg.BaseURL),
        model:  cfg.Model,
    }, nil
}

func init() {
    llm.Register("openai", func(cfg llm.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}

func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    req := m.buildRequest(msgs, opts...)
    resp, err := m.client.ChatCompletion(ctx, req)
    if err != nil {
        return nil, m.mapError("llm.generate", err)
    }
    return m.convertResponse(resp), nil
}

func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        req := m.buildRequest(msgs, opts...)
        req.Stream = true

        stream, err := m.client.ChatCompletionStream(ctx, req)
        if err != nil {
            yield(schema.StreamChunk{}, m.mapError("llm.stream", err))
            return
        }
        defer stream.Close()

        for {
            chunk, err := stream.Recv()
            if err != nil {
                if err == io.EOF { return }
                yield(schema.StreamChunk{}, m.mapError("llm.stream", err))
                return
            }
            if !yield(m.convertChunk(chunk), nil) {
                return // consumer stopped
            }
        }
    }
}

func (m *Model) BindTools(tools []tool.Tool) llm.ChatModel {
    return &Model{client: m.client, model: m.model, tools: tools}
}

func (m *Model) ModelID() string { return "openai/" + m.model }
```

## Error Mapping

```go
func (m *Model) mapError(op string, err error) error {
    var apiErr *APIError
    if !errors.As(err, &apiErr) {
        return &core.Error{Op: op, Code: core.ErrProviderDown, Err: err}
    }
    code := core.ErrProviderDown
    switch apiErr.StatusCode {
    case 401: code = core.ErrAuth
    case 429: code = core.ErrRateLimit
    case 408, 504: code = core.ErrTimeout
    case 400: code = core.ErrInvalidInput
    }
    return &core.Error{Op: op, Code: code, Message: apiErr.Message, Err: err}
}
```

## Testing with Recorded Responses

```go
func TestGenerate(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        data, _ := os.ReadFile("testdata/chat_completion.json")
        w.Header().Set("Content-Type", "application/json")
        w.Write(data)
    }))
    defer server.Close()

    model, err := New(llm.ProviderConfig{
        APIKey:  "test-key",
        Model:   "gpt-4o",
        BaseURL: server.URL,
    })
    require.NoError(t, err)

    resp, err := model.Generate(context.Background(), []schema.Message{
        schema.HumanMessage("hello"),
    })
    require.NoError(t, err)
    assert.NotEmpty(t, resp.Text())
}
```

## Reference

See `docs/providers.md` for provider categories and the extension guide.
