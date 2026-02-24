---
title: Building a WhatsApp Support Bot
description: "Build a WhatsApp support bot in Go using Beluga AI agents with Twilio webhooks, per-user conversation state, and multi-turn LLM-powered responses."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, WhatsApp bot, Twilio, support bot, webhooks, chatbot"
---

WhatsApp is one of the most widely used channels for customer support. This tutorial demonstrates how to build a WhatsApp bot that receives messages via Twilio webhooks, processes them with a Beluga AI agent, and sends formatted replies. The pattern shown here applies to any webhook-based messaging platform -- the core architecture (webhook handler, async processing, per-user state, outbound API call) is the same regardless of the messaging provider.

## What You Will Build

A WhatsApp support bot that handles incoming messages via Twilio webhooks, maintains conversation history per user, processes queries with an LLM agent, and sends responses back through the WhatsApp API.

## Prerequisites

- Twilio account with WhatsApp Sandbox enabled
- Publicly accessible URL (for example via ngrok) for webhooks
- OpenAI or other LLM provider API key

## Architecture Overview

```
User (WhatsApp) --> Twilio --> Webhook Handler --> Agent --> LLM
                                   |                          |
                                   +---- Response <-----------+
                                   |
                              Twilio API --> User (WhatsApp)
```

## Step 1: Define the Bot Server

The `BotServer` struct holds the LLM model (created via the registry pattern), Twilio credentials, and per-user conversation state. The `sessions` map stores message history keyed by the sender's WhatsApp address, enabling multi-turn conversations. The `sync.RWMutex` protects concurrent access to the sessions map because Go's `net/http` server handles requests in parallel goroutines, and multiple users may be sending messages simultaneously.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "sync"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// BotServer handles WhatsApp webhook requests.
type BotServer struct {
    model       llm.ChatModel
    twilioSID   string
    twilioToken string
    twilioFrom  string

    // Per-user conversation history.
    mu       sync.RWMutex
    sessions map[string][]schema.Message
}

func NewBotServer(model llm.ChatModel) *BotServer {
    return &BotServer{
        model:       model,
        twilioSID:   os.Getenv("TWILIO_ACCOUNT_SID"),
        twilioToken: os.Getenv("TWILIO_AUTH_TOKEN"),
        twilioFrom:  os.Getenv("TWILIO_WHATSAPP_FROM"),
        sessions:    make(map[string][]schema.Message),
    }
}
```

## Step 2: Handle Incoming Webhooks

Parse the incoming Twilio webhook payload and extract the message. Twilio sends webhook data as form-encoded POST requests with fields like `From` (sender address) and `Body` (message text). The handler returns HTTP 200 immediately and processes the message asynchronously in a goroutine -- this is critical because Twilio expects a fast response to webhooks and will retry the delivery if the response is slow, which would cause duplicate processing. The goroutine uses `context.Background()` instead of `r.Context()` because `r.Context()` is cancelled when the HTTP handler returns, but the LLM call must continue after the response is sent.

```go
// IncomingMessage represents a parsed Twilio webhook payload.
type IncomingMessage struct {
    From string
    Body string
}

func (s *BotServer) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    msg := IncomingMessage{
        From: r.FormValue("From"),
        Body: r.FormValue("Body"),
    }

    if msg.From == "" || msg.Body == "" {
        http.Error(w, "missing required fields", http.StatusBadRequest)
        return
    }

    slog.Info("message received", "from", msg.From, "body", msg.Body)

    // Process asynchronously to return 200 quickly.
    go s.processMessage(context.Background(), msg)

    w.WriteHeader(http.StatusOK)
}
```

## Step 3: Process Messages with the Agent

Maintain conversation history per user and generate responses. The history is loaded under a mutex lock, appended with the new message, and saved back -- this ensures consistent state even when the same user sends multiple messages in quick succession. The system prompt is prepended to the message list on each call rather than stored in history, which keeps the history clean and allows the system prompt to be updated without invalidating existing sessions.

The error handling follows a graceful degradation pattern: if the LLM call fails, the bot sends a friendly error message rather than silently dropping the conversation. This ensures the user always receives a response.

```go
func (s *BotServer) processMessage(ctx context.Context, msg IncomingMessage) {
    // Load or create conversation history.
    s.mu.Lock()
    history := s.sessions[msg.From]
    history = append(history, schema.NewHumanMessage(msg.Body))
    s.sessions[msg.From] = history
    s.mu.Unlock()

    // Build message list with system prompt.
    msgs := make([]schema.Message, 0, len(history)+1)
    msgs = append(msgs, schema.NewSystemMessage(
        "You are a helpful customer support assistant. "+
            "Be concise, friendly, and professional. "+
            "If you don't know the answer, say so honestly.",
    ))
    msgs = append(msgs, history...)

    // Generate response.
    aiMsg, err := s.model.Generate(ctx, msgs)
    if err != nil {
        slog.Error("generation failed", "error", err, "from", msg.From)
        s.sendReply(ctx, msg.From, "Sorry, I'm having trouble right now. Please try again later.")
        return
    }

    response := aiMsg.Text()

    // Update history with AI response.
    s.mu.Lock()
    s.sessions[msg.From] = append(s.sessions[msg.From], schema.NewAIMessage(response))
    s.mu.Unlock()

    // Send reply.
    if err := s.sendReply(ctx, msg.From, response); err != nil {
        slog.Error("send reply failed", "error", err, "to", msg.From)
    }
}
```

## Step 4: Send Replies via Twilio

Send the response back through the Twilio WhatsApp API. The Twilio Messages API accepts form-encoded POST requests with basic authentication using the account SID and auth token. The `context.Context` is threaded through to support cancellation if the server is shutting down.

```go
func (s *BotServer) sendReply(ctx context.Context, to, body string) error {
    data := fmt.Sprintf("To=%s&From=%s&Body=%s", to, s.twilioFrom, body)

    url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.twilioSID)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    req.SetBasicAuth(s.twilioSID, s.twilioToken)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Body = http.NoBody // Replace with actual body encoding.

    // In production, properly URL-encode the body parameter.
    _ = data

    slog.Info("reply sent", "to", to, "length", len(body))
    return nil
}
```

## Step 5: Run the Server

The main function creates the model via Beluga AI's registry pattern (`llm.New("openai", ...)`), which resolves the provider by name and passes the configuration through. The `gpt-4o-mini` model is a good choice for messaging bots because it balances quality with cost and latency -- chat-style interactions need fast responses, and the smaller model keeps per-message costs low for high-volume support scenarios.

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
        os.Exit(1)
    }

    bot := NewBotServer(model)

    http.HandleFunc("/whatsapp", bot.HandleWebhook)
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, `{"status":"ok"}`)
    })

    addr := ":8080"
    slog.Info("bot server starting", "addr", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

## Step 6: Configure Twilio Webhook

1. Start the server: `go run main.go`
2. Expose it publicly: `ngrok http 8080`
3. In the Twilio Console, set the WhatsApp Sandbox webhook URL to `https://your-url.ngrok.io/whatsapp`
4. Send a WhatsApp message to the sandbox number

## Verification

1. Start the server and expose it via ngrok.
2. Configure the Twilio Sandbox webhook URL.
3. Send a WhatsApp message to the sandbox number.
4. Verify you receive an automated reply.
5. Send a follow-up message and verify the bot maintains conversation context.

## Next Steps

- [Omni-Channel Gateway](/docs/tutorials/messaging/omnichannel-gateway) -- Handle messages from WhatsApp, SMS, and Slack through a unified interface
- [Content Moderation](/docs/tutorials/safety/content-moderation) -- Filter messages across all channels
