# Context Package

The context package provides utilities for building RAG (Retrieval-Augmented Generation) context. It offers a fluent builder pattern for assembling context from retrieved documents, conversation history, and templates.

## Features

- **Fluent Builder Pattern**: Chain methods for easy context construction
- **Document Management**: Add and sort documents by relevance score
- **Conversation History**: Maintain conversation history with role-based messages
- **Template Support**: Custom templates with placeholder substitution
- **Token Estimation**: Simple token count estimation for context sizing

## Usage

### Basic Context Building

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/context"

// Create a new context builder
builder := context.NewBuilder()

// Add documents with relevance scores
docs := []context.Document{
    {ID: "1", Content: "Document content here", Metadata: map[string]any{"source": "web"}},
    {ID: "2", Content: "Another document", Metadata: map[string]any{"source": "file"}},
}
scores := []float64{0.9, 0.8}

ctx := builder.
    AddDocuments(docs, scores).
    AddHistory([]context.Message{
        {Role: "user", Content: "Previous question"},
        {Role: "assistant", Content: "Previous answer"},
    }).
    WithSystemPrompt("You are a helpful assistant.").
    Build()
```

### Building Context for Questions

```go
// Build context specifically for a question
contextStr := builder.
    WithSystemPrompt("You are helpful").
    AddDocument(context.Document{ID: "1", Content: "Context info"}, 0.9).
    BuildForQuestion("What is AI?")

// Output includes:
// - System prompt
// - Context label with documents
// - Conversation history (if any)
// - Question
```

### Using Templates

```go
// Custom template with placeholders
builder := context.NewBuilder().
    WithSystemPrompt("You are an expert assistant").
    WithTemplate("System: {{system}}\n\nContext:\n{{documents}}\n\nHistory:\n{{history}}\n\nQuestion: {{question}}").
    AddDocument(context.Document{ID: "1", Content: "Info"}, 0.9).
    WithMetadata("user", "John")

result := builder.BuildForQuestion("How does it work?")
```

### Sorting Documents

```go
// Sort documents by relevance score (highest first)
builder.AddDocuments(docs, scores).SortByScore()
```

### Full Context Struct

```go
// Get a full context struct with metadata
ctx := builder.BuildContext("What is AI?")

fmt.Println(ctx.Content)     // The formatted context string
fmt.Println(ctx.Documents)   // Documents with scores
fmt.Println(ctx.History)     // Conversation history
fmt.Println(ctx.Metadata)    // Custom metadata
fmt.Println(ctx.TokenCount)  // Estimated token count
```

## Configuration Options

### Document Length Limits

```go
// Limit document content length (default: 10000)
builder.WithMaxDocumentLength(5000)
```

### History Size Limits

```go
// Limit number of history messages (default: 50)
builder.WithMaxHistorySize(20)
```

## Types

### Document

```go
type Document struct {
    Content  string
    Metadata map[string]any
    ID       string
}
```

### Message

```go
type Message struct {
    Role    string
    Content string
}
```

### Context

```go
type Context struct {
    Content    string
    Documents  []DocumentWithScore
    History    []Message
    Metadata   map[string]any
    TokenCount int
}
```

## Template Placeholders

Available placeholders in templates:
- `{{system}}` - System prompt
- `{{documents}}` - Formatted documents
- `{{history}}` - Formatted conversation history
- `{{question}}` - The question (when using BuildForQuestion)
- `{{key}}` - Any custom metadata key
