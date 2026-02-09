---
title: "REST & OpenAI Agents"
description: "REST/SSE API server and OpenAI Agents protocol compatibility"
---

## openai_agents

```go
import "github.com/lookatitude/beluga-ai/protocol/openai_agents"
```

Package openai_agents provides a compatibility layer between Beluga AI agents
and the OpenAI Agents SDK format. It converts Beluga agents, tools, and
handoffs into the OpenAI Agents SDK wire format for interoperability.

This allows Beluga agents to be exposed via an API compatible with the OpenAI
Agents SDK, enabling clients built for that SDK to interact with Beluga agents.

## Agent Conversion

FromAgent converts a Beluga Agent into an OpenAI Agents SDK AgentDef,
preserving the agent's identity, tools, and handoff relationships:

```go
agentDef := openai_agents.FromAgent(belugaAgent)
jsonBytes, err := json.Marshal(agentDef)
```

## Tool Conversion

FromTools converts Beluga tools into OpenAI Agents SDK ToolDefs:

```go
toolDefs := openai_agents.FromTools(myTools)
```

## Runner

Runner executes agents using the OpenAI Agents SDK request/response format.
It maintains a registry of agents and dispatches requests by agent name.

```go
runner := openai_agents.NewRunner(agentA, agentB)

resp, err := runner.Run(ctx, openai_agents.RunRequest{
    AgentName: "agentA",
    Input:     "Hello",
})

agents := runner.ListAgents() // returns AgentDefs for all registered agents
```

## Key Types

- AgentDef — agent definition in the OpenAI Agents SDK format
- ToolDef / FunctionDef — tool definitions
- Handoff — agent handoff descriptor
- RunRequest / RunResponse — execution request and response types
- Runner — dispatches agent execution requests
- ToolCallResult — tool call result in the response

---

## rest

```go
import "github.com/lookatitude/beluga-ai/protocol/rest"
```

Package rest provides a REST/SSE API server for exposing Beluga agents over
HTTP. It supports both synchronous invocation and real-time streaming via
Server-Sent Events (SSE).

Agents are registered at path prefixes and automatically get two endpoints:

- POST /{path}/invoke — synchronous invocation returning a JSON response
- POST /{path}/stream — streaming invocation via Server-Sent Events

## Usage

```go
srv := rest.NewServer()
if err := srv.RegisterAgent("assistant", myAgent); err != nil {
    log.Fatal(err)
}
srv.Serve(ctx, ":8080")
```

Clients can then invoke the agent synchronously:

```go
// POST /assistant/invoke
// {"input": "Hello"}
// Response: {"result": "Hi there!"}
```

Or stream responses via SSE:

```go
// POST /assistant/stream
// {"input": "Hello"}
// Response: text/event-stream with agent events
```

## SSE Support

The package includes SSEWriter for writing Server-Sent Events to HTTP
responses. It handles event formatting, multi-line data splitting, and
connection keep-alive heartbeats per the SSE specification.

```go
sse, err := rest.NewSSEWriter(w)
if err != nil {
    // response writer does not support flushing
}
sse.WriteEvent(rest.SSEEvent{Event: "message", Data: "hello"})
sse.WriteHeartbeat()
```

## Key Types

- RESTServer — serves Beluga agents as REST/SSE HTTP endpoints
- SSEWriter — writes Server-Sent Events to an HTTP response
- SSEEvent — represents a single SSE event (event, data, id fields)
- InvokeRequest / InvokeResponse — synchronous invocation types
- StreamRequest / StreamEvent — streaming invocation types
