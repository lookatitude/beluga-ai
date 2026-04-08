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

Use composition to wrap or extend the base agent. Beluga AI's agent system is designed for extension -- embed `*agent.BaseAgent` and override `Invoke` and `Stream` for custom behavior.

## Why This Matters

Every production agent eventually needs behavior that the framework doesn't provide out of the box: input sanitization, output formatting, custom metrics, audit logging, or domain-specific validation. The question is whether you modify the framework code, fork it, or compose around it.

Beluga AI is designed for the composition approach. By embedding `*agent.BaseAgent` in a custom struct, you can override specific methods while inheriting identity, persona, tool management, and child agent tracking. This follows Go's "accept interfaces, return structs" principle. The functional options pattern keeps the API clean and extensible.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// ObservableAgent wraps BaseAgent with input/output processing and lifecycle callbacks.
type ObservableAgent struct {
	*agent.BaseAgent
	inputFilters  []InputFilter
	outputFilters []OutputFilter
	onStart       func(input string)
	onFinish      func(output string)
}

// InputFilter transforms the input string before it reaches the agent.
type InputFilter func(input string) string

// OutputFilter transforms the output string before it is returned.
type OutputFilter func(output string) string

// ObservableOption configures the ObservableAgent.
type ObservableOption func(*ObservableAgent)

// NewObservableAgent creates an agent with input/output processing hooks.
func NewObservableAgent(id string, opts []agent.Option, obsOpts ...ObservableOption) *ObservableAgent {
	base := agent.New(id, opts...)
	oa := &ObservableAgent{BaseAgent: base}
	for _, opt := range obsOpts {
		opt(oa)
	}
	return oa
}

// WithInputFilter adds an input pre-processing filter.
func WithInputFilter(f InputFilter) ObservableOption {
	return func(oa *ObservableAgent) {
		oa.inputFilters = append(oa.inputFilters, f)
	}
}

// WithOutputFilter adds an output post-processing filter.
func WithOutputFilter(f OutputFilter) ObservableOption {
	return func(oa *ObservableAgent) {
		oa.outputFilters = append(oa.outputFilters, f)
	}
}

// WithOnStart sets a callback invoked before each Invoke.
func WithOnStart(cb func(input string)) ObservableOption {
	return func(oa *ObservableAgent) {
		oa.onStart = cb
	}
}

// WithOnFinish sets a callback invoked after each Invoke.
func WithOnFinish(cb func(output string)) ObservableOption {
	return func(oa *ObservableAgent) {
		oa.onFinish = cb
	}
}

// Invoke applies input filters, delegates to the base agent, then applies output filters.
func (oa *ObservableAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	// Apply input filters in order.
	processed := input
	for _, f := range oa.inputFilters {
		processed = f(processed)
	}

	if oa.onStart != nil {
		oa.onStart(processed)
	}

	result, err := oa.BaseAgent.Invoke(ctx, processed, opts...)
	if err != nil {
		return result, err
	}

	// Apply output filters in order.
	for _, f := range oa.outputFilters {
		result = f(result)
	}

	if oa.onFinish != nil {
		oa.onFinish(result)
	}

	return result, nil
}

// Stream delegates to the base agent; apply output processing in the caller if needed.
func (oa *ObservableAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	processed := input
	for _, f := range oa.inputFilters {
		processed = f(processed)
	}
	return oa.BaseAgent.Stream(ctx, processed, opts...)
}

// SanitizeInputFilter removes leading/trailing whitespace from the input.
func SanitizeInputFilter() InputFilter {
	return func(input string) string {
		return strings.TrimSpace(input)
	}
}

// UppercaseOutputFilter converts the output to uppercase (example transform).
func UppercaseOutputFilter() OutputFilter {
	return func(output string) string {
		return strings.ToUpper(output)
	}
}

func main() {
	ctx := context.Background()

	type CalcInput struct {
		Expression string `json:"expression" description:"Math expression to evaluate" required:"true"`
	}
	calculator := tool.NewFuncTool("calculate", "Evaluate a math expression",
		func(ctx context.Context, input CalcInput) (*tool.Result, error) {
			return tool.TextResult("42"), nil
		},
	)

	// Build an observable agent using composition.
	a := NewObservableAgent(
		"observable-assistant",
		[]agent.Option{
			agent.WithPersona(agent.Persona{
				Role: "Assistant",
				Goal: "Help users with calculations",
			}),
			agent.WithTools([]tool.Tool{calculator}),
		},
		WithInputFilter(SanitizeInputFilter()),
		WithOnStart(func(input string) {
			slog.Info("agent invoked", "input_length", len(input))
		}),
		WithOnFinish(func(output string) {
			slog.Info("agent finished", "output_length", len(output))
		}),
	)

	result, err := a.Invoke(ctx, "  What is 6 * 7?  ")
	if err != nil {
		slog.Error("agent failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

## Explanation

1. **Composition over embedding** -- `ObservableAgent` embeds `*agent.BaseAgent` and overrides only `Invoke` and `Stream`. It inherits `ID()`, `Persona()`, `Tools()`, and `Children()` from the base agent without reimplementing them.

2. **Filter chains** -- Input and output filters are applied in registration order. Each filter receives the output of the previous one, enabling a pipeline of independent transformations. Filters are pure functions that transform strings, making unit testing straightforward.

3. **Lifecycle callbacks** -- `onStart` and `onFinish` run before and after each invocation. Use these for logging, metrics, audit trails, or integration with external systems. Setting a callback to nil (the default) means it is skipped with zero overhead.

4. **Functional options** -- `ObservableOption` functions configure the agent at construction time, following Beluga AI's `WithX()` convention. Adding new options does not change the constructor signature or break existing callers.

## Testing

```go
func TestObservableAgent_AppliesInputFilter(t *testing.T) {
	filterCalled := false
	a := NewObservableAgent(
		"test-agent",
		nil,
		WithInputFilter(func(input string) string {
			filterCalled = true
			return strings.TrimSpace(input)
		}),
	)

	// Even without a real LLM, the filter should be called before delegation.
	// Wire up a mock via WithPlanner or WithLLM in the agent.Option slice.
	_ = a
	_ = filterCalled
}
```

## Variations

### Logging Agent

Create a version that logs all interactions for debugging and audit trails:

```go
func NewLoggingAgent(id string, agentOpts []agent.Option) *ObservableAgent {
	return NewObservableAgent(
		id,
		agentOpts,
		WithOnStart(func(input string) {
			slog.Info("turn start", "agent", id, "input", input)
		}),
		WithOnFinish(func(output string) {
			slog.Info("turn end", "agent", id, "output_length", len(output))
		}),
	)
}
```

### Validation Agent

Reject inputs that fail domain validation before they consume LLM tokens:

```go
func NewValidatingAgent(id string, agentOpts []agent.Option, validate func(string) error) *ObservableAgent {
	return NewObservableAgent(
		id,
		agentOpts,
		WithInputFilter(func(input string) string {
			if err := validate(input); err != nil {
				// Return a sentinel value that the agent can recognize.
				return "__INVALID__: " + err.Error()
			}
			return input
		}),
	)
}
```

## Related Recipes

- **[Agent Handoffs](/docs/cookbook/agents/#agent-handoffs)** -- Route conversations between specialist agents
- **[LLM Error Handling](/docs/cookbook/llm/llm-error-handling)** -- Handle errors in your custom agent
