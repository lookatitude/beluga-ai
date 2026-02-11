---
title: Omni-Channel Messaging Gateway
description: Build a unified messaging gateway that handles messages from WhatsApp, SMS, and Slack through a single agent interface with shared conversation history.
---

Users communicate across multiple channels -- WhatsApp, SMS, Slack, and more. Building a separate bot for each channel creates maintenance overhead and inconsistent experiences. A unified messaging gateway solves this by normalizing incoming messages from different providers into a common format, routing them through a single AI agent, and dispatching responses back through the correct channel. This architecture follows the same interface-first design used throughout Beluga AI: define a `ChannelProvider` interface, and each messaging platform becomes a pluggable implementation.

## What You Will Build

A multi-channel gateway that normalizes messages from different sources, routes them through a single AI agent, and sends responses back through the correct channel provider. Users get a consistent experience whether they contact support via SMS or WhatsApp.

## Prerequisites

- Completion of the [WhatsApp Bot](/tutorials/messaging/whatsapp-bot) tutorial
- API keys for multiple messaging services (Twilio for SMS/WhatsApp)

## Architecture

```
WhatsApp --> |                  |                  |
SMS      --> | Gateway (Normalize) --> Agent --> Send Reply
Slack    --> |                  |                  |
                 |                         |
            Shared Memory           Provider Router
```

## Step 1: Define the Gateway Interface

Create a provider-agnostic messaging interface. The `NormalizedMessage` struct decouples your agent logic from any specific messaging platform's payload format. The `UserID` field is a canonical identifier (such as a phone number) that remains the same regardless of which channel the user contacts from -- this is what enables cross-channel conversation continuity. The `ChannelProvider` interface has only two methods (`Send` and `Channel`), keeping the contract minimal for new provider implementations.

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "strings"
    "sync"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// NormalizedMessage represents a channel-agnostic message.
type NormalizedMessage struct {
    UserID    string // Canonical user identifier.
    Channel   string // "whatsapp", "sms", "slack".
    Body      string
    RawFrom   string // Original sender address.
    Metadata  map[string]any
}

// ChannelProvider handles sending messages on a specific channel.
type ChannelProvider interface {
    Send(ctx context.Context, to, body string) error
    Channel() string
}
```

## Step 2: Build the Gateway

The gateway maps channel names to providers and routes responses. The `providers` map uses the channel name as the key, enabling O(1) lookup when routing responses back to the correct provider. The `sessions` map is keyed by canonical user ID (not by channel), which means a user who starts a conversation on SMS and continues on WhatsApp shares the same conversation history. The `sync.RWMutex` protects the sessions map because multiple webhook handlers may process messages concurrently.

```go
// Gateway handles messages from multiple channels.
type Gateway struct {
    providers map[string]ChannelProvider
    model     llm.ChatModel
    mu        sync.RWMutex
    sessions  map[string][]schema.Message // keyed by canonical user ID.
}

func NewGateway(model llm.ChatModel) *Gateway {
    return &Gateway{
        providers: make(map[string]ChannelProvider),
        model:     model,
        sessions:  make(map[string][]schema.Message),
    }
}

func (g *Gateway) RegisterProvider(p ChannelProvider) {
    g.providers[p.Channel()] = p
}
```

## Step 3: Normalize Incoming Messages

Different providers send data in different formats. Twilio sends form-encoded webhook payloads; Slack sends JSON; other providers have their own conventions. The normalization layer translates each provider's format into the common `NormalizedMessage` struct, isolating the rest of the gateway from provider-specific parsing logic. The channel is inferred from the sender address format -- Twilio prefixes WhatsApp numbers with `whatsapp:`, making it easy to distinguish from SMS on the same webhook endpoint.

```go
// normalizeFromTwilio parses a Twilio webhook into a NormalizedMessage.
func normalizeFromTwilio(r *http.Request) (NormalizedMessage, error) {
    if err := r.ParseForm(); err != nil {
        return NormalizedMessage{}, fmt.Errorf("parse form: %w", err)
    }

    from := r.FormValue("From")
    body := r.FormValue("Body")

    if from == "" || body == "" {
        return NormalizedMessage{}, fmt.Errorf("missing required fields")
    }

    // Determine channel from the sender address format.
    channel := "sms"
    if strings.HasPrefix(from, "whatsapp:") {
        channel = "whatsapp"
    }

    // Extract canonical user ID (phone number without prefix).
    userID := strings.TrimPrefix(from, "whatsapp:")

    return NormalizedMessage{
        UserID:  userID,
        Channel: channel,
        Body:    body,
        RawFrom: from,
    }, nil
}
```

## Step 4: Process and Route Messages

Process messages through the shared agent and route responses. The conversation history is loaded and updated under a mutex lock because multiple webhook handlers may be processing messages from the same user concurrently (for example, if the user sends two messages in quick succession). The system prompt is prepended fresh on each call rather than stored in history, keeping the history clean and allowing the prompt to be updated without invalidating existing sessions. The response is routed back through the original channel using the `RawFrom` address, ensuring the reply reaches the user on the same platform they used.

```go
func (g *Gateway) ProcessMessage(ctx context.Context, msg NormalizedMessage) error {
    // Load conversation history by canonical user ID.
    // This means the same user on WhatsApp and SMS shares history.
    g.mu.Lock()
    history := g.sessions[msg.UserID]
    history = append(history, schema.NewHumanMessage(msg.Body))
    g.sessions[msg.UserID] = history
    g.mu.Unlock()

    // Build messages with system prompt.
    msgs := make([]schema.Message, 0, len(history)+1)
    msgs = append(msgs, schema.NewSystemMessage(
        "You are a helpful support assistant. "+
            "Be concise and professional. "+
            "The user may switch between messaging channels.",
    ))
    msgs = append(msgs, history...)

    // Generate response.
    aiMsg, err := g.model.Generate(ctx, msgs)
    if err != nil {
        return fmt.Errorf("generate: %w", err)
    }

    response := aiMsg.Text()

    // Update history.
    g.mu.Lock()
    g.sessions[msg.UserID] = append(g.sessions[msg.UserID], schema.NewAIMessage(response))
    g.mu.Unlock()

    // Route response to the correct channel.
    provider, ok := g.providers[msg.Channel]
    if !ok {
        return fmt.Errorf("unsupported channel: %s", msg.Channel)
    }

    return provider.Send(ctx, msg.RawFrom, response)
}
```

## Step 5: Create Webhook Handlers

Set up HTTP handlers for each incoming channel. The handler returns HTTP 200 immediately and processes the message asynchronously in a goroutine. This is important for webhook-based integrations because most providers (Twilio, Slack) expect a quick response to the webhook request and will retry if the response is slow. The goroutine uses `context.Background()` rather than `r.Context()` because the processing must continue after the HTTP response is sent -- `r.Context()` is cancelled when the handler returns.

```go
func (g *Gateway) HandleTwilio(w http.ResponseWriter, r *http.Request) {
    msg, err := normalizeFromTwilio(r)
    if err != nil {
        slog.Error("normalize failed", "error", err)
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    slog.Info("message received",
        "user", msg.UserID,
        "channel", msg.Channel,
        "body", msg.Body,
    )

    go func() {
        if err := g.ProcessMessage(context.Background(), msg); err != nil {
            slog.Error("process failed", "error", err, "user", msg.UserID)
        }
    }()

    w.WriteHeader(http.StatusOK)
}
```

## Step 6: Run the Gateway

```go
func main() {
    model, err := llm.New("openai", llm.ProviderConfig{
        Options: map[string]any{
            "api_key": os.Getenv("OPENAI_API_KEY"),
            "model":   "gpt-4o-mini",
        },
    })
    if err != nil {
        slog.Error("model creation failed", "error", err)
        return
    }

    gw := NewGateway(model)

    // Register channel providers (implementations would wrap Twilio, Slack SDKs).
    // gw.RegisterProvider(newTwilioWhatsAppProvider())
    // gw.RegisterProvider(newTwilioSMSProvider())
    // gw.RegisterProvider(newSlackProvider())

    http.HandleFunc("/twilio", gw.HandleTwilio)
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    addr := ":8080"
    slog.Info("gateway starting", "addr", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        slog.Error("server error", "error", err)
    }
}
```

## Shared Memory Across Channels

The gateway uses the canonical user ID (phone number) to key conversation history. This means a user who starts a conversation on SMS and continues on WhatsApp sees a seamless experience -- the agent remembers what was discussed regardless of which channel the user switches to.

For production deployments, replace the in-memory `sessions` map with a persistent store. The `memory` package provides ready-made store implementations (Redis, PostgreSQL, SQLite) that support the same session-keyed access pattern with TTL-based expiration and concurrent access safety.

```go
// Use Redis or PostgreSQL for production session storage.
// The memory package provides ready-made store implementations.
import "github.com/lookatitude/beluga-ai/memory"
```

## Verification

1. Send an SMS to the bot. Verify the agent responds.
2. Send a WhatsApp message from the same phone number. Verify the agent responds and recalls the SMS conversation.
3. Verify both channels receive responses through their respective providers.

## Next Steps

- [Content Moderation](/tutorials/safety/content-moderation) -- Filter messages across all channels before processing
- [Human-in-the-Loop](/tutorials/safety/human-in-loop) -- Escalate sensitive requests to human agents
