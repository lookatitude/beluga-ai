// Package rest provides a REST/SSE API server for exposing Beluga agents over
// HTTP. It supports both synchronous invocation and real-time streaming via
// Server-Sent Events (SSE).
//
// Agents are registered at path prefixes and automatically get two endpoints:
//
//   - POST /{path}/invoke — synchronous invocation returning a JSON response
//   - POST /{path}/stream — streaming invocation via Server-Sent Events
//
// # Usage
//
//	srv := rest.NewServer()
//	if err := srv.RegisterAgent("assistant", myAgent); err != nil {
//	    log.Fatal(err)
//	}
//	srv.Serve(ctx, ":8080")
//
// Clients can then invoke the agent synchronously:
//
//	// POST /assistant/invoke
//	// {"input": "Hello"}
//	// Response: {"result": "Hi there!"}
//
// Or stream responses via SSE:
//
//	// POST /assistant/stream
//	// {"input": "Hello"}
//	// Response: text/event-stream with agent events
//
// # SSE Support
//
// The package includes SSEWriter for writing Server-Sent Events to HTTP
// responses. It handles event formatting, multi-line data splitting, and
// connection keep-alive heartbeats per the SSE specification.
//
//	sse, err := rest.NewSSEWriter(w)
//	if err != nil {
//	    // response writer does not support flushing
//	}
//	sse.WriteEvent(rest.SSEEvent{Event: "message", Data: "hello"})
//	sse.WriteHeartbeat()
//
// # Key Types
//
//   - RESTServer — serves Beluga agents as REST/SSE HTTP endpoints
//   - SSEWriter — writes Server-Sent Events to an HTTP response
//   - SSEEvent — represents a single SSE event (event, data, id fields)
//   - InvokeRequest / InvokeResponse — synchronous invocation types
//   - StreamRequest / StreamEvent — streaming invocation types
package rest
