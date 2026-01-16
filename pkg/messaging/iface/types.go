package iface

import (
	"time"
)

// Channel represents the message channel type.
type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelChat     Channel = "chat"
	ChannelEmail    Channel = "email"
)

// ConversationState represents the state of a conversation.
type ConversationState string

const (
	ConversationStateActive   ConversationState = "active"
	ConversationStateClosed   ConversationState = "closed"
	ConversationStateInactive ConversationState = "inactive"
)

// DeliveryStatus represents the delivery status of a message.
type DeliveryStatus string

const (
	DeliveryStatusSent        DeliveryStatus = "sent"
	DeliveryStatusDelivered   DeliveryStatus = "delivered"
	DeliveryStatusRead        DeliveryStatus = "read"
	DeliveryStatusFailed      DeliveryStatus = "failed"
	DeliveryStatusUndelivered DeliveryStatus = "undelivered"
)

// HealthStatus represents the health status of a backend.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// Binding represents a messaging binding for a participant.
type Binding struct {
	Type         Channel `json:"type"`          // Binding type (sms, whatsapp, chat, email)
	Address      string  `json:"address"`       // Binding address (phone number, email, etc.)
	ProxyAddress string  `json:"proxy_address"` // Proxy address (if applicable)
}

// Conversation represents a multi-channel conversation thread.
type Conversation struct {
	ConversationSID string            `json:"conversation_sid"` // Unique conversation identifier
	AccountSID      string            `json:"account_sid"`      // Account identifier
	FriendlyName    string            `json:"friendly_name"`    // Human-readable conversation name
	UniqueName      string            `json:"unique_name"`      // Unique identifier (optional)
	State           ConversationState `json:"state"`            // Conversation state
	DateCreated     time.Time         `json:"date_created"`     // When conversation was created
	DateUpdated     time.Time         `json:"date_updated"`     // When conversation was last updated
	Attributes      string            `json:"attributes"`       // JSON string with additional attributes
	Metadata        map[string]any    `json:"metadata"`         // Additional metadata
	Bindings        []Binding         `json:"bindings"`         // Messaging bindings
}

// Message represents a message in a conversation.
type Message struct {
	MessageSID      string         `json:"message_sid"`      // Unique message identifier
	ConversationSID string         `json:"conversation_sid"` // Parent conversation identifier
	AccountSID      string         `json:"account_sid"`      // Account identifier
	Channel         Channel        `json:"channel"`          // Message channel
	From            string         `json:"from"`             // Sender identifier
	To              string         `json:"to"`               // Recipient identifier
	Body            string         `json:"body"`             // Message text content
	MediaURLs       []string       `json:"media_urls"`       // URLs to media attachments
	Index           int            `json:"index"`            // Message index within conversation (0-based)
	Author          string         `json:"author"`           // Author identifier (participant SID or identity)
	DateCreated     time.Time      `json:"date_created"`     // When message was created
	DateUpdated     time.Time      `json:"date_updated"`     // When message was last updated
	DeliveryStatus  DeliveryStatus `json:"delivery_status"`  // Delivery status
	Attributes      string         `json:"attributes"`       // JSON string with additional attributes
	Metadata        map[string]any `json:"metadata"`         // Additional metadata
}

// Participant represents a participant in a conversation.
type Participant struct {
	ParticipantSID   string    `json:"participant_sid"`   // Unique participant identifier
	ConversationSID  string    `json:"conversation_sid"`  // Parent conversation identifier
	AccountSID       string    `json:"account_sid"`       // Account identifier
	Identity         string    `json:"identity"`          // Participant identity (phone number, email, etc.)
	Attributes       string    `json:"attributes"`        // JSON string with additional attributes
	DateCreated      time.Time `json:"date_created"`      // When participant was added
	DateUpdated      time.Time `json:"date_updated"`      // When participant was last updated
	RoleSID          string    `json:"role_sid"`          // Role identifier (if using roles)
	MessagingBinding Binding   `json:"messaging_binding"` // Messaging binding
}

// WebhookEvent represents an event received from the provider via webhook.
type WebhookEvent struct {
	EventID     string         `json:"event_id"`     // Unique event identifier (generated)
	EventType   string         `json:"event_type"`   // Event type
	EventData   map[string]any `json:"event_data"`   // Event payload (varies by event type)
	AccountSID  string         `json:"account_sid"`  // Account identifier
	Timestamp   time.Time      `json:"timestamp"`    // When event was received
	Signature   string         `json:"signature"`    // Webhook signature (for validation)
	Source      string         `json:"source"`       // Event source (conversations-api, etc.)
	ResourceSID string         `json:"resource_sid"` // Related resource identifier
	Metadata    map[string]any `json:"metadata"`     // Additional metadata
}

// ConversationConfig represents configuration for creating a conversation.
type ConversationConfig struct {
	FriendlyName string            `json:"friendly_name"` // Human-readable conversation name
	UniqueName   string            `json:"unique_name"`   // Unique identifier (optional)
	Attributes   string            `json:"attributes"`    // JSON attributes
	State        ConversationState `json:"state"`         // Initial state
	Metadata     map[string]any    `json:"metadata"`      // Additional metadata
}

// HealthStatusDetail represents detailed health status information.
type HealthStatusDetail struct {
	Status    HealthStatus   `json:"status"`     // Health status
	LastCheck time.Time      `json:"last_check"` // Last health check timestamp
	Details   map[string]any `json:"details"`    // Health check details
	Errors    []string       `json:"errors"`     // Health check errors (if any)
}
