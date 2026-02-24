---
title: Deploying Agents as a REST API
description: "Expose Beluga AI agents as a REST API in Go with synchronous and streaming SSE endpoints, concurrent request handling, middleware, and Docker containerization."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, REST API, SSE, streaming, deployment, Docker, HTTP server"
---

Having a capable agent is step one; making it accessible to users is step two. The `server` package provides HTTP framework adapters, while the `protocol` package handles REST/SSE transport. This tutorial shows how to wrap agents in HTTP handlers with both synchronous and streaming responses. The synchronous endpoint suits simple request-response clients, while the SSE endpoint provides real-time token streaming for chat UIs where users expect to see the response appear progressively.

## What You Will Build

A REST API server that exposes an agent via two endpoints: a synchronous `/chat` endpoint and a streaming `/stream` endpoint using Server-Sent Events (SSE). You will handle concurrent requests, add middleware, and containerize the service.

## Prerequisites

- Familiarity with the `agent` and `llm` packages
- Basic Go HTTP knowledge

## Step 1: Define Request and Response Types

The request and response types are simple JSON-serializable structs. The `SessionID` field is optional -- when provided, it enables conversation continuity across requests by allowing the server to load and maintain per-session message history. Keeping the API surface minimal makes it easier for clients to integrate.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "os"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

type ChatRequest struct {
    Input     string `json:"input"`
    SessionID string `json:"session_id,omitempty"`
}

type ChatResponse struct {
    Output string `json:"output"`
    Error  string `json:"error,omitempty"`
}
```

## Step 2: Create the HTTP Handler (Synchronous)

Wrap the LLM `Generate` call in an HTTP handler. The handler uses `r.Context()` for the LLM call, which means client disconnections automatically cancel in-flight LLM requests -- this prevents wasting tokens on responses nobody will read. Each HTTP request is handled in its own goroutine by Go's `net/http` server, so concurrent requests are naturally supported without additional synchronization at this level.

```go
type Server struct {
    model llm.ChatModel
}

func (s *Server) HandleChat(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    msgs := []schema.Message{
        schema.NewHumanMessage(req.Input),
    }

    aiMsg, err := s.model.Generate(r.Context(), msgs)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(ChatResponse{Error: err.Error()})
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(ChatResponse{Output: aiMsg.Text()})
}
```

## Step 3: Streaming with Server-Sent Events

Streaming provides a better user experience for chat interfaces because users see the response forming in real time rather than waiting for the entire generation to complete. SSE is the right transport for this because it works over standard HTTP, passes through proxies and load balancers without special configuration, and is natively supported by browser `EventSource` APIs.

The streaming handler uses `model.Stream()` which returns an `iter.Seq2[StreamChunk, error]` -- Beluga AI's standard streaming pattern. Each chunk's `Delta` field contains the incremental text, which is written as an SSE `data` event and flushed immediately. The `http.Flusher` interface check ensures the underlying `ResponseWriter` supports streaming; most Go HTTP servers do, but reverse proxies may buffer unless configured otherwise.

```go
func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    // Set SSE headers.
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    msgs := []schema.Message{
        schema.NewHumanMessage(req.Input),
    }

    for chunk, err := range s.model.Stream(r.Context(), msgs) {
        if err != nil {
            fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
            flusher.Flush()
            return
        }
        fmt.Fprintf(w, "data: %s\n\n", chunk.Delta)
        flusher.Flush()
    }

    fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
    flusher.Flush()
}
```

## Step 4: Middleware for Auth and Logging

Add standard HTTP middleware for authentication and request logging. These follow Go's idiomatic `func(http.Handler) http.Handler` middleware pattern, which composes naturally with `net/http` and third-party routers. The logging middleware uses `slog` (Go's structured logging package), consistent with Beluga AI's observability approach. The auth middleware uses a simple bearer token check -- in production, you would replace this with JWT validation or an API gateway.

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        slog.Info("request received",
            "method", r.Method,
            "path", r.URL.Path,
            "remote", r.RemoteAddr,
        )
        next.ServeHTTP(w, r)
    })
}

func authMiddleware(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("Authorization")
            if key != "Bearer "+apiKey {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Step 5: Wire It Together

The main function creates the model via the registry pattern (`llm.New("openai", ...)`), wires the handlers, and applies middleware. The health endpoint is important for container orchestration -- Kubernetes and Docker Compose use it to determine when the service is ready to receive traffic.

```go
func main() {
    model, err := llm.New("openai", llm.ProviderConfig{
        Options: map[string]any{
            "api_key": os.Getenv("OPENAI_API_KEY"),
            "model":   "gpt-4o",
        },
    })
    if err != nil {
        slog.Error("failed to create model", "error", err)
        os.Exit(1)
    }

    srv := &Server{model: model}

    mux := http.NewServeMux()
    mux.HandleFunc("/chat", srv.HandleChat)
    mux.HandleFunc("/stream", srv.HandleStream)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, `{"status":"ok"}`)
    })

    apiKey := os.Getenv("API_KEY")
    handler := loggingMiddleware(authMiddleware(apiKey)(mux))

    addr := ":8080"
    slog.Info("starting server", "addr", addr)
    if err := http.ListenAndServe(addr, handler); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

## Step 6: Docker Deployment

Create a multi-stage `Dockerfile` for a minimal production image. The multi-stage build compiles the Go binary in a full SDK image, then copies only the binary into a minimal Alpine image. `CGO_ENABLED=0` produces a statically linked binary that runs without libc dependencies. The `ca-certificates` package is required for HTTPS calls to LLM provider APIs.

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/server /usr/local/bin/server
EXPOSE 8080
CMD ["server"]
```

Build and run:

```bash
docker build -t beluga-server .
docker run -p 8080:8080 \
    -e OPENAI_API_KEY="sk-..." \
    -e API_KEY="my-secret-key" \
    beluga-server
```

## Testing the API

Test the synchronous endpoint:

```bash
curl -X POST http://localhost:8080/chat \
    -H "Authorization: Bearer my-secret-key" \
    -H "Content-Type: application/json" \
    -d '{"input": "What is the capital of France?"}'
```

Test the streaming endpoint:

```bash
curl -N -X POST http://localhost:8080/stream \
    -H "Authorization: Bearer my-secret-key" \
    -H "Content-Type: application/json" \
    -d '{"input": "Explain quantum computing"}'
```

## Verification

1. Start the server with `go run main.go`.
2. Send a synchronous request to `/chat`. Verify you receive a JSON response.
3. Send a streaming request to `/stream`. Verify you receive SSE data chunks.
4. Test without the `Authorization` header. Verify you receive a 401 response.
5. Build the Docker image and verify it runs.

## Next Steps

- [MCP Tool Server](/docs/tutorials/server/mcp-tools) -- Expose tools via the Model Context Protocol
- [Content Moderation](/docs/tutorials/safety/content-moderation) -- Add safety guardrails to your API
