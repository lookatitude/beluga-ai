# Building a Research Agent

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll build an autonomous research agent capable of breaking down complex questions, searching for information, and synthesizing a comprehensive report using a multi-step reasoning loop (ReAct).

## Learning Objectives
- ✅ Implement a multi-step reasoning loop (ReAct)
- ✅ Integrate search tools (Google/DuckDuckGo)
- ✅ Manage agent scratchpad (intermediate steps)
- ✅ Synthesize final output

## Introduction
Welcome, colleague! Building a research agent is one of the most powerful ways to use Beluga AI. We're going to move beyond simple chat and create something that can actually browse the web and think through a problem.

## Prerequisites

- Completed [Creating Your First Agent](../../getting-started/03-first-agent.md)
- Search API Key (SerpApi or similar)

## Step 1: Define the Tools

A researcher needs search and a calculator (for data analysis).
```go
func getTools() []agentsiface.Tool {
    return []agentsiface.Tool{
        tools.NewSerpApiTool(os.Getenv("SERPAPI_KEY")),
        tools.NewCalculatorTool(),
        tools.NewWebScraperTool(), // Visit pages found
    }
}

## Step 2: The Researcher Persona
const researcherPrompt = `You are a Senior Research Analyst.
Your goal is to answer the user's question comprehensively.
1. Break down the question into search queries.
2. Use the Search tool to find information.
3. Use the WebScraper to read details.
4. Analyze the findings.
5. Provide a final report with citations.`
```

## Step 3: Configuring the Agent

We'll use a `ReAct` agent which is optimized for reasoning.
agent, err := agents.NewReActAgent(
```
    "researcher",
    llm,
    getTools(),
    researcherPrompt,
    agents.WithMaxIterations(15), // Research takes time
)

## Step 4: Execution Loop
```go
input := map[string]any{
    "input": "What is the current state of Solid State Batteries in 2026? Key players and breakthroughs.",
}
// We use Stream to show progress to the user
stream, _ := agent.Stream(ctx, input)
go
for chunk := range stream {
    if chunk.Type == "tool_start" {
        fmt.Printf("🔎 Researching: %s\n", chunk.Content)
    } else if chunk.Type == "tool_end" {
        fmt.Printf("✅ Found data\n")
    }
}
```

## Verification

Run the agent with a complex query. Verify it performs multiple searches (e.g., "Solid state battery companies", "Toyota battery roadmap", "QuantumScape stock") before answering.

## Next Steps

- **[Multi-Agent Orchestration](./agents-multi-agent-orchestration.md)** - Team of researchers
- **[Memory Persistence](./memory-redis-persistence.md)** - Save research sessions
