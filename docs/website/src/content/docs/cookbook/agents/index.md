---
title: Agent Recipes
description: Practical recipes for building, customizing, and orchestrating agents.
sidebar:
  order: 0
---

## Handle Tool Failures Gracefully

**Problem:** An agent calls a tool that fails or returns an error, and you need to handle this without crashing the agent loop.

**Solution:** Use tool middleware with retry logic and hooks to intercept errors before they propagate.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
)

// Wrap tools with error-handling middleware before passing to the agent.
func main() {
	ctx := context.Background()

	// Define a tool that may fail.
	type SearchInput struct {
		Query string `json:"query" description:"Search query" required:"true"`
	}
	search := tool.NewFuncTool("search", "Search the web",
		func(ctx context.Context, input SearchInput) (*tool.Result, error) {
			// This might fail due to network issues.
			return tool.TextResult("results for: " + input.Query), nil
		},
	)

	// Apply retry middleware so transient failures are retried automatically.
	resilientSearch := tool.ApplyMiddleware(search, tool.WithRetry(3))

	// Add hooks to log failures without blocking execution.
	hookedSearch := tool.WithHooks(resilientSearch, tool.Hooks{
		OnError: func(ctx context.Context, name string, err error) error {
			slog.Error("tool failed", "tool", name, "error", err)
			if core.IsRetryable(err) {
				return err // Let retry middleware handle it.
			}
			// For non-retryable errors, return a user-friendly message.
			return fmt.Errorf("the %s tool is temporarily unavailable", name)
		},
	})

	a := agent.New("assistant",
		agent.WithTools([]tool.Tool{hookedSearch}),
	)

	result, err := a.Invoke(ctx, "Search for Go concurrency patterns")
	if err != nil {
		slog.Error("agent failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

**Key decisions:**
- `tool.WithRetry(3)` retries up to 3 times with exponential backoff for retryable errors.
- `OnError` hooks let you log, transform, or suppress errors before they reach the agent.
- Non-retryable errors get a user-friendly message instead of raw stack traces.

---

## Prevent Tool Hallucinations

**Problem:** The LLM generates tool calls with invalid arguments or calls nonexistent tools, causing runtime errors.

**Solution:** Validate tool calls before execution using agent hooks and input schema validation.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// validateToolCall checks that tool arguments match the expected schema.
func validateToolCall(registry *tool.Registry, call schema.ToolCall) error {
	t, ok := registry.Get(call.Name)
	if !ok {
		return fmt.Errorf("unknown tool: %q", call.Name)
	}

	// Marshal the arguments and check against the input schema.
	inputSchema := t.InputSchema()
	requiredFields, _ := inputSchema["required"].([]any)
	properties, _ := inputSchema["properties"].(map[string]any)

	for _, req := range requiredFields {
		fieldName, _ := req.(string)
		if _, ok := call.Arguments[fieldName]; !ok {
			return fmt.Errorf("tool %q missing required field %q", call.Name, fieldName)
		}
	}

	// Check for unexpected fields.
	for key := range call.Arguments {
		if _, ok := properties[key]; !ok {
			slog.Warn("unexpected tool argument", "tool", call.Name, "field", key)
		}
	}

	return nil
}

func main() {
	ctx := context.Background()

	reg := tool.NewRegistry()

	type WeatherInput struct {
		City string `json:"city" description:"City name" required:"true"`
	}
	weather := tool.NewFuncTool("get_weather", "Get current weather",
		func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
			return tool.TextResult(fmt.Sprintf("Weather in %s: sunny, 22°C", input.City)), nil
		},
	)
	reg.Add(weather)

	// Validate tool calls via hooks.
	a := agent.New("assistant",
		agent.WithTools(reg.All()),
		agent.WithHooks(agent.Hooks{
			OnToolCall: func(ctx context.Context, call schema.ToolCall) error {
				if err := validateToolCall(reg, call); err != nil {
					slog.Error("invalid tool call", "error", err)
					return err // Agent will see the error and retry with corrected args.
				}
				return nil
			},
		}),
	)

	result, err := a.Invoke(ctx, "What's the weather in Tokyo?")
	if err != nil {
		slog.Error("agent failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

---

## Execute Agent Steps in Parallel

**Problem:** You have independent subtasks that can run concurrently to reduce total execution time.

**Solution:** Use the `workflow.ParallelAgent` to fan out work to multiple agents simultaneously.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/agent/workflow"
)

func main() {
	ctx := context.Background()

	// Create specialist agents for independent tasks.
	researcher := agent.New("researcher",
		agent.WithPersona(agent.Persona{
			Role: "Research Analyst",
			Goal: "Find relevant information about the topic",
		}),
	)

	writer := agent.New("writer",
		agent.WithPersona(agent.Persona{
			Role: "Content Writer",
			Goal: "Draft compelling content about the topic",
		}),
	)

	factChecker := agent.New("fact-checker",
		agent.WithPersona(agent.Persona{
			Role: "Fact Checker",
			Goal: "Verify claims and identify inaccuracies",
		}),
	)

	// ParallelAgent runs all children concurrently and collects results.
	parallel := workflow.NewParallelAgent("parallel-team",
		researcher, writer, factChecker,
	)

	result, err := parallel.Invoke(ctx, "Write about quantum computing advances in 2025")
	if err != nil {
		slog.Error("parallel execution failed", "error", err)
		return
	}

	fmt.Println(result)
}
```

**Key decisions:**
- Each child agent receives the same input and runs independently.
- Results are aggregated after all agents complete (or the first error is returned).
- Use `workflow.SequentialAgent` instead when steps depend on each other.

---

## Build a Custom Agent

**Problem:** The `BaseAgent` doesn't fit your use case, and you need specialized behavior such as custom reasoning loops or domain-specific logic.

**Solution:** Implement the `Agent` interface directly, embedding `BaseAgent` for shared functionality.

```go
package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// ReviewAgent performs code review with a multi-pass approach.
type ReviewAgent struct {
	*agent.BaseAgent
	maxPasses int
}

// NewReviewAgent creates a code review agent.
func NewReviewAgent(model interface{}, tools []tool.Tool) *ReviewAgent {
	base := agent.New("code-reviewer",
		agent.WithPersona(agent.Persona{
			Role:      "Senior Code Reviewer",
			Goal:      "Find bugs, security issues, and style problems",
			Backstory: "You are a meticulous engineer with 20 years of Go experience.",
		}),
		agent.WithTools(tools),
	)

	return &ReviewAgent{
		BaseAgent: base,
		maxPasses: 3,
	}
}

// Invoke performs multi-pass code review.
func (r *ReviewAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	var allFindings string

	for pass := 1; pass <= r.maxPasses; pass++ {
		prompt := fmt.Sprintf(
			"Pass %d of %d. Review this code for %s:\n\n%s\n\nPrevious findings:\n%s",
			pass, r.maxPasses, r.passFocus(pass), input, allFindings,
		)

		result, err := r.BaseAgent.Invoke(ctx, prompt, opts...)
		if err != nil {
			return "", fmt.Errorf("review pass %d failed: %w", pass, err)
		}

		allFindings += fmt.Sprintf("\n## Pass %d: %s\n%s", pass, r.passFocus(pass), result)
	}

	return allFindings, nil
}

// Stream delegates to the base agent's stream for the final pass.
func (r *ReviewAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return r.BaseAgent.Stream(ctx, input, opts...)
}

func (r *ReviewAgent) passFocus(pass int) string {
	switch pass {
	case 1:
		return "correctness and bugs"
	case 2:
		return "security vulnerabilities"
	case 3:
		return "style and performance"
	default:
		return "general review"
	}
}

func main() {
	ctx := context.Background()
	reviewer := NewReviewAgent(nil, nil)

	result, err := reviewer.Invoke(ctx, `func process(data []byte) {
		var result map[string]any
		json.Unmarshal(data, &result)
		fmt.Println(result["key"])
	}`)
	if err != nil {
		slog.Error("review failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

**Key decisions:**
- Embed `*agent.BaseAgent` to inherit `ID()`, `Persona()`, `Tools()`, and `Children()`.
- Override `Invoke` and `Stream` for custom behavior.
- The multi-pass pattern is a common way to get thorough results from a single agent.

---

## Agent Handoffs

**Problem:** You need specialized agents that can transfer control to each other based on the conversation topic.

**Solution:** Use handoffs-as-tools. Beluga automatically generates `transfer_to_{id}` tools from handoff declarations.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
)

func main() {
	ctx := context.Background()

	// Create specialist agents.
	billing := agent.New("billing-agent",
		agent.WithPersona(agent.Persona{
			Role: "Billing Specialist",
			Goal: "Handle billing inquiries, refunds, and payment issues",
		}),
	)

	technical := agent.New("technical-agent",
		agent.WithPersona(agent.Persona{
			Role: "Technical Support",
			Goal: "Resolve technical issues and troubleshoot problems",
		}),
	)

	// Create a triage agent that can hand off to specialists.
	triage := agent.New("triage-agent",
		agent.WithPersona(agent.Persona{
			Role: "Customer Support Triage",
			Goal: "Classify the customer issue and route to the right specialist",
		}),
		agent.WithHandoffs([]agent.Handoff{
			agent.HandoffTo(billing, "Transfer to billing for payment and refund issues"),
			agent.HandoffTo(technical, "Transfer to technical support for product issues"),
		}),
	)

	// The triage agent's LLM will see transfer_to_billing-agent and
	// transfer_to_technical-agent as available tools.
	result, err := triage.Invoke(ctx, "I was charged twice for my subscription")
	if err != nil {
		slog.Error("triage failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

**Advanced handoffs with filters and callbacks:**

```go
// Handoff with input transformation and lifecycle hooks.
handoff := agent.Handoff{
	TargetAgent: billing,
	Description: "Transfer to billing for payment issues",
	InputFilter: func(input agent.HandoffInput) agent.HandoffInput {
		// Add context from the triage conversation.
		input.Context["priority"] = "high"
		input.Context["category"] = "billing"
		return input
	},
	OnHandoff: func(ctx context.Context) error {
		slog.Info("handoff triggered", "target", "billing")
		return nil
	},
	IsEnabled: func(ctx context.Context) bool {
		// Disable during maintenance windows.
		return true
	},
}
```

---

## Stream Agent Responses to a UI

**Problem:** You need to display agent responses in real-time as they are generated, including tool calls and handoff events.

**Solution:** Use `agent.Stream()` and handle each `EventType` appropriately.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
)

func main() {
	ctx := context.Background()

	a := agent.New("assistant",
		agent.WithPersona(agent.Persona{
			Role: "Helpful Assistant",
			Goal: "Answer questions clearly and concisely",
		}),
	)

	// Stream returns iter.Seq2[agent.Event, error].
	for event, err := range a.Stream(ctx, "Explain Go generics in 3 sentences") {
		if err != nil {
			slog.Error("stream error", "error", err)
			break
		}

		switch event.Type {
		case agent.EventText:
			// Print text chunks as they arrive (no newline for streaming).
			fmt.Print(event.Text)

		case agent.EventToolCall:
			// Display tool invocation.
			args, _ := json.Marshal(event.ToolCall.Arguments)
			fmt.Printf("\n[calling tool: %s(%s)]\n", event.ToolCall.Name, args)

		case agent.EventToolResult:
			// Display tool result.
			fmt.Printf("[tool result: %s]\n", event.Text)

		case agent.EventHandoff:
			// Display handoff notification.
			fmt.Printf("\n[transferring to: %s]\n", event.Text)

		case agent.EventDone:
			fmt.Println("\n--- done ---")

		case agent.EventError:
			fmt.Printf("\n[error: %s]\n", event.Text)
		}
	}
}
```

---

## Compose Agents with Sequential Workflow

**Problem:** You need agents to execute in a defined order, each receiving the output of the previous agent.

**Solution:** Use `workflow.SequentialAgent` to chain agents into a pipeline.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/agent/workflow"
)

func main() {
	ctx := context.Background()

	// Step 1: Research the topic.
	researcher := agent.New("researcher",
		agent.WithPersona(agent.Persona{
			Role: "Researcher",
			Goal: "Gather key facts and data about the topic",
		}),
	)

	// Step 2: Draft content from research.
	writer := agent.New("writer",
		agent.WithPersona(agent.Persona{
			Role: "Writer",
			Goal: "Write a clear, engaging article from the research provided",
		}),
	)

	// Step 3: Review and polish.
	editor := agent.New("editor",
		agent.WithPersona(agent.Persona{
			Role: "Editor",
			Goal: "Improve clarity, fix grammar, and ensure accuracy",
		}),
	)

	// Chain them: researcher → writer → editor.
	pipeline := workflow.NewSequentialAgent("content-pipeline",
		researcher, writer, editor,
	)

	result, err := pipeline.Invoke(ctx, "The impact of large language models on software engineering")
	if err != nil {
		slog.Error("pipeline failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

---

## Loop Agent for Iterative Refinement

**Problem:** You need an agent to repeatedly refine its output until a quality threshold is met.

**Solution:** Use `workflow.LoopAgent` with a termination condition.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/agent/workflow"
)

func main() {
	ctx := context.Background()

	improver := agent.New("improver",
		agent.WithPersona(agent.Persona{
			Role: "Content Improver",
			Goal: "Improve the text based on the critique provided",
		}),
	)

	// Loop until the output contains "APPROVED" or max 5 iterations.
	loop := workflow.NewLoopAgent("refine-loop",
		improver,
		workflow.WithMaxIterations(5),
		workflow.WithTermination(func(output string) bool {
			return strings.Contains(output, "APPROVED")
		}),
	)

	result, err := loop.Invoke(ctx, "Draft: Go is a programming language.")
	if err != nil {
		slog.Error("loop failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

---

## Agent with Memory

**Problem:** You want an agent that remembers previous conversations and uses that context in future interactions.

**Solution:** Attach a memory instance to the agent and use the `WithMemory` option.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/memory"
	_ "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
)

func main() {
	ctx := context.Background()

	// Create an in-memory store for development.
	mem, err := memory.New("inmemory", config.ProviderConfig{})
	if err != nil {
		slog.Error("memory creation failed", "error", err)
		return
	}

	a := agent.New("assistant",
		agent.WithPersona(agent.Persona{
			Role: "Personal Assistant",
			Goal: "Remember user preferences and provide personalized help",
		}),
		agent.WithMemory(mem),
	)

	// First conversation — agent learns a preference.
	result, err := a.Invoke(ctx, "My favorite programming language is Go")
	if err != nil {
		slog.Error("first invoke failed", "error", err)
		return
	}
	fmt.Println("Turn 1:", result)

	// Second conversation — agent recalls the preference.
	result, err = a.Invoke(ctx, "What's my favorite language?")
	if err != nil {
		slog.Error("second invoke failed", "error", err)
		return
	}
	fmt.Println("Turn 2:", result)
}
```
