package iface

import (
	"context"
)

// ConversationalBackend defines the interface for conversational backend instances.
// A conversational backend manages multi-channel messaging conversations (SMS, WhatsApp, etc.)
// and integrates with AI agents for automated responses.
type ConversationalBackend interface {
	// Lifecycle
	// Start starts the conversational backend.
	Start(ctx context.Context) error

	// Stop stops the conversational backend gracefully, completing in-flight conversations.
	Stop(ctx context.Context) error

	// Conversation Management
	// CreateConversation creates a new conversation with the given configuration.
	CreateConversation(ctx context.Context, config *ConversationConfig) (*Conversation, error)

	// GetConversation retrieves a conversation by ID.
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)

	// ListConversations returns all conversations (with optional filtering).
	ListConversations(ctx context.Context) ([]*Conversation, error)

	// CloseConversation closes a conversation and cleans up resources.
	CloseConversation(ctx context.Context, conversationID string) error

	// Message Operations
	// SendMessage sends a message to a conversation.
	SendMessage(ctx context.Context, conversationID string, message *Message) error

	// ReceiveMessages returns a channel for receiving messages from a conversation.
	ReceiveMessages(ctx context.Context, conversationID string) (<-chan *Message, error)

	// Participant Management
	// AddParticipant adds a participant to a conversation.
	AddParticipant(ctx context.Context, conversationID string, participant *Participant) error

	// RemoveParticipant removes a participant from a conversation.
	RemoveParticipant(ctx context.Context, conversationID string, participantID string) error

	// Webhook Handling
	// HandleWebhook handles a webhook event from the provider.
	HandleWebhook(ctx context.Context, event *WebhookEvent) error

	// Health & Status
	// HealthCheck performs a health check on the backend instance.
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// GetConfig returns the backend configuration.
	GetConfig() interface{} // Returns *messaging.Config - using interface{} to avoid circular import
}
