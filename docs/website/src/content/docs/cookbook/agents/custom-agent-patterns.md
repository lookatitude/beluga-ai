---
title: "Custom Agent Patterns"
description: "Recipe for extending Go AI agents with custom pre-processing, post-processing, and observability hooks using composition patterns in Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, custom agent patterns, Go agent hooks, agent composition, BaseAgent extension, agent observability, pre-processing"
---

## Problem

You need to add custom logic to an agent without modifying framework code. For example, you want to add pre-processing of inputs, post-processing of outputs, or custom logging.

## Solution

Use composition to wrap or extend the base agent. Beluga AI's agent system is designed for extension -- embed the base agent and add your custom behavior around it.

## Why This Matters

Every production agent eventually needs behavior that the framework doesn't provide out of the box: input sanitization, output formatting, custom metrics, audit logging, or domain-specific validation. The question is whether you modify the framework code, fork it, or compose around it.

Beluga AI is designed for the composition approach. By wrapping `agent.Agent` with a custom struct, you can intercept every stage of the agent lifecycle without touching framework internals. This follows Go's "accept interfaces, return structs" principle -- your `CustomAgent` accepts any implementation of `agent.Agent` and adds behavior around it. The functional options pattern (`WithInputFilter`, `WithOutputFilter`, `WithThoughtCallback`) keeps the API clean and extensible: adding new options doesn't change existing code or break callers.

The filter chain pattern used here (input filters applied in order, output filters applied in order) creates a processing pipeline that is easy to reason about and test. Each filter is a pure function that transforms data, making unit testing straightforward.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "strings"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
)

var tracer = otel.Tracer("beluga.agents.custom")

// CustomAgent wraps a base agent with custom behavior.
// Composition is used rather than inheritance for flexibility.
type CustomAgent struct {
    baseAgent     agent.Agent
    name          string
    inputFilters  []InputFilter
    outputFilters []OutputFilter
    onThought     func(string)
    onAction      func(agent.AgentAction)
}

// InputFilter pre-processes inputs before they reach the agent
type InputFilter func(inputs map[string]any) map[string]any

// OutputFilter post-processes results before returning
type OutputFilter func(result map[string]any) map[string]any

// CustomAgentOption configures the custom agent
type CustomAgentOption func(*CustomAgent)

// NewCustomAgent creates a new custom agent wrapping a base agent
func NewCustomAgent(name string, base agent.Agent, opts ...CustomAgentOption) *CustomAgent {
    ca := &CustomAgent{
        baseAgent:     base,
        name:          name,
        inputFilters:  make([]InputFilter, 0),
        outputFilters: make([]OutputFilter, 0),
    }

    for _, opt := range opts {
        opt(ca)
    }

    return ca
}

// WithInputFilter adds an input pre-processing filter
func WithInputFilter(filter InputFilter) CustomAgentOption {
    return func(ca *CustomAgent) {
        ca.inputFilters = append(ca.inputFilters, filter)
    }
}

// WithOutputFilter adds an output post-processing filter
func WithOutputFilter(filter OutputFilter) CustomAgentOption {
    return func(ca *CustomAgent) {
        ca.outputFilters = append(ca.outputFilters, filter)
    }
}

// WithThoughtCallback sets a callback for when the agent "thinks"
func WithThoughtCallback(cb func(string)) CustomAgentOption {
    return func(ca *CustomAgent) {
        ca.onThought = cb
    }
}

// WithActionCallback sets a callback for when the agent takes action
func WithActionCallback(cb func(agent.AgentAction)) CustomAgentOption {
    return func(ca *CustomAgent) {
        ca.onAction = cb
    }
}

// Plan implements agent.Agent by delegating to the base agent
// with custom pre/post processing
func (ca *CustomAgent) Plan(ctx context.Context, intermediateSteps []agent.IntermediateStep, inputs map[string]any) (agent.AgentAction, agent.AgentFinish, error) {
    ctx, span := tracer.Start(ctx, "custom_agent.plan",
        trace.WithAttributes(
            attribute.String("agent_name", ca.name),
        ))
    defer span.End()

    // Apply input filters
    processedInputs := ca.applyInputFilters(inputs)

    // Log the thought process
    if ca.onThought != nil {
        ca.onThought(fmt.Sprintf("Processing inputs: %v", processedInputs))
    }

    // Delegate to base agent
    action, finish, err := ca.baseAgent.Plan(ctx, intermediateSteps, processedInputs)
    if err != nil {
        span.RecordError(err)
        return action, finish, fmt.Errorf("base agent plan failed: %w", err)
    }

    // Notify action callback
    if action.Tool != "" && ca.onAction != nil {
        ca.onAction(action)
    }

    // Apply output filters if we have a finish
    if finish.ReturnValues != nil {
        finish.ReturnValues = ca.applyOutputFilters(finish.ReturnValues)
    }

    return action, finish, nil
}

// applyInputFilters runs all input filters in order
func (ca *CustomAgent) applyInputFilters(inputs map[string]any) map[string]any {
    result := inputs
    for _, filter := range ca.inputFilters {
        result = filter(result)
    }
    return result
}

// applyOutputFilters runs all output filters in order
func (ca *CustomAgent) applyOutputFilters(outputs map[string]any) map[string]any {
    result := outputs
    for _, filter := range ca.outputFilters {
        result = filter(result)
    }
    return result
}

// Name returns the agent name
func (ca *CustomAgent) Name() string {
    return ca.name
}

// InputVariables delegates to the base agent
func (ca *CustomAgent) InputVariables() []string {
    return ca.baseAgent.InputVariables()
}

// OutputVariables delegates to the base agent
func (ca *CustomAgent) OutputVariables() []string {
    return ca.baseAgent.OutputVariables()
}

// SanitizeInputFilter removes potentially harmful content
func SanitizeInputFilter() InputFilter {
    return func(inputs map[string]any) map[string]any {
        result := make(map[string]any)
        for k, v := range inputs {
            if str, ok := v.(string); ok {
                cleaned := strings.TrimSpace(str)
                result[k] = cleaned
            } else {
                result[k] = v
            }
        }
        return result
    }
}

// AddTimestampFilter adds a timestamp to inputs for logging
func AddTimestampFilter() InputFilter {
    return func(inputs map[string]any) map[string]any {
        result := make(map[string]any)
        for k, v := range inputs {
            result[k] = v
        }
        result["_timestamp"] = time.Now().Format(time.RFC3339)
        return result
    }
}

// AddMetadataFilter adds metadata to outputs
func AddMetadataFilter(agentName string) OutputFilter {
    return func(outputs map[string]any) map[string]any {
        result := make(map[string]any)
        for k, v := range outputs {
            result[k] = v
        }
        result["_agent"] = agentName
        result["_completed_at"] = time.Now().Format(time.RFC3339)
        return result
    }
}

func main() {
    ctx := context.Background()

    // Create base LLM and agent
    llmClient, _ := llm.New("openai", llm.ProviderConfig{
        APIKey: "your-api-key",
    })

    calculator := tool.NewFuncTool(
        "calculator",
        "Perform calculations",
        func(ctx context.Context, args map[string]any) (string, error) {
            return `{"result": 4}`, nil
        },
    )

    baseAgent, _ := agent.New("react", agent.Config{
        Name:  "base",
        Model: llmClient,
        Tools: []tool.Tool{calculator},
    })

    // Create custom agent with extensions
    customAgent := NewCustomAgent(
        "custom-assistant",
        baseAgent,
        WithInputFilter(SanitizeInputFilter()),
        WithInputFilter(AddTimestampFilter()),
        WithOutputFilter(AddMetadataFilter("custom-assistant")),
        WithThoughtCallback(func(thought string) {
            log.Printf("[THOUGHT] %s", thought)
        }),
        WithActionCallback(func(action agent.AgentAction) {
            log.Printf("[ACTION] Using tool: %s", action.Tool)
        }),
    )

    // Use the custom agent
    action, finish, err := customAgent.Plan(ctx, nil, map[string]any{
        "input": "What is 2 + 2?",
    })
    if err != nil {
        log.Fatalf("Agent failed: %v", err)
    }

    if finish.ReturnValues != nil {
        fmt.Printf("Result: %v\n", finish.ReturnValues)
    } else {
        fmt.Printf("Action: %s with %v\n", action.Tool, action.ToolInput)
    }
}
```

## Explanation

1. **Composition over inheritance** -- The `CustomAgent` wraps `agent.Agent` rather than extending a struct. This works with any agent type (ReAct, PlanExecute, or future types) without modification, because it depends on the interface, not a specific implementation.

2. **Filter chains** -- The chain of responsibility pattern is used for both inputs and outputs. Filters are applied in order, enabling complex processing pipelines. Each filter receives the output of the previous filter, so you can stack sanitization, enrichment, and validation independently.

3. **Callbacks for observability** -- The `onThought` and `onAction` callbacks let you hook into the agent's decision-making process without modifying the agent itself. This is useful for building debug UIs, recording agent trajectories, or feeding data into monitoring systems.

4. **Functional options pattern** -- `CustomAgentOption` functions configure the agent at construction time. This follows Beluga AI's `WithX()` convention and makes the API clean and extensible -- adding new options doesn't change existing code or break callers.

## Testing

```go
func TestCustomAgent_AppliesInputFilters(t *testing.T) {
    filterCalled := false
    inputFilter := func(inputs map[string]any) map[string]any {
        filterCalled = true
        inputs["filtered"] = true
        return inputs
    }

    mockAgent := &MockAgent{
        planFunc: func(ctx context.Context, steps []agent.IntermediateStep, inputs map[string]any) (agent.AgentAction, agent.AgentFinish, error) {
            if _, ok := inputs["filtered"]; !ok {
                t.Error("Input filter was not applied")
            }
            return agent.AgentAction{}, agent.AgentFinish{ReturnValues: map[string]any{"output": "done"}}, nil
        },
    }

    customAgent := NewCustomAgent("test", mockAgent, WithInputFilter(inputFilter))

    _, _, err := customAgent.Plan(context.Background(), nil, map[string]any{"input": "test"})
    if err != nil {
        t.Fatalf("Plan failed: %v", err)
    }

    if !filterCalled {
        t.Error("Input filter was not called")
    }
}

func TestCustomAgent_CallbacksAreCalled(t *testing.T) {
    thoughtCalled := false
    actionCalled := false

    mockAgent := &MockAgent{
        planFunc: func(ctx context.Context, steps []agent.IntermediateStep, inputs map[string]any) (agent.AgentAction, agent.AgentFinish, error) {
            return agent.AgentAction{Tool: "test_tool"}, agent.AgentFinish{}, nil
        },
    }

    customAgent := NewCustomAgent(
        "test",
        mockAgent,
        WithThoughtCallback(func(thought string) { thoughtCalled = true }),
        WithActionCallback(func(action agent.AgentAction) { actionCalled = true }),
    )

    _, _, _ = customAgent.Plan(context.Background(), nil, map[string]any{"input": "test"})

    if !thoughtCalled {
        t.Error("Thought callback was not called")
    }
    if !actionCalled {
        t.Error("Action callback was not called")
    }
}
```

## Variations

### Logging Agent

Create a version that logs all interactions for debugging and audit trails:

```go
func NewLoggingAgent(base agent.Agent, logger *log.Logger) *CustomAgent {
    return NewCustomAgent(
        "logging-"+base.Name(),
        base,
        WithThoughtCallback(func(thought string) {
            logger.Printf("[THOUGHT] %s", thought)
        }),
        WithActionCallback(func(action agent.AgentAction) {
            logger.Printf("[ACTION] %s: %v", action.Tool, action.ToolInput)
        }),
    )
}
```

### Rate-Limited Agent

Add rate limiting to prevent API abuse when the agent makes many tool calls in rapid succession:

```go
func NewRateLimitedAgent(base agent.Agent, rps float64) *CustomAgent {
    limiter := rate.NewLimiter(rate.Limit(rps), 1)

    return NewCustomAgent(
        "limited-"+base.Name(),
        base,
        WithInputFilter(func(inputs map[string]any) map[string]any {
            limiter.Wait(context.Background())
            return inputs
        }),
    )
}
```

## Related Recipes

- **[LLM Error Handling](/cookbook/llm/llm-error-handling)** -- Handle errors in your custom agent
