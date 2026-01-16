package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- SHA1 required by Twilio API for webhook signature validation
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// HandleInboundCall handles an inbound call webhook event from Twilio.
func (b *TwilioBackend) HandleInboundCall(ctx context.Context, webhookData map[string]string) (vbiface.VoiceSession, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.HandleInboundCall")
	defer span.End()

	startTime := time.Now()

	// Validate webhook signature
	if err := b.validateWebhookSignature(webhookData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, "call.inbound", time.Since(startTime), false)
		}
		return nil, NewTwilioError("HandleInboundCall", ErrCodeInvalidSignature, err)
	}

	// Parse webhook data
	callSID := webhookData["CallSid"]
	from := webhookData["From"]
	to := webhookData["To"]
	callStatus := webhookData["CallStatus"]

	span.SetAttributes(
		attribute.String("call_sid", callSID),
		attribute.String("call_from", from),
		attribute.String("call_to", to),
		attribute.String("call_status", callStatus),
	)

	// Create session config from webhook data
	sessionConfig := &vbiface.SessionConfig{
		Metadata: map[string]any{
			"to":   to,
			"from": from,
		},
		// Additional config from webhook
	}

	// Create voice session
	session, err := b.CreateSession(ctx, sessionConfig)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, "call.inbound", time.Since(startTime), false)
		}
		return nil, err
	}

	if b.metrics != nil {
		b.metrics.RecordWebhook(ctx, "call.inbound", time.Since(startTime), true)
	}

	span.SetStatus(codes.Ok, "inbound call handled")
	return session, nil
}

// validateWebhookSignature validates the Twilio webhook signature.
func (b *TwilioBackend) validateWebhookSignature(webhookData map[string]string) error {
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
	// Note: Twilio requires SHA1 for webhook signature validation (Twilio API requirement)
	// #nosec G505 -- SHA1 required by Twilio API for webhook signature validation
	mac := hmac.New(sha1.New, []byte(b.config.AuthToken))
	mac.Write([]byte(signatureString))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Compare signatures (use constant-time comparison)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("invalid webhook signature")
	}

	return nil
}

// ParseWebhookEvent parses a webhook event from Twilio.
func (b *TwilioBackend) ParseWebhookEvent(ctx context.Context, webhookData map[string]string) (*WebhookEvent, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.ParseWebhookEvent")
	defer span.End()

	eventType := webhookData["EventType"]
	if eventType == "" {
		// Infer event type from other fields
		if callSID := webhookData["CallSid"]; callSID != "" {
			callStatus := webhookData["CallStatus"]
			eventType = fmt.Sprintf("call.%s", callStatus)
		} else if streamSID := webhookData["StreamSid"]; streamSID != "" {
			eventType = "stream.event"
		}
	}

	event := &WebhookEvent{
		EventID:     fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType:   eventType,
		EventData:   make(map[string]any),
		AccountSID:  webhookData["AccountSid"],
		Timestamp:   time.Now(),
		Signature:   webhookData["X-Twilio-Signature"],
		Source:      "voice-api",
		ResourceSID: webhookData["CallSid"],
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

// WebhookEvent represents a webhook event from Twilio.
type WebhookEvent struct {
	EventID     string
	EventType   string
	EventData   map[string]any
	AccountSID  string
	Timestamp   time.Time
	Signature   string
	Source      string
	ResourceSID string
	Metadata    map[string]any
}

// HandleWebhook handles a generic webhook event from Twilio.
func (b *TwilioBackend) HandleWebhook(ctx context.Context, webhookData map[string]string) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.HandleWebhook")
	defer span.End()

	startTime := time.Now()

	// Validate signature
	if err := b.validateWebhookSignature(webhookData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, "unknown", time.Since(startTime), false)
		}
		return NewTwilioError("HandleWebhook", ErrCodeInvalidSignature, err)
	}

	// Parse event
	event, err := b.ParseWebhookEvent(ctx, webhookData)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, "unknown", time.Since(startTime), false)
		}
		return err
	}

	span.SetAttributes(attribute.String("event_type", event.EventType))

	// Route event to appropriate handler
	switch {
	case strings.HasPrefix(event.EventType, "call."):
		// Use specialized handler
		if event.EventType == "call.status" {
			return b.HandleCallStatusEvent(ctx, event)
		}
		return b.handleCallEvent(ctx, event)
	case strings.HasPrefix(event.EventType, "stream."):
		return b.HandleStreamEvent(ctx, event)
	case strings.HasPrefix(event.EventType, "transcription."):
		if event.EventType == "transcription.completed" {
			return b.HandleTranscriptionCompletedEvent(ctx, event)
		}
		return b.handleTranscriptionEvent(ctx, event)
	default:
		// Unknown event type - log but don't error
		span.SetStatus(codes.Ok, "event handled (unknown type)")
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, event.EventType, time.Since(startTime), true)
		}
		return nil
	}
}

// handleCallEvent handles call-related webhook events.
func (b *TwilioBackend) handleCallEvent(ctx context.Context, event *WebhookEvent) error {
	callSID := event.ResourceSID
	callStatus := ""
	if status, ok := event.EventData["CallStatus"].(string); ok {
		callStatus = status
	} else if status, ok := event.EventData["CallStatus"]; ok {
		callStatus = fmt.Sprintf("%v", status)
	}

	switch callStatus {
	case "initiated":
		// Call initiated
	case "ringing":
		// Call ringing
	case "answered":
		// Call answered - create session if needed
		_, err := b.HandleInboundCall(ctx, convertEventDataToMap(event.EventData))
		return err
	case "completed":
		// Call completed - close session
		b.mu.RLock()
		_, exists := b.sessions[callSID]
		b.mu.RUnlock()
		if exists {
			return b.CloseSession(ctx, callSID)
		}
	case "failed", "busy", "no-answer", "canceled":
		// Call failed - cleanup
		b.mu.RLock()
		_, exists := b.sessions[callSID]
		b.mu.RUnlock()
		if exists {
			return b.CloseSession(ctx, callSID)
		}
	}

	return nil
}

// handleStreamEvent handles stream-related webhook events.
func (b *TwilioBackend) handleStreamEvent(ctx context.Context, event *WebhookEvent) error {
	// Handle stream events (start, stop, etc.)
	return nil
}

// handleTranscriptionEvent handles transcription-related webhook events.
func (b *TwilioBackend) handleTranscriptionEvent(ctx context.Context, event *WebhookEvent) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.handleTranscriptionEvent")
	defer span.End()

	startTime := time.Now()

	transcriptionSID := event.ResourceSID
	span.SetAttributes(attribute.String("transcription_sid", transcriptionSID))

	// Get transcription manager
	transcriptionMgr := NewTranscriptionManager(b)

	// Retrieve transcription from Twilio API
	transcription, err := transcriptionMgr.RetrieveTranscription(ctx, transcriptionSID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, event.EventType, time.Since(startTime), false)
		}
		return err
	}

	// Store transcription in vector store for RAG
	if err := transcriptionMgr.StoreTranscription(ctx, transcription); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordWebhook(ctx, event.EventType, time.Since(startTime), false)
		}
		return err
	}

	if b.metrics != nil {
		b.metrics.RecordWebhook(ctx, event.EventType, time.Since(startTime), true)
		b.metrics.RecordTranscription(ctx, transcriptionSID, true)
	}

	span.SetStatus(codes.Ok, "transcription event handled")
	return nil
}

// convertEventDataToMap converts event data to a string map for webhook handling.
func convertEventDataToMap(eventData map[string]any) map[string]string {
	result := make(map[string]string)
	for key, value := range eventData {
		if str, ok := value.(string); ok {
			result[key] = str
		} else {
			result[key] = fmt.Sprintf("%v", value)
		}
	}
	return result
}

// BuildWebhookURL builds a webhook URL for Twilio configuration.
func (b *TwilioBackend) BuildWebhookURL(path string) string {
	baseURL := b.config.WebhookURL
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
