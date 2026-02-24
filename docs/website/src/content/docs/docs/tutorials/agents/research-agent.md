---
title: Building a Research Agent
description: "Build an autonomous research agent in Go with Beluga AI — decompose questions, search multiple sources with the ReAct pattern, and synthesize cited reports."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, research agent, ReAct, autonomous agent, tool use, report synthesis"
---

A research agent goes beyond simple chat -- it breaks down complex questions into search queries, retrieves information from multiple sources, and synthesizes a comprehensive report. This is an example of the **ReAct (Reason + Act) pattern**, where the model alternates between reasoning about what to do next and executing actions via tools. The agent loop continues until the model decides it has gathered enough information to produce a final answer, at which point it responds without any tool calls. This self-terminating behavior is what makes the agent autonomous rather than requiring explicit orchestration.

## What You Will Build

An autonomous research agent using the ReAct pattern that searches for information, analyzes findings, and produces a structured report with citations.

## Prerequisites

- Understanding of the [ChatModel interface](/docs/guides/llm) and [tool system](/docs/guides/tools)
- A configured LLM provider

## Step 1: Define the Tools

Tools are the agent's interface to the external world. Each tool has two components: a `schema.ToolDefinition` that tells the model what the tool does and what arguments it accepts (via JSON Schema), and an execution function that performs the actual work. The model never calls the execution function directly -- it generates a tool call with arguments, and your code dispatches it. This separation between tool description and tool execution is what makes the system provider-agnostic: the same tool definitions work with any LLM that supports function calling.

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

The system prompt shapes the agent's behavior within the ReAct loop. By instructing the model to break down questions into specific search queries and to refine queries when results are insufficient, you are encoding a research methodology into the agent's reasoning. The model will follow these instructions when deciding which tool to call next and how to interpret results. A well-crafted persona prompt is often the difference between an agent that makes one search and stops, and one that iterates until it has comprehensive coverage of a topic.

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

The agent loop is the core of the ReAct pattern. Each iteration: (1) send the full conversation history to the model, (2) check if the response contains tool calls, and (3) if so, execute them and append results to the history. The loop terminates when the model responds with plain text and no tool calls, signaling that it has gathered enough information to answer.

The `maxIterations` bound is a safety mechanism. Without it, a model that keeps generating tool calls (due to ambiguous instructions or hallucinated tools) would loop indefinitely. In production, you would also monitor token usage per iteration and set timeouts via context cancellation.

Note how the entire conversation history -- including all tool calls and their results -- is sent to the model on every iteration. This gives the model full visibility into what it has already tried, preventing redundant searches and enabling it to build on previous findings.

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

In a synchronous agent loop, the user sees nothing until the final response is ready. For long research tasks that involve multiple tool calls, this creates a poor experience. The `Stream` method returns an `iter.Seq2[schema.StreamChunk, error]` iterator that delivers tokens as they are generated, giving users real-time visibility into the agent's reasoning and progress. You accumulate the streamed chunks into a complete response, then check for tool calls just as you would with the synchronous `Generate` method.

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

- [Multi-Agent Orchestration](/docs/tutorials/agents/multi-agent-orchestration) -- Coordinate teams of agents
- [Tools Registry](/docs/tutorials/agents/tools-registry) -- Build a reusable tool library
