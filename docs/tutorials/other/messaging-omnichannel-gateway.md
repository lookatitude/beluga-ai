# Omni-channel Gateway Setup

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll build a gateway that handles messages from multiple channels (WhatsApp, SMS, Slack) using a single unified interface. You'll learn how to normalize message formats and route them to the correct provider while sharing the same agent logic.

## Learning Objectives
- ✅ Implement a multi-provider gateway
- ✅ Normalize message formats across channels
- ✅ Route messages to the correct provider
- ✅ Shared agent logic for all channels

## Introduction
Welcome, colleague! Your users are everywhere—WhatsApp, SMS, Slack, and more. Building a separate bot for each is a maintenance nightmare. Let's build a unified messaging gateway that lets us write our agent logic once and deploy it to any channel.

## Prerequisites

- [Building a WhatsApp Support Bot](./messaging-whatsapp-bot.md)
- API Keys for multiple services (Twilio, Slack)

## Step 1: The Gateway Pattern

Instead of separate handlers for each channel, use a central gateway.
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/messaging"
)

type Gateway struct {
    providers map[string]messaging.Provider
}

func (g *Gateway) Send(ctx context.Context, msg messaging.Message) error {
    provider, ok := g.providers[msg.Channel]
    if !ok {
        return fmt.Errorf("unsupported channel: %s", msg.Channel)
    }
    return provider.SendMessage(ctx, msg)
}

## Step 2: Registering Providers
func main() {
    gateway := &Gateway{
        providers: make(map[string]messaging.Provider),
    }

    // WhatsApp
    whatsapp, _ := twilio.NewProvider(whatsappConfig)
    gateway.providers["whatsapp"] = whatsapp

    // SMS
    sms, _ := twilio.NewProvider(smsConfig)
    gateway.providers["sms"] = sms
}
```

## Step 3: Unified Webhook Handler

Twilio uses the same format for SMS and WhatsApp, making normalization easy.
```go
func (g *Gateway) HandleTwilio(w http.ResponseWriter, r *http.Request) {
    msg, _ := g.providers["whatsapp"].ParseIncoming(r)
    
    // Logic to determine channel
    channel := "sms"
    if strings.HasPrefix(msg.From, "whatsapp:") {
        channel = "whatsapp"
    }

    // Universal Agent Logic
    response := myAgent.Invoke(ctx, msg.Body)

    // Send back through gateway
    g.Send(ctx, messaging.Message{
        To:      msg.From,
        Body:    response,
        Channel: channel,
    })
}
```

## Step 4: Shared Memory across Channels

Ensure user `+1234567890` has the same history whether they use SMS or WhatsApp.
```go
func getMemory(userID string) memory.Memory {
    // Return memory from a shared Redis store based on userID
    return redisStore.Get(userID)
}
```

## Verification

1. Send an SMS. Verify the agent responds.
2. Send a WhatsApp message. Verify the agent responds AND remembers the SMS context (if using shared memory).

## Next Steps

- **[Redis Persistence](../higher-level/memory-redis-persistence.md)** - Persist multi-channel conversations.
- **[Content Moderation](../higher-level/safety-content-moderation.md)** - Filter messages across all channels.
