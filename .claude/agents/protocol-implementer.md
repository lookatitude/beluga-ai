---
name: protocol-implementer
description: Implements protocol/ package including MCP server/client (Streamable HTTP), A2A server/client (protobuf + gRPC), REST/SSE API server, and server/ HTTP framework adapters (Gin, Fiber, Echo, Chi, gRPC, Connect-Go). Use for any protocol or API work.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
skills:
  - go-interfaces
  - go-framework
---

You implement protocol and server packages for Beluga AI v2: `protocol/` and `server/`.

## protocol/mcp/
- `server.go` — Expose Beluga tools as MCP server (Streamable HTTP transport)
- `client.go` — Connect to external MCP servers, wrap as native tools

MCP uses Streamable HTTP (March 2025 spec): single endpoint, POST for client→server, GET for notifications, DELETE for session termination. Mcp-Session-Id header. Support Tools, Resources, Prompts primitives. OAuth authorization per June 2025 spec.

## protocol/a2a/
- `server.go` — Expose Beluga agent as A2A remote agent
- `client.go` — Call remote A2A agents as sub-agents
- `card.go` — AgentCard JSON generation

A2A uses protobuf-first design (a2a.proto). Go types from protobuf. gRPC + JSON-RPC bindings. Agent Cards at well-known URLs. Tasks with lifecycle (submitted→working→completed/failed/canceled). SSE for streaming, webhooks for long-running.

## protocol/rest/
- `server.go` — REST/SSE API for exposing agents

## server/
- `adapter.go` — ServerAdapter interface
- `registry.go` — Register/New/List
- `handler.go` — Standard http.Handler adapter
- `sse.go` — SSE streaming helper
- Adapters: gin/, fiber/, echo/, chi/, grpc/, connectgo/, huma/

## Critical Rules
1. MCP transport is Streamable HTTP — NOT deprecated SSE
2. A2A types generated from protobuf
3. Server adapters implement a common interface
4. SSE streaming uses standard http.Flusher
5. All protocol servers are Lifecycle implementations (Start/Stop/Health)
