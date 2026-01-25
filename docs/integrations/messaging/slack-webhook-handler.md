# Slack Webhook Handler

Welcome, colleague! In this integration guide, we're going to integrate Slack webhooks with Beluga AI's messaging package. This enables AI agents to interact with Slack channels and direct messages.

## What you will build

You will configure Beluga AI to handle Slack webhooks, enabling AI agents to receive messages from Slack and send responses, creating conversational AI experiences in Slack workspaces.

## Learning Objectives

- ✅ Configure Slack webhooks with Beluga AI
- ✅ Handle Slack events (messages, mentions)
- ✅ Send messages to Slack channels
- ✅ Integrate with AI agents

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Slack workspace
- Slack app with webhook URL

## Step 1: Setup and Installation

Create a Slack app at https://api.slack.com/apps

Configure webhook URL:
- Go to "Incoming Webhooks"
- Enable and create webhook
- Copy webhook URL

Set environment variable:
bash
```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
```

## Step 2: Create Slack Webhook Handler

Create a Slack webhook handler:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/messaging/iface"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type SlackWebhookHandler struct {
    botToken string
    agent    agents.Agent
    tracer   trace.Tracer
}

type SlackEvent struct {
    Type    string `json:"type"`
    Event   struct {
        Type    string `json:"type"`
        Text    string `json:"text"`
        User    string `json:"user"`
        Channel string `json:"channel"`
    } `json:"event"`
    Challenge string `json:"challenge"`
}

func NewSlackWebhookHandler(botToken string, agent agents.Agent) *SlackWebhookHandler {
    return &SlackWebhookHandler{
        botToken: botToken,
        agent:    agent,
        tracer:   otel.Tracer("beluga.messaging.slack"),
    }
}

func (h *SlackWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ctx, span := h.tracer.Start(ctx, "slack.webhook")
    defer span.End()

    var event SlackEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        span.RecordError(err)
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Handle URL verification challenge
    if event.Type == "url_verification" {
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte(event.Challenge))
        return
    }

    // Handle message events
    if event.Type == "event_callback" && event.Event.Type == "message" {
        // Process message with agent
        response, err := h.agent.Run(ctx, event.Event.Text)
        if err != nil {
            span.RecordError(err)
            log.Printf("Agent error: %v", err)
            return
        }

        // Send response to Slack
        if err := h.sendSlackMessage(ctx, event.Event.Channel, response.Content); err != nil {
            span.RecordError(err)
            log.Printf("Send error: %v", err)
        }
    }


    w.WriteHeader(http.StatusOK)
}
```

## Step 3: Send Messages to Slack

Implement Slack message sending:
```go
func (h *SlackWebhookHandler) sendSlackMessage(ctx context.Context, channel, text string) error {
    ctx, span := h.tracer.Start(ctx, "slack.send",
        trace.WithAttributes(
            attribute.String("channel", channel),
        ),
    )
    defer span.End()

    payload := map[string]string{
        "channel": channel,
        "text":   text,
    }

    jsonData, _ := json.Marshal(payload)
    
    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://slack.com/api/chat.postMessage",
        strings.NewReader(string(jsonData)))
    req.Header.Set("Authorization", "Bearer "+h.botToken)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        err := fmt.Errorf("slack API error: %d", resp.StatusCode)
        span.RecordError(err)
        return err
    }


    return nil
}
```

## Step 4: Use with Beluga AI

Integrate with Beluga AI agents:
```go
func main() {
    ctx := context.Background()

    // Create AI agent
    agent, err := agents.NewAgent(ctx, agents.Config{
        LLMProvider: "openai",
        SystemPrompt: "You are a helpful Slack assistant.",
    })
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Create Slack handler
    handler := NewSlackWebhookHandler(
        os.Getenv("SLACK_BOT_TOKEN"),
        agent,
    )

    // Set up webhook endpoint
    http.HandleFunc("/slack/webhook", handler.HandleWebhook)


    fmt.Println("Slack webhook server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionSlackHandler struct {
    botToken string
    agent    agents.Agent
    tracer   trace.Tracer
}

func NewProductionSlackHandler(botToken string, agent agents.Agent) *ProductionSlackHandler {
    return &ProductionSlackHandler{
        botToken: botToken,
        agent:    agent,
        tracer:   otel.Tracer("beluga.messaging.slack"),
    }
}

func (h *ProductionSlackHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ctx, span := h.tracer.Start(ctx, "slack.webhook")
    defer span.End()

    var event SlackEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        span.RecordError(err)
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // URL verification
    if event.Type == "url_verification" {
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte(event.Challenge))
        return
    }

    // Process messages
    if event.Type == "event_callback" && event.Event.Type == "message" {
        // Ignore bot messages
        if event.Event.User == "" {
            w.WriteHeader(http.StatusOK)
            return
        }

        span.SetAttributes(
            attribute.String("channel", event.Event.Channel),
            attribute.String("user", event.Event.User),
        )

        // Process with agent
        response, err := h.agent.Run(ctx, event.Event.Text)
        if err != nil {
            span.RecordError(err)
            log.Printf("Agent error: %v", err)
            w.WriteHeader(http.StatusOK)
            return
        }

        // Send response
        if err := h.sendMessage(ctx, event.Event.Channel, response.Content); err != nil {
            span.RecordError(err)
            log.Printf("Send error: %v", err)
        }
    }

    w.WriteHeader(http.StatusOK)
}

func (h *ProductionSlackHandler) sendMessage(ctx context.Context, channel, text string) error {
    ctx, span := h.tracer.Start(ctx, "slack.send")
    defer span.End()

    payload := map[string]string{
        "channel": channel,
        "text":   text,
    }

    jsonData, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://slack.com/api/chat.postMessage",
        strings.NewReader(string(jsonData)))
    req.Header.Set("Authorization", "Bearer "+h.botToken)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        span.RecordError(err)
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        err := fmt.Errorf("API error: %d", resp.StatusCode)
        span.RecordError(err)
        return err
    }

    return nil
}

func main() {
    ctx := context.Background()

    agent, _ := agents.NewAgent(ctx, agents.Config{
        LLMProvider: "openai",
    })

    handler := NewProductionSlackHandler(os.Getenv("SLACK_BOT_TOKEN"), agent)


    http.HandleFunc("/slack/webhook", handler.HandleWebhook)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `BotToken` | Slack bot token | - | Yes |
| `WebhookURL` | Incoming webhook URL | - | No |
| `SigningSecret` | Webhook signing secret | - | No |

## Common Issues

### "Invalid token"

**Problem**: Wrong bot token.

**Solution**: Verify bot token:export SLACK_BOT_TOKEN="xoxb-your-token"
```

### "URL verification failed"

**Problem**: Challenge response incorrect.

**Solution**: Return challenge value exactly as received.

## Production Considerations

When using Slack in production:

- **Webhook security**: Verify webhook signatures
- **Rate limiting**: Handle Slack rate limits
- **Error handling**: Handle API failures gracefully
- **Mention handling**: Respond to mentions appropriately
- **Threading**: Support threaded conversations

## Next Steps

Congratulations! You've integrated Slack with Beluga AI. Next, learn how to:

- **[Twilio Conversations API](./twilio-conversations-api.md)** - Twilio integration
- **Messaging Package Documentation** - Deep dive into messaging package
- **[Messaging Use Cases](../../use-cases/)** - Messaging patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
