# Multi-Agent Orchestration Patterns

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll coordinate multiple specialized agents to solve complex problems. We'll explore two primary patterns: the Router (Handoff) pattern and the Supervisor (Manager) pattern.

## Learning Objectives
- ✅ Understand Handoff vs. Supervisor patterns
- ✅ Implement a Router (Classifier) Agent
- ✅ Implement a Supervisor (Manager) Agent
- ✅ Shared memory/state between agents

## Introduction
Welcome, colleague! Sometimes one agent isn't enough. For complex tasks like software development or market research, you need a team. Let's look at how to orchestrate multiple agents to work together seamlessly.

## Prerequisites

- [Building a Research Agent](./agents-research-agent.md)

## Pattern 1: Router (Handoff)

A "Receptionist" agent decides which expert to call.
graph TD
```mermaid
    User --> Receptionist
    Receptionist -->|Coding| Developer
    Receptionist -->|Writing| Copywriter
    Receptionist -->|Math| Analyst
go
// 1. Define Experts
coder := agents.NewBaseAgent("coder", ...)
writer := agents.NewBaseAgent("writer", ...)

// 2. Define Router
routerPrompt := `Classify the user input into: [CODING, WRITING, OTHER]`
router := chatmodels.NewChatModel(llm)

// 3. Orchestration Logic
func handleRequest(input string) {
    classification := router.Predict(input)
    switch classification {
    case "CODING":
        coder.Invoke(ctx, input)
    case "WRITING":
        writer.Invoke(ctx, input)
    }
}
```

## Pattern 2: Supervisor (Manager)

A "Manager" creates a plan and delegates tasks, aggregating results.
```go
func runSupervisor(goal string) {
    // 1. Manager plans
    plan := manager.Plan(goal) // Returns ["Research X", "Write Code Y"]
    
    // 2. Execute steps
    var context string
    for _, step := range plan {
        if step.Type == "Research" {
            res, _ := researcher.Invoke(ctx, step.Content)
            context += res
        } else if step.Type == "Code" {
            coder.Invoke(ctx, step.Content + "\nContext: " + context)
        }
    }
}
```

## Step 3: Shared State

Use a shared `ChatHistory` or `Memory` accessible by all agents (or passed in context).
```go
sharedMem := memory.NewBufferMemory()

coder.Initialize(map[string]any{"memory": sharedMem})
tester.Initialize(map[string]any{"memory": sharedMem})
```

## Verification

Create a "Software House" simulator:
1. **Product Manager**: Defines specs.
2. **Developer**: Writes code.
3. **QA**: Reviews code.

Input: "Build a snake game in Python". Watch the agents interact.

## Next Steps

- **[DAG-based Agents](./orchestration-dag-agents.md)** - Formalize the flow
- **[Redis Persistence](./memory-redis-persistence.md)** - Persist the team's state
