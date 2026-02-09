---
title: Mock ChatModel for UI Testing
description: Create mock ChatModel implementations for fast, deterministic UI testing without external API calls or costs.
---

Testing AI-powered UIs against live LLM APIs is slow, expensive, and non-deterministic. Mock `ChatModel` implementations let you test your application's UI components, conversation flows, and error handling with fast, repeatable results and zero API costs.

## Overview

This guide covers three mock patterns:

1. **Simple mock** — maps user input to predefined responses
2. **Scenario mock** — matches conditions and simulates errors or delays
3. **Streaming mock** — produces chunked responses using `iter.Seq2`

All mocks implement the `llm.ChatModel` interface, making them drop-in replacements for any real provider.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`go get github.com/lookatitude/beluga-ai`)
- A test suite or UI application that accepts `llm.ChatModel`

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Simple Mock

A basic mock that returns predefined responses based on user input.

```go
package mock

import (
    "context"
    "iter"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// SimpleMock maps user messages to predefined responses.
type SimpleMock struct {
    mu        sync.RWMutex
    responses map[string]string
    delay     time.Duration
}

// NewSimpleMock creates a mock with no predefined responses.
func NewSimpleMock() *SimpleMock {
    return &SimpleMock{
        responses: make(map[string]string),
    }
}

// SetResponse maps a user input string to a response.
func (m *SimpleMock) SetResponse(input, output string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.responses[input] = output
}

// SetDelay adds a simulated latency to every Generate call.
func (m *SimpleMock) SetDelay(d time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.delay = d
}

func (m *SimpleMock) Generate(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) (*schema.AIMessage, error) {
    m.mu.RLock()
    delay := m.delay
    m.mu.RUnlock()

    if delay > 0 {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(delay):
        }
    }

    // Find the last user message
    var userInput string
    for i := len(msgs) - 1; i >= 0; i-- {
        if msgs[i].Role() == schema.RoleHuman {
            userInput = msgs[i].(schema.HasText).Text()
            break
        }
    }

    m.mu.RLock()
    response, ok := m.responses[userInput]
    m.mu.RUnlock()

    if !ok {
        response = "This is a mock response."
    }

    return schema.NewAIMessage(schema.WithText(response)), nil
}

func (m *SimpleMock) Stream(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        result, err := m.Generate(ctx, msgs, opts...)
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }
        yield(schema.StreamChunk{Delta: result.Text()}, nil)
    }
}

func (m *SimpleMock) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return m
}

func (m *SimpleMock) ModelID() string {
    return "mock"
}
```

## Scenario Mock

A condition-based mock that matches message patterns and can simulate errors and delays per scenario.

```go
package mock

import (
    "context"
    "iter"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// Scenario defines a conditional response.
type Scenario struct {
    // Condition checks whether this scenario matches the input.
    // A nil Condition always matches.
    Condition func([]schema.Message) bool

    // Response is the text to return when matched.
    Response string

    // Err causes Generate to return this error when matched.
    Err error

    // Delay adds latency before responding.
    Delay time.Duration
}

// ScenarioMock evaluates scenarios in order and returns the first match.
type ScenarioMock struct {
    mu        sync.Mutex
    scenarios []Scenario
}

// NewScenarioMock creates a mock with no scenarios.
func NewScenarioMock() *ScenarioMock {
    return &ScenarioMock{}
}

// AddScenario appends a scenario. Scenarios are evaluated in insertion order.
func (m *ScenarioMock) AddScenario(s Scenario) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.scenarios = append(m.scenarios, s)
}

func (m *ScenarioMock) Generate(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) (*schema.AIMessage, error) {
    m.mu.Lock()
    scenarios := make([]Scenario, len(m.scenarios))
    copy(scenarios, m.scenarios)
    m.mu.Unlock()

    for _, s := range scenarios {
        if s.Condition != nil && !s.Condition(msgs) {
            continue
        }
        if s.Delay > 0 {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(s.Delay):
            }
        }
        if s.Err != nil {
            return nil, s.Err
        }
        return schema.NewAIMessage(schema.WithText(s.Response)), nil
    }

    return schema.NewAIMessage(schema.WithText("Default mock response")), nil
}

func (m *ScenarioMock) Stream(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        result, err := m.Generate(ctx, msgs, opts...)
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }
        yield(schema.StreamChunk{Delta: result.Text()}, nil)
    }
}

func (m *ScenarioMock) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return m
}

func (m *ScenarioMock) ModelID() string {
    return "mock-scenario"
}
```

## Streaming Mock

A mock that produces multiple chunks to test streaming UI behavior.

```go
package mock

import (
    "context"
    "iter"
    "time"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// StreamingMock produces token-by-token output from a list of chunks.
type StreamingMock struct {
    chunks     []string
    chunkDelay time.Duration
}

// NewStreamingMock creates a streaming mock with the given chunks and delay.
func NewStreamingMock(chunks []string, chunkDelay time.Duration) *StreamingMock {
    return &StreamingMock{
        chunks:     chunks,
        chunkDelay: chunkDelay,
    }
}

func (m *StreamingMock) Generate(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) (*schema.AIMessage, error) {
    var full string
    for _, chunk := range m.chunks {
        full += chunk
    }
    return schema.NewAIMessage(schema.WithText(full)), nil
}

func (m *StreamingMock) Stream(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        for _, chunk := range m.chunks {
            if m.chunkDelay > 0 {
                select {
                case <-ctx.Done():
                    yield(schema.StreamChunk{}, ctx.Err())
                    return
                case <-time.After(m.chunkDelay):
                }
            }
            if !yield(schema.StreamChunk{Delta: chunk}, nil) {
                return
            }
        }
    }
}

func (m *StreamingMock) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return m
}

func (m *StreamingMock) ModelID() string {
    return "mock-streaming"
}
```

## Using Mocks in Tests

### Basic Test

```go
package myapp_test

import (
    "context"
    "testing"

    "myapp/mock"
)

func TestChatUI(t *testing.T) {
    m := mock.NewSimpleMock()
    m.SetResponse("Hello", "Hi there! How can I help?")
    m.SetDelay(50 * time.Millisecond) // simulate network latency

    ui := NewChatUI(m)

    response, err := ui.SendMessage(context.Background(), "Hello")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if response != "Hi there! How can I help?" {
        t.Errorf("got %q, want %q", response, "Hi there! How can I help?")
    }
}
```

### Error Handling Test

```go
func TestChatUI_Error(t *testing.T) {
    m := mock.NewScenarioMock()
    m.AddScenario(mock.Scenario{
        Err: fmt.Errorf("rate limit exceeded"),
    })

    ui := NewChatUI(m)

    _, err := ui.SendMessage(context.Background(), "anything")
    if err == nil {
        t.Fatal("expected error, got nil")
    }
}
```

### Streaming UI Test

```go
func TestStreamingUI(t *testing.T) {
    m := mock.NewStreamingMock(
        []string{"Hello", " ", "world", "!"},
        10*time.Millisecond,
    )

    ctx := context.Background()
    var received []string
    for chunk, err := range m.Stream(ctx, nil) {
        if err != nil {
            t.Fatalf("stream error: %v", err)
        }
        received = append(received, chunk.Delta)
    }

    if len(received) != 4 {
        t.Errorf("got %d chunks, want 4", len(received))
    }
}
```

## Advanced Topics

### Using `internal/testutil` Mocks

Beluga AI ships with `MockChatModel` in `internal/testutil/` for framework-level testing. For application-level mocks where you need custom behavior, the patterns in this guide give you full control.

### Observability in Mocks

Add OpenTelemetry spans to track mock usage in integration test environments.

```go
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("beluga.mock")

func (m *SimpleMock) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    ctx, span := tracer.Start(ctx, "mock.generate")
    defer span.End()

    // ... existing logic
}
```

### Deterministic Test Fixtures

Use a fixture file for repeatable test data instead of hardcoding responses.

```go
func LoadFixtures(path string) *SimpleMock {
    data, err := os.ReadFile(path)
    if err != nil {
        panic(err)
    }
    var fixtures map[string]string
    if err := json.Unmarshal(data, &fixtures); err != nil {
        panic(err)
    }
    m := NewSimpleMock()
    for input, output := range fixtures {
        m.SetResponse(input, output)
    }
    return m
}
```

## Configuration Reference

| Mock Type | Use Case | Thread-Safe | Streaming |
|-----------|----------|-------------|-----------|
| `SimpleMock` | Basic input-output mapping | Yes | Single chunk |
| `ScenarioMock` | Conditional responses, error injection | Yes | Single chunk |
| `StreamingMock` | Token-by-token UI testing | Yes | Multi-chunk |

## Troubleshooting

### Mock returns default response

**Problem**: No response configured for the given input.

**Solution**: Call `SetResponse(input, output)` with the exact input string the UI sends. Use logging to capture the actual input being passed.

### Tests pass too quickly

**Problem**: No delay configured, so tests do not exercise timeout or loading-state logic.

**Solution**: Use `SetDelay()` to simulate realistic network latency. For streaming, set `chunkDelay` to test progressive rendering.

### Context cancellation not respected

**Problem**: Mock does not stop when the context is canceled.

**Solution**: Check for `ctx.Done()` in delay loops as shown in the examples above. Use `select` with `time.After` instead of `time.Sleep`.

## Related Resources

- [OpenAI Assistants API Bridge](/integrations/openai-assistants-bridge) — Integrate OpenAI Assistants
- [LLM Providers](/integrations/llm-providers) — Production LLM provider options
- [Testing Patterns](/guides/testing) — Comprehensive testing guide
