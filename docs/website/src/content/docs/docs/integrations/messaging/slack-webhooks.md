---
title: Slack Webhook Integration
description: "Integrate Slack webhooks with Beluga AI to build intelligent bots that receive and respond to messages in channels and DMs in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Slack integration, Slack bot, Beluga AI, webhook handler, Slack Events API, AI chatbot Go, team messaging"
---

## Overview

Slack is where most engineering and operations teams already communicate. By connecting a Beluga AI agent to Slack, your team can query knowledge bases, trigger workflows, and get AI-generated summaries without leaving their primary communication tool. This is particularly valuable for internal support bots, on-call assistants, and DevOps automation. This guide walks through integrating Slack webhooks with Beluga AI, enabling AI agents to receive messages from Slack workspaces and send intelligent responses, with full OpenTelemetry instrumentation for observability.

## Prerequisites

- Go 1.23 or later
- A Beluga AI application
- A Slack workspace with admin access
- A Slack app configured with a bot token and webhook URL

## Installation

Create a Slack app at [api.slack.com/apps](https://api.slack.com/apps) and configure the following:

1. Navigate to **Incoming Webhooks** and enable webhooks
2. Navigate to **Event Subscriptions** and subscribe to `message.channels` and `message.im` events
3. Navigate to **OAuth & Permissions** and add the `chat:write` scope

Set the required environment variables:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `BotToken` | Slack bot OAuth token (`xoxb-...`) | - | Yes |
| `WebhookURL` | Incoming webhook URL | - | No |
| `SigningSecret` | Webhook request signing secret | - | Recommended |

## Usage

### Define the Webhook Handler

The handler receives Slack events, routes them to a Beluga AI agent, and sends the agent's response back to the originating channel.

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

	"github.com/lookatitude/beluga-ai/agent"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SlackEvent represents an incoming Slack Events API payload.
type SlackEvent struct {
	Type  string `json:"type"`
	Event struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		User    string `json:"user"`
		Channel string `json:"channel"`
	} `json:"event"`
	Challenge string `json:"challenge"`
}

// SlackWebhookHandler routes Slack events to a Beluga AI agent.
type SlackWebhookHandler struct {
	botToken string
	agent    agent.Agent
	tracer   trace.Tracer
}

func NewSlackWebhookHandler(botToken string, a agent.Agent) *SlackWebhookHandler {
	return &SlackWebhookHandler{
		botToken: botToken,
		agent:    a,
		tracer:   otel.Tracer("beluga.integration.slack"),
	}
}
```

### Handle Incoming Events

The `HandleWebhook` method processes two event types: the initial URL verification challenge and subsequent message events.

```go
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

	// Handle Slack URL verification challenge.
	if event.Type == "url_verification" {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(event.Challenge))
		return
	}

	// Route message events to the agent.
	if event.Type == "event_callback" && event.Event.Type == "message" {
		// Ignore bot-generated messages to prevent loops.
		if event.Event.User == "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		span.SetAttributes(
			attribute.String("slack.channel", event.Event.Channel),
			attribute.String("slack.user", event.Event.User),
		)

		response, err := h.agent.Run(ctx, event.Event.Text)
		if err != nil {
			span.RecordError(err)
			log.Printf("agent error: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		if err := h.sendMessage(ctx, event.Event.Channel, response.Content); err != nil {
			span.RecordError(err)
			log.Printf("slack send error: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}
```

### Send Messages to Slack

The `sendMessage` method posts a reply to a Slack channel using the Web API.

```go
func (h *SlackWebhookHandler) sendMessage(ctx context.Context, channel, text string) error {
	ctx, span := h.tracer.Start(ctx, "slack.send",
		trace.WithAttributes(
			attribute.String("slack.channel", channel),
		),
	)
	defer span.End()

	payload := map[string]string{
		"channel": channel,
		"text":    text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://slack.com/api/chat.postMessage",
		strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("slack API error: status %d", resp.StatusCode)
		span.RecordError(err)
		return err
	}

	return nil
}
```

### Wire It Together

```go
func main() {
	ctx := context.Background()

	a, err := agent.NewAgent(ctx, agent.Config{
		LLMProvider:  "openai",
		SystemPrompt: "You are a helpful Slack assistant.",
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	handler := NewSlackWebhookHandler(os.Getenv("SLACK_BOT_TOKEN"), a)

	http.HandleFunc("/slack/webhook", handler.HandleWebhook)

	log.Println("Slack webhook server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Advanced Topics

### Webhook Signature Verification

In production, verify that incoming requests originate from Slack by validating the `X-Slack-Signature` header against your app's signing secret. This prevents unauthorized parties from sending fabricated events to your endpoint.

### Rate Limiting

Slack enforces rate limits on the Web API (typically one message per second per channel for `chat.postMessage`). Use the `resilience` package to add retry logic with exponential backoff:

```go
import "github.com/lookatitude/beluga-ai/resilience"
```

### Threaded Conversations

To reply in a thread rather than the channel, include the `thread_ts` field from the original message event in your response payload. This keeps agent conversations organized within threads.

### Bot Message Filtering

The handler above filters messages with an empty `User` field to avoid responding to its own messages. For more robust filtering, check the `bot_id` field or maintain a set of known bot user IDs.

## Troubleshooting

### "Invalid token"

Verify that the `SLACK_BOT_TOKEN` environment variable contains a valid `xoxb-` prefixed token and that the token has the required OAuth scopes.

### "URL verification failed"

The challenge response must return the `challenge` value from the request body exactly as received, with a `text/plain` content type.

## Related Resources

- [Twilio Conversations](/integrations/twilio-conversations) -- Multi-channel messaging integration
- [Observability and Tracing](/guides/observability-tracing) -- OpenTelemetry setup for Beluga AI
