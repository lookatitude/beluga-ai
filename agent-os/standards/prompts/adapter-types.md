# Adapter Types

Two adapters for different output formats.

## DefaultPromptAdapter
```go
adapter, _ := NewDefaultPromptAdapter("name", "Hello {{.name}}!", []string{"name"})
result, _ := adapter.Format(ctx, map[string]any{"name": "World"})
// result: "Hello World!" (string)
```
- Returns `string`
- Simple placeholder replacement

## ChatPromptAdapter
```go
adapter, _ := NewChatPromptAdapter("name", "You are helpful.", "{{.query}}", []string{"query"})
result, _ := adapter.Format(ctx, map[string]any{"query": "Hello"})
// result: []schema.Message{SystemMessage, UserMessage}
```
- Returns `[]schema.Message`
- Builds system + history + user message sequence
- `HistoryKey` defaults to "history" for inserting chat history

## Why Two Adapters?
1. **ISP compliance** - Different outputs need different interfaces
2. **Performance** - No runtime type checks
3. **Clarity** - Output type obvious at construction
