---
title: Orchestration & Workflows
description: "Coordinate agents with sequential pipelines, parallel fan-out, LLM-driven handoffs, supervisor delegation, and durable workflow execution with checkpointing."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, orchestration, workflows, sequential agent, parallel, handoffs, durable execution"
---

Beluga provides multiple orchestration patterns for coordinating agents, each designed for a specific type of coordination problem. Deterministic patterns (sequential, parallel, loop) provide predictable execution when you know the steps in advance. LLM-driven patterns (handoffs, supervisor) enable dynamic routing when the execution path depends on the input. Durable workflows add persistence and resumability for long-running processes that must survive restarts.

Choosing the right pattern depends on two questions: Does the execution path change based on input? And does the workflow need to survive process restarts? The decision table at the end of this guide maps common scenarios to recommended patterns.

## Workflow Agents

The `agent/workflow` package provides deterministic orchestration patterns that compose child agents without LLM reasoning. These patterns execute in a predictable order, making them suitable for pipelines where the steps are known at build time and only the content changes between runs.

### Sequential Agent

Sequential agents run a series of agents in order, passing the output of each as the input to the next. This is the natural choice for pipelines where each step transforms or enriches the result: research feeds into writing, writing feeds into review. The key constraint is that each agent receives only the previous agent's output, not the full history.

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

When subtasks are independent of each other, running them concurrently reduces total latency. The parallel agent fans out the same input to all child agents simultaneously and aggregates their results. This pattern works well for tasks like multi-perspective analysis, where each agent examines the input from a different angle and the combined output is richer than any single agent's response.

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

The loop agent repeatedly invokes the same agent until a stop condition is met or a maximum iteration count is reached. This pattern is essential for iterative refinement tasks like code review, where the agent improves its output over multiple passes. The stop condition function inspects each iteration's output to decide whether the result is satisfactory.

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

Handoffs let the LLM dynamically decide when to route a conversation to a specialist agent. Unlike the deterministic patterns above, handoffs delegate the routing decision to the model, which makes them suitable for scenarios where the right specialist depends on conversational context rather than a predefined rule.

Handoffs are implemented as tools because the LLM already knows how to call tools. Each handoff automatically generates a `transfer_to_{name}` tool that the model can invoke when it determines a specialist would handle the conversation better. This design means no new routing concepts are needed: the model uses the same tool-calling mechanism it already understands.

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

Not all handoffs should be available at all times. Conditional handoffs control availability dynamically based on runtime context, such as user tier, feature flags, or system state. The `IsEnabled` function is evaluated before each tool call, so handoffs can appear and disappear based on the current request context.

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

The supervisor pattern combines LLM-driven planning with structured delegation. A supervisor agent receives a high-level goal, uses LLM reasoning to decompose it into tasks, and delegates each task to a worker agent via handoffs. This pattern is suited for complex goals where the decomposition itself requires intelligence, not just rule-based routing.

Unlike simple handoffs where the model routes to one specialist, the supervisor orchestrates multiple specialists in sequence or parallel, synthesizing their outputs into a coherent result.

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

The agent `EventBus` enables decoupled, asynchronous communication between agents. Instead of direct invocations, agents publish events to named topics and subscribe to the topics they care about. This pattern supports reactive architectures where adding a new agent requires only subscribing to existing topics, with no changes to publishers.

The tradeoff is reduced visibility into the overall workflow: since there is no central orchestrator, you must rely on event tracing and correlation IDs to reconstruct the execution path.

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

The `workflow` package provides Beluga's own durable execution engine for long-running, resumable workflows. The framework owns this engine rather than defaulting to an external system like Temporal, because most agent workflows do not need the complexity of a distributed orchestration platform. For simple to moderate workflows, an embedded engine with persistent checkpointing avoids the operational overhead of deploying and maintaining external infrastructure.

For teams that need Temporal's advanced features (versioning, cross-language workers, large-scale fan-out), Temporal is available as a provider option. The workflow abstraction is the same either way: define steps, connect them, and let the engine handle persistence and resumability.

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

The engine persists workflow state after each step completes. If the process crashes mid-workflow, restarting it resumes from the last successful checkpoint rather than replaying the entire workflow. This is particularly important for AI workflows where individual steps may involve expensive LLM calls that should not be repeated unnecessarily.

| Feature | Description |
|---------|-------------|
| **Checkpointing** | State persisted after each step |
| **Resumability** | Resume from last checkpoint after failure |
| **Timeouts** | Per-step and total workflow timeouts |
| **Retries** | Configurable retry with backoff |
| **Signals** | External events trigger workflow transitions |
| **Timers** | Schedule future actions within workflows |

## Streaming Across Orchestration

All workflow agents support Beluga's `iter.Seq2` streaming pattern. Streaming is particularly valuable in orchestration because it provides real-time visibility into which agent is currently active and what it is producing. Without streaming, the user would see nothing until the entire multi-agent pipeline completes.

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

Use this table to match your coordination requirements to the appropriate pattern. The "Decision Maker" column indicates whether execution order is fixed at build time (deterministic) or determined by the LLM at runtime.

| Pattern | Decision Maker | Best For |
|---------|---------------|----------|
| **Sequential** | Deterministic | Linear pipelines where each step transforms the previous output |
| **Parallel** | Deterministic | Independent subtasks that can run concurrently for lower latency |
| **Loop** | Condition-based | Iterative refinement until a quality threshold is met |
| **Handoff** | LLM-driven | Dynamic routing to specialists based on conversational context |
| **Supervisor** | LLM-driven | Complex goals requiring decomposition, delegation, and synthesis |
| **Durable Workflow** | Deterministic | Long-running processes that must survive restarts with checkpointed state |

## Next Steps

- [Building Your First Agent](/guides/first-agent/) — Single agent fundamentals
- [Tools & MCP](/guides/tools-and-mcp/) — Tool system for agents
- [Safety & Guards](/guides/production/safety-and-guards/) — Guard multi-agent systems
- [Deploying to Production](/guides/production/deployment/) — Production orchestration
