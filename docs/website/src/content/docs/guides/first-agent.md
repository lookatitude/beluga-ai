---
title: Building Your First Agent
description: Create a complete AI agent from scratch with tools, streaming, and reasoning.
---

Build a fully functional AI agent that can reason about tasks, use tools, and stream responses. This guide covers the `agent` package from first principles.

## Core Concepts

An agent in Beluga combines three things:

| Component | Purpose |
|-----------|---------|
| **ChatModel** | The LLM that powers reasoning |
| **Tools** | Functions the agent can call |
| **Persona** | Identity that shapes behavior |

The agent uses a **Planner** (default: ReAct) to decide when to call tools versus respond to the user, and an **Executor** that runs the reasoning loop.

## Creating a Basic Agent

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

Every agent implements this interface:

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

`Invoke` returns the final text result. `Stream` returns an iterator of events, giving you real-time visibility into the agent's reasoning process.

## Defining a Persona

The Persona uses a **Role-Goal-Backstory** framework to shape the agent's behavior:

```go
persona := agent.Persona{
	Role:      "senior data analyst",
	Goal:      "provide accurate data insights with clear visualizations",
	Backstory: "You have 10 years of experience with financial data analysis.",
	Traits:    []string{"precise", "methodical", "detail-oriented"},
}
```

This converts to a system message automatically. Empty fields are omitted.

## Adding Tools

Tools give your agent the ability to interact with the outside world. Use `FuncTool` to wrap any Go function:

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

The JSON Schema for the tool input is generated automatically from struct tags. The LLM sees the tool name, description, and schema, and decides when to call it.

## Streaming Responses

Stream events to see the agent's reasoning in real-time:

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

| Event | Description |
|-------|-------------|
| `EventText` | A text chunk from the agent's response |
| `EventToolCall` | The agent is requesting a tool invocation |
| `EventToolResult` | The result of a tool invocation |
| `EventHandoff` | An agent-to-agent transfer |
| `EventDone` | The agent has finished execution |
| `EventError` | An error occurred during execution |

## The ReAct Reasoning Loop

By default, agents use the **ReAct** (Reasoning + Acting) planner. The loop works as follows:

1. Send conversation to the LLM (with tool definitions)
2. If the LLM returns tool calls → execute them, add results to conversation, go to step 1
3. If the LLM returns text with no tool calls → return the text as the final answer

```
User Input → [LLM] → Tool Calls? → Yes → Execute Tools → [LLM] → ...
                                  → No  → Final Answer
```

The maximum number of iterations prevents infinite loops:

```go
a := agent.New("researcher",
	agent.WithLLM(model),
	agent.WithTools(tools),
	agent.WithMaxIterations(20),          // Default: 10
	agent.WithTimeout(10 * time.Minute),  // Default: 5 minutes
)
```

## Alternative Reasoning Strategies

Beluga includes several planner implementations beyond ReAct:

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

Create multi-agent systems where agents transfer conversations to specialists:

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

Handoffs are automatically converted to tools named `transfer_to_{agent_id}`. When the LLM calls one, the target agent is invoked with the provided message.

## Lifecycle Hooks

Monitor and control agent execution with hooks:

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

- [Working with LLMs](/guides/working-with-llms) — Deep dive into the ChatModel interface
- [Tools & MCP](/guides/tools-and-mcp) — Advanced tool patterns and MCP integration
- [Orchestration & Workflows](/guides/orchestration) — Multi-agent coordination
- [Memory System](/guides/memory-system) — Persistent agent memory
