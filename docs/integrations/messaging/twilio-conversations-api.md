# Twilio Conversations API

Welcome, colleague! In this integration guide, we're going to integrate Twilio Conversations API with Beluga AI's messaging package. Twilio Conversations provides multi-channel messaging (SMS, WhatsApp, etc.) with conversation management.

## What you will build

You will configure Beluga AI to use Twilio Conversations API for multi-channel messaging, enabling SMS, WhatsApp, and other channels with AI agent integration and conversation persistence.

## Learning Objectives

- ✅ Configure Twilio Conversations with Beluga AI messaging
- ✅ Create and manage conversations
- ✅ Send and receive messages across channels
- ✅ Integrate with AI agents

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Twilio account and credentials
- Twilio Conversations API enabled

## Step 1: Setup and Installation

Install Twilio Go SDK:
bash
```bash
go get github.com/twilio/twilio-go
```

Get Twilio credentials:
- Account SID
- Auth Token
- Conversations Service SID

Set environment variables:
bash
```bash
export TWILIO_ACCOUNT_SID="your-account-sid"
export TWILIO_AUTH_TOKEN="your-auth-token"
export TWILIO_CONVERSATIONS_SERVICE_SID="your-service-sid"
```

## Step 2: Basic Twilio Configuration

Create a Twilio messaging backend:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/messaging"
    "github.com/lookatitude/beluga-ai/pkg/messaging/iface"
    "github.com/lookatitude/beluga-ai/pkg/messaging/providers/twilio"
)

func main() {
    ctx := context.Background()

    // Create Twilio configuration
    config := &twilio.TwilioConfig{
        AccountSID:            os.Getenv("TWILIO_ACCOUNT_SID"),
        AuthToken:            os.Getenv("TWILIO_AUTH_TOKEN"),
        ConversationsServiceSID: os.Getenv("TWILIO_CONVERSATIONS_SERVICE_SID"),
        EnableMetrics:        true,
    }

    // Create Twilio provider
    provider, err := twilio.NewTwilioProvider(config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Start backend
    if err := provider.Start(ctx); err != nil {
        log.Fatalf("Failed to start: %v", err)
    }
    defer provider.Stop(ctx)

    // Create conversation
    conversation, err := provider.CreateConversation(ctx, &iface.ConversationConfig{
        FriendlyName: "Customer Support",
    })
    if err != nil {
        log.Fatalf("Failed to create conversation: %v", err)
    }

    fmt.Printf("Created conversation: %s\n", conversation.ID)
}
```

### Verification

Run the example:
bash
```bash
export TWILIO_ACCOUNT_SID="your-sid"
export TWILIO_AUTH_TOKEN="your-token"
export TWILIO_CONVERSATIONS_SERVICE_SID="your-service-sid"
go run main.go
```

You should see a conversation created.

## Step 3: Send and Receive Messages

Handle messaging:
```go
func handleMessaging(ctx context.Context, provider *twilio.TwilioProvider, conversationID string) error {
    // Send message
    message := &iface.Message{
        Content: "Hello from Beluga AI!",
        From:    "system",
        To:      "customer",
    }
    
    if err := provider.SendMessage(ctx, conversationID, message); err != nil {
        return fmt.Errorf("send failed: %w", err)
    }

    // Receive messages
    msgChan, err := provider.ReceiveMessages(ctx, conversationID)
    if err != nil {
        return fmt.Errorf("receive failed: %w", err)
    }

    // Process incoming messages
    for msg := range msgChan {
        fmt.Printf("Received: %s\n", msg.Content)
        // Process with AI agent
    }


    return nil
}
```

## Step 4: Webhook Handling

Handle Twilio webhooks:
```go
func handleWebhook(ctx context.Context, provider *twilio.TwilioProvider, event *iface.WebhookEvent) error {
    // Twilio sends webhook events for new messages
    if event.Type == "message.new" {
        // Process new message
        message := event.Data.(*iface.Message)
        
        // Send to AI agent
        response, err := processWithAgent(ctx, message.Content)
        if err != nil {
            return err
        }
        
        // Send response
        reply := &iface.Message{
            Content: response,
            From:    "agent",
            To:      message.From,
        }

        

        return provider.SendMessage(ctx, event.ConversationID, reply)
    }
    
    return nil
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/messaging"
    "github.com/lookatitude/beluga-ai/pkg/messaging/iface"
    "github.com/lookatitude/beluga-ai/pkg/messaging/providers/twilio"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx := context.Background()

    // Create Twilio provider
    config := &twilio.TwilioConfig{
        AccountSID:            os.Getenv("TWILIO_ACCOUNT_SID"),
        AuthToken:            os.Getenv("TWILIO_AUTH_TOKEN"),
        ConversationsServiceSID: os.Getenv("TWILIO_CONVERSATIONS_SERVICE_SID"),
        EnableMetrics:        true,
    }

    provider, err := twilio.NewTwilioProvider(config)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    if err := provider.Start(ctx); err != nil {
        log.Fatalf("Failed: %v", err)
    }
    defer provider.Stop(ctx)

    // Create conversation
    conversation, err := provider.CreateConversation(ctx, &iface.ConversationConfig{
        FriendlyName: "Support Chat",
    })
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Set up webhook handler
    http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
        event := parseWebhookEvent(r)
        provider.HandleWebhook(ctx, event)
    })


    fmt.Printf("Conversation created: %s\n", conversation.ID)
    fmt.Println("Webhook server running on :8080")
    http.ListenAndServe(":8080", nil)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `AccountSID` | Twilio account SID | - | Yes |
| `AuthToken` | Twilio auth token | - | Yes |
| `ConversationsServiceSID` | Conversations service SID | - | Yes |
| `WebhookURL` | Webhook endpoint URL | - | No |

## Common Issues

### "Invalid credentials"

**Problem**: Wrong account SID or auth token.

**Solution**: Verify credentials:export TWILIO_ACCOUNT_SID="your-sid"
bash
```bash
export TWILIO_AUTH_TOKEN="your-token"
```

### "Service not found"

**Problem**: Conversations service not created.

**Solution**: Create service in Twilio console.

## Production Considerations

When using Twilio in production:

- **Webhook security**: Verify webhook signatures
- **Rate limiting**: Handle Twilio rate limits
- **Error handling**: Handle API failures gracefully
- **Multi-channel**: Support SMS, WhatsApp, etc.
- **Cost management**: Monitor message costs

## Next Steps

Congratulations! You've integrated Twilio with Beluga AI. Next, learn how to:

- **[Slack Webhook Handler](./slack-webhook-handler.md)** - Slack integration
- **[Messaging Package Documentation](../../api/packages/messaging.md)** - Deep dive into messaging package
- **[Messaging Use Cases](../../use-cases/)** - Messaging patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
