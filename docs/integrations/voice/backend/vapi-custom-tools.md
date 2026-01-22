# Vapi Custom Tools Integration

Welcome, colleague! In this guide we'll integrate **Vapi custom tools** with Beluga AI. Vapi supports custom tool definitions that can call your backend (e.g. Beluga agents, APIs). You'll define tools in Vapi, implement HTTP handlers that invoke Beluga agents or logic, and return results for TTS or structured responses.

## What you will build

You will create HTTP endpoints that Vapi calls as "custom tools" during a conversation. Each endpoint receives tool arguments from Vapi, runs Beluga agent or custom logic, and returns a result. Vapi then uses that result for the next turn (e.g. speak via TTS or update context). This allows you to combine Vapi's managed voice pipeline with Beluga's agents and tools.

## Learning Objectives

- ✅ Define custom tools in Vapi (name, parameters, endpoint URL)
- ✅ Implement HTTP handlers for each tool
- ✅ Call Beluga agents or custom logic from handlers
- ✅ Return structured results (e.g. JSON) for Vapi to consume
- ✅ Handle errors and timeouts

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- Vapi account and assistant configured for custom tools
- Voice backend using the Vapi provider (optional; tools can be used with Vapi-only flows)

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

Refer to [Vapi custom tools](https://docs.vapi.ai/tools) for tool schema and request format.

## Step 2: Define Tools in Vapi

In the Vapi dashboard, add a custom tool for each capability (e.g. `lookup_order`, `book_appointment`). Set the endpoint URL to your backend (e.g. `https://your-app.example.com/vapi/tools/lookup_order`). Define parameters (e.g. `order_id`) as JSON schema.

## Step 3: Implement Tool Handlers

Create an HTTP handler that receives Vapi's tool-call payload, extracts arguments, runs your logic, and returns a result:
```go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
)

func vapiToolLookupOrder(agent agentsiface.Agent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Message struct {
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"toolCalls"`
			} `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Find lookup_order call, parse arguments (e.g. order_id)
		// Invoke agent or custom logic
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		out, err := agent.Invoke(ctx, /* input from tool args */)
		if err != nil {
			writeToolError(w, err)
			return
		}
		writeToolResult(w, out)
	}
}

func writeToolResult(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
}

func writeToolError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
```

## Step 4: Wire Beluga Agents

Use `pkg/agents` to create an agent (e.g. ReAct, plan-execute) with tools. In your tool handler, call `agent.Invoke` with the parsed tool arguments. Map Vapi tool names to your agent tools so that both sides agree on semantics.

## Step 5: Register Routes and Run Server
```text
go
go
	mux := http.NewServeMux()
	mux.Handle("/vapi/tools/lookup_order", vapiToolLookupOrder(agent))
	// ... more tools
	_ = http.ListenAndServe(":8080", mux)
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Tool base URL | Base path for Vapi tool endpoints | `/vapi/tools` |
| Timeout | Handler timeout for agent/logic | 10 s |
| Auth | Optional API key or HMAC verification | Per Vapi docs |

## Common Issues

### "Vapi never calls my endpoint"

**Problem**: URL incorrect, or tool not attached to assistant.

**Solution**: Verify Vapi assistant tool config and endpoint URL. Ensure the server is reachable (HTTPS, firewall).

### "Agent returns but Vapi doesn't use it"

**Problem**: Response format doesn't match what Vapi expects.

**Solution**: Follow Vapi's custom tool response schema (e.g. `result` or `content`). Check Vapi docs for the exact structure.

### "Timeouts or slow responses"

**Problem**: Agent or downstream calls are slow.

**Solution**: Set reasonable timeouts; return quickly or use async patterns if Vapi supports them.

## Production Considerations

- **Auth**: Verify requests (e.g. Vapi webhook secret or API key) before processing.
- **Idempotency**: Handle duplicate tool calls if Vapi retries.
- **Observability**: Log and metric tool invocations, latency, and errors.

## Next Steps

- **[LiveKit Webhooks](./livekit-webhooks-integration.md)** — LiveKit webhook handling.
- **[Voice Backends Tutorial](../../../../tutorials/voice/voice-backends-livekit-vapi.md)** — Backend and Vapi setup.
- **[Scaling Concurrent Streams](../../../../cookbook/voice-backend-scaling-concurrent-streams.md)** — Production scaling.
