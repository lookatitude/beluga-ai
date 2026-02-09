---
title: Building a Research Agent
description: Build an autonomous research agent that decomposes questions, searches for information, and synthesizes reports.
---

A research agent goes beyond simple chat — it breaks down complex questions into search queries, retrieves information from multiple sources, and synthesizes a comprehensive report. This tutorial demonstrates building an agent with a reasoning loop, tool integration, and streaming progress output.

## What You Will Build

An autonomous research agent using the ReAct (Reason + Act) pattern that searches for information, analyzes findings, and produces a structured report with citations.

## Prerequisites

- Understanding of the [ChatModel interface](/guides/llm) and [tool system](/guides/tools)
- A configured LLM provider

## Step 1: Define the Tools

A research agent needs tools for search, web scraping, and computation:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    "github.com/lookatitude/beluga-ai/schema"
)

// Define tool definitions for the model
var searchTool = schema.ToolDefinition{
    Name:        "web_search",
    Description: "Search the web for information. Use this to find current data and facts.",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "query": map[string]any{
                "type":        "string",
                "description": "The search query",
            },
        },
        "required": []any{"query"},
    },
}

var calculatorTool = schema.ToolDefinition{
    Name:        "calculator",
    Description: "Perform mathematical calculations.",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "expression": map[string]any{
                "type":        "string",
                "description": "The math expression to evaluate",
            },
        },
        "required": []any{"expression"},
    },
}
```

## Step 2: Define the Researcher Persona

```go
const researcherPrompt = `You are a Senior Research Analyst.
Your goal is to answer the user's question comprehensively.

Process:
1. Break down the question into specific search queries.
2. Use the web_search tool to find relevant information.
3. Use the calculator tool for any numerical analysis.
4. Synthesize findings into a structured report.
5. Include citations for all claims.

Always search multiple angles before concluding. If initial results are insufficient, refine your queries.`
```

## Step 3: Build the Agent Loop

Implement a ReAct-style reasoning loop that alternates between thinking and acting:

```go
func runResearchAgent(ctx context.Context, model llm.ChatModel, question string) (string, error) {
    // Bind tools to the model
    toolModel := model.BindTools([]schema.ToolDefinition{searchTool, calculatorTool})

    // Initialize conversation
    messages := []schema.Message{
        schema.NewSystemMessage(researcherPrompt),
        schema.NewHumanMessage(question),
    }

    maxIterations := 10
    for i := 0; i < maxIterations; i++ {
        // Generate a response (may include tool calls)
        resp, err := toolModel.Generate(ctx, messages)
        if err != nil {
            return "", fmt.Errorf("iteration %d: %w", i, err)
        }

        // Append the AI response to history
        messages = append(messages, resp)

        // If no tool calls, the agent is done
        if len(resp.ToolCalls) == 0 {
            return resp.Text(), nil
        }

        // Execute each tool call
        for _, tc := range resp.ToolCalls {
            fmt.Printf("  Tool: %s\n", tc.Name)

            result, err := executeTool(ctx, tc)
            if err != nil {
                result = fmt.Sprintf("Error: %v", err)
            }

            // Append tool result to history
            messages = append(messages, schema.NewToolMessage(tc.ID, result))
        }
    }

    return "", fmt.Errorf("agent exceeded max iterations (%d)", maxIterations)
}

func executeTool(ctx context.Context, tc schema.ToolCall) (string, error) {
    var args map[string]string
    if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
        return "", fmt.Errorf("parse arguments: %w", err)
    }

    switch tc.Name {
    case "web_search":
        // Replace with actual search API call
        return fmt.Sprintf("Search results for: %s\n[Simulated results]", args["query"]), nil
    case "calculator":
        return fmt.Sprintf("Result: %s = [computed]", args["expression"]), nil
    default:
        return "", fmt.Errorf("unknown tool: %s", tc.Name)
    }
}
```

## Step 4: Streaming Progress

Use `Stream` to show progress to the user in real-time:

```go
func runResearchAgentStreaming(ctx context.Context, model llm.ChatModel, question string) error {
    toolModel := model.BindTools([]schema.ToolDefinition{searchTool, calculatorTool})

    messages := []schema.Message{
        schema.NewSystemMessage(researcherPrompt),
        schema.NewHumanMessage(question),
    }

    for i := 0; i < 10; i++ {
        // Stream the response
        var fullText string
        var toolCalls []schema.ToolCall

        for chunk, err := range toolModel.Stream(ctx, messages) {
            if err != nil {
                return fmt.Errorf("stream error: %w", err)
            }
            if chunk.Delta != "" {
                fmt.Print(chunk.Delta) // Print tokens as they arrive
                fullText += chunk.Delta
            }
            toolCalls = append(toolCalls, chunk.ToolCalls...)
        }
        fmt.Println()

        // Build the AI message from accumulated chunks
        aiMsg := &schema.AIMessage{
            Parts:     []schema.ContentPart{schema.TextPart{Text: fullText}},
            ToolCalls: toolCalls,
        }
        messages = append(messages, aiMsg)

        if len(toolCalls) == 0 {
            return nil // Done
        }

        // Execute tools
        for _, tc := range toolCalls {
            result, err := executeTool(ctx, tc)
            if err != nil {
                result = fmt.Sprintf("Error: %v", err)
            }
            messages = append(messages, schema.NewToolMessage(tc.ID, result))
        }
    }

    return nil
}
```

## Step 5: Run the Agent

```go
func main() {
    ctx := context.Background()

    model, err := llm.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "gpt-4o",
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    question := "What is the current state of solid-state batteries? Who are the key players and what recent breakthroughs have occurred?"

    report, err := runResearchAgent(ctx, model, question)
    if err != nil {
        fmt.Printf("Agent error: %v\n", err)
        return
    }

    fmt.Println("\n--- Research Report ---")
    fmt.Println(report)
}
```

## Verification

Run the agent with a complex question. Verify it:
1. Makes multiple search queries covering different aspects of the topic.
2. Refines queries when initial results are insufficient.
3. Produces a structured report with citations.

## Next Steps

- [Multi-Agent Orchestration](/tutorials/agents/multi-agent-orchestration) — Coordinate teams of agents
- [Tools Registry](/tutorials/agents/tools-registry) — Build a reusable tool library
