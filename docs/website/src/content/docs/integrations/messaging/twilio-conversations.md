---
title: Twilio Conversations
description: Integrate Twilio Conversations API with Beluga AI for multi-channel messaging across SMS, WhatsApp, and other channels.
---

## Overview

Customers reach out through SMS, WhatsApp, and web chat -- often switching between channels mid-conversation. Twilio Conversations unifies these channels behind a single API, so one Beluga AI agent handles all of them without channel-specific logic. This reduces development effort and ensures consistent AI behavior regardless of how a customer contacts you. This guide covers integrating Twilio Conversations API with Beluga AI to build multi-channel AI messaging experiences.

## Prerequisites

- Go 1.23 or later
- A Beluga AI application
- A Twilio account with Conversations API enabled
- Twilio Account SID, Auth Token, and Conversations Service SID

## Installation

Install the Twilio Go SDK:

```bash
go get github.com/twilio/twilio-go
```

Retrieve your credentials from the [Twilio Console](https://console.twilio.com) and set the required environment variables:

```bash
export TWILIO_ACCOUNT_SID="your-account-sid"
export TWILIO_AUTH_TOKEN="your-auth-token"
export TWILIO_CONVERSATIONS_SERVICE_SID="your-service-sid"
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `AccountSID` | Twilio account SID | - | Yes |
| `AuthToken` | Twilio auth token | - | Yes |
| `ConversationsServiceSID` | Conversations service SID | - | Yes |
| `WebhookURL` | Webhook endpoint for inbound events | - | No |

## Usage

### Create a Twilio Messaging Provider

Set up a Twilio provider and create a conversation:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	twilioprovider "github.com/lookatitude/beluga-ai/protocol/twilio"
)

func main() {
	ctx := context.Background()

	config := &twilioprovider.Config{
		AccountSID:              os.Getenv("TWILIO_ACCOUNT_SID"),
		AuthToken:               os.Getenv("TWILIO_AUTH_TOKEN"),
		ConversationsServiceSID: os.Getenv("TWILIO_CONVERSATIONS_SERVICE_SID"),
	}

	provider, err := twilioprovider.NewProvider(config)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	if err := provider.Start(ctx); err != nil {
		log.Fatalf("failed to start provider: %v", err)
	}
	defer provider.Stop(ctx)

	conversation, err := provider.CreateConversation(ctx, &twilioprovider.ConversationConfig{
		FriendlyName: "Customer Support",
	})
	if err != nil {
		log.Fatalf("failed to create conversation: %v", err)
	}

	fmt.Printf("Created conversation: %s\n", conversation.ID)
}
```

### Send and Receive Messages

Use the provider to send messages and listen for incoming ones:

```go
func handleMessaging(ctx context.Context, provider *twilioprovider.Provider, conversationID string) error {
	// Send a message.
	msg := &twilioprovider.Message{
		Content: "Hello from Beluga AI!",
		From:    "system",
		To:      "customer",
	}

	if err := provider.SendMessage(ctx, conversationID, msg); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}

	// Receive messages from the conversation.
	msgChan, err := provider.ReceiveMessages(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("receive failed: %w", err)
	}

	for msg := range msgChan {
		fmt.Printf("Received: %s\n", msg.Content)
		// Route to an AI agent for processing.
	}

	return nil
}
```

### Handle Webhooks

Twilio sends webhook events for new messages. Use a webhook handler to route inbound messages to a Beluga AI agent:

```go
func handleWebhook(ctx context.Context, provider *twilioprovider.Provider, event *twilioprovider.WebhookEvent) error {
	if event.Type != "message.new" {
		return nil
	}

	message := event.Data.(*twilioprovider.Message)

	// Process with an AI agent.
	response, err := processWithAgent(ctx, message.Content)
	if err != nil {
		return fmt.Errorf("agent processing failed: %w", err)
	}

	reply := &twilioprovider.Message{
		Content: response,
		From:    "agent",
		To:      message.From,
	}

	return provider.SendMessage(ctx, event.ConversationID, reply)
}
```

### Complete Example

A full integration that creates a conversation and listens for webhook events:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/lookatitude/beluga-ai/agent"
	twilioprovider "github.com/lookatitude/beluga-ai/protocol/twilio"
)

func main() {
	ctx := context.Background()

	config := &twilioprovider.Config{
		AccountSID:              os.Getenv("TWILIO_ACCOUNT_SID"),
		AuthToken:               os.Getenv("TWILIO_AUTH_TOKEN"),
		ConversationsServiceSID: os.Getenv("TWILIO_CONVERSATIONS_SERVICE_SID"),
	}

	provider, err := twilioprovider.NewProvider(config)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	if err := provider.Start(ctx); err != nil {
		log.Fatalf("failed to start provider: %v", err)
	}
	defer provider.Stop(ctx)

	conversation, err := provider.CreateConversation(ctx, &twilioprovider.ConversationConfig{
		FriendlyName: "Support Chat",
	})
	if err != nil {
		log.Fatalf("failed to create conversation: %v", err)
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		event, err := twilioprovider.ParseWebhookEvent(r)
		if err != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		if err := provider.HandleWebhook(ctx, event); err != nil {
			log.Printf("webhook handling error: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("Conversation created: %s\n", conversation.ID)
	log.Println("Webhook server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Advanced Topics

### Multi-Channel Support

Twilio Conversations unifies SMS, WhatsApp, and web chat behind a single API. Add participants from different channels to the same conversation:

- **SMS**: Add a phone number participant
- **WhatsApp**: Add a WhatsApp-enabled number participant
- **Web Chat**: Add a chat-based participant

The Beluga AI agent sees all messages through the same interface regardless of originating channel.

### Webhook Signature Verification

In production, validate Twilio webhook signatures to ensure requests originate from Twilio. Use the `X-Twilio-Signature` header and your Auth Token to verify each request.

### Conversation Lifecycle

Manage conversation state transitions (active, inactive, closed) to control resource usage and agent availability. Use Twilio's conversation state API to archive completed support sessions.

### Cost Management

Monitor message volume and costs through the Twilio Console. Consider implementing rate limiting with the `resilience` package to control outbound message throughput.

## Troubleshooting

### "Invalid credentials"

Verify that `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN` are set correctly. Tokens can be regenerated from the Twilio Console.

### "Service not found"

Ensure the Conversations service is created in the Twilio Console and that `TWILIO_CONVERSATIONS_SERVICE_SID` matches the correct service.

## Related Resources

- [Slack Webhooks](/integrations/slack-webhooks) -- Slack messaging integration
- [Observability and Tracing](/guides/observability-tracing) -- OpenTelemetry setup for Beluga AI
