# Building a WhatsApp Support Bot

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll build an automated support bot using the Beluga AI Messaging package and Twilio WhatsApp integration. You'll learn how to handle incoming messages, maintain conversation state, and send formatted replies to users.

## Learning Objectives
- ✅ Configure Twilio WhatsApp provider
- ✅ Handle incoming WhatsApp messages
- ✅ Send formatted responses
- ✅ Maintain conversation state

## Introduction
Welcome, colleague! WhatsApp is one of the most popular channels for customer support. Let's look at how to use the Beluga AI messaging package to build a responsive, context-aware support bot that can handle real user queries.

## Prerequisites

- Twilio Account with WhatsApp Sandbox enabled
- Go 1.24+
- Publicly accessible URL (e.g., via ngrok) for webhooks

## Step 1: Initialize Twilio Provider
```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/messaging"
    "github.com/lookatitude/beluga-ai/pkg/messaging/providers/twilio"
)

func main() {
    config := &twilio.Config{
        AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
        AuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
        PhoneNumber: "whatsapp:+14155238886", // Twilio Sandbox number
    }
    
    provider, _ := twilio.NewProvider(config)
}
```

## Step 2: Handle Incoming Messages

You need to set up a webhook handler to receive messages from Twilio.
```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. Parse incoming message
    msg, err := provider.ParseIncoming(r)
    if err != nil {
        http.Error(w, "Bad Request", 400)
        return
    }

    fmt.Printf("Message from %s: %s\n", msg.From, msg.Body)

    // 2. Process with Agent (Optional)
    // response := agent.Invoke(ctx, msg.Body)


    // 3. Send Reply
    err = provider.SendMessage(context.Background(), messaging.Message{
        To:   msg.From,
```
        Body: "Hello from Beluga AI! How can I help you today?",
    })
}

## Step 3: Conversation Management

Use the `MessagingSession` to keep track of user context.
session, _ := messaging.NewSession(msg.From,






    messaging.WithProvider(provider),
    messaging.WithMemory(memory.NewBufferMemory()),
)

```
session.HandleMessage(msg)

## Step 4: Running the Server
```go
func main() {
    // ... init provider ...
    http.HandleFunc("/whatsapp", handleWebhook)
    http.ListenAndServe(":8080", nil)
}
```

## Verification

1. Start your server and expose it via ngrok: `ngrok http 8080`.
2. Configure Twilio Sandbox webhook URL to `https://your-url.ngrok.io/whatsapp`.
3. Send a WhatsApp message to the sandbox number.
4. Verify you receive the automated reply.

## Next Steps

- **[Omni-channel Gateway Setup](./messaging-omnichannel-gateway.md)** - Add SMS and Slack.
- **[Building a Research Agent](../higher-level/agents-research-agent.md)** - Connect your bot to a smart agent.
