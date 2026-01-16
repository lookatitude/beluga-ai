# Quickstart Guide: Twilio API Integration

**Date**: 2025-01-07  
**Feature**: Twilio API Integration (001-twilio-integration)  
**Version**: 1.0.0

## Overview

This guide provides step-by-step instructions for setting up and using Twilio Voice API and Conversations API integration with Beluga AI Framework.

## Prerequisites

- Go 1.24+ installed
- Twilio account with API credentials (Account SID, Auth Token)
- Phone number provisioned in Twilio (for Voice API)
- SMS/WhatsApp enabled phone number (for Conversations API)
- Public webhook endpoint (or webhook tunneling service like ngrok for development)

## Setup

### 1. Install Dependencies

```bash
go get github.com/twilio/twilio-go@v1.29.1
go get github.com/lookatitude/beluga-ai/pkg/voice/backend
go get github.com/lookatitude/beluga-ai/pkg/messaging
```

### 2. Configure Twilio Credentials

Set environment variables:

```bash
export TWILIO_ACCOUNT_SID="AC1234567890abcdef"
export TWILIO_AUTH_TOKEN="your_auth_token"
export TWILIO_PHONE_NUMBER="+15551234567"
export TWILIO_WEBHOOK_URL="https://your-domain.com/webhooks/twilio"
```

Or use configuration file (`config.yaml`):

```yaml
twilio:
  account_sid: "AC1234567890abcdef"
  auth_token: "your_auth_token"
  phone_number: "+15551234567"
  webhook_url: "https://your-domain.com/webhooks/twilio"
```

## Voice API Integration

### Basic Voice Agent

Create a voice-enabled agent that handles phone calls:

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
    vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    ctx := context.Background()
    
    // Create Twilio voice backend
    config := &vbiface.Config{
        Provider: "twilio",
        AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
        AuthToken: os.Getenv("TWILIO_AUTH_TOKEN"),
        PhoneNumber: os.Getenv("TWILIO_PHONE_NUMBER"),
    }
    
    backend, err := backend.NewBackend(ctx, "twilio", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start backend
    if err := backend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    // Create agent
    llm, _ := llms.NewLLM(ctx, "openai", &llms.Config{...})
    agent, _ := agents.NewAgent(ctx, llm, &agents.Config{...})
    
    // Create session configuration
    sessionConfig := &vbiface.SessionConfig{
        AgentInstance: agent,
    }
    
    // Create voice session (for outbound call)
    session, err := backend.CreateSession(ctx, sessionConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start session
    if err := session.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Session handles call automatically
    // For inbound calls, use webhook handler (see below)
}
```

### Inbound Call Webhook Handler

Handle inbound calls via webhook:

```go
func handleInboundCall(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Parse webhook data
    webhookData := parseWebhookData(r)
    
    // Get backend instance
    backend := getBackendInstance()
    
    // Handle inbound call (creates session automatically)
    session, err := backend.HandleInboundCall(ctx, webhookData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Session is now active and handling the call
    w.WriteHeader(http.StatusOK)
}
```

## Conversations API Integration

### Basic Messaging Agent

Create a messaging agent that handles SMS/WhatsApp:

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/messaging"
    messagingiface "github.com/lookatitude/beluga-ai/pkg/messaging/iface"
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/memory"
)

func main() {
    ctx := context.Background()
    
    // Create Twilio messaging backend
    config := &messagingiface.Config{
        Provider: "twilio",
        AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
        AuthToken: os.Getenv("TWILIO_AUTH_TOKEN"),
    }
    
    backend, err := messaging.NewBackend(ctx, "twilio", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start backend
    if err := backend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    // Create agent with memory
    llm, _ := llms.NewLLM(ctx, "openai", &llms.Config{...})
    mem, _ := memory.NewMemory(memory.MemoryTypeVectorStore, ...)
    agent, _ := agents.NewAgent(ctx, llm, &agents.Config{
        Memory: mem,
    })
    
    // Create conversation
    convConfig := &messagingiface.ConversationConfig{
        FriendlyName: "Customer Support",
        AgentInstance: agent,
    }
    
    conversation, err := backend.CreateConversation(ctx, convConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Receive messages
    messageChan, err := backend.ReceiveMessages(ctx, conversation.ID())
    if err != nil {
        log.Fatal(err)
    }
    
    // Process messages
    for message := range messageChan {
        // Agent processes message automatically
        // Response is sent via agent callback
    }
}
```

### Message Webhook Handler

Handle incoming messages via webhook:

```go
func handleMessageWebhook(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Parse webhook event
    event := parseWebhookEvent(r)
    
    // Get backend instance
    backend := getMessagingBackendInstance()
    
    // Handle webhook
    if err := backend.HandleWebhook(ctx, event); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}
```

## Integration with Agents and Memory

### Voice Agent with Memory

```go
// Create memory instance
memory, _ := memory.NewMemory(memory.MemoryTypeVectorStore, ...)

// Create agent with memory
agent, _ := agents.NewAgent(ctx, llm, &agents.Config{
    Memory: memory,
})

// Memory is automatically used for conversation context
```

### Messaging Agent with Multi-Channel Memory

```go
// Create memory instance with vector store
vectorStore, _ := vectorstores.NewVectorStore(ctx, "pgvector", ...)
memory, _ := memory.NewMemory(memory.MemoryTypeVectorStore, 
    memory.WithVectorStore(vectorStore),
)

// Create agent
agent, _ := agents.NewAgent(ctx, llm, &agents.Config{
    Memory: memory,
})

// Memory persists across channels (SMS, WhatsApp)
// Context is maintained when customer switches channels
```

## Webhook Configuration

### Twilio Console Configuration

1. Log in to Twilio Console
2. Navigate to Phone Numbers → Manage → Active Numbers
3. Select your phone number
4. Configure webhooks:
   - **Voice**: Set "A CALL COMES IN" webhook URL
   - **Messaging**: Set webhook URL in Conversations API configuration

### Webhook Endpoint Setup

```go
func setupWebhookHandlers() {
    http.HandleFunc("/webhooks/twilio/voice/status", handleCallStatus)
    http.HandleFunc("/webhooks/twilio/voice/stream", handleStreamEvent)
    http.HandleFunc("/webhooks/twilio/conversations/events", handleConversationEvent)
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Development with ngrok

For local development, use ngrok to expose webhook endpoint:

```bash
ngrok http 8080
# Use ngrok URL in Twilio webhook configuration
```

## Common Use Cases

### Use Case 1: Customer Support Voice Agent

```go
// Create voice agent for customer support
agent := createCustomerSupportAgent(ctx)

// Backend handles inbound calls automatically
// Agent processes speech and responds
```

### Use Case 2: Multi-Channel Support Agent

```go
// Create messaging agent
agent := createSupportAgent(ctx)

// Agent handles SMS and WhatsApp
// Context is maintained across channels
```

### Use Case 3: RAG-Enabled Voice Agent

```go
// Create vector store for knowledge base
vectorStore := createKnowledgeBase(ctx)

// Create agent with RAG
agent := createRAGAgent(ctx, vectorStore)

// Agent can answer questions using knowledge base
```

## Error Handling

```go
// Handle errors
if err != nil {
    if errors.Is(err, ErrCodeRateLimit) {
        // Retry with backoff
        time.Sleep(time.Second * 2)
        // Retry operation
    } else if errors.Is(err, ErrCodeAuthError) {
        // Check credentials
        log.Fatal("Invalid Twilio credentials")
    } else {
        // Log and handle other errors
        log.Printf("Error: %v", err)
    }
}
```

## Observability

### Metrics

Access metrics via OpenTelemetry:

```go
// Metrics are automatically exported
// View in your observability platform (Prometheus, etc.)
```

### Tracing

Traces are automatically created for all operations:

```go
// View traces in your tracing backend (Jaeger, etc.)
// All operations include trace IDs and span IDs
```

## Next Steps

- See [Voice Backend API Contract](./contracts/voice-backend-api.md) for detailed API documentation
- See [Conversational Backend API Contract](./contracts/conversational-backend-api.md) for messaging API documentation
- See [Webhook API Contract](./contracts/webhook-api.md) for webhook handling
- See examples in `examples/voice/twilio/`
