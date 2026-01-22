# Deploying via REST

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll expose your Beluga AI agents and chains as a production-ready REST API. You'll learn how to wrap agents in HTTP handlers, implement real-time streaming using Server-Sent Events (SSE), and containerize the whole thing with Docker.

## Learning Objectives
- ✅ Wrap an Agent in an HTTP Handler
- ✅ Handle streaming responses (SSE)
- ✅ Manage concurrent requests
- ✅ Deploy with Docker

## Introduction
Welcome, colleague! Having a great agent is step one; making it accessible to your users is step two. Let's build a clean, scalable REST API around our AI logic so we can integrate it into any web or mobile frontend.

## Prerequisites

- [Creating Your First Agent](../../getting-started/03-first-agent.md)
- Basic Go HTTP knowledge

## Step 1: The Request/Response Model
```go
type ChatRequest struct {
    Input     string `json:"input"`
    SessionID string `json:"session_id"`
}

type ChatResponse struct {
    Output string `json:"output"`
    Error  string `json:"error,omitempty"`
}

## Step 2: HTTP Handler
func (s *Server) HandleChat(w http.ResponseWriter, r *http.Request) {
    var req ChatRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // 1. Load/Create Agent
    agent := s.agentFactory.GetAgent(req.SessionID)
    
    // 2. Invoke
    res, err := agent.Invoke(r.Context(), req.Input)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    
    // 3. Respond
    json.NewEncoder(w).Encode(ChatResponse{Output: res.(string)})
}
```

## Step 3: Streaming Responses (Server-Sent Events)

Streaming provides a better UX.
```go
func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
    // Set headers for SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Get agent stream
    stream, _ := agent.Stream(r.Context(), input)
    
    for chunk := range stream {
        // Format as SSE data
        fmt.Fprintf(w, "data: %s\n\n", chunk)
        w.(http.Flusher).Flush()
    }
}
```

## Step 4: Dockerizing

Create a `Dockerfile`:
FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o server main.go
CMD ["./server"]
```

## Step 5: Middleware (Auth/Logging)

Don't forget standard middleware!
```
mux.Handle("/chat", AuthMiddleware(LoggingMiddleware(http.HandlerFunc(HandleChat))))

## Verification

1. Start server: `go run main.go`.
2. Use `curl`:
```bash
   curl -X POST -d '{"input":"Hello"}' http://localhost:8080/chat
```
3. Test streaming with a browser or `curl -N`.

## Next Steps

- **[Building an MCP Server](./server-mcp-tools.md)** - Standardized Tool Protocol
- **[Production Deployment](../../getting-started/07-production-deployment.md)** - Cloud deployment
