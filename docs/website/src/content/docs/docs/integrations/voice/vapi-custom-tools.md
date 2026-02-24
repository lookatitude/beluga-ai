---
title: Vapi Custom Tools Integration
description: "Connect Vapi custom tool webhooks to Beluga AI agents for extending voice conversations with backend logic, APIs, and databases."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Vapi integration, custom tools, voice AI, Beluga AI, Vapi webhooks, voice agent Go, telephony AI"
---

Vapi is a popular hosted voice AI platform. When your Vapi assistant needs to perform actions beyond conversation -- looking up orders, booking appointments, or querying databases -- it calls out to your backend via HTTP. By connecting these custom tool endpoints to Beluga AI agents, you combine Vapi's managed telephony and voice handling with the full power of Beluga AI's agent runtime, tools, and memory system. This guide covers implementing HTTP endpoints that Vapi invokes as custom tools, connecting them to Beluga AI agents and logic, and returning structured results for the voice pipeline to consume.

## Overview

Vapi's custom tool system sends HTTP POST requests to your backend when the assistant decides to use a tool. Each request includes tool arguments extracted from the conversation. Your endpoint processes the request -- using Beluga AI agents, database lookups, or any custom logic -- and returns a result that Vapi uses for the next conversation turn.

This pattern enables:
- Calling Beluga AI agents from voice conversations
- Database lookups and API integrations triggered by voice
- Structured data retrieval during live calls
- Multi-step workflows driven by conversational context

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Vapi account with an assistant configured for custom tools ([docs.vapi.ai](https://docs.vapi.ai/tools))
- A publicly accessible HTTPS endpoint

## Installation

Install the Beluga AI module:

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

| Option | Description | Default |
|--------|-------------|---------|
| Tool base URL | Base path for Vapi tool endpoints | `/vapi/tools` |
| Timeout | Handler timeout for agent or logic execution | `10s` |
| Auth | Optional API key or HMAC verification | Per Vapi configuration |

## Usage

### Defining Tools in Vapi

In the Vapi dashboard, add a custom tool for each backend capability:

1. Set a tool name (e.g., `lookup_order`)
2. Define parameters as a JSON schema (e.g., `order_id` as a string)
3. Set the endpoint URL to your backend (e.g., `https://your-app.example.com/vapi/tools/lookup_order`)

### Implementing Tool Handlers

Create an HTTP handler that receives Vapi's tool-call payload, extracts arguments, invokes Beluga AI logic, and returns a result:

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    // Create a Beluga AI agent for order lookups.
    model, err := llm.New("openai", llm.WithModel("gpt-4o"))
    if err != nil {
        log.Fatalf("Failed to create model: %v", err)
    }

    orderAgent, err := agent.New("order-lookup",
        agent.WithModel(model),
        agent.WithSystemPrompt("You are an order lookup assistant. Given an order ID, return the order status and details."),
    )
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/vapi/tools/lookup_order", vapiToolHandler(orderAgent))

    log.Println("Listening on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}

// vapiToolRequest represents the incoming Vapi tool call payload.
type vapiToolRequest struct {
    Message struct {
        ToolCalls []struct {
            Function struct {
                Name      string `json:"name"`
                Arguments string `json:"arguments"`
            } `json:"function"`
        } `json:"toolCalls"`
    } `json:"message"`
}

func vapiToolHandler(a *agent.Agent) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        var req vapiToolRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "bad request", http.StatusBadRequest)
            return
        }

        if len(req.Message.ToolCalls) == 0 {
            http.Error(w, "no tool calls in request", http.StatusBadRequest)
            return
        }

        // Extract tool arguments.
        args := req.Message.ToolCalls[0].Function.Arguments

        // Invoke the Beluga AI agent with a timeout.
        ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
        defer cancel()

        result, err := a.Run(ctx, []schema.Message{
            schema.NewHumanMessage(args),
        })
        if err != nil {
            writeToolError(w, err)
            return
        }

        writeToolResult(w, result)
    }
}

func writeToolResult(w http.ResponseWriter, result any) {
    w.Header().Set("Content-Type", "application/json")
    err := json.NewEncoder(w).Encode(map[string]any{"result": result})
    if err != nil {
        log.Printf("Failed to write tool result: %v", err)
    }
}

func writeToolError(w http.ResponseWriter, err error) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    encErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
    if encErr != nil {
        log.Printf("Failed to write tool error: %v", encErr)
    }
}
```

### Multiple Tool Endpoints

Register a handler for each tool defined in Vapi:

```go
mux := http.NewServeMux()
mux.HandleFunc("/vapi/tools/lookup_order", vapiToolHandler(orderAgent))
mux.HandleFunc("/vapi/tools/book_appointment", vapiToolHandler(bookingAgent))
mux.HandleFunc("/vapi/tools/check_inventory", vapiToolHandler(inventoryAgent))

log.Println("Listening on :8080")
if err := http.ListenAndServe(":8080", mux); err != nil {
    log.Fatalf("Server failed: %v", err)
}
```

Each endpoint can use a different Beluga AI agent, or route to shared logic based on the tool name.

## Advanced Topics

### Request Authentication

Verify incoming requests to ensure they originate from Vapi. Use an API key header or HMAC signature depending on your Vapi configuration:

```go
func requireAPIKey(next http.HandlerFunc, expectedKey string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("X-API-Key")
        if key != expectedKey {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

// Usage:
mux.HandleFunc("/vapi/tools/lookup_order",
    requireAPIKey(vapiToolHandler(orderAgent), os.Getenv("VAPI_TOOL_API_KEY")),
)
```

### Response Format

Vapi expects a specific response structure. Consult the [Vapi custom tool documentation](https://docs.vapi.ai/tools) for the exact schema. A typical successful response:

```json
{
  "result": "Order #12345 is shipped and arriving on March 15."
}
```

A typical error response:

```json
{
  "error": "Order not found"
}
```

### Idempotent Handlers

Vapi may retry tool calls on timeout or failure. Design handlers to be idempotent -- repeated calls with the same arguments should produce the same result without side effects.

### Production Considerations

- **Timeouts**: Return results within Vapi's tool timeout window (typically 10-30 seconds). Use `context.WithTimeout` to enforce deadlines.
- **Observability**: Log and instrument tool invocations with OpenTelemetry. Track latency, error rates, and argument patterns.
- **Error handling**: Return structured error responses rather than generic HTTP 500 errors. Include actionable error messages that Vapi can relay to the user.
- **Scaling**: Each tool call is an independent HTTP request. Use standard Go HTTP server scaling techniques (connection pooling, load balancing).

## Troubleshooting

### Vapi Never Calls the Endpoint

The tool endpoint URL is incorrect, or the tool is not attached to the assistant. Verify:
- The endpoint URL in the Vapi dashboard matches your server's public URL
- The tool is assigned to the active assistant
- The server is reachable over HTTPS from Vapi's infrastructure

### Agent Returns but Vapi Ignores the Result

The response format does not match Vapi's expected schema. Check the [Vapi custom tool documentation](https://docs.vapi.ai/tools) for the exact response structure. Ensure the `Content-Type` header is `application/json`.

### Timeout or Slow Responses

The agent or downstream logic exceeds Vapi's timeout. Reduce processing time by:
- Caching frequently requested data
- Using faster models for latency-sensitive tools
- Precomputing results where possible

## Related Resources

- [Voice Services Overview](/docs/integrations/voice-services) -- All supported voice providers
- [LiveKit Webhooks Integration](/docs/integrations/livekit-webhooks) -- LiveKit webhook handling
