---
title: Orchestration & Workflows
description: Coordinate multi-agent systems with workflow agents, handoffs, DAG execution, and durable workflows.
---

Beluga provides multiple orchestration patterns for coordinating agents — from simple sequential pipelines to durable workflow engines with persistent state.

## Workflow Agents

The `agent/workflow` package provides deterministic orchestration patterns that compose child agents without LLM reasoning:

### Sequential Agent

Run agents in sequence, passing the output of each as input to the next:

```go
import (
	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/agent/workflow"
)

researcher := agent.New("researcher",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "research analyst"}),
	agent.WithTools([]tool.Tool{searchTool}),
)

writer := agent.New("writer",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "technical writer"}),
)

reviewer := agent.New("reviewer",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "editor"}),
)

// Run: researcher → writer → reviewer
pipeline := workflow.NewSequentialAgent("content-pipeline",
	[]agent.Agent{researcher, writer, reviewer},
)

result, err := pipeline.Invoke(ctx, "Write a blog post about Go generics")
```

### Parallel Agent

Run agents concurrently and aggregate results:

```go
sentimentAgent := agent.New("sentiment",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "sentiment analyst"}),
)

topicAgent := agent.New("topics",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "topic classifier"}),
)

entityAgent := agent.New("entities",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "entity extractor"}),
)

// Run all three in parallel
parallel := workflow.NewParallelAgent("text-analysis",
	[]agent.Agent{sentimentAgent, topicAgent, entityAgent},
)

result, err := parallel.Invoke(ctx, "Analyze this customer feedback: ...")
// Result contains aggregated output from all three agents
```

### Loop Agent

Repeat an agent until a condition is met:

```go
improver := agent.New("improver",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{
		Role: "code reviewer and improver",
		Goal: "iteratively improve code quality",
	}),
)

loop := workflow.NewLoopAgent("refinement-loop",
	improver,
	workflow.WithMaxIterations(3),
	workflow.WithStopCondition(func(result string) bool {
		return strings.Contains(result, "APPROVED")
	}),
)

result, err := loop.Invoke(ctx, "Review and improve this code: ...")
```

## Handoffs (Agent Transfers)

Handoffs let the LLM decide when to route to a specialist. They are automatically converted to `transfer_to_{name}` tools:

```go
billingAgent := agent.New("billing",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "billing specialist"}),
)

techAgent := agent.New("tech-support",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{Role: "technical support engineer"}),
)

// Triage agent routes dynamically based on user input
triage := agent.New("triage",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{
		Role: "customer service triage",
		Goal: "route customers to the right specialist",
	}),
	agent.WithHandoffs([]agent.Handoff{
		agent.HandoffTo(billingAgent, "Transfer billing questions to the billing specialist"),
		agent.HandoffTo(techAgent, "Transfer technical issues to tech support"),
	}),
)

// The LLM sees these as tools:
//   transfer_to_billing: "Transfer billing questions..."
//   transfer_to_tech-support: "Transfer technical issues..."
result, err := triage.Invoke(ctx, "I was charged twice on my last invoice")
```

### Conditional Handoffs

Control handoff availability dynamically:

```go
handoff := agent.Handoff{
	TargetAgent: premiumAgent,
	Description: "Transfer to premium support",
	IsEnabled: func(ctx context.Context) bool {
		user := auth.UserFromContext(ctx)
		return user != nil && user.Tier == "premium"
	},
	OnHandoff: func(ctx context.Context) error {
		log.Println("Transferring to premium support")
		return nil
	},
	InputFilter: func(input agent.HandoffInput) agent.HandoffInput {
		// Add context for the target agent
		input.Context = map[string]any{"priority": "high"}
		return input
	},
}
```

## Supervisor Pattern

Use a supervisor agent to coordinate workers:

```go
// Worker agents
worker1 := agent.New("data-collector",
	agent.WithLLM(model),
	agent.WithTools([]tool.Tool{apiTool}),
)

worker2 := agent.New("analyzer",
	agent.WithLLM(model),
	agent.WithTools([]tool.Tool{calculatorTool}),
)

// Supervisor decides how to delegate
supervisor := agent.New("supervisor",
	agent.WithLLM(model),
	agent.WithPersona(agent.Persona{
		Role: "project supervisor",
		Goal: "coordinate workers to complete complex tasks",
	}),
	agent.WithChildren([]agent.Agent{worker1, worker2}),
	agent.WithHandoffs([]agent.Handoff{
		agent.HandoffTo(worker1, "Delegate data collection tasks"),
		agent.HandoffTo(worker2, "Delegate analysis tasks"),
	}),
)
```

## Event Bus

The agent `EventBus` enables decoupled communication between agents:

```go
bus := agent.NewEventBus()

// Subscribe to events
bus.Subscribe("task.complete", func(event agent.BusEvent) {
	log.Printf("Task completed: %s", event.Data)
})

// Publish events from agents
bus.Publish(agent.BusEvent{
	Topic:   "task.complete",
	AgentID: "researcher",
	Data:    "Research phase finished",
})
```

## Durable Workflows

The `workflow` package provides a durable execution engine for long-running, resumable workflows:

```go
import "github.com/lookatitude/beluga-ai/workflow"

// Define workflow steps
wf := workflow.New("document-processing",
	workflow.Step("ingest", func(ctx workflow.Context, input string) (string, error) {
		// Load and validate documents
		return "documents loaded", nil
	}),
	workflow.Step("process", func(ctx workflow.Context, input string) (string, error) {
		// Process with AI
		return "documents processed", nil
	}),
	workflow.Step("publish", func(ctx workflow.Context, input string) (string, error) {
		// Publish results
		return "published", nil
	}),
)

// Run with automatic checkpointing
engine := workflow.NewEngine(workflow.EngineConfig{
	Store: workflowStore, // Persistent state store
})

result, err := engine.Execute(ctx, wf, "start processing")
```

### Durable Execution Features

| Feature | Description |
|---------|-------------|
| **Checkpointing** | State persisted after each step |
| **Resumability** | Resume from last checkpoint after failure |
| **Timeouts** | Per-step and total workflow timeouts |
| **Retries** | Configurable retry with backoff |
| **Signals** | External events trigger workflow transitions |
| **Timers** | Schedule future actions within workflows |

## Streaming Across Orchestration

All workflow agents support streaming:

```go
for event, err := range pipeline.Stream(ctx, "Build a marketing report") {
	if err != nil {
		log.Fatal(err)
		break
	}

	switch event.Type {
	case agent.EventText:
		fmt.Print(event.Text)
	case agent.EventHandoff:
		fmt.Printf("\n[Handed off to: %s]\n", event.AgentID)
	case agent.EventToolCall:
		fmt.Printf("\n[Tool: %s]\n", event.ToolCall.Name)
	}
}
```

## Choosing an Orchestration Pattern

| Pattern | Decision Maker | Best For |
|---------|---------------|----------|
| **Sequential** | Deterministic | Pipelines with ordered steps |
| **Parallel** | Deterministic | Independent tasks, aggregation |
| **Loop** | Condition-based | Iterative refinement |
| **Handoff** | LLM-driven | Dynamic routing to specialists |
| **Supervisor** | LLM-driven | Complex delegation with oversight |
| **Durable Workflow** | Deterministic | Long-running, resumable processes |

## Next Steps

- [Building Your First Agent](/guides/first-agent/) — Single agent fundamentals
- [Tools & MCP](/guides/tools-and-mcp/) — Tool system for agents
- [Safety & Guards](/guides/safety-and-guards/) — Guard multi-agent systems
- [Deploying to Production](/guides/deployment/) — Production orchestration
