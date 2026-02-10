// Package server provides HTTP framework adapters for serving Beluga AI agents.
// It defines a ServerAdapter interface backed by a registry of implementations,
// and includes a built-in stdlib net/http adapter. Framework-specific adapters
// for Gin, Fiber, Echo, Chi, gRPC, Connect-Go, and Huma are available as
// sub-packages under server/adapters/.
//
// # ServerAdapter Interface
//
// Every HTTP framework adapter implements the ServerAdapter interface:
//
//   - RegisterAgent(path, agent) — registers an agent with invoke/stream endpoints
//   - RegisterHandler(path, handler) — registers a raw http.Handler
//   - Serve(ctx, addr) — starts the server, blocks until done
//   - Shutdown(ctx) — gracefully shuts down the server
//
// # Registry Pattern
//
// Adapters register themselves via init() using the standard Beluga registry
// pattern. Import the adapter package to register it, then create instances
// via server.New:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/gin"
//
//	adapter, err := server.New("gin", server.Config{
//	    ReadTimeout:  10 * time.Second,
//	    WriteTimeout: 30 * time.Second,
//	})
//
// The built-in "stdlib" adapter is registered automatically.
//
// # Agent Handler
//
// NewAgentHandler creates an http.Handler that exposes an agent via two
// sub-paths:
//
//   - POST {prefix}/invoke — synchronous invocation returning JSON
//   - POST {prefix}/stream — SSE stream of agent events
//
// # SSE Support
//
// The package provides SSEWriter for writing Server-Sent Events. It handles
// event formatting, multi-line data per the SSE specification, reconnection
// hints, and keep-alive heartbeats.
//
// # Middleware and Hooks
//
// ServerAdapter supports middleware composition via ApplyMiddleware, which wraps
// adapters with cross-cutting behavior. The Hooks type provides optional
// callbacks (BeforeRequest, AfterRequest, OnError) that are composable via
// ComposeHooks.
//
// # Key Types
//
//   - ServerAdapter — interface for HTTP framework adapters
//   - Config — adapter configuration (timeouts, extras)
//   - Factory — creates a ServerAdapter from Config
//   - StdlibAdapter — built-in net/http implementation
//   - Middleware — wraps a ServerAdapter to add behavior
//   - Hooks — optional lifecycle callbacks for request processing
//   - SSEWriter / SSEEvent — Server-Sent Events support
//   - NewAgentHandler — creates HTTP handler for an agent
//   - InvokeRequest / InvokeResponse / StreamEvent — request/response types
package server
