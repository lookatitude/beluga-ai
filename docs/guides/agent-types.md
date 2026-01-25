# Agent Types in Beluga AI

## Introduction

Welcome to this guide on the different agent types available in Beluga AI. By the end, you'll understand how to choose between agent types and implement both ReAct and PlanExecute patterns for your AI applications.

**What you'll learn:**
- The difference between ReAct and PlanExecute agent strategies
- When to use each agent type for optimal results
- How to create and configure each agent type
- Best practices for tool binding and execution

**Why this matters:**
While simple LLM calls work for straightforward questions, complex tasks require agents that can reason, plan, and take multiple actions. Choosing the right agent type for your use case can dramatically improve results and efficiency.

## Prerequisites

Before we begin, make sure you have:

- **Go 1.24+** installed ([installation guide](https://go.dev/doc/install))
- **Beluga AI Framework** installed (`go get github.com/lookatitude/beluga-ai`)
- **OpenAI API key** (or another provider) - you'll need this for LLM access
- **Understanding of basic LLM concepts** - if you're new to this, check out our [LLM concepts guide](../concepts/llms.md)
- **Understanding of tools** - see our [tools tutorial](../getting-started/04-working-with-tools.md)

## Concepts

Before we start coding, let's understand the key concepts:

### The Agent Spectrum

Think of agents on a spectrum from simple to complex:

```
Simple ◄─────────────────────────────────────────────────────────► Complex

┌─────────────┐   ┌─────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Single LLM │   │    ReAct    │   │   PlanExecute   │   │   Multi-Agent   │
│    Call     │   │    Agent    │   │     Agent       │   │   Orchestration │
└─────────────┘   └─────────────┘   └─────────────────┘   └─────────────────┘
     │                  │                   │                     │
     ▼                  ▼                   ▼                     ▼
  Q&A, simple      Multi-step        Complex tasks          Collaborative
  generation       reasoning         with planning          team workflows
```

### ReAct: Reasoning + Acting

ReAct (Reasoning + Acting) agents interleave thinking and doing in a tight loop:

```
┌─────────────────────────────────────────────────────────────────┐
│                        ReAct Loop                                │
│                                                                  │
│   ┌──────────┐    ┌─────────┐    ┌────────────┐                 │
│   │  Think   │───▶│   Act   │───▶│  Observe   │──┐              │
│   └──────────┘    └─────────┘    └────────────┘  │              │
│        ▲                                          │              │
│        └──────────────────────────────────────────┘              │
│                                                                  │
│   Repeat until done or max iterations                           │
└─────────────────────────────────────────────────────────────────┘
```

**Best for:**
- Tasks requiring adaptive reasoning
- When the next step depends on previous results
- Exploratory tasks where the path isn't clear upfront

### PlanExecute: Plan First, Then Execute

PlanExecute agents create a complete plan before executing, then follow it:

```
┌─────────────────────────────────────────────────────────────────┐
│                    PlanExecute Strategy                          │
│                                                                  │
│   Phase 1: Planning                                              │
│   ┌──────────────────────────────────────────────────────┐      │
│   │  ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐   │      │
│   │  │ S1  │──▶│ S2  │──▶│ S3  │──▶│ S4  │──▶│ S5  │   │      │
│   │  └─────┘   └─────┘   └─────┘   └─────┘   └─────┘   │      │
│   └──────────────────────────────────────────────────────┘      │
│                              │                                   │
│                              ▼                                   │
│   Phase 2: Execution                                             │
│   ┌──────────────────────────────────────────────────────┐      │
│   │  Execute S1 → Execute S2 → Execute S3 → Execute S4 → S5    │
│   └──────────────────────────────────────────────────────┘      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Best for:**
- Well-defined multi-step tasks
- When you need predictable execution
- Tasks that benefit from upfront reasoning
- Debugging (easier to inspect the plan)

### Choosing Between ReAct and PlanExecute

| Factor | ReAct | PlanExecute |
|--------|-------|-------------|
| **Task clarity** | Exploratory, unclear path | Well-defined steps |
| **Adaptability** | High - adjusts each step | Lower - follows plan |
| **Predictability** | Less predictable | More predictable |
| **Token efficiency** | More tokens (repeated context) | Fewer tokens (plan once) |
| **Debugging** | Harder (emergent behavior) | Easier (inspect plan) |
| **Use cases** | Research, Q&A | Workflows, automation |

## Step-by-Step Tutorial

Now let's build agents step by step.

### Step 1: Create a ReAct Agent

First, we'll create a ReAct agent that can reason about problems and use tools:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/providers/react"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    ctx := context.Background()

    // Create the LLM client
    llmClient, err := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        log.Fatalf("Failed to create LLM: %v", err)
    }

    // Create tools for the agent
    calculator := tools.NewSimpleTool(
        "calculator",
        "Perform arithmetic calculations. Input should be a math expression.",
        func(ctx context.Context, args map[string]any) (string, error) {
            expression, _ := args["expression"].(string)
            // In production, use a proper math evaluator
            return fmt.Sprintf(`{"expression": "%s", "result": 4}`, expression), nil
        },
        tools.WithParameter("expression", "string", "The math expression", true),
    )

    webSearch := tools.NewSimpleTool(
        "web_search",
        "Search the web for information. Returns relevant search results.",
        func(ctx context.Context, args map[string]any) (string, error) {
            query, _ := args["query"].(string)
            return fmt.Sprintf(`{"query": "%s", "results": ["Result 1", "Result 2"]}`, query), nil
        },
        tools.WithParameter("query", "string", "The search query", true),
    )

    // Create the ReAct agent with a prompt template
    promptTemplate := `You are a helpful research assistant. You can reason about problems
and use tools to find information or perform calculations.

Available tools: {tools}

Question: {input}

Think through this step by step. Use tools when needed. When you have the final answer,
respond with "Final Answer: [your answer]".`

    reactAgent, err := react.NewReActAgent(
        "research-assistant",
        llmClient,
        []tools.Tool{calculator, webSearch},
        promptTemplate,
    )
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    fmt.Printf("Created ReAct agent: %s\n", reactAgent.Name())
}
```

**What you'll see:**
```
Created ReAct agent: research-assistant
```

**Why this works:** The ReAct agent embeds the reasoning loop - it will think about the problem, decide which tool to use (if any), observe the result, and repeat until it has a final answer.

### Step 2: Create a PlanExecute Agent

Now let's create a PlanExecute agent for more structured tasks:

```go
    // Create a PlanExecute agent for structured tasks
    planAgent, err := planexecute.NewPlanExecuteAgent(
        "task-planner",
        llmClient,
        []tools.Tool{calculator, webSearch},
    )
    if err != nil {
        log.Fatalf("Failed to create PlanExecute agent: %v", err)
    }

    // Configure the agent with options
    planAgent = planAgent.
        WithMaxPlanSteps(10).      // Limit plan complexity
        WithMaxIterations(20)       // Limit execution iterations

    fmt.Printf("Created PlanExecute agent: %s\n", planAgent.Name())
```

**What you'll see:**
```
Created PlanExecute agent: task-planner
```

**Why this works:** The PlanExecute agent separates planning from execution. It first creates a detailed plan, then executes each step. This gives you more control and makes debugging easier.

### Step 3: Using Different LLMs for Planning and Execution

A powerful pattern is using different models for planning vs execution:

```go
    // Use a more capable model for planning, faster one for execution
    plannerLLM, _ := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-4"),  // Best reasoning for planning
    )

    executorLLM, _ := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-3.5-turbo"),  // Faster for execution
    )

    planAgent = planAgent.
        WithPlannerLLM(plannerLLM).
        WithExecutorLLM(executorLLM)
```

**Why this works:** Planning benefits from more sophisticated reasoning, while execution often just follows instructions. Using a faster model for execution reduces latency and cost while maintaining quality.

### Step 4: Running an Agent with the Executor

To actually run an agent, use the Executor:

```go
    import "github.com/lookatitude/beluga-ai/pkg/agents/internal/executor"

    // Create an executor to run the agent
    agentExecutor := executor.NewAgentExecutor(
        reactAgent,
        reactAgent.GetTools(),
        executor.WithMaxIterations(10),
        executor.WithHandleParsingErrors(true),
    )

    // Run the agent with input
    result, err := agentExecutor.Run(ctx, map[string]any{
        "input": "What is 2 + 2, and what is the capital of France?",
    })
    if err != nil {
        log.Fatalf("Agent execution failed: %v", err)
    }

    fmt.Printf("Result: %v\n", result)
```

**What you'll see:**
The agent will:
1. Think about the question
2. Use the calculator for "2 + 2"
3. Use web_search for the capital of France
4. Combine the results into a final answer

## Code Examples

Here's a complete, production-ready example showing both agent types:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/pkg/agents/providers/planexecute"
    "github.com/lookatitude/beluga-ai/pkg/agents/providers/react"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

var tracer = otel.Tracer("beluga.agents.example")

// AgentDemo demonstrates both agent types
type AgentDemo struct {
    llmClient iface.ChatModel
    tools     []tools.Tool
}

// NewAgentDemo creates a new demo with the given LLM client
func NewAgentDemo(client iface.ChatModel, agentTools []tools.Tool) *AgentDemo {
    return &AgentDemo{
        llmClient: client,
        tools:     agentTools,
    }
}

// RunReActDemo demonstrates a ReAct agent
func (d *AgentDemo) RunReActDemo(ctx context.Context, question string) error {
    ctx, span := tracer.Start(ctx, "demo.react",
        trace.WithAttributes(
            attribute.String("question", question),
        ))
    defer span.End()

    start := time.Now()

    // Create the ReAct agent
    promptTemplate := `You are a helpful assistant with access to tools.
Think through problems step by step. When you have the final answer,
respond with "Final Answer: [your answer]".

Question: {input}`

    agent, err := react.NewReActAgent(
        "react-demo",
        d.llmClient,
        d.tools,
        promptTemplate,
    )
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("failed to create ReAct agent: %w", err)
    }

    // Plan (this starts the reasoning loop)
    action, finish, err := agent.Plan(ctx, nil, map[string]any{"input": question})
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("planning failed: %w", err)
    }

    // Log results
    if finish.ReturnValues != nil {
        fmt.Printf("ReAct Result: %v\n", finish.ReturnValues)
    } else {
        fmt.Printf("ReAct Action: %s with input %v\n", action.Tool, action.ToolInput)
    }

    span.SetAttributes(
        attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())),
    )
    span.SetStatus(codes.Ok, "")

    return nil
}

// RunPlanExecuteDemo demonstrates a PlanExecute agent
func (d *AgentDemo) RunPlanExecuteDemo(ctx context.Context, task string) error {
    ctx, span := tracer.Start(ctx, "demo.planexecute",
        trace.WithAttributes(
            attribute.String("task", task),
        ))
    defer span.End()

    start := time.Now()

    // Create the PlanExecute agent
    agent, err := planexecute.NewPlanExecuteAgent(
        "planexecute-demo",
        d.llmClient,
        d.tools,
    )
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("failed to create PlanExecute agent: %w", err)
    }

    // Configure the agent
    agent = agent.
        WithMaxPlanSteps(5).
        WithMaxIterations(10)

    // Plan first
    action, _, err := agent.Plan(ctx, nil, map[string]any{"input": task})
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("planning failed: %w", err)
    }

    // Extract and display the plan
    if planJSON, ok := action.ToolInput["plan"].(string); ok {
        var plan planexecute.ExecutionPlan
        if err := json.Unmarshal([]byte(planJSON), &plan); err == nil {
            fmt.Printf("Generated Plan (%d steps):\n", plan.TotalSteps)
            for _, step := range plan.Steps {
                fmt.Printf("  %d. %s (tool: %s)\n", step.StepNumber, step.Action, step.Tool)
            }
        }
    }

    span.SetAttributes(
        attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())),
    )
    span.SetStatus(codes.Ok, "")

    return nil
}

func main() {
    ctx := context.Background()

    // Create LLM client
    client, err := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        log.Fatalf("Failed to create LLM: %v", err)
    }

    // Create tools
    calculator := tools.NewSimpleTool(
        "calculator",
        "Perform arithmetic calculations",
        func(ctx context.Context, args map[string]any) (string, error) {
            expr, _ := args["expression"].(string)
            return fmt.Sprintf(`{"result": "evaluated: %s"}`, expr), nil
        },
        tools.WithParameter("expression", "string", "Math expression", true),
    )

    demo := NewAgentDemo(client, []tools.Tool{calculator})

    fmt.Println("=== ReAct Agent Demo ===")
    if err := demo.RunReActDemo(ctx, "What is 15 * 7?"); err != nil {
        log.Printf("ReAct demo error: %v", err)
    }

    fmt.Println("\n=== PlanExecute Agent Demo ===")
    if err := demo.RunPlanExecuteDemo(ctx, "Calculate 15 * 7, then add 10 to the result"); err != nil {
        log.Printf("PlanExecute demo error: %v", err)
    }
}
```

## Testing

Testing agents requires mocking the LLM responses. Here's how:

### Unit Tests for ReAct Agent

```go
func TestReActAgent_Plan(t *testing.T) {
    tests := []struct {
        name         string
        input        string
        mockResponse string
        wantAction   bool
        wantFinish   bool
    }{
        {
            name:         "generates tool action",
            input:        "Calculate 2+2",
            mockResponse: "Thought: I should use the calculator.\nAction: calculator\nAction Input: 2+2",
            wantAction:   true,
            wantFinish:   false,
        },
        {
            name:         "generates final answer",
            input:        "What is 1+1?",
            mockResponse: "Thought: This is simple.\nFinal Answer: 2",
            wantAction:   false,
            wantFinish:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockLLM := llms.NewAdvancedMockChatModel(
                llms.WithResponses(tt.mockResponse),
            )

            agent, err := react.NewReActAgent(
                "test",
                mockLLM,
                []tools.Tool{testCalculator},
                "Test prompt: {input}",
            )
            if err != nil {
                t.Fatalf("Failed to create agent: %v", err)
            }

            action, finish, err := agent.Plan(context.Background(), nil, map[string]any{"input": tt.input})
            if err != nil {
                t.Fatalf("Plan() error: %v", err)
            }

            gotAction := action.Tool != ""
            gotFinish := finish.ReturnValues != nil

            if gotAction != tt.wantAction {
                t.Errorf("gotAction = %v, wantAction = %v", gotAction, tt.wantAction)
            }
            if gotFinish != tt.wantFinish {
                t.Errorf("gotFinish = %v, wantFinish = %v", gotFinish, tt.wantFinish)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v -run TestReActAgent ./...
```

## Best Practices

### Do

- **Start with ReAct for exploratory tasks** - It's more flexible and adapts to unexpected situations
- **Use PlanExecute for well-defined workflows** - The upfront planning catches issues early
- **Set reasonable iteration limits** - Prevent infinite loops with `WithMaxIterations()`
- **Use different LLMs for planning vs execution** - Optimize for quality and speed
- **Add OTEL instrumentation** - Track agent behavior for debugging and optimization

### Don't

- **Don't use agents for simple tasks** - A single LLM call is faster and cheaper
- **Don't ignore iteration limits** - Agents can loop forever without them
- **Don't give agents too many tools** - Context length and decision quality suffer
- **Don't skip error handling** - Tool failures and LLM errors need graceful handling

### Performance Tips

- **Cache tool results** - Same inputs often produce same outputs
- **Use streaming for long operations** - Users appreciate feedback during waits
- **Parallelize independent steps** - PlanExecute can run parallel steps concurrently

## Troubleshooting

### Q: My ReAct agent loops forever

**A:** This usually means the agent can't find a satisfactory answer. Check:
1. Are your tools returning useful results?
2. Is the prompt clear about when to finish?
3. Set `WithMaxIterations()` to prevent infinite loops

### Q: The PlanExecute agent's plans are too vague

**A:** The planner LLM needs more guidance:
1. Provide more detailed tool descriptions
2. Include examples in the system prompt
3. Use a more capable model (GPT-4 > GPT-3.5)

### Q: Tool calls are failing

**A:** Debug tool execution:
1. Check tool input parsing - JSON must be valid
2. Verify tool is registered in the tool map
3. Look at the exact arguments the LLM is generating

### Q: How do I debug agent reasoning?

**A:** Enable verbose logging:
```go
// Add logging to see agent thinking
agent.OnThought(func(thought string) {
    log.Printf("Agent thought: %s", thought)
})
agent.OnAction(func(action AgentAction) {
    log.Printf("Agent action: %s with %v", action.Tool, action.ToolInput)
})
```

## Related Resources

Now that you understand agent types, explore:

- **[Streaming LLM Guide](./llm-streaming-tool-calls.md)** - Combine streaming with agents for responsive UIs
- **[PlanExecute Example](https://github.com/lookatitude/beluga-ai/blob/main/examples/agents/planexecute/README.md)** - Complete implementation with tests
- **[Custom Agent Cookbook](../cookbook/custom-agent.md)** - Extend agents with custom behavior
- **[Multi-Agent Use Case](../use-cases/11-batch-processing.md)** - Coordinate multiple agents
- **[Agent Concepts](../concepts/agents.md)** - Deep dive into agent architecture
