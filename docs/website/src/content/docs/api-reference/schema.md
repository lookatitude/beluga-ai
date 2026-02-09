---
title: Schema Package API
description: API documentation for shared message types, multimodal content, and session types.
---

```go
import "github.com/lookatitude/beluga-ai/schema"
```

Package schema defines the shared types used throughout the Beluga AI framework. It contains message types, multimodal content parts, tool call/result types, document types for RAG, event types for streaming, and session types.

This package has zero external dependencies beyond the standard library and contains no business logic — only type definitions and constructors.

## Messages

### Message Interface

All messages implement the `Message` interface:

```go
type Message interface {
    GetRole() Role
    GetContent() []ContentPart
    GetMetadata() map[string]any
}
```

### Roles

```go
const (
    RoleSystem Role = "system"  // System instructions
    RoleHuman  Role = "human"   // User input
    RoleAI     Role = "ai"      // Model output
    RoleTool   Role = "tool"    // Tool result
)
```

### SystemMessage

System-level instructions:

```go
type SystemMessage struct {
    Parts    []ContentPart
    Metadata map[string]any
}

sys := schema.NewSystemMessage("You are a helpful assistant")
```

### HumanMessage

User input:

```go
type HumanMessage struct {
    Parts    []ContentPart
    Metadata map[string]any
}

msg := schema.NewHumanMessage("Hello, how are you?")
```

### AIMessage

Model output with tool calls and token usage:

```go
type AIMessage struct {
    Parts     []ContentPart
    ToolCalls []ToolCall
    Usage     Usage
    ModelID   string
    Metadata  map[string]any
}

resp := schema.NewAIMessage("I'm doing well, thank you!")
text := resp.Text() // Extract all text content
```

### ToolMessage

Tool execution result:

```go
type ToolMessage struct {
    ToolCallID string
    Parts      []ContentPart
    Metadata   map[string]any
}

result := schema.NewToolMessage(callID, "42")
```

## Multimodal Content

### ContentPart Interface

All content types implement `ContentPart`:

```go
type ContentPart interface {
    PartType() ContentType
}
```

### Content Types

```go
const (
    ContentText  ContentType = "text"
    ContentImage ContentType = "image"
    ContentAudio ContentType = "audio"
    ContentVideo ContentType = "video"
    ContentFile  ContentType = "file"
)
```

### TextPart

Plain text content:

```go
type TextPart struct {
    Text string
}

part := schema.TextPart{Text: "Hello world"}
```

### ImagePart

Image data (inline or URL):

```go
type ImagePart struct {
    Data     []byte // Raw image bytes (nil if URL provided)
    MimeType string // "image/png", "image/jpeg", etc.
    URL      string // Optional URL (empty if Data provided)
}
```

### AudioPart

Audio data for speech:

```go
type AudioPart struct {
    Data       []byte // Raw audio bytes
    Format     string // "wav", "mp3", "pcm16", etc.
    SampleRate int    // 16000, 44100, etc.
}
```

### VideoPart

Video data (inline or URL):

```go
type VideoPart struct {
    Data     []byte // Raw video bytes (nil if URL provided)
    MimeType string // "video/mp4", etc.
    URL      string // Optional URL (empty if Data provided)
}
```

### FilePart

Generic file attachment:

```go
type FilePart struct {
    Data     []byte // Raw file bytes
    Name     string // "report.pdf"
    MimeType string // "application/pdf"
}
```

## Tools

### ToolDefinition

Describes a tool's interface for model consumption:

```go
type ToolDefinition struct {
    Name        string
    Description string
    InputSchema map[string]any // JSON Schema
}
```

### ToolCall

Request from model to invoke a tool:

```go
type ToolCall struct {
    ID        string // Unique call ID
    Name      string // Tool name
    Arguments string // JSON-encoded arguments
}
```

### ToolResult

Output from tool execution:

```go
type ToolResult struct {
    CallID  string
    Content []ContentPart
    IsError bool
}
```

## Documents (RAG)

### Document

Unit of content for RAG pipeline:

```go
type Document struct {
    ID        string
    Content   string
    Metadata  map[string]any
    Score     float64    // Relevance score from retrieval
    Embedding []float32  // Optional vector embedding
}
```

## Sessions

### Session

Conversation session with turns and state:

```go
type Session struct {
    ID        string
    Turns     []Turn
    State     map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Turn

Single conversational exchange:

```go
type Turn struct {
    Input     Message
    Output    Message
    Timestamp time.Time
    Metadata  map[string]any
}
```

## Streaming

### StreamChunk

Incremental piece of streaming response:

```go
type StreamChunk struct {
    Delta        string      // Text delta
    ToolCalls    []ToolCall  // Incremental tool calls
    FinishReason string      // "stop", "tool_calls", "length"
    Usage        *Usage      // Token usage (may be nil)
    ModelID      string
}
```

## Usage

### Usage

Token consumption tracking:

```go
type Usage struct {
    InputTokens  int // Prompt tokens
    OutputTokens int // Generated tokens
    TotalTokens  int // Input + output
    CachedTokens int // Tokens from cache
}
```

## Agent Events

### AgentEvent

Event emitted during agent execution:

```go
type AgentEvent struct {
    Type      string // "agent_start", "tool_call", "thought", "handoff"
    AgentID   string
    Payload   any
    Timestamp time.Time
}
```

## Usage Examples

### Multimodal Messages

```go
// Text + image message
msg := &schema.HumanMessage{
    Parts: []schema.ContentPart{
        schema.TextPart{Text: "What's in this image?"},
        schema.ImagePart{
            Data:     imageBytes,
            MimeType: "image/png",
        },
    },
}
```

### Extract Text

```go
// Extract all text from any message
text := msg.Text()
```

### Type Assertions

```go
for _, part := range msg.GetContent() {
    switch p := part.(type) {
    case schema.TextPart:
        fmt.Println("Text:", p.Text)
    case schema.ImagePart:
        fmt.Println("Image size:", len(p.Data))
    case schema.AudioPart:
        fmt.Println("Audio format:", p.Format)
    }
}
```

## See Also

- [Core Package](./core.md) for foundational types
- [LLM Package](./llm.md) for LLM abstraction
- [RAG Package](./rag.md) for document types
