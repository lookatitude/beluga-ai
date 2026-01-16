# Conversational Backend API Contract: Twilio Provider

**Date**: 2025-01-07  
**Feature**: Twilio API Integration (001-twilio-integration)  
**Version**: 1.0.0  
**Provider**: Twilio Conversations API (v1)

## Overview

This document defines the API contract for the Twilio Conversations API provider implementation of the `ConversationalBackend` interface. The provider integrates Twilio Conversations API with Beluga AI Framework's messaging system.

## Interface Definition

The `ConversationalBackend` interface is defined in `pkg/messaging/iface/backend.go`:

```go
package iface

import (
    "context"
)

// ConversationalBackend defines the interface for conversational backend instances.
type ConversationalBackend interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Conversation Management
    CreateConversation(ctx context.Context, config *ConversationConfig) (Conversation, error)
    GetConversation(ctx context.Context, conversationID string) (Conversation, error)
    ListConversations(ctx context.Context) ([]Conversation, error)
    CloseConversation(ctx context.Context, conversationID string) error
    
    // Message Operations
    SendMessage(ctx context.Context, conversationID string, message *Message) error
    ReceiveMessages(ctx context.Context, conversationID string) (<-chan *Message, error)
    
    // Participant Management
    AddParticipant(ctx context.Context, conversationID string, participant *Participant) error
    RemoveParticipant(ctx context.Context, conversationID string, participantID string) error
    
    // Webhook Handling
    HandleWebhook(ctx context.Context, event *WebhookEvent) error
    
    // Health & Status
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    GetConfig() *Config
}
```

## Core Methods

### Start

Initializes and starts the Twilio conversational backend.

**Signature**:
```go
func (p *TwilioProvider) Start(ctx context.Context) error
```

**Behavior**:
- Validates Twilio configuration (AccountSID, AuthToken)
- Initializes Twilio REST client
- Verifies API connectivity
- Sets up webhook endpoint registration (if configured)
- Returns error if backend cannot be started

**Errors**:
- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeAuthError`: Authentication failure
- `ErrCodeNetworkError`: Network connectivity issue

**OTEL**:
- Span: `twilio.messaging.backend.start`
- Metrics: `twilio_messaging_backend_start_total`, `twilio_messaging_backend_start_duration_seconds`

### Stop

Gracefully shuts down the backend, completing in-flight conversations.

**Signature**:
```go
func (p *TwilioProvider) Stop(ctx context.Context) error
```

**Behavior**:
- Completes all active conversations gracefully
- Closes all message channels
- Cleans up resources
- Returns error if shutdown fails

**Errors**:
- `ErrCodeShutdownError`: Shutdown failure

**OTEL**:
- Span: `twilio.messaging.backend.stop`
- Metrics: `twilio_messaging_backend_stop_total`, `twilio_messaging_backend_stop_duration_seconds`

### CreateConversation

Creates a new conversation with the given configuration.

**Signature**:
```go
func (p *TwilioProvider) CreateConversation(ctx context.Context, config *ConversationConfig) (Conversation, error)
```

**Behavior**:
- Creates Twilio Conversation resource via API
- Sets up conversation attributes
- Creates MessagingSession instance
- Associates agent instance (if provided)
- Returns error if conversation cannot be created

**Configuration**:
- `FriendlyName`: Human-readable conversation name
- `UniqueName`: Unique identifier (optional)
- `Attributes`: JSON attributes
- `State`: Initial state (active, closed, inactive)

**Errors**:
- `ErrCodeInvalidConfig`: Invalid conversation configuration
- `ErrCodeRateLimit`: Twilio rate limit exceeded
- `ErrCodeNetworkError`: Network error creating conversation
- `ErrCodeConversationFailed`: Conversation creation failed

**OTEL**:
- Span: `twilio.conversation.create`
- Metrics: `twilio_conversation_create_total`, `twilio_conversation_create_duration_seconds`
- Attributes: `conversation.friendly_name`, `conversation.state`

### GetConversation

Retrieves an existing conversation by ID.

**Signature**:
```go
func (p *TwilioProvider) GetConversation(ctx context.Context, conversationID string) (Conversation, error)
```

**Behavior**:
- Fetches conversation from Twilio API
- Returns conversation if found
- Returns error if conversation not found

**Errors**:
- `ErrCodeConversationNotFound`: Conversation not found
- `ErrCodeNetworkError`: Network error fetching conversation

**OTEL**:
- Span: `twilio.conversation.get`
- Metrics: `twilio_conversation_get_total`, `twilio_conversation_get_duration_seconds`

### ListConversations

Returns all conversations (with optional filtering).

**Signature**:
```go
func (p *TwilioProvider) ListConversations(ctx context.Context) ([]Conversation, error)
```

**Behavior**:
- Fetches conversations from Twilio API
- Returns list of conversations
- Supports pagination (if needed)

**OTEL**:
- Span: `twilio.conversations.list`
- Metrics: `twilio_conversations_list_total`, `twilio_conversations_list_duration_seconds`

### CloseConversation

Closes a conversation and cleans up resources.

**Signature**:
```go
func (p *TwilioProvider) CloseConversation(ctx context.Context, conversationID string) error
```

**Behavior**:
- Updates Twilio Conversation resource state to "closed"
- Closes message channels
- Removes session from active sessions
- Cleans up resources
- Returns error if conversation cannot be closed

**Errors**:
- `ErrCodeConversationNotFound`: Conversation not found
- `ErrCodeCloseFailed`: Failed to close conversation

**OTEL**:
- Span: `twilio.conversation.close`
- Metrics: `twilio_conversation_close_total`, `twilio_conversation_close_duration_seconds`

### SendMessage

Sends a message to a conversation.

**Signature**:
```go
func (p *TwilioProvider) SendMessage(ctx context.Context, conversationID string, message *Message) error
```

**Behavior**:
- Creates Twilio Message resource via API
- Sends message to conversation
- Supports text and media messages
- Returns error if message cannot be sent

**Message Types**:
- Text message: `Body` field
- Media message: `MediaURLs` field (images, videos, audio)

**Errors**:
- `ErrCodeConversationNotFound`: Conversation not found
- `ErrCodeInvalidMessage`: Invalid message content
- `ErrCodeRateLimit`: Twilio rate limit exceeded
- `ErrCodeNetworkError`: Network error sending message

**OTEL**:
- Span: `twilio.message.send`
- Metrics: `twilio_message_send_total`, `twilio_message_send_duration_seconds`
- Attributes: `message.channel`, `message.has_media`

### ReceiveMessages

Returns a channel for receiving messages from a conversation.

**Signature**:
```go
func (p *TwilioProvider) ReceiveMessages(ctx context.Context, conversationID string) (<-chan *Message, error)
```

**Behavior**:
- Creates message channel for conversation
- Listens for webhook events (onMessageAdded)
- Streams messages to channel
- Closes channel when conversation closes or context cancels

**Errors**:
- `ErrCodeConversationNotFound`: Conversation not found
- `ErrCodeChannelFailed`: Failed to create message channel

**OTEL**:
- Span: `twilio.messages.receive`
- Metrics: `twilio_messages_receive_total`, `twilio_messages_receive_duration_seconds`

### AddParticipant

Adds a participant to a conversation.

**Signature**:
```go
func (p *TwilioProvider) AddParticipant(ctx context.Context, conversationID string, participant *Participant) error
```

**Behavior**:
- Creates Twilio Participant resource via API
- Adds participant to conversation
- Sets up messaging bindings (SMS, WhatsApp, etc.)
- Returns error if participant cannot be added

**Participant Configuration**:
- `Identity`: Participant identity (phone number, email, etc.)
- `MessagingBinding`: Binding type and address
- `Attributes`: JSON attributes

**Errors**:
- `ErrCodeConversationNotFound`: Conversation not found
- `ErrCodeInvalidParticipant`: Invalid participant configuration
- `ErrCodeRateLimit`: Twilio rate limit exceeded

**OTEL**:
- Span: `twilio.participant.add`
- Metrics: `twilio_participant_add_total`, `twilio_participant_add_duration_seconds`

### RemoveParticipant

Removes a participant from a conversation.

**Signature**:
```go
func (p *TwilioProvider) RemoveParticipant(ctx context.Context, conversationID string, participantID string) error
```

**Behavior**:
- Deletes Twilio Participant resource via API
- Removes participant from conversation
- Returns error if participant cannot be removed

**Errors**:
- `ErrCodeConversationNotFound`: Conversation not found
- `ErrCodeParticipantNotFound`: Participant not found

**OTEL**:
- Span: `twilio.participant.remove`
- Metrics: `twilio_participant_remove_total`, `twilio_participant_remove_duration_seconds`

### HandleWebhook

Handles a webhook event from Twilio.

**Signature**:
```go
func (p *TwilioProvider) HandleWebhook(ctx context.Context, event *WebhookEvent) error
```

**Behavior**:
- Validates webhook signature
- Parses webhook event data
- Routes event to appropriate handler
- Triggers orchestration workflows (if configured)
- Returns error if webhook cannot be processed

**Event Types**:
- `conversation.created`: Conversation created
- `conversation.updated`: Conversation updated
- `message.added`: Message added to conversation
- `message.updated`: Message updated
- `message.delivery.updated`: Message delivery status updated
- `participant.added`: Participant added
- `participant.removed`: Participant removed
- `typing.started`: Typing indicator started

**Errors**:
- `ErrCodeInvalidSignature`: Invalid webhook signature
- `ErrCodeInvalidWebhook`: Invalid webhook data

**OTEL**:
- Span: `twilio.messaging.webhook.handle`
- Metrics: `twilio_messaging_webhook_total`, `twilio_messaging_webhook_duration_seconds`
- Attributes: `webhook.event_type`, `webhook.conversation_sid`

### HealthCheck

Performs a health check on the backend instance.

**Signature**:
```go
func (p *TwilioProvider) HealthCheck(ctx context.Context) (*HealthStatus, error)
```

**Behavior**:
- Verifies Twilio API connectivity
- Checks active conversation count
- Validates configuration
- Returns health status

**Health Status**:
- `healthy`: API accessible, configuration valid
- `degraded`: API accessible but some issues (rate limits, etc.)
- `unhealthy`: API inaccessible or configuration invalid

**OTEL**:
- Span: `twilio.messaging.backend.health_check`
- Metrics: `twilio_messaging_backend_health_check_total`, `twilio_messaging_backend_health_check_duration_seconds`

### GetConfig

Returns the backend configuration.

**Signature**:
```go
func (p *TwilioProvider) GetConfig() *Config
```

**Behavior**:
- Returns current configuration
- Thread-safe access

## Configuration

### TwilioConfig

Twilio-specific configuration for Conversations API.

```go
type TwilioConfig struct {
    *Config
    
    // Twilio credentials
    AccountSID string `mapstructure:"account_sid" yaml:"account_sid" env:"TWILIO_ACCOUNT_SID" validate:"required"`
    AuthToken  string `mapstructure:"auth_token" yaml:"auth_token" env:"TWILIO_AUTH_TOKEN" validate:"required"`
    
    // API configuration
    APIVersion string `mapstructure:"api_version" yaml:"api_version" env:"TWILIO_CONVERSATIONS_API_VERSION" default:"v1"`
    BaseURL    string `mapstructure:"base_url" yaml:"base_url" env:"TWILIO_BASE_URL" default:"https://conversations.twilio.com"`
    
    // Webhook configuration
    WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url" env:"TWILIO_WEBHOOK_URL"`
    
    // Retry configuration
    MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"TWILIO_MAX_RETRIES" default:"3"`
    RetryDelay time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" env:"TWILIO_RETRY_DELAY" default:"1s"`
}
```

## Error Codes

- `ErrCodeTwilioRateLimit`: Twilio rate limit exceeded
- `ErrCodeTwilioAuthError`: Twilio authentication error
- `ErrCodeTwilioNetworkError`: Twilio network error
- `ErrCodeTwilioTimeout`: Twilio request timeout
- `ErrCodeTwilioConversationFailed`: Conversation operation failed
- `ErrCodeTwilioMessageFailed`: Message operation failed
- `ErrCodeTwilioInvalidWebhook`: Invalid webhook data or signature

## Observability

### Metrics

- `twilio_messaging_backend_operations_total`: Total operations (counter)
- `twilio_messaging_backend_operation_duration_seconds`: Operation duration (histogram)
- `twilio_messaging_backend_errors_total`: Total errors (counter)
- `twilio_conversation_create_total`: Conversations created (counter)
- `twilio_message_send_total`: Messages sent (counter)
- `twilio_message_send_duration_seconds`: Message send duration (histogram)
- `twilio_conversation_active`: Active conversations (gauge)
- `twilio_participant_active`: Active participants (gauge)

### Tracing

All methods create OTEL spans with attributes:
- `twilio.account_sid`: Twilio account identifier
- `twilio.conversation_sid`: Conversation identifier (when applicable)
- `twilio.message_sid`: Message identifier (when applicable)
- `twilio.operation`: Operation name

### Logging

Structured logging with OTEL context:
- Trace ID and span ID included
- Log levels: DEBUG (detailed), INFO (operations), WARN (errors), ERROR (failures)
