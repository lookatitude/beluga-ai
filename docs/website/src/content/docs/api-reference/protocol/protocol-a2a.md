---
title: "A2A API — Agent-to-Agent Protocol"
description: "A2A protocol API reference for Beluga AI. Agent-to-Agent collaboration with Agent Cards, task lifecycle management, and official SDK integration."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "A2A API, Agent-to-Agent, AgentCard, task lifecycle, multi-agent, collaboration, SDK, Beluga AI, Go, reference"
---

## a2a

```go
import "github.com/lookatitude/beluga-ai/protocol/a2a"
```

Package a2a implements the Agent-to-Agent (A2A) protocol for multi-agent
collaboration. It provides an A2A server that exposes a Beluga agent as a
remote agent with an Agent Card, task lifecycle management, and HTTP endpoints,
as well as a client for connecting to remote A2A agents.

The A2A protocol enables agents to discover each other via Agent Cards served
at /.well-known/agent.json, submit tasks asynchronously, poll for task status,
and cancel running tasks. Tasks follow a lifecycle: submitted -> working ->
completed/failed/canceled.

## Server

A2AServer exposes a Beluga agent as a remote A2A agent via HTTP. It provides
endpoints for the Agent Card, task creation, status polling, and cancellation.

```go
card := a2a.AgentCard{
    Name:     "assistant",
    Endpoint: "http://localhost:9090",
}
srv := a2a.NewServer(myAgent, card)
srv.Serve(ctx, ":9090")
```

The server exposes the following HTTP endpoints:

- GET  /.well-known/agent.json — returns the Agent Card
- POST /tasks — creates a new task
- GET  /tasks/{id} — returns task status
- POST /tasks/{id}/cancel — cancels a running task

## Client

A2AClient connects to a remote A2A agent and provides methods for retrieving
the Agent Card, creating tasks, polling status, and cancellation.

```go
client := a2a.NewClient("http://localhost:9090")
card, err := client.GetCard(ctx)
task, err := client.CreateTask(ctx, a2a.TaskRequest{Input: "Hello"})
task, err = client.GetTask(ctx, task.ID)
```

## Remote Agent

NewRemoteAgent wraps an A2A endpoint as a local agent.Agent, enabling
transparent use of remote agents in local orchestration:

```go
remote, err := a2a.NewRemoteAgent("http://localhost:9090")
result, err := remote.Invoke(ctx, "Hello")
```

## Key Types

- A2AServer — serves a Beluga agent via the A2A protocol
- A2AClient — connects to remote A2A agents
- AgentCard — describes a remote agent's identity and capabilities
- Task — represents an A2A task with lifecycle state
- TaskStatus — lifecycle state (submitted, working, completed, failed, canceled)
- TaskRequest / TaskResponse / ErrorResponse — API message types

---

## sdk

```go
import "github.com/lookatitude/beluga-ai/protocol/a2a/sdk"
```

Package sdk provides integration between the official A2A Go SDK
(github.com/a2aproject/a2a-go) and Beluga's A2A protocol layer.
It bridges Beluga's agent.Agent interface with the SDK's server and client,
enabling exposure of Beluga agents as A2A remote agents and consumption
of remote A2A agents as Beluga agents.

This package is useful when you need full compliance with the official A2A
SDK behavior, including JSON-RPC messaging, event queues, and the standard
AgentCard format.

## Server

NewServer creates an A2A request handler and agent card from a Beluga agent.
The returned handler should be mounted on an HTTP server. The agent's tools
are automatically converted to A2A skills.

```go
handler, card := sdk.NewServer(myAgent, sdk.ServerConfig{
    Name:        "my-agent",
    Version:     "1.0.0",
    Description: "A helpful assistant",
    URL:         "http://localhost:9090",
})
http.ListenAndServe(":9090", handler)
```

## Client

NewRemoteAgent creates a Beluga agent.Agent that delegates to a remote A2A
agent via the official SDK client. It fetches the AgentCard to populate the
agent's identity.

```go
remote, err := sdk.NewRemoteAgent(ctx, "http://remote-agent:9090")
if err != nil {
    log.Fatal(err)
}
result, err := remote.Invoke(ctx, "Hello")
```

## Key Types

- ServerConfig — configuration for creating an A2A SDK server
- NewServer — creates handler and card from a Beluga agent
- NewRemoteAgent — wraps a remote A2A agent as a local agent.Agent
