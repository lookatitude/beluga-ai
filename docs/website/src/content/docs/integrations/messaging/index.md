---
title: Messaging Platforms
description: Connect Beluga AI agents to Twilio, Slack, and custom messaging channels via the protocol and server packages.
sidebar:
  order: 0
---

Beluga AI agents can serve conversations across messaging platforms by combining the `protocol` package (for HTTP/SSE endpoints) with the `server` package (for framework adapters). This page covers common patterns for connecting to Twilio, Slack, and custom messaging backends.

## Architecture

```
Messaging Platform → Webhook → Server Adapter → Agent → Response → Platform API
     (Twilio)         POST       (Gin/Chi)      Generate    Text     (Twilio REST)
     (Slack)          POST       (Echo/Fiber)   Stream      Blocks   (Slack API)
```

Beluga does not include dedicated messaging SDKs. Instead, it provides HTTP server adapters and agent runtime — you bring the platform SDK and webhook handling for your specific channels.

## Twilio Integration

### WhatsApp and SMS

Connect a Beluga AI agent to Twilio for WhatsApp and SMS conversations.

**Prerequisites**: Twilio account, phone number with SMS/WhatsApp enabled.

```bash
export TWILIO_ACCOUNT_SID="AC..."
export TWILIO_AUTH_TOKEN="..."
```

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    twilio "github.com/twilio/twilio-go"
    twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

func main() {
    ctx := context.Background()

    model, err := llm.New("openai", config.ProviderConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    client := twilio.NewRestClientWithParams(twilio.ClientParams{
        Username: os.Getenv("TWILIO_ACCOUNT_SID"),
        Password: os.Getenv("TWILIO_AUTH_TOKEN"),
    })

    http.HandleFunc("/webhook/twilio", func(w http.ResponseWriter, r *http.Request) {
        if err := r.ParseForm(); err != nil {
            http.Error(w, "bad request", http.StatusBadRequest)
            return
        }

        from := r.FormValue("From")
        body := r.FormValue("Body")

        // Generate response with Beluga AI
        resp, err := model.Generate(ctx, []schema.Message{
            schema.NewSystemMessage("You are a helpful assistant responding via SMS."),
            schema.NewUserMessage(schema.Text(body)),
        })
        if err != nil {
            log.Printf("generate error: %v", err)
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }

        // Send reply via Twilio
        responseText := resp.Content()
        params := &twilioApi.CreateMessageParams{}
        params.SetTo(from)
        params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
        params.SetBody(responseText)

        if _, err := client.Api.CreateMessage(params); err != nil {
            log.Printf("twilio send error: %v", err)
        }

        w.WriteHeader(http.StatusOK)
    })

    fmt.Println("Twilio webhook listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Webhook Security

Validate Twilio webhook signatures to prevent spoofed requests:

```go
import "github.com/twilio/twilio-go/client"

func validateTwilioSignature(r *http.Request, authToken string) bool {
    validator := client.NewRequestValidator(authToken)
    url := "https://your-domain.com" + r.URL.String()
    params := make(map[string]string)
    for key, values := range r.PostForm {
        params[key] = values[0]
    }
    signature := r.Header.Get("X-Twilio-Signature")
    return validator.Validate(url, params, signature)
}
```

## Slack Integration

### Incoming Webhooks

The simplest Slack integration: post agent responses to a channel via webhook.

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func postToSlack(webhookURL, text string) error {
    payload, err := json.Marshal(map[string]string{"text": text})
    if err != nil {
        return err
    }
    resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(payload))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("slack returned %d", resp.StatusCode)
    }
    return nil
}

func main() {
    ctx := context.Background()
    webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

    model, err := llm.New("openai", config.ProviderConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := model.Generate(ctx, []schema.Message{
        schema.NewUserMessage(schema.Text("Summarize today's key metrics.")),
    })
    if err != nil {
        log.Fatal(err)
    }

    if err := postToSlack(webhookURL, resp.Content()); err != nil {
        log.Fatal(err)
    }
}
```

### Slack Events API

Handle interactive conversations via Slack's Events API:

```go
http.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
    var event struct {
        Type      string `json:"type"`
        Challenge string `json:"challenge"`
        Event     struct {
            Type    string `json:"type"`
            Text    string `json:"text"`
            User    string `json:"user"`
            Channel string `json:"channel"`
        } `json:"event"`
    }

    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    // Handle URL verification challenge
    if event.Type == "url_verification" {
        w.Header().Set("Content-Type", "text/plain")
        fmt.Fprint(w, event.Challenge)
        return
    }

    // Handle message events
    if event.Event.Type == "message" {
        go handleSlackMessage(context.Background(), model, event.Event.Channel, event.Event.Text)
    }

    w.WriteHeader(http.StatusOK)
})
```

## Omnichannel Gateway Pattern

For applications that serve multiple messaging platforms, create a gateway that normalizes incoming messages and routes them to a shared agent:

```go
// IncomingMessage normalizes messages from any platform
type IncomingMessage struct {
    Platform  string // "twilio", "slack", "web"
    ChannelID string
    UserID    string
    Text      string
    Metadata  map[string]any
}

// OutgoingMessage is the agent's response
type OutgoingMessage struct {
    ChannelID string
    Text      string
}

// Gateway routes messages from platforms to the agent
type Gateway struct {
    model llm.ChatModel
    senders map[string]func(ctx context.Context, msg OutgoingMessage) error
}

func (g *Gateway) Handle(ctx context.Context, msg IncomingMessage) error {
    resp, err := g.model.Generate(ctx, []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewUserMessage(schema.Text(msg.Text)),
    })
    if err != nil {
        return fmt.Errorf("generate: %w", err)
    }

    sender, ok := g.senders[msg.Platform]
    if !ok {
        return fmt.Errorf("unknown platform: %s", msg.Platform)
    }

    return sender(ctx, OutgoingMessage{
        ChannelID: msg.ChannelID,
        Text:      resp.Content(),
    })
}
```

Register platform-specific senders:

```go
gateway := &Gateway{
    model: model,
    senders: map[string]func(ctx context.Context, msg OutgoingMessage) error{
        "twilio": func(ctx context.Context, msg OutgoingMessage) error {
            // Send via Twilio API
            return nil
        },
        "slack": func(ctx context.Context, msg OutgoingMessage) error {
            // Send via Slack API
            return nil
        },
        "web": func(ctx context.Context, msg OutgoingMessage) error {
            // Send via WebSocket/SSE
            return nil
        },
    },
}
```

## Using Server Adapters

Beluga's `server` package provides adapters for popular HTTP frameworks. Use them to mount webhook handlers alongside your existing web application:

```go
import (
    "github.com/lookatitude/beluga-ai/server/adapters/gin"
    ginfw "github.com/gin-gonic/gin"
)

router := ginfw.Default()
router.POST("/webhook/twilio", func(c *ginfw.Context) {
    // Handle Twilio webhook using Gin
})
router.POST("/webhook/slack", func(c *ginfw.Context) {
    // Handle Slack events using Gin
})
```

Available adapters: `gin`, `fiber`, `echo`, `chi`, `huma`, `grpc`, `connect`.

## Conversation State

For multi-turn messaging conversations, maintain state using Beluga's memory system:

```go
import "github.com/lookatitude/beluga-ai/memory"

// Store conversation history per user
mem, err := memory.New("redis", config.ProviderConfig{
    Options: map[string]any{
        "address": "localhost:6379",
    },
})

// Retrieve history for a user
history, err := mem.Recall(ctx, userID, 20)

// Build messages from history
messages := append([]schema.Message{systemPrompt}, history...)
messages = append(messages, schema.NewUserMessage(schema.Text(incomingText)))

resp, err := model.Generate(ctx, messages)
```

## Production Considerations

- **Webhook timeouts**: Twilio requires a response within 15 seconds. For long-running agent calls, respond with `202 Accepted` and send the reply asynchronously via the platform API.
- **Rate limiting**: Both Twilio and Slack enforce rate limits. Use Beluga's `resilience` package for client-side rate limiting.
- **Idempotency**: Webhook retries can cause duplicate messages. Track message IDs to deduplicate.
- **Security**: Always validate webhook signatures. Never expose API keys in webhook responses.
