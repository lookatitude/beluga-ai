---
title: "Server Adapters"
description: "HTTP framework adapters: Gin, Fiber, Echo, Chi, gRPC, Connect, Huma"
---

## server

```go
import "github.com/lookatitude/beluga-ai/server"
```

Package server provides HTTP framework adapters for serving Beluga AI agents.
It defines a ServerAdapter interface backed by a registry of implementations,
and includes a built-in stdlib net/http adapter. Framework-specific adapters
for Gin, Fiber, Echo, Chi, gRPC, Connect-Go, and Huma are available as
sub-packages under server/adapters/.

## ServerAdapter Interface

Every HTTP framework adapter implements the ServerAdapter interface:

- RegisterAgent(path, agent) — registers an agent with invoke/stream endpoints
- RegisterHandler(path, handler) — registers a raw http.Handler
- Serve(ctx, addr) — starts the server, blocks until done
- Shutdown(ctx) — gracefully shuts down the server

## Registry Pattern

Adapters register themselves via init() using the standard Beluga registry
pattern. Import the adapter package to register it, then create instances
via server.New:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/gin"

adapter, err := server.New("gin", server.Config{
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 30 * time.Second,
})
```

The built-in "stdlib" adapter is registered automatically.

## Agent Handler

NewAgentHandler creates an http.Handler that exposes an agent via two
sub-paths:

- POST {prefix}/invoke — synchronous invocation returning JSON
- POST {prefix}/stream — SSE stream of agent events

## SSE Support

The package provides SSEWriter for writing Server-Sent Events. It handles
event formatting, multi-line data per the SSE specification, reconnection
hints, and keep-alive heartbeats.

## Middleware and Hooks

ServerAdapter supports middleware composition via ApplyMiddleware, which wraps
adapters with cross-cutting behavior. The Hooks type provides optional
callbacks (BeforeRequest, AfterRequest, OnError) that are composable via
ComposeHooks.

## Key Types

- ServerAdapter — interface for HTTP framework adapters
- Config — adapter configuration (timeouts, extras)
- Factory — creates a ServerAdapter from Config
- StdlibAdapter — built-in net/http implementation
- Middleware — wraps a ServerAdapter to add behavior
- Hooks — optional lifecycle callbacks for request processing
- SSEWriter / SSEEvent — Server-Sent Events support
- NewAgentHandler — creates HTTP handler for an agent
- InvokeRequest / InvokeResponse / StreamEvent — request/response types

---

## chi

```go
import "github.com/lookatitude/beluga-ai/server/adapters/chi"
```

Package chi provides a Chi-based ServerAdapter for the Beluga AI server
package. It wraps the github.com/go-chi/chi/v5 router, enabling Beluga
agents to be served using Chi's lightweight, composable routing.

The adapter registers itself as "chi" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/chi"

adapter, err := server.New("chi", server.Config{})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":8080")
```

The underlying chi.Router is accessible via the Router() method for advanced
configuration such as adding Chi middleware or custom routes.

## Key Types

- Adapter — implements server.ServerAdapter using Chi
- New — creates a new Chi adapter with the given configuration

---

## connect

```go
import "github.com/lookatitude/beluga-ai/server/adapters/connect"
```

Package connect provides a Connect-Go based ServerAdapter for the Beluga AI
server package. Connect-Go enables HTTP/1.1 and HTTP/2 communication that is
compatible with gRPC, gRPC-Web, and Connect protocol clients.

The adapter registers itself as "connect" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/connect"

adapter, err := server.New("connect", server.Config{})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":8080")
```

The adapter uses the standard net/http server under the hood, supporting both
HTTP/1.1 and HTTP/2. This makes it compatible with Connect, gRPC, and gRPC-Web
clients without requiring separate servers.

The underlying http.ServeMux is accessible via the Mux() method for advanced
configuration such as registering Connect-Go service handlers directly.

## Key Types

- Adapter — implements server.ServerAdapter using Connect-Go
- New — creates a new Connect-Go adapter with the given configuration

---

## echo

```go
import "github.com/lookatitude/beluga-ai/server/adapters/echo"
```

Package echo provides an Echo-based ServerAdapter for the Beluga AI server
package. It wraps the github.com/labstack/echo/v4 HTTP framework, enabling
Beluga agents to be served using Echo's routing and middleware ecosystem.

The adapter registers itself as "echo" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/echo"

adapter, err := server.New("echo", server.Config{})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":8080")
```

The underlying echo.Echo instance is accessible via the Echo() method for
advanced configuration such as adding Echo-specific middleware or custom
routes.

## Key Types

- Adapter — implements server.ServerAdapter using Echo
- New — creates a new Echo adapter with the given configuration

---

## fiber

```go
import "github.com/lookatitude/beluga-ai/server/adapters/fiber"
```

Package fiber provides a Fiber v3-based ServerAdapter for the Beluga AI
server package. It wraps the github.com/gofiber/fiber/v3 HTTP framework,
enabling Beluga agents to be served using Fiber's high-performance routing.

The adapter registers itself as "fiber" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/fiber"

adapter, err := server.New("fiber", server.Config{})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":8080")
```

The underlying fiber.App is accessible via the App() method for advanced
configuration such as adding Fiber-specific middleware or custom routes.

## Key Types

- Adapter — implements server.ServerAdapter using Fiber v3
- New — creates a new Fiber adapter with the given configuration

---

## gin

```go
import "github.com/lookatitude/beluga-ai/server/adapters/gin"
```

Package gin provides a Gin-based ServerAdapter for the Beluga AI server
package. It wraps the github.com/gin-gonic/gin HTTP framework, enabling
Beluga agents to be served using Gin's routing and middleware ecosystem.

The adapter registers itself as "gin" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/gin"

adapter, err := server.New("gin", server.Config{})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":8080")
```

The underlying gin.Engine is accessible via the Engine() method for advanced
configuration such as adding Gin-specific middleware or custom routes.

## Key Types

- Adapter — implements server.ServerAdapter using Gin
- New — creates a new Gin adapter with the given configuration

---

## grpc

```go
import "github.com/lookatitude/beluga-ai/server/adapters/grpc"
```

Package grpc provides a gRPC-based ServerAdapter for the Beluga AI server
package. It exposes agents via unary (Invoke) and server-streaming (Stream)
RPCs using JSON encoding over gRPC. No .proto file is required — the service
descriptor is defined programmatically.

The adapter registers itself as "grpc" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/grpc"

adapter, err := server.New("grpc", server.Config{})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":50051")
```

The gRPC service exposes the following methods under beluga.AgentService:

- Invoke (unary) — synchronous agent invocation
- Stream (server-streaming) — streaming agent events

Requests and responses use JSON encoding. The ClientCodecOption function
returns a gRPC dial option for connecting with the JSON codec.

Note: RegisterHandler is not supported for gRPC adapters. Use RegisterAgent
to expose agents.

The underlying grpc.Server is accessible via the Server() method for
advanced configuration such as adding interceptors.

## Key Types

- Adapter — implements server.ServerAdapter using gRPC
- New — creates a new gRPC adapter with the given configuration
- InvokeRequest / InvokeResponse — unary RPC message types
- StreamEvent — streaming RPC event type
- ClientCodecOption — returns a dial option for JSON codec clients

---

## huma

```go
import "github.com/lookatitude/beluga-ai/server/adapters/huma"
```

Package huma provides a Huma-based ServerAdapter for the Beluga AI server
package. Huma is an OpenAPI-first framework that wraps standard net/http with
automatic API documentation generation.

The adapter registers itself as "huma" in the server registry via init().
Import this package to make the adapter available:

```go
import _ "github.com/lookatitude/beluga-ai/server/adapters/huma"

adapter, err := server.New("huma", server.Config{
    Extra: map[string]any{
        "title":   "My API",
        "version": "2.0.0",
    },
})
if err != nil {
    log.Fatal(err)
}
adapter.RegisterAgent("/chat", myAgent)
adapter.Serve(ctx, ":8080")
```

The Config.Extra map supports the following keys:

- "title" (string) — API title for OpenAPI docs (default: "Beluga AI")
- "version" (string) — API version for OpenAPI docs (default: "1.0.0")

The underlying huma.API and http.ServeMux are accessible via the API() and
Mux() methods for advanced configuration such as registering Huma operations
or custom routes.

## Key Types

- Adapter — implements server.ServerAdapter using Huma
- New — creates a new Huma adapter with the given configuration
