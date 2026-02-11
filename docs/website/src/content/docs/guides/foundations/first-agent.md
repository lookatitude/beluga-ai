---
title: Building Your First Agent
description: Create a complete AI agent from scratch with tools, streaming, and reasoning.
---

Build a fully functional AI agent that can reason about tasks, use tools, and stream responses. This guide covers the `agent` package from first principles, walking through the design decisions that make Beluga's agent system composable and production-ready.

## Core Concepts

An agent in Beluga combines three components into a reasoning loop. The LLM provides the intelligence, tools give the agent the ability to take actions in the world, and the persona shapes how the agent communicates and reasons. This separation of concerns means you can swap any component independently — use a different model, add new tools, or adjust the persona — without changing the rest of the agent's configuration.

| Component | Purpose |
|-----------|---------|
| **ChatModel** | The LLM that powers reasoning |
| **Tools** | Functions the agent can call |
| **Persona** | Identity that shapes behavior |

The agent uses a **Planner** (default: ReAct) to decide when to call tools versus respond to the user, and an **Executor** that runs the reasoning loop. This design decouples the reasoning strategy from the execution mechanics, allowing you to switch between planners like ReAct, Reflexion, or Tree of Thoughts without changing how tools are invoked or results are collected.

## Creating a Basic Agent

The following example creates the simplest possible agent: an LLM wrapped in an agent runtime with a persona. The blank import of the OpenAI provider package triggers its `init()` function, which registers the provider with the LLM registry. This is Beluga's registry pattern — providers self-register on import, and `llm.New()` looks them up by name. The functional options pattern (`agent.WithLLM`, `agent.WithPersona`) keeps the constructor clean while allowing any combination of configuration.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/llm"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
	ctx := context.Background()

	// Create the LLM
	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create the agent
	a := agent.New("assistant",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role: "helpful assistant",
			Goal: "answer questions accurately and concisely",
		}),
	)

	// Invoke synchronously
	result, err := a.Invoke(ctx, "What is the capital of France?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
```

## The Agent Interface

Every agent implements a common interface that provides both synchronous and streaming execution paths. `Invoke` returns the final text result, making it suitable for simple request-response patterns. `Stream` returns an `iter.Seq2[Event, error]` — Beluga's standard streaming type — giving you real-time visibility into the agent's reasoning process as it interleaves tool calls and text generation.

The interface also exposes the agent's identity (`ID`, `Persona`), its capabilities (`Tools`), and its position in a hierarchy (`Children`). This uniform structure enables introspection and orchestration — a parent agent can enumerate a child's tools, and monitoring systems can log the agent's persona without special-casing.

```go
type Agent interface {
	ID() string
	Persona() Persona
	Tools() []tool.Tool
	Children() []Agent
	Invoke(ctx context.Context, input string, opts ...Option) (string, error)
	Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error]
}
```

## Defining a Persona

The Persona uses a **Role-Goal-Backstory** framework derived from research on prompt-based agent specialization. Role defines what the agent is, Goal defines what it optimizes for, and Backstory provides context that shapes tone and decision-making. Traits add fine-grained behavioral modifiers. The framework converts automatically to a system message — empty fields are omitted, so you only specify what matters for your use case.

```go
persona := agent.Persona{
	Role:      "senior data analyst",
	Goal:      "provide accurate data insights with clear visualizations",
	Backstory: "You have 10 years of experience with financial data analysis.",
	Traits:    []string{"precise", "methodical", "detail-oriented"},
}
```

## Adding Tools

Tools give your agent the ability to interact with the outside world — querying APIs, reading databases, performing calculations, or any other side effect. `FuncTool` wraps any Go function into a tool that the agent can call. The function's input struct defines the tool's parameter schema: struct tags specify JSON field names, descriptions, and validation constraints. Beluga generates the JSON Schema automatically from these tags, so the LLM sees a properly documented schema and can decide when and how to invoke the tool.

```go
import "github.com/lookatitude/beluga-ai/tool"

type WeatherInput struct {
	City string `json:"city" description:"City name" required:"true"`
}

weather := tool.NewFuncTool("get_weather", "Get current weather for a city",
	func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
		// Call your weather API here
		return tool.TextResult(fmt.Sprintf("Weather in %s: 22°C, sunny", input.City)), nil
	},
)

a := agent.New("weather-bot",
	agent.WithLLM(model),
	agent.WithTools([]tool.Tool{weather}),
	agent.WithPersona(agent.Persona{
		Role: "weather assistant",
		Goal: "provide accurate weather information",
	}),
)
```

## Streaming Responses

Streaming lets you observe the agent's reasoning process as it happens, rather than waiting for the final result. This is critical for user-facing applications where perceived latency matters — you can display text chunks as they arrive and show tool invocation status in real time. Beluga uses `iter.Seq2[Event, error]` for streaming instead of channels. This choice avoids the complexity of channel lifecycle management (who closes the channel, buffering, goroutine leaks) and integrates naturally with Go's `for range` syntax.

```go
for event, err := range a.Stream(ctx, "What's the weather in Tokyo?") {
	if err != nil {
		log.Printf("error: %v", err)
		break
	}

	switch event.Type {
	case agent.EventText:
		fmt.Print(event.Text) // Print response chunks as they arrive
	case agent.EventToolCall:
		fmt.Printf("\n[Calling tool: %s]\n", event.ToolCall.Name)
	case agent.EventToolResult:
		fmt.Printf("[Tool result received]\n")
	case agent.EventDone:
		fmt.Println("\n[Done]")
	}
}
```

### Event Types

Each event type corresponds to a distinct phase in the agent's execution. Text events carry incremental response content. Tool events mark the boundary between reasoning and action. The Done event signals that the reasoning loop has terminated, either because the LLM produced a final answer or because the maximum iteration count was reached.

| Event | Description |
|-------|-------------|
| `EventText` | A text chunk from the agent's response |
| `EventToolCall` | The agent is requesting a tool invocation |
| `EventToolResult` | The result of a tool invocation |
| `EventHandoff` | An agent-to-agent transfer |
| `EventDone` | The agent has finished execution |
| `EventError` | An error occurred during execution |

## The ReAct Reasoning Loop

By default, agents use the **ReAct** (Reasoning + Acting) planner, which interleaves reasoning steps with tool use in a loop. The approach mirrors how humans solve problems: think about what information is needed, take an action to get it, then think again with the new information. This loop continues until the LLM decides it has enough information to produce a final answer, or until the maximum iteration count is reached.

1. Send conversation to the LLM (with tool definitions)
2. If the LLM returns tool calls → execute them, add results to conversation, go to step 1
3. If the LLM returns text with no tool calls → return the text as the final answer

```
User Input → [LLM] → Tool Calls? → Yes → Execute Tools → [LLM] → ...
                                  → No  → Final Answer
```

The maximum iteration count and timeout prevent runaway loops — for example, an LLM that keeps calling tools without converging on an answer. Set these based on the complexity of your task: simple Q&A may need only 3-5 iterations, while multi-step research tasks may need 15-20.

```go
a := agent.New("researcher",
	agent.WithLLM(model),
	agent.WithTools(tools),
	agent.WithMaxIterations(20),          // Default: 10
	agent.WithTimeout(10 * time.Minute),  // Default: 5 minutes
)
```

## Alternative Reasoning Strategies

Not all tasks are best served by the ReAct loop. Beluga includes several planner implementations that use different reasoning strategies. Each planner follows the same `Planner` interface, so switching strategies requires only changing the configuration — the rest of your agent code stays the same. Choose a planner based on the nature of your task: ReAct for general-purpose use, Reflexion when output quality is critical, and Tree of Thoughts or Graph of Thoughts for problems that benefit from exploring multiple solution paths.

```go
// Use Reflexion (self-critique and retry)
a := agent.New("writer",
	agent.WithLLM(model),
	agent.WithPlannerName("reflexion"),
)

// Use Tree of Thoughts (explore multiple reasoning paths)
a := agent.New("solver",
	agent.WithLLM(model),
	agent.WithPlannerName("tot"),
)
```

Available planners:

| Planner | Strategy | Best For |
|---------|----------|----------|
| `react` | Reasoning + Acting (default) | General-purpose tasks |
| `reflexion` | Self-critique and retry | Quality-sensitive outputs |
| `tot` | Tree of Thoughts | Complex problem solving |
| `got` | Graph of Thoughts | Non-linear reasoning |
| `selfdiscover` | Self-Discover | Novel task structures |
| `lats` | Language Agent Tree Search | Search-based planning |
| `moa` | Mixture of Agents | Multi-model consensus |

## Agent Handoffs

In many applications, a single agent cannot cover all domains effectively. Handoffs solve this by letting agents transfer conversations to specialists. A triage agent can route requests to a coder, writer, or analyst based on the task. Beluga implements handoffs as tools — each handoff target generates a `transfer_to_{agent_id}` tool. This means the LLM uses the same mechanism it already knows (tool calling) to perform transfers, without requiring any special protocol.

```go
// Create specialist agents
coder := agent.New("coder",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "software engineer"}),
)

writer := agent.New("writer",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "technical writer"}),
)

// Create a triage agent that can hand off to specialists
triage := agent.New("triage",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{
		Role: "triage agent",
		Goal: "route requests to the right specialist",
	}),
	agent.WithHandoffs([]agent.Handoff{
		agent.HandoffTo(coder, "Hand off coding tasks to the software engineer"),
		agent.HandoffTo(writer, "Hand off writing tasks to the technical writer"),
	}),
)
```

## Lifecycle Hooks

Hooks provide observation and control points at each phase of agent execution without requiring you to implement a full custom planner or middleware layer. All hook fields are optional — set only the ones you need, and nil hooks are skipped with zero overhead. The `OnError` hook deserves special attention: returning `nil` from it suppresses the error, while returning the error (or a different one) lets it propagate. This gives you fine-grained control over error handling and recovery.

```go
a := agent.New("monitored",
	agent.WithLLM(model),
	agent.WithHooks(agent.Hooks{
		OnStart: func(ctx context.Context, input any) error {
			log.Printf("Agent started with: %v", input)
			return nil
		},
		OnEnd: func(ctx context.Context, result any, err error) {
			log.Printf("Agent finished: result=%v, err=%v", result, err)
		},
		OnToolCall: func(ctx context.Context, call any) error {
			log.Printf("Tool call: %v", call)
			return nil
		},
		OnError: func(ctx context.Context, err error) error {
			log.Printf("Error: %v", err)
			return err // Return nil to suppress the error
		},
	}),
)
```

## Configuration Options

The full set of functional options for `agent.New`. Each option targets a specific aspect of agent behavior — LLM selection, tool binding, persona definition, loop constraints, observability, and multi-agent coordination. Options compose freely: you can combine any subset without conflicts.

| Option | Default | Description |
|--------|---------|-------------|
| `WithLLM(model)` | required | The language model to use |
| `WithTools(tools)` | none | Available tools |
| `WithPersona(p)` | empty | Agent identity (Role/Goal/Backstory) |
| `WithMaxIterations(n)` | 10 | Max reasoning loop iterations |
| `WithTimeout(d)` | 5 min | Max execution duration |
| `WithHooks(h)` | none | Lifecycle callbacks |
| `WithHandoffs(h)` | none | Agent transfer targets |
| `WithMemory(m)` | none | Conversation persistence |
| `WithPlanner(p)` | ReAct | Reasoning strategy (direct) |
| `WithPlannerName(name)` | "react" | Reasoning strategy (by name) |
| `WithChildren(agents)` | none | Child agents for orchestration |

## Complete Example: Research Agent

This example brings everything together: a research agent with two tools, a detailed persona, streaming output, and a higher iteration limit to allow for multi-step research. The agent uses the default ReAct planner to alternate between searching for information and summarizing findings until it has enough material to produce a final answer.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/tool"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

type SearchInput struct {
	Query string `json:"query" description:"Search query" required:"true"`
}

type SummarizeInput struct {
	Text string `json:"text" description:"Text to summarize" required:"true"`
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	search := tool.NewFuncTool("web_search", "Search the web for information",
		func(ctx context.Context, input SearchInput) (*tool.Result, error) {
			// Integrate with your search API
			return tool.TextResult("Search results for: " + input.Query), nil
		},
	)

	summarize := tool.NewFuncTool("summarize", "Summarize a piece of text",
		func(ctx context.Context, input SummarizeInput) (*tool.Result, error) {
			return tool.TextResult("Summary: " + input.Text[:50] + "..."), nil
		},
	)

	researcher := agent.New("researcher",
		agent.WithLLM(model),
		agent.WithTools([]tool.Tool{search, summarize}),
		agent.WithPersona(agent.Persona{
			Role:      "research analyst",
			Goal:      "gather and synthesize information from multiple sources",
			Backstory: "You are thorough and always verify information from multiple sources.",
			Traits:    []string{"analytical", "thorough", "objective"},
		}),
		agent.WithMaxIterations(15),
	)

	// Stream the research process
	for event, err := range researcher.Stream(ctx, "Research the latest advances in quantum computing") {
		if err != nil {
			log.Printf("error: %v", err)
			break
		}

		switch event.Type {
		case agent.EventText:
			fmt.Print(event.Text)
		case agent.EventToolCall:
			fmt.Printf("\n[Researching: %s]\n", event.ToolCall.Name)
		case agent.EventDone:
			fmt.Println()
		}
	}
}
```

## Next Steps

- [Working with LLMs](/guides/foundations/working-with-llms/) — Deep dive into the ChatModel interface
- [Tools & MCP](/guides/tools-and-mcp/) — Advanced tool patterns and MCP integration
- [Orchestration & Workflows](/guides/orchestration/) — Multi-agent coordination
- [Memory System](/guides/memory-system/) — Persistent agent memory
