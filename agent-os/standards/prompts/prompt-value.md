# PromptValue Interface

Intermediate representation for prompt output.

```go
type PromptValue interface {
    ToString() string
    ToMessages() []schema.Message
}
```

## Purpose
- Allows same prompt to be used with both string-based and message-based LLMs
- Consumer chooses output format at call site
- Decouples prompt construction from LLM requirements

## Implementation Pattern
```go
type StringPromptValue struct {
    content string
}

func (v *StringPromptValue) ToString() string {
    return v.content
}

func (v *StringPromptValue) ToMessages() []schema.Message {
    return []schema.Message{schema.NewHumanMessage(v.content)}
}
```

## When to Use
- Return PromptValue from templates when output format is unknown
- Use adapters (DefaultPromptAdapter, ChatPromptAdapter) when format is known
