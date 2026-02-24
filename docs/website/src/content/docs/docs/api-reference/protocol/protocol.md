---
title: "Protocol API — MCP, A2A, REST Overview"
description: "Protocol package API reference for Beluga AI. MCP, A2A, and REST/SSE protocol abstractions for agent and tool interoperability."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "protocol API, MCP, A2A, REST, SSE, interoperability, OpenAI Agents, JSON-RPC, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/protocol"
```

Package protocol provides interoperability protocols for the Beluga AI
framework, enabling AI agents and tools to communicate across system
boundaries using standardized protocols.

Protocol implements three complementary communication mechanisms:

- MCP (Model Context Protocol) for tool and resource sharing between AI
  systems via JSON-RPC 2.0 over Streamable HTTP transport.
- A2A (Agent-to-Agent) for multi-agent collaboration with task lifecycle
  management, Agent Cards, and HTTP-based communication.
- REST/SSE for exposing Beluga agents over HTTP with both synchronous
  invocation and real-time streaming via Server-Sent Events.

Additionally, the openai_agents sub-package provides a compatibility layer
for the OpenAI Agents SDK wire format.

## Sub-packages

- protocol/mcp — MCP server and client using Streamable HTTP transport
- protocol/mcp/sdk — Integration with the official MCP Go SDK
- protocol/mcp/registry — MCP server discovery and tool aggregation
- protocol/mcp/providers/composio — Composio MCP integration
- protocol/a2a — A2A server and client for agent collaboration
- protocol/a2a/sdk — Integration with the official A2A Go SDK
- protocol/openai_agents — OpenAI Agents SDK compatibility layer
- protocol/rest — REST/SSE API server for exposing agents over HTTP

## Example

Expose an agent over all three protocols:

```go
// MCP: share tools with other AI systems
mcpSrv := mcp.NewServer("my-tools", "1.0.0")
mcpSrv.AddTool(searchTool)
go mcpSrv.Serve(ctx, ":8081")

// A2A: enable multi-agent collaboration
card := a2a.AgentCard{Name: "assistant", Endpoint: "http://localhost:8082"}
a2aSrv := a2a.NewServer(myAgent, card)
go a2aSrv.Serve(ctx, ":8082")

// REST/SSE: serve an HTTP API
restSrv := rest.NewServer()
restSrv.RegisterAgent("assistant", myAgent)
restSrv.Serve(ctx, ":8080")
```
