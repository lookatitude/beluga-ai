# Custom Mock for UI Testing

Welcome, colleague! In this integration guide, we're going to create custom mock implementations of Beluga AI's ChatModel interface for UI testing. This enables you to test your UI components without making real API calls.

## What you will build

You will create mock ChatModel implementations that simulate LLM responses for UI testing, enabling fast, reliable tests without external dependencies or API costs.

## Learning Objectives

- ✅ Create custom ChatModel mocks
- ✅ Simulate different response scenarios
- ✅ Use mocks in UI tests
- ✅ Understand mock patterns

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Understanding of testing patterns

## Step 1: Basic Mock Implementation

Create a simple mock:
```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type MockChatModel struct {
    responses map[string]string
    delay    time.Duration
}

func NewMockChatModel() *MockChatModel {
    return &MockChatModel{
        responses: make(map[string]string),
        delay:     0,
    }
}

func (m *MockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
    // Simulate delay
    if m.delay > 0 {
        time.Sleep(m.delay)
    }
    
    // Get last user message
    var userInput string
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].GetRole() == schema.RoleHuman {
            userInput = messages[i].GetContent()
            break
        }
    }
    
    // Return mock response
    response := m.responses[userInput]
    if response == "" {
        response = "This is a mock response."
    }
    
    return []schema.Message{schema.NewAIMessage(response)}, nil
}

func (m *MockChatModel) SetResponse(input, output string) {
    m.responses[input] = output
}

func (m *MockChatModel) SetDelay(delay time.Duration) {
    m.delay = delay
}
```

## Step 2: Advanced Mock with Scenarios

Create a scenario-based mock:
```go
type ScenarioMockChatModel struct {
    scenarios []MockScenario
    current   int
    mu        sync.Mutex
}

type MockScenario struct {
    Condition func([]schema.Message) bool
    Response  string
    Error     error
    Delay     time.Duration
}

func NewScenarioMockChatModel() *ScenarioMockChatModel {
    return &ScenarioMockChatModel{
        scenarios: make([]MockScenario, 0),
    }
}

func (m *ScenarioMockChatModel) AddScenario(scenario MockScenario) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.scenarios = append(m.scenarios, scenario)
}

func (m *ScenarioMockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Find matching scenario
    for _, scenario := range m.scenarios {
        if scenario.Condition == nil || scenario.Condition(messages) {
            if scenario.Delay > 0 {
                time.Sleep(scenario.Delay)
            }
            if scenario.Error != nil {
                return nil, scenario.Error
            }
            return []schema.Message{schema.NewAIMessage(scenario.Response)}, nil
        }
    }
    
    // Default response
    return []schema.Message{schema.NewAIMessage("Mock response")}, nil
}
```

## Step 3: Use in UI Tests

Use mock in UI testing:
```go
func TestUIWithMock(t *testing.T) {
    // Create mock
    mock := NewMockChatModel()
    mock.SetResponse("Hello", "Hi there! How can I help?")
    mock.SetDelay(100 * time.Millisecond) // Simulate network delay
    
    // Use in UI component
    uiComponent := NewUIComponent(mock)
    
    // Test interaction
    response, err := uiComponent.SendMessage("Hello")
    if err != nil {
        t.Fatalf("Failed: %v", err)
    }
    
    if response != "Hi there! How can I help?" {
        t.Errorf("Expected 'Hi there! How can I help?', got '%s'", response)
    }
}
```

## Step 4: Streaming Mock

Create a streaming mock:
```go
type StreamingMockChatModel struct {
    responses []string
    delay     time.Duration
}

func (m *StreamingMockChatModel) StreamMessages(ctx context.Context, messages []schema.Message) (<-chan schema.Message, error) {
    ch := make(chan schema.Message, len(m.responses))
    
    go func() {
        defer close(ch)
        for _, response := range m.responses {
            if m.delay > 0 {
                time.Sleep(m.delay)
            }
            ch <- schema.NewAIMessage(response)
        }
    }()

    
    return ch, nil
}
```

## Step 5: Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionMockChatModel struct {
    responses  map[string]string
    delay      time.Duration
    errorRate  float64
    mu         sync.RWMutex
    tracer     trace.Tracer
    callCount  int64
}

func NewProductionMockChatModel() *ProductionMockChatModel {
    return &ProductionMockChatModel{
        responses: make(map[string]string),
        delay:     50 * time.Millisecond,
        errorRate: 0.0,
        tracer:    otel.Tracer("beluga.chatmodels.mock"),
    }
}

func (m *ProductionMockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
    ctx, span := m.tracer.Start(ctx, "mock.generate_messages",
        trace.WithAttributes(attribute.Int("message_count", len(messages))),
    )
    defer span.End()
    
    m.mu.Lock()
    m.callCount++
    callCount := m.callCount
    m.mu.Unlock()
    
    span.SetAttributes(attribute.Int64("call_count", callCount))
    
    // Simulate delay
    if m.delay > 0 {
        time.Sleep(m.delay)
    }
    
    // Simulate errors
    if m.errorRate > 0 && float64(callCount%10)/10.0 < m.errorRate {
        err := fmt.Errorf("mock error")
        span.RecordError(err)
        return nil, err
    }
    
    // Get response
    var userInput string
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].GetRole() == schema.RoleHuman {
            userInput = messages[i].GetContent()
            break
        }
    }
    
    m.mu.RLock()
    response := m.responses[userInput]
    m.mu.RUnlock()
    
    if response == "" {
        response = fmt.Sprintf("Mock response to: %s", userInput)
    }
    
    span.SetAttributes(attribute.String("response_length", fmt.Sprintf("%d", len(response))))
    return []schema.Message{schema.NewAIMessage(response)}, nil
}

func (m *ProductionMockChatModel) SetResponse(input, output string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.responses[input] = output
}

func (m *ProductionMockChatModel) SetDelay(delay time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.delay = delay
}

func (m *ProductionMockChatModel) SetErrorRate(rate float64) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.errorRate = rate
}

func main() {
    ctx := context.Background()
    
    mock := NewProductionMockChatModel()
    mock.SetResponse("Hello", "Hi! How can I help?")
    mock.SetDelay(100 * time.Millisecond)
    
    messages := []schema.Message{
        schema.NewHumanMessage("Hello"),
    }
    
    responses, err := mock.GenerateMessages(ctx, messages)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    
    fmt.Printf("Response: %s\n", responses[0].GetContent())
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Delay` | Response delay | `0` | No |
| `ErrorRate` | Error simulation rate | `0.0` | No |
| `Responses` | Predefined responses | `\{\}` | No |

## Common Issues

### "Mock not responding"

**Problem**: No response configured.

**Solution**: Set responses:mock.SetResponse("input", "output")
```

### "Tests too fast"

**Problem**: No delay, tests don't simulate real behavior.

**Solution**: Add delay:mock.SetDelay(100 * time.Millisecond)
```

## Production Considerations

When using mocks in production:

- **Realistic delays**: Simulate network latency
- **Error scenarios**: Test error handling
- **State management**: Handle conversation state
- **Performance**: Keep mocks fast for tests
- **Coverage**: Test all code paths

## Next Steps

Congratulations! You've created custom mocks for UI testing. Next, learn how to:

- **[OpenAI Assistants API Bridge](./openai-assistants-api-bridge.md)** - Assistants integration
- **[ChatModels Package Documentation](../../api/packages/chatmodels.md)** - Deep dive into chatmodels
- **[Testing Guide](../../testing-guide.md)** - Testing patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
