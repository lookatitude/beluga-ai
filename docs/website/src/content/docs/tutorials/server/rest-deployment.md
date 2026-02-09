---
title: Deploying Agents as a REST API
description: Expose Beluga AI agents as a production-ready REST API with streaming Server-Sent Events, concurrent request handling, and Docker deployment.
---

Having a capable agent is step one; making it accessible to users is step two. The `server` package provides HTTP framework adapters, while the `protocol` package handles REST/SSE transport. This tutorial shows how to wrap agents in HTTP handlers with both synchronous and streaming responses.

## What You Will Build

A REST API server that exposes an agent via two endpoints: a synchronous `/chat` endpoint and a streaming `/stream` endpoint using Server-Sent Events (SSE). You will handle concurrent requests, add middleware, and containerize the service.

## Prerequisites

- Familiarity with the `agent` and `llm` packages
- Basic Go HTTP knowledge

## Step 1: Define Request and Response Types

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

Wrap the LLM `Generate` call in an HTTP handler:

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

Streaming provides a better user experience. Use SSE to stream response chunks:

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

Add standard HTTP middleware for authentication and request logging:

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

Create a multi-stage `Dockerfile` for a minimal production image:

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

- [MCP Tool Server](/tutorials/server/mcp-tools) -- Expose tools via the Model Context Protocol
- [Content Moderation](/tutorials/safety/content-moderation) -- Add safety guardrails to your API
