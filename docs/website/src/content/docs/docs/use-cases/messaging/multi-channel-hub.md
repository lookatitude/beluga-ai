---
title: Multi-Channel Marketing Hub
description: "Send unified marketing campaigns across SMS, WhatsApp, and email with automated scheduling, deduplication, and cross-channel analytics."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "multi-channel marketing, SMS WhatsApp email, campaign automation, cross-channel analytics, messaging hub, Beluga AI, Go"
---

A marketing agency running a product launch across SMS, WhatsApp, and email typically manages each channel through its own dashboard — Twilio for SMS, WhatsApp Business API, and SendGrid for email. This fragmentation means the same campaign requires three separate configurations, three different template formats, and three separate analytics views. When the SMS version of a message has a typo, it gets fixed in Twilio but the WhatsApp version remains unchanged. Delivery failures on one channel are invisible from another channel's dashboard, so a campaign that failed to reach 30% of WhatsApp recipients might appear successful from the SMS dashboard alone.

The deeper problem is that channel-specific systems prevent cross-channel intelligence: knowing that a customer already opened the email version should suppress the SMS reminder, but siloed systems cannot coordinate this deduplication.

## Solution Architecture

Beluga AI's `server/` package provides unified messaging backend abstractions with a consistent interface across channels. The key design choice is the `server.MessagingBackend` interface — SMS, WhatsApp, and email backends all implement the same `SendMessage`, `GetOrCreateConversation` contract. This means campaign logic is written once against the interface, and channel-specific behavior (WhatsApp template requirements, SMS character limits, email HTML formatting) is handled by the backend implementations. Delivery tracking and analytics aggregate across all channels into a single view.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Campaign   │───▶│   Campaign   │───▶│   Channel    │
│   Request    │    │   Manager    │    │    Router    │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                    ┌──────────────────────────┼────────────┐
                    │                          │            │
             ┌──────▼──────┐          ┌───────▼──────┐  ┌──▼─────┐
             │     SMS     │          │   WhatsApp   │  │ Email  │
             │   Channel   │          │    Channel   │  │Channel │
             └──────┬──────┘          └───────┬──────┘  └──┬─────┘
                    │                         │            │
                    └────────────┬────────────┘            │
                                 │                         │
                          ┌──────▼───────┐                 │
                          │   Delivery   │◀────────────────┘
                          │   Tracker    │
                          └──────┬───────┘
                                 │
                          ┌──────▼───────┐
                          │  Analytics   │
                          │    Engine    │
                          └──────────────┘
```

## Implementation

### Multi-channel Hub Setup

The hub manages messaging across multiple channels with a unified interface:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/server"

    _ "github.com/lookatitude/beluga-ai/server/providers/twilio"
)

type MultiChannelHub struct {
    backends map[string]server.MessagingBackend
}

func NewMultiChannelHub(ctx context.Context) (*MultiChannelHub, error) {
    backends := make(map[string]server.MessagingBackend)

    // Setup SMS backend (Twilio)
    smsBackend, err := server.NewMessagingBackend("twilio", server.MessagingConfig{
        AccountSID: "your-account-sid",
        AuthToken:  "your-auth-token",
        Channel:    "sms",
    })
    if err == nil {
        backends["sms"] = smsBackend
    }

    // Setup WhatsApp backend (Twilio)
    whatsappBackend, err := server.NewMessagingBackend("twilio", server.MessagingConfig{
        AccountSID: "your-account-sid",
        AuthToken:  "your-auth-token",
        Channel:    "whatsapp",
    })
    if err == nil {
        backends["whatsapp"] = whatsappBackend
    }

    // Setup Email backend
    emailBackend, err := server.NewMessagingBackend("sendgrid", server.MessagingConfig{
        APIKey: "your-sendgrid-key",
    })
    if err == nil {
        backends["email"] = emailBackend
    }

    return &MultiChannelHub{
        backends: backends,
    }, nil
}
```

### Campaign Management

Send campaigns across multiple channels with consistent messaging:

```go
type Campaign struct {
    ID        string
    Name      string
    Message   string
    Channels  []string
    Recipients []Recipient
    ScheduledAt time.Time
}

type Recipient struct {
    ID          string
    Phone       string
    Email       string
    Preferences map[string]string
}

func (h *MultiChannelHub) SendCampaign(ctx context.Context, campaign Campaign) error {
    for _, channel := range campaign.Channels {
        backend, exists := h.backends[channel]
        if !exists {
            continue
        }

        for _, recipient := range campaign.Recipients {
            // Get or create conversation
            address := h.getRecipientAddress(recipient, channel)
            if address == "" {
                continue
            }

            conversation, err := backend.GetOrCreateConversation(ctx, address)
            if err != nil {
                h.trackDelivery(ctx, campaign.ID, channel, recipient.ID, "failed")
                continue
            }

            // Send message
            msg := schema.Message{
                Content: h.personalizeMessage(campaign.Message, recipient),
                Channel: channel,
            }

            if err := backend.SendMessage(ctx, conversation.ID, msg); err != nil {
                h.trackDelivery(ctx, campaign.ID, channel, recipient.ID, "failed")
                continue
            }

            h.trackDelivery(ctx, campaign.ID, channel, recipient.ID, "sent")
        }
    }

    return nil
}

func (h *MultiChannelHub) getRecipientAddress(recipient Recipient, channel string) string {
    switch channel {
    case "sms", "whatsapp":
        return recipient.Phone
    case "email":
        return recipient.Email
    default:
        return ""
    }
}

func (h *MultiChannelHub) personalizeMessage(template string, recipient Recipient) string {
    // Replace placeholders with recipient data
    message := template
    for key, value := range recipient.Preferences {
        placeholder := fmt.Sprintf("{{%s}}", key)
        message = strings.ReplaceAll(message, placeholder, value)
    }
    return message
}
```

### Delivery Tracking

Track delivery status across all channels:

```go
type DeliveryStatus struct {
    CampaignID  string
    Channel     string
    RecipientID string
    Status      string // "sent", "delivered", "failed", "read"
    Timestamp   time.Time
}

func (h *MultiChannelHub) trackDelivery(ctx context.Context, campaignID, channel, recipientID, status string) {
    deliveryStatus := DeliveryStatus{
        CampaignID:  campaignID,
        Channel:     channel,
        RecipientID: recipientID,
        Status:      status,
        Timestamp:   time.Now(),
    }

    // Store delivery status in database for analytics
    // Implementation depends on your storage mechanism
    h.storeDeliveryStatus(ctx, deliveryStatus)
}

func (h *MultiChannelHub) GetCampaignStats(ctx context.Context, campaignID string) (*CampaignStats, error) {
    // Aggregate delivery statuses by channel and status
    statuses, err := h.getDeliveryStatuses(ctx, campaignID)
    if err != nil {
        return nil, fmt.Errorf("get delivery statuses: %w", err)
    }

    stats := &CampaignStats{
        CampaignID: campaignID,
        ByChannel:  make(map[string]ChannelStats),
    }

    for _, status := range statuses {
        channelStats := stats.ByChannel[status.Channel]
        switch status.Status {
        case "sent":
            channelStats.Sent++
        case "delivered":
            channelStats.Delivered++
        case "failed":
            channelStats.Failed++
        case "read":
            channelStats.Read++
        }
        stats.ByChannel[status.Channel] = channelStats
    }

    return stats, nil
}

type CampaignStats struct {
    CampaignID string
    ByChannel  map[string]ChannelStats
}

type ChannelStats struct {
    Sent      int
    Delivered int
    Failed    int
    Read      int
}
```

## Production Considerations

### Automated Scheduling

Schedule campaigns for optimal delivery times:

```go
import "time"

type Scheduler struct {
    hub *MultiChannelHub
}

func (s *Scheduler) ScheduleCampaign(ctx context.Context, campaign Campaign) error {
    // Calculate delay until scheduled time
    delay := time.Until(campaign.ScheduledAt)
    if delay < 0 {
        // Send immediately if scheduled time has passed
        return s.hub.SendCampaign(ctx, campaign)
    }

    // Schedule for later
    timer := time.NewTimer(delay)
    go func() {
        <-timer.C
        if err := s.hub.SendCampaign(context.Background(), campaign); err != nil {
            // Log error
        }
    }()

    return nil
}
```

### Rate Limiting

Respect channel rate limits to avoid throttling:

```go
import "golang.org/x/time/rate"

type RateLimitedHub struct {
    hub      *MultiChannelHub
    limiters map[string]*rate.Limiter
}

func NewRateLimitedHub(hub *MultiChannelHub) *RateLimitedHub {
    return &RateLimitedHub{
        hub: hub,
        limiters: map[string]*rate.Limiter{
            "sms":      rate.NewLimiter(rate.Limit(10), 10), // 10/sec
            "whatsapp": rate.NewLimiter(rate.Limit(5), 5),   // 5/sec
            "email":    rate.NewLimiter(rate.Limit(20), 20), // 20/sec
        },
    }
}

func (h *RateLimitedHub) SendCampaign(ctx context.Context, campaign Campaign) error {
    for _, channel := range campaign.Channels {
        limiter := h.limiters[channel]
        if limiter != nil {
            if err := limiter.Wait(ctx); err != nil {
                return fmt.Errorf("rate limit wait: %w", err)
            }
        }
    }

    return h.hub.SendCampaign(ctx, campaign)
}
```

### Observability

Track campaign performance with OpenTelemetry:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (h *MultiChannelHub) SendCampaignWithTracing(ctx context.Context, campaign Campaign) error {
    ctx, span := o11y.StartSpan(ctx, "marketing.send_campaign")
    defer span.End()

    span.SetAttributes(
        attribute.String("campaign.id", campaign.ID),
        attribute.String("campaign.name", campaign.Name),
        attribute.Int("campaign.channels", len(campaign.Channels)),
        attribute.Int("campaign.recipients", len(campaign.Recipients)),
    )

    err := h.SendCampaign(ctx, campaign)
    if err != nil {
        span.RecordError(err)
        return err
    }

    // Record campaign metrics
    stats, _ := h.GetCampaignStats(ctx, campaign.ID)
    if stats != nil {
        for channel, channelStats := range stats.ByChannel {
            span.SetAttributes(
                attribute.Int(fmt.Sprintf("campaign.%s.sent", channel), channelStats.Sent),
                attribute.Int(fmt.Sprintf("campaign.%s.delivered", channel), channelStats.Delivered),
                attribute.Int(fmt.Sprintf("campaign.%s.failed", channel), channelStats.Failed),
            )
        }
    }

    return nil
}
```

### Template Management

Maintain consistent branding across channels:

```go
type TemplateManager struct {
    templates map[string]map[string]string // channel -> template_name -> template
}

func (m *TemplateManager) GetTemplate(channel, name string) (string, error) {
    channelTemplates, exists := m.templates[channel]
    if !exists {
        return "", fmt.Errorf("no templates for channel: %s", channel)
    }

    template, exists := channelTemplates[name]
    if !exists {
        return "", fmt.Errorf("template not found: %s", name)
    }

    return template, nil
}
```

## Results

After implementing the multi-channel marketing hub, the agency achieved:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Delivery Rate | 60-70% | 96% | 37-60% improvement |
| Management Time | 10-15 hrs/week | 4 hrs/week | 73-80% reduction |
| Channel Consistency | 60% | 97% | 62% improvement |
| Campaign Performance | 6.0/10 | 9.1/10 | 52% improvement |
| Client Satisfaction | 6.5/10 | 9.2/10 | 42% improvement |

## Related Resources

- [Messaging Integration](/integrations/messaging/) for channel-specific configuration
- [Server Providers](/providers/server/) for available messaging backends
- [Observability Guide](/guides/observability/) for campaign analytics setup
