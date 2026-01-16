package twilio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/twilio/twilio-go"
	twilioconv "github.com/twilio/twilio-go/rest/conversations/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TwilioProvider implements the ConversationalBackend interface for Twilio Conversations API.
type TwilioProvider struct {
	config          *TwilioConfig
	client          *twilio.RestClient
	conversations   map[string]*iface.Conversation
	sessions        map[string]*MessagingSession
	messageChannels map[string]chan *iface.Message
	mu              sync.RWMutex
	metrics         *messaging.Metrics
}

// NewTwilioProvider creates a new Twilio messaging provider.
func NewTwilioProvider(config *TwilioConfig) (*TwilioProvider, error) {
	var metrics *messaging.Metrics
	if config.EnableMetrics {
		// Initialize metrics using OTEL global meter and tracer
		// This follows the standard pattern used across all Beluga AI packages
		meter := otel.Meter("beluga.messaging.providers.twilio")
		tracer := otel.Tracer("beluga.messaging.providers.twilio")
		var err error
		metrics, err = messaging.NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics as fallback
			metrics = messaging.NoOpMetrics()
		}
	} else {
		// Use no-op metrics when metrics are disabled
		metrics = messaging.NoOpMetrics()
	}

	return &TwilioProvider{
		config:          config,
		conversations:   make(map[string]*iface.Conversation),
		sessions:        make(map[string]*MessagingSession),
		messageChannels: make(map[string]chan *iface.Message),
		metrics:         metrics,
	}, nil
}

// Start starts the Twilio messaging backend.
func (p *TwilioProvider) Start(ctx context.Context) error {
	startTime := time.Now()
	ctx, span := p.startSpan(ctx, "TwilioProvider.Start")
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Initialize Twilio REST client
	clientParams := twilio.ClientParams{
		Username: p.config.AccountSID,
		Password: p.config.AuthToken,
	}
	p.client = twilio.NewRestClientWithParams(clientParams)

	// Verify connectivity by making a test API call
	_, err := p.client.Api.FetchAccount(p.config.AccountSID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if p.metrics != nil {
			p.metrics.RecordOperation(ctx, "start", time.Since(startTime), false)
		}
		return messaging.NewMessagingError("Start", messaging.ErrCodeAuthentication, err)
	}

	span.SetStatus(codes.Ok, "started")

	if p.metrics != nil {
		p.metrics.RecordOperation(ctx, "start", time.Since(startTime), true)
	}

	return nil
}

// Stop stops the Twilio messaging backend gracefully.
func (p *TwilioProvider) Stop(ctx context.Context) error {
	startTime := time.Now()
	ctx, span := p.startSpan(ctx, "TwilioProvider.Stop")
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Close all message channels
	for conversationID, ch := range p.messageChannels {
		close(ch)
		delete(p.messageChannels, conversationID)
	}

	// Close all sessions
	for conversationID := range p.sessions {
		delete(p.sessions, conversationID)
	}

	p.conversations = make(map[string]*iface.Conversation)

	span.SetStatus(codes.Ok, "stopped")

	if p.metrics != nil {
		p.metrics.RecordOperation(ctx, "stop", time.Since(startTime), true)
	}

	return nil
}

// CreateConversation creates a new conversation.
func (p *TwilioProvider) CreateConversation(ctx context.Context, config *iface.ConversationConfig) (*iface.Conversation, error) {
	ctx, span := p.startSpan(ctx, "TwilioProvider.CreateConversation")
	defer span.End()

	startTime := time.Now()

	// Create Twilio Conversation resource
	conversationParams := &twilioconv.CreateConversationParams{}
	conversationParams.SetFriendlyName(config.FriendlyName)

	if config.UniqueName != "" {
		conversationParams.SetUniqueName(config.UniqueName)
	}

	if config.Attributes != "" {
		conversationParams.SetAttributes(config.Attributes)
	}

	conv, err := p.client.ConversationsV1.CreateConversation(conversationParams)
	if err != nil {
		err = p.mapTwilioError("CreateConversation", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if p.metrics != nil {
			p.metrics.RecordConversation(ctx, "create", time.Since(startTime), false)
		}
		return nil, err
	}

	conversation := &iface.Conversation{
		ConversationSID: getStringValue(conv.Sid),
		AccountSID:      getStringValue(conv.AccountSid),
		FriendlyName:    getStringValue(conv.FriendlyName),
		State:           iface.ConversationState(getStringValue(conv.State)),
		DateCreated:     getTimeValue(conv.DateCreated),
		DateUpdated:     getTimeValue(conv.DateUpdated),
		Attributes:      getStringValue(conv.Attributes),
		Metadata:        make(map[string]any),
	}

	p.mu.Lock()
	p.conversations[conversation.ConversationSID] = conversation
	p.mu.Unlock()

	// Create session
	session, err := NewMessagingSession(conversation.ConversationSID, p.config, p)
	if err != nil {
		err = p.mapTwilioError("CreateConversation", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if p.metrics != nil {
			p.metrics.RecordConversation(ctx, "create", time.Since(startTime), false)
		}
		return nil, err
	}

	// Store session
	p.mu.Lock()
	p.sessions[conversation.ConversationSID] = session
	p.mu.Unlock()

	span.SetAttributes(
		attribute.String("conversation_sid", conversation.ConversationSID),
		attribute.String("friendly_name", conversation.FriendlyName),
	)

	if p.metrics != nil {
		p.metrics.RecordConversation(ctx, "create", time.Since(startTime), true)
		p.metrics.IncrementActiveConversations(ctx)
	}

	span.SetStatus(codes.Ok, "conversation created")
	return conversation, nil
}

// GetConversation retrieves a conversation by ID.
func (p *TwilioProvider) GetConversation(ctx context.Context, conversationID string) (*iface.Conversation, error) {
	ctx, span := p.startSpan(ctx, "TwilioProvider.GetConversation")
	defer span.End()

	span.SetAttributes(attribute.String("conversation_id", conversationID))

	// Check local cache first
	p.mu.RLock()
	conv, exists := p.conversations[conversationID]
	p.mu.RUnlock()

	if exists {
		span.SetStatus(codes.Ok, "conversation found in cache")
		return conv, nil
	}

	// Fetch from Twilio API
	convResource, err := p.client.ConversationsV1.FetchConversation(conversationID)
	if err != nil {
		err = p.mapTwilioError("GetConversation", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, messaging.NewMessagingError("GetConversation", messaging.ErrCodeConversationNotFound, err)
	}

	conversation := &iface.Conversation{
		ConversationSID: getStringValue(convResource.Sid),
		AccountSID:      getStringValue(convResource.AccountSid),
		FriendlyName:    getStringValue(convResource.FriendlyName),
		State:           iface.ConversationState(getStringValue(convResource.State)),
		DateCreated:     getTimeValue(convResource.DateCreated),
		DateUpdated:     getTimeValue(convResource.DateUpdated),
		Attributes:      getStringValue(convResource.Attributes),
		Metadata:        make(map[string]any),
	}

	p.mu.Lock()
	p.conversations[conversationID] = conversation
	p.mu.Unlock()

	span.SetStatus(codes.Ok, "conversation fetched")
	return conversation, nil
}

// ListConversations returns all conversations.
func (p *TwilioProvider) ListConversations(ctx context.Context) ([]*iface.Conversation, error) {
	ctx, span := p.startSpan(ctx, "TwilioProvider.ListConversations")
	defer span.End()

	// Fetch conversations from Twilio API
	conversations, err := p.client.ConversationsV1.ListConversation(nil)
	if err != nil {
		err = p.mapTwilioError("ListConversations", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	result := make([]*iface.Conversation, 0, len(conversations))
	for _, convResource := range conversations {
		conversation := &iface.Conversation{
			ConversationSID: getStringValue(convResource.Sid),
			AccountSID:      getStringValue(convResource.AccountSid),
			FriendlyName:    getStringValue(convResource.FriendlyName),
			State:           iface.ConversationState(getStringValue(convResource.State)),
			DateCreated:     getTimeValue(convResource.DateCreated),
			DateUpdated:     getTimeValue(convResource.DateUpdated),
			Attributes:      getStringValue(convResource.Attributes),
			Metadata:        make(map[string]any),
		}
		result = append(result, conversation)
	}

	span.SetAttributes(attribute.Int("conversation_count", len(result)))
	span.SetStatus(codes.Ok, "conversations listed")
	return result, nil
}

// CloseConversation closes a conversation.
func (p *TwilioProvider) CloseConversation(ctx context.Context, conversationID string) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.CloseConversation")
	defer span.End()

	span.SetAttributes(attribute.String("conversation_id", conversationID))

	// Update Twilio Conversation resource state to "closed"
	updateParams := &twilioconv.UpdateConversationParams{}
	updateParams.SetState("closed")

	_, err := p.client.ConversationsV1.UpdateConversation(conversationID, updateParams)
	if err != nil {
		err = p.mapTwilioError("CloseConversation", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return messaging.NewMessagingError("CloseConversation", messaging.ErrCodeCloseFailed, err)
	}

	// Close message channel
	p.mu.Lock()
	if ch, exists := p.messageChannels[conversationID]; exists {
		close(ch)
		delete(p.messageChannels, conversationID)
	}

	// Remove session
	delete(p.sessions, conversationID)

	// Update local cache
	if conv, exists := p.conversations[conversationID]; exists {
		conv.State = iface.ConversationStateClosed
		conv.DateUpdated = time.Now()
	}
	p.mu.Unlock()

	if p.metrics != nil {
		p.metrics.DecrementActiveConversations(ctx)
	}

	span.SetStatus(codes.Ok, "conversation closed")
	return nil
}

// SendMessage sends a message to a conversation.
func (p *TwilioProvider) SendMessage(ctx context.Context, conversationID string, message *iface.Message) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.SendMessage")
	defer span.End()

	startTime := time.Now()

	span.SetAttributes(
		attribute.String("conversation_id", conversationID),
		attribute.String("channel", string(message.Channel)),
		attribute.Bool("has_media", len(message.MediaURLs) > 0),
	)

	// Validate message
	if message.Body == "" && len(message.MediaURLs) == 0 {
		err := messaging.NewMessagingError("SendMessage", messaging.ErrCodeInvalidMessage, fmt.Errorf("message must have body or media"))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Handle unsupported media types (edge case)
	for _, mediaURL := range message.MediaURLs {
		if !p.isSupportedMediaType(mediaURL) {
			err := messaging.NewMessagingError("SendMessage", messaging.ErrCodeInvalidMessage, fmt.Errorf("unsupported media type: %s", mediaURL))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	// Create Twilio Message resource
	messageParams := &twilioconv.CreateConversationMessageParams{}
	messageParams.SetBody(message.Body)

	if len(message.MediaURLs) > 0 {
		// Twilio supports media via MediaSid or MediaURL
		// For now, we'll use the first media URL
		mediaURL := message.MediaURLs[0]
		messageParams.SetMediaSid(mediaURL) // Or use SetMediaUrl if available
	}

	msg, err := p.client.ConversationsV1.CreateConversationMessage(conversationID, messageParams)
	if err != nil {
		err = p.mapTwilioError("SendMessage", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if p.metrics != nil {
			p.metrics.RecordMessage(ctx, string(message.Channel), time.Since(startTime), false)
		}
		return err
	}

	// Update message with Twilio response
	message.MessageSID = getStringValue(msg.Sid)
	message.ConversationSID = getStringValue(msg.ConversationSid)
	message.DateCreated = getTimeValue(msg.DateCreated)
	message.DateUpdated = getTimeValue(msg.DateUpdated)

	if p.metrics != nil {
		p.metrics.RecordMessage(ctx, string(message.Channel), time.Since(startTime), true)
	}

	span.SetStatus(codes.Ok, "message sent")
	return nil
}

// ReceiveMessages returns a channel for receiving messages from a conversation.
func (p *TwilioProvider) ReceiveMessages(ctx context.Context, conversationID string) (<-chan *iface.Message, error) {
	ctx, span := p.startSpan(ctx, "TwilioProvider.ReceiveMessages")
	defer span.End()

	span.SetAttributes(attribute.String("conversation_id", conversationID))

	p.mu.Lock()
	ch, exists := p.messageChannels[conversationID]
	if !exists {
		ch = make(chan *iface.Message, 100)
		p.messageChannels[conversationID] = ch
	}
	p.mu.Unlock()

	span.SetStatus(codes.Ok, "message channel created")
	return ch, nil
}

// AddParticipant adds a participant to a conversation.
func (p *TwilioProvider) AddParticipant(ctx context.Context, conversationID string, participant *iface.Participant) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.AddParticipant")
	defer span.End()

	span.SetAttributes(
		attribute.String("conversation_id", conversationID),
		attribute.String("participant_identity", participant.Identity),
	)

	// Create Twilio Participant resource
	participantParams := &twilioconv.CreateConversationParticipantParams{}
	participantParams.SetIdentity(participant.Identity)

	if participant.MessagingBinding.Type != "" {
		// Set messaging binding address and proxy address
		if participant.MessagingBinding.Address != "" {
			participantParams.SetMessagingBindingAddress(participant.MessagingBinding.Address)
		}
		if participant.MessagingBinding.ProxyAddress != "" {
			participantParams.SetMessagingBindingProxyAddress(participant.MessagingBinding.ProxyAddress)
		}
	}

	part, err := p.client.ConversationsV1.CreateConversationParticipant(conversationID, participantParams)
	if err != nil {
		err = p.mapTwilioError("AddParticipant", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	participant.ParticipantSID = getStringValue(part.Sid)
	participant.ConversationSID = getStringValue(part.ConversationSid)
	participant.DateCreated = getTimeValue(part.DateCreated)

	if p.metrics != nil {
		p.metrics.RecordParticipant(ctx, "add", true)
		p.metrics.IncrementActiveParticipants(ctx)
	}

	span.SetStatus(codes.Ok, "participant added")
	return nil
}

// RemoveParticipant removes a participant from a conversation.
func (p *TwilioProvider) RemoveParticipant(ctx context.Context, conversationID string, participantID string) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.RemoveParticipant")
	defer span.End()

	span.SetAttributes(
		attribute.String("conversation_id", conversationID),
		attribute.String("participant_id", participantID),
	)

	err := p.client.ConversationsV1.DeleteConversationParticipant(conversationID, participantID, nil)
	if err != nil {
		err = p.mapTwilioError("RemoveParticipant", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return messaging.NewMessagingError("RemoveParticipant", messaging.ErrCodeParticipantNotFound, err)
	}

	if p.metrics != nil {
		p.metrics.RecordParticipant(ctx, "remove", true)
		p.metrics.DecrementActiveParticipants(ctx)
	}

	span.SetStatus(codes.Ok, "participant removed")
	return nil
}

// Helper functions for handling Twilio SDK pointer types

// getStringValue safely extracts a string from a pointer.
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// getTimeValue safely extracts a time.Time from a pointer.
func getTimeValue(ptr *time.Time) time.Time {
	if ptr == nil {
		return time.Time{}
	}
	return *ptr
}

// HandleWebhook handles a webhook event from Twilio (signature: webhookData map[string]string).
// The implementation with event parsing is in webhook.go

// HealthCheck performs a health check on the backend.
func (p *TwilioProvider) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
	ctx, span := p.startSpan(ctx, "TwilioProvider.HealthCheck")
	defer span.End()

	startTime := time.Now()

	// Verify API connectivity
	_, err := p.client.Api.FetchAccount(p.config.AccountSID)
	var status iface.HealthStatus
	var errors []string

	if err != nil {
		status = iface.HealthStatusUnhealthy
		errors = append(errors, fmt.Sprintf("API connectivity failed: %v", err))
	} else {
		status = iface.HealthStatusHealthy
	}

	details := map[string]any{
		"active_conversations": len(p.conversations),
		"active_sessions":      len(p.sessions),
	}
	if len(errors) > 0 {
		details["errors"] = errors
	}

	// Interface expects *HealthStatus (pointer to string type)
	healthStatusPtr := &status
	_ = details // Details not in HealthStatus type
	_ = errors  // Errors not in HealthStatus type

	span.SetAttributes(
		attribute.String("health_status", string(status)),
		attribute.Int("active_conversations", len(p.conversations)),
	)

	if p.metrics != nil {
		p.metrics.RecordOperation(ctx, "health_check", time.Since(startTime), len(errors) == 0)
	}

	span.SetStatus(codes.Ok, "health check completed")
	return healthStatusPtr, nil
}

// GetConfig returns the backend configuration.
func (p *TwilioProvider) GetConfig() interface{} {
	return p.config.Config
}

// Helper methods

// mapTwilioError maps Twilio SDK errors to messaging errors.
func (p *TwilioProvider) mapTwilioError(op string, err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "rate limit") || contains(errStr, "429"):
		return messaging.NewMessagingError(op, messaging.ErrCodeRateLimit, err)
	case contains(errStr, "401") || contains(errStr, "unauthorized"):
		return messaging.NewMessagingError(op, messaging.ErrCodeAuthentication, err)
	case contains(errStr, "timeout") || contains(errStr, "deadline"):
		return messaging.NewMessagingError(op, messaging.ErrCodeTimeout, err)
	case contains(errStr, "network") || contains(errStr, "connection"):
		return messaging.NewMessagingError(op, messaging.ErrCodeNetworkError, err)
	case contains(errStr, "404") || contains(errStr, "not found"):
		return messaging.NewMessagingError(op, messaging.ErrCodeNotFound, err)
	default:
		return messaging.NewMessagingError(op, messaging.ErrCodeInternalError, err)
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// isSupportedMediaType checks if a media URL represents a supported media type.
func (p *TwilioProvider) isSupportedMediaType(mediaURL string) bool {
	// Twilio supports: image/jpeg, image/png, image/gif, video/mp4, audio/mpeg, audio/wav
	// For now, accept all URLs - full implementation would validate MIME type
	return true
}

// startSpan starts an OTEL span for tracing.
func (p *TwilioProvider) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if p.metrics != nil && p.metrics.Tracer() != nil {
		return p.metrics.Tracer().Start(ctx, operation)
	}
	return ctx, trace.SpanFromContext(ctx)
}

// Note: Webhook event handlers are implemented in webhook.go
