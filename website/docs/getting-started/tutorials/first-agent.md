---
title: First Agent
sidebar_position: 3
---

# Part 3: Creating Your First Agent

In this tutorial, you'll learn how to create autonomous AI agents that can reason, plan, and execute tasks using tools. Agents are the building blocks for creating intelligent, interactive AI systems.

## Learning Objectives

- ✅ Understand agent architecture and lifecycle
- ✅ Create a base agent with tools
- ✅ Execute agent tasks
- ✅ Handle agent responses and errors
- ✅ Understand the ReAct pattern

## Prerequisites

- Completed [Part 1: Your First LLM Call](./01-first-llm-call)
- Basic understanding of tools (we'll cover this in detail in Part 4)
- API key for an LLM provider

## What is an Agent?

An agent is an autonomous entity that:
- **Plans** actions to achieve goals
- **Executes** actions using tools
- **Observes** results and adapts
- **Reasons** about next steps

Agents can solve complex tasks by breaking them down into smaller steps and using tools to gather information or perform actions.

## Step 1: Project Setup

Create a new file `agent_example.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/llms"
)
```

## Step 2: Setup LLM Provider

```go
func setupLLM(ctx context.Context) (llmsiface.ChatModel, error) {
	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llms.WithTemperatureConfig(0.7),
	)

	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return provider, nil
}
```

## Step 3: Create Tools

Agents use tools to perform actions. Let's create some basic tools:

```go
func createTools() []tools.Tool {
	// Calculator tool for math operations
	calculator := tools.NewCalculatorTool()

	// Echo tool for testing
	echoTool := tools.NewEchoTool()

	return []tools.Tool{
		calculator,
		echoTool,
	}
}
```

## Step 4: Create an Agent

```go
func createAgent(llm llmsiface.ChatModel, agentTools []tools.Tool) (agentsiface.CompositeAgent, error) {
	agent, err := agents.NewBaseAgent(
		"my-assistant",           // Agent name
		llm,                     // LLM provider
		agentTools,              // Available tools
		agents.WithMaxRetries(3), // Max retry attempts
		agents.WithMaxIterations(10), // Max reasoning iterations
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}
```

## Step 5: Initialize and Execute Agent

```go
func main() {
	ctx := context.Background()

	// Step 1: Setup LLM
	fmt.Println("Setting up LLM...")
	llm, err := setupLLM(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Step 2: Create tools
	fmt.Println("Creating tools...")
	agentTools := createTools()
	fmt.Printf("Created %d tools\n", len(agentTools))

	// Step 3: Create agent
	fmt.Println("Creating agent...")
	agent, err := createAgent(llm, agentTools)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		return
	}

	// Step 4: Initialize agent
	fmt.Println("Initializing agent...")
	initConfig := map[string]interface{}{
		"max_retries": 3,
		"max_iterations": 10,
	}
	if err := agent.Initialize(initConfig); err != nil {
		fmt.Printf("Error initializing agent: %v\n", err)
		return
	}

	// Step 5: Execute agent with a task
	fmt.Println("\nExecuting agent task...")
	input := map[string]interface{}{
		"input": "Calculate 15 * 23 and then echo the result",
	}

	result, err := agent.Invoke(ctx, input)
	if err != nil {
		fmt.Printf("Error executing agent: %v\n", err)
		return
	}

	// Step 6: Display result
	fmt.Printf("\nAgent Result: %v\n", result)
}
```

## Step 6: Understanding Agent Lifecycle

Agents follow a specific lifecycle:

### 1. Initialization

```go
// Initialize with configuration
config := map[string]interface{}{
	"max_retries": 3,
	"max_iterations": 10,
	"temperature": 0.7,
}
agent.Initialize(config)
```

### 2. Execution

```go
// Execute with input
input := map[string]interface{}{
	"input": "Your task description here",
}
result, err := agent.Invoke(ctx, input)
```

### 3. Finalization

```go
// Cleanup resources
defer agent.Finalize()
```

## Step 7: Agent Configuration Options

```go
agent, err := agents.NewBaseAgent(
	"my-agent",
	llm,
	tools,
	// Retry configuration
	agents.WithMaxRetries(5),
	agents.WithRetryDelay(2 * time.Second),
	
	// Execution limits
	agents.WithMaxIterations(20),
	agents.WithTimeout(60 * time.Second),
	
	// Event handlers
	agents.WithEventHandler("execution_started", func(payload interface{}) error {
		fmt.Printf("Execution started: %v\n", payload)
		return nil
	}),
	agents.WithEventHandler("execution_completed", func(payload interface{}) error {
		fmt.Printf("Execution completed: %v\n", payload)
		return nil
	}),
)
```

## Step 8: Handling Agent Responses

```go
result, err := agent.Invoke(ctx, input)
if err != nil {
	// Check error type
	if agents.IsAgentError(err) {
		code := agents.GetAgentErrorCode(err)
		fmt.Printf("Agent error [%s]: %v\n", code, err)
	} else {
		fmt.Printf("Unexpected error: %v\n", err)
	}
	return
}

// Process result
if output, ok := result["output"].(string); ok {
	fmt.Printf("Agent output: %s\n", output)
}
```

## Step 9: Complete Example with Error Handling

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup
	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)
	factory := llms.NewFactory()
	llm, _ := factory.CreateProvider("openai", config)

	// Create agent
	tools := []tools.Tool{
		tools.NewCalculatorTool(),
		tools.NewEchoTool(),
	}
	agent, _ := agents.NewBaseAgent("assistant", llm, tools,
		agents.WithMaxRetries(3),
		agents.WithMaxIterations(10),
	)

	// Initialize
	agent.Initialize(map[string]interface{}{
		"max_retries": 3,
	})

	// Execute
	input := map[string]interface{}{
		"input": "Calculate 42 * 17",
	}
	result, err := agent.Invoke(ctx, input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)
}
```

## Step 10: Advanced: ReAct Agent Pattern

The ReAct (Reasoning + Acting) pattern enables agents to reason about actions:

```go
// ReAct agents are currently in development
// This is a preview of the API

reactAgent, err := agents.NewReActAgent(
	"researcher",
	llm,
	tools,
	"You are a helpful assistant that can use tools to answer questions.",
)
```

## Exercises

1. **Create a math agent**: Build an agent that solves math problems step by step
2. **Add more tools**: Integrate additional tools (Shell, API, etc.)
3. **Experiment with iterations**: Try different max_iterations values
4. **Add event handlers**: Monitor agent execution with custom handlers
5. **Error recovery**: Implement retry logic for failed operations

## Common Issues

### Agent not executing

- Check that tools are properly registered
- Verify LLM provider is working
- Ensure agent is initialized before execution

### Tool execution errors

- Verify tool implementations are correct
- Check tool input/output formats
- Review tool error messages

### Timeout errors

- Increase timeout duration
- Reduce max_iterations
- Optimize tool execution time

## Next Steps

Congratulations! You've created your first agent. Next, learn how to:

- **[Part 4: Working with Tools](./working-with-tools)** - Deep dive into tool integration
- **[Part 5: Memory Management](./memory-management)** - Add conversation memory
- **[Concepts: Agents](../../concepts/agents)** - Deep dive into agent concepts

---

**Ready for the next step?** Continue to [Part 4: Working with Tools](./working-with-tools)!

