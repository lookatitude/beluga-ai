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
	DateCreated     time.Time      `json:"date_created"`
	DateUpdated     time.Time      `json:"date_updated"`
	Metadata        map[string]any `json:"metadata"`
	Body            string         `json:"body"`
	From            string         `json:"from"`
	To              string         `json:"to"`
	MessageSID      string         `json:"message_sid"`
	Author          string         `json:"author"`
	Channel         Channel        `json:"channel"`
	AccountSID      string         `json:"account_sid"`
	DeliveryStatus  DeliveryStatus `json:"delivery_status"`
	Attributes      string         `json:"attributes"`
	ConversationSID string         `json:"conversation_sid"`
	MediaURLs       []string       `json:"media_urls"`
	Index           int            `json:"index"`
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
	Timestamp   time.Time      `json:"timestamp"`
	EventData   map[string]any `json:"event_data"`
	Metadata    map[string]any `json:"metadata"`
	EventID     string         `json:"event_id"`
	EventType   string         `json:"event_type"`
	AccountSID  string         `json:"account_sid"`
	Signature   string         `json:"signature"`
	Source      string         `json:"source"`
	ResourceSID string         `json:"resource_sid"`
}

// ConversationConfig represents configuration for creating a conversation.
type ConversationConfig struct {
	Metadata     map[string]any    `json:"metadata"`
	FriendlyName string            `json:"friendly_name"`
	UniqueName   string            `json:"unique_name"`
	Attributes   string            `json:"attributes"`
	State        ConversationState `json:"state"`
}

// HealthStatusDetail represents detailed health status information.
type HealthStatusDetail struct {
	Status    HealthStatus   `json:"status"`     // Health status
	LastCheck time.Time      `json:"last_check"` // Last health check timestamp
	Details   map[string]any `json:"details"`    // Health check details
	Errors    []string       `json:"errors"`     // Health check errors (if any)
}
