package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// HandleWebhook handles a webhook event from Twilio Conversations API.
func (p *TwilioProvider) HandleWebhook(ctx context.Context, event *iface.WebhookEvent) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.HandleWebhook")
	defer span.End()

	startTime := time.Now()

	// Event is already parsed - validate signature if needed
	// Note: In a full implementation, signature validation would happen before parsing

	span.SetAttributes(attribute.String("event_type", event.EventType))

	// Route event to appropriate handler with retry logic for webhook delivery failures
	maxRetries := 3
	retryDelay := time.Second
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err = p.handleWebhookEvent(ctx, event)
		if err == nil {
			break
		}

		// Retry with exponential backoff for webhook delivery failures
		if attempt < maxRetries-1 {
			time.Sleep(retryDelay * time.Duration(1<<attempt))
		}
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if p.metrics != nil {
			p.metrics.RecordWebhook(ctx, event.EventType, time.Since(startTime), false)
		}
		return err
	}

	if p.metrics != nil {
		p.metrics.RecordWebhook(ctx, event.EventType, time.Since(startTime), true)
	}

	span.SetStatus(codes.Ok, "webhook handled")
	return nil
}

// handleWebhookEvent routes the event to the appropriate handler.
func (p *TwilioProvider) handleWebhookEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Route to specialized handlers
	switch {
	case strings.HasPrefix(event.EventType, "conversation."):
		return p.HandleConversationEvent(ctx, event)
	case strings.HasPrefix(event.EventType, "message."):
		return p.HandleMessageEvent(ctx, event)
	case strings.HasPrefix(event.EventType, "participant."):
		return p.HandleParticipantEvent(ctx, event)
	case strings.HasPrefix(event.EventType, "typing."):
		return p.HandleTypingEvent(ctx, event)
	default:
		// Unknown event type - log but don't error
		return nil
	}
}

// ParseWebhookEvent parses a webhook event from Twilio.
func (p *TwilioProvider) ParseWebhookEvent(ctx context.Context, webhookData map[string]string) (*iface.WebhookEvent, error) {
	ctx, span := p.startSpan(ctx, "TwilioProvider.ParseWebhookEvent")
	defer span.End()

	eventType := webhookData["EventType"]
	if eventType == "" {
		// Infer event type from other fields
		if messageSID := webhookData["MessageSid"]; messageSID != "" {
			eventType = "message.added"
		} else if conversationSID := webhookData["ConversationSid"]; conversationSID != "" {
			eventType = "conversation.created"
		}
	}

	event := &iface.WebhookEvent{
		EventID:     fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType:   eventType,
		EventData:   make(map[string]any),
		AccountSID:  webhookData["AccountSid"],
		Timestamp:   time.Now(),
		Signature:   webhookData["X-Twilio-Signature"],
		Source:      "conversations-api",
		ResourceSID: webhookData["MessageSid"],
		Metadata:    make(map[string]any),
	}

	// Copy webhook data to event data
	for key, value := range webhookData {
		if key != "X-Twilio-Signature" {
			event.EventData[key] = value
		}
	}

	span.SetAttributes(
		attribute.String("event_type", eventType),
		attribute.String("resource_sid", event.ResourceSID),
	)

	span.SetStatus(codes.Ok, "webhook event parsed")
	return event, nil
}

// validateWebhookSignature validates the Twilio webhook signature.
func (p *TwilioProvider) validateWebhookSignature(webhookData map[string]string) error {
	signature := webhookData["X-Twilio-Signature"]
	if signature == "" {
		return errors.New("missing X-Twilio-Signature header")
	}

	// Get the full URL from webhook data (should be set by HTTP handler)
	// The HTTP handler should extract the URL from the request and add it as "_url" key
	url := webhookData["_url"]
	if url == "" {
		return errors.New("missing webhook URL for signature validation")
	}

	// Build signature string
	// Twilio signature is computed as: HMAC-SHA1(AuthToken, URL + sorted parameters)
	params := make([]string, 0, len(webhookData))
	for key, value := range webhookData {
		if key != "X-Twilio-Signature" && key != "_url" {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
	}
	sort.Strings(params)
	paramString := strings.Join(params, "")

	signatureString := url + paramString

	// Compute expected signature
	mac := hmac.New(sha1.New, []byte(p.config.AuthToken))
	mac.Write([]byte(signatureString))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Compare signatures (use constant-time comparison)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("invalid webhook signature")
	}

	return nil
}

// handleMessageAddedEvent handles message.added webhook events.
func (p *TwilioProvider) handleMessageAddedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	conversationID := ""
	if convSID, ok := event.EventData["ConversationSid"].(string); ok {
		conversationID = convSID
	} else if convSID, ok := event.EventData["ConversationSid"].(string); ok {
		conversationID = convSID
	}

	if conversationID == "" {
		return fmt.Errorf("missing ConversationSid in event data")
	}

	// Get or create session
	p.mu.RLock()
	session, exists := p.sessions[conversationID]
	p.mu.RUnlock()

	if !exists {
		// Create new session
		var err error
		session, err = NewMessagingSession(conversationID, p.config, p)
		if err != nil {
			return err
		}

		p.mu.Lock()
		p.sessions[conversationID] = session
		p.mu.Unlock()

		// Start session
		if err := session.Start(ctx); err != nil {
			return err
		}
	}

	// Parse message from event data
	message := &iface.Message{
		MessageSID:      getStringFromEvent(event.EventData, "MessageSid"),
		ConversationSID: conversationID,
		Body:            getStringFromEvent(event.EventData, "Body"),
		Channel:         iface.Channel(getStringFromEvent(event.EventData, "Channel")),
		From:            getStringFromEvent(event.EventData, "Author"),
		DateCreated:     time.Now(),
	}

	// Send to message channel
	p.mu.RLock()
	ch, exists := p.messageChannels[conversationID]
	p.mu.RUnlock()

	if exists {
		select {
		case ch <- message:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full - log warning
		}
	}

	// Process message through session
	return session.ProcessMessage(ctx, message)
}

// handleConversationCreatedEvent handles conversation.created webhook events.
func (p *TwilioProvider) handleConversationCreatedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Update local cache
	conversationID := getStringFromEvent(event.EventData, "ConversationSid")
	if conversationID != "" {
		// Fetch and cache conversation
		_, err := p.GetConversation(ctx, conversationID)
		return err
	}
	return nil
}

// handleConversationUpdatedEvent handles conversation.updated webhook events.
func (p *TwilioProvider) handleConversationUpdatedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Update local cache
	conversationID := getStringFromEvent(event.EventData, "ConversationSid")
	if conversationID != "" {
		// Fetch and update cache
		_, err := p.GetConversation(ctx, conversationID)
		return err
	}
	return nil
}

// handleMessageDeliveryUpdatedEvent handles message.delivery.updated webhook events.
func (p *TwilioProvider) handleMessageDeliveryUpdatedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Update message delivery status
	return nil
}

// handleParticipantAddedEvent handles participant.added webhook events.
func (p *TwilioProvider) handleParticipantAddedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Update participant list
	return nil
}

// handleParticipantRemovedEvent handles participant.removed webhook events.
func (p *TwilioProvider) handleParticipantRemovedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Update participant list
	return nil
}

// handleTypingStartedEvent handles typing.started webhook events.
func (p *TwilioProvider) handleTypingStartedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Handle typing indicator
	return nil
}

// getStringFromEvent extracts a string value from event data.
func getStringFromEvent(eventData map[string]any, key string) string {
	if value, ok := eventData[key].(string); ok {
		return value
	}
	if value, ok := eventData[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// BuildWebhookURL builds a webhook URL for Twilio configuration.
func (p *TwilioProvider) BuildWebhookURL(path string) string {
	baseURL := p.config.WebhookURL
	if baseURL == "" {
		return ""
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	u.Path = path
	return u.String()
}
