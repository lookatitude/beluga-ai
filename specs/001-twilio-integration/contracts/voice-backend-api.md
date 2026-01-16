# Voice Backend API Contract: Twilio Provider

**Date**: 2025-01-07  
**Feature**: Twilio API Integration (001-twilio-integration)  
**Version**: 1.0.0  
**Provider**: Twilio Voice API (2010-04-01)

## Overview

This document defines the API contract for the Twilio Voice API provider implementation of the `VoiceBackend` interface. The provider integrates Twilio Programmable Voice API with Beluga AI Framework's voice backend system.

## Interface Implementation

The Twilio provider implements `pkg/voice/backend/iface.VoiceBackend`:

```go
package twilio

import (
    "context"
    vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// TwilioProvider implements VoiceBackend interface
type TwilioProvider struct {
    // Implementation details
}
```

## Core Methods

### Start

Initializes and starts the Twilio voice backend.

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
- Span: `twilio.backend.start`
- Metrics: `twilio_backend_start_total`, `twilio_backend_start_duration_seconds`

### Stop

Gracefully shuts down the backend, completing in-flight conversations.

**Signature**:
```go
func (p *TwilioProvider) Stop(ctx context.Context) error
```

**Behavior**:
- Completes all active calls gracefully
- Closes all WebSocket streams
- Cleans up resources
- Returns error if shutdown fails

**Errors**:
- `ErrCodeShutdownError`: Shutdown failure

**OTEL**:
- Span: `twilio.backend.stop`
- Metrics: `twilio_backend_stop_total`, `twilio_backend_stop_duration_seconds`

### CreateSession

Creates a new voice session (Twilio call) with the given configuration.

**Signature**:
```go
func (p *TwilioProvider) CreateSession(ctx context.Context, config *vbiface.SessionConfig) (vbiface.VoiceSession, error)
```

**Behavior**:
- Creates Twilio Call resource via API
- Establishes WebSocket media stream connection
- Creates VoiceSession instance
- Associates agent instance (if provided)
- Returns error if session cannot be created

**Configuration Mapping**:
- `To`: Twilio "To" phone number
- `From`: Twilio "From" phone number (from config)
- `URL`: Webhook URL for call status updates
- `StatusCallback`: Webhook URL for status callbacks
- `StatusCallbackEvent`: Events to receive (initiated, ringing, answered, completed)

**Errors**:
- `ErrCodeInvalidConfig`: Invalid session configuration
- `ErrCodeRateLimit`: Twilio rate limit exceeded
- `ErrCodeNetworkError`: Network error creating call
- `ErrCodeCallFailed`: Call creation failed

**OTEL**:
- Span: `twilio.call.create`
- Metrics: `twilio_call_create_total`, `twilio_call_create_duration_seconds`
- Attributes: `call.to`, `call.from`, `call.direction`

### GetSession

Retrieves an existing voice session by session ID.

**Signature**:
```go
func (p *TwilioProvider) GetSession(ctx context.Context, sessionID string) (vbiface.VoiceSession, error)
```

**Behavior**:
- Looks up session by ID in active sessions
- Returns session if found
- Returns error if session not found

**Errors**:
- `ErrCodeSessionNotFound`: Session not found

**OTEL**:
- Span: `twilio.session.get`
- Metrics: `twilio_session_get_total`, `twilio_session_get_duration_seconds`

### ListSessions

Returns all active voice sessions.

**Signature**:
```go
func (p *TwilioProvider) ListSessions(ctx context.Context) ([]vbiface.VoiceSession, error)
```

**Behavior**:
- Returns list of all active sessions
- Thread-safe access to session map

**OTEL**:
- Span: `twilio.sessions.list`
- Metrics: `twilio_sessions_list_total`, `twilio_sessions_list_duration_seconds`

### CloseSession

Closes a voice session and cleans up resources.

**Signature**:
```go
func (p *TwilioProvider) CloseSession(ctx context.Context, sessionID string) error
```

**Behavior**:
- Updates Twilio Call resource status (if needed)
- Closes WebSocket stream connection
- Removes session from active sessions
- Cleans up resources
- Returns error if session cannot be closed

**Errors**:
- `ErrCodeSessionNotFound`: Session not found
- `ErrCodeCloseFailed`: Failed to close session

**OTEL**:
- Span: `twilio.session.close`
- Metrics: `twilio_session_close_total`, `twilio_session_close_duration_seconds`

### HealthCheck

Performs a health check on the backend instance.

**Signature**:
```go
func (p *TwilioProvider) HealthCheck(ctx context.Context) (*vbiface.HealthStatus, error)
```

**Behavior**:
- Verifies Twilio API connectivity
- Checks active session count
- Validates configuration
- Returns health status

**Health Status**:
- `healthy`: API accessible, configuration valid
- `degraded`: API accessible but some issues (rate limits, etc.)
- `unhealthy`: API inaccessible or configuration invalid

**OTEL**:
- Span: `twilio.backend.health_check`
- Metrics: `twilio_backend_health_check_total`, `twilio_backend_health_check_duration_seconds`

### GetConnectionState

Returns the current connection state.

**Signature**:
```go
func (p *TwilioProvider) GetConnectionState() vbiface.ConnectionState
```

**Behavior**:
- Returns current connection state (connected, disconnected, error)
- Thread-safe access

**OTEL**:
- Metrics: `twilio_backend_connection_state` (gauge)

### GetActiveSessionCount

Returns the number of active sessions.

**Signature**:
```go
func (p *TwilioProvider) GetActiveSessionCount() int
```

**Behavior**:
- Returns count of active sessions
- Thread-safe access

**OTEL**:
- Metrics: `twilio_backend_active_sessions` (gauge)

### GetConfig

Returns the backend configuration.

**Signature**:
```go
func (p *TwilioProvider) GetConfig() *vbiface.Config
```

**Behavior**:
- Returns current configuration
- Thread-safe access

### UpdateConfig

Updates the backend configuration with validation.

**Signature**:
```go
func (p *TwilioProvider) UpdateConfig(ctx context.Context, config *vbiface.Config) error
```

**Behavior**:
- Validates new configuration
- Updates configuration if valid
- Returns error if validation fails

**Errors**:
- `ErrCodeInvalidConfig`: Invalid configuration

**OTEL**:
- Span: `twilio.backend.update_config`
- Metrics: `twilio_backend_update_config_total`, `twilio_backend_update_config_duration_seconds`

## Twilio-Specific Extensions

### HandleInboundCall

Handles an inbound call webhook event.

**Signature**:
```go
func (p *TwilioProvider) HandleInboundCall(ctx context.Context, webhookData map[string]string) (vbiface.VoiceSession, error)
```

**Behavior**:
- Validates webhook signature
- Parses webhook data
- Creates voice session for inbound call
- Establishes media stream
- Returns session

**Errors**:
- `ErrCodeInvalidSignature`: Invalid webhook signature
- `ErrCodeInvalidWebhook`: Invalid webhook data

### StreamAudio

Manages WebSocket audio streaming for a call.

**Signature**:
```go
func (p *TwilioProvider) StreamAudio(ctx context.Context, sessionID string) (AudioStream, error)
```

**Behavior**:
- Establishes WebSocket connection to Twilio Media Stream
- Handles bidirectional audio streaming
- Manages mu-law codec encoding/decoding
- Returns audio stream interface

**Errors**:
- `ErrCodeSessionNotFound`: Session not found
- `ErrCodeStreamFailed`: Stream connection failed

## Configuration

### TwilioConfig

Twilio-specific configuration extending base VoiceBackend config.

```go
type TwilioConfig struct {
    *vbiface.Config
    
    // Twilio credentials
    AccountSID string `mapstructure:"account_sid" yaml:"account_sid" env:"TWILIO_ACCOUNT_SID" validate:"required"`
    AuthToken  string `mapstructure:"auth_token" yaml:"auth_token" env:"TWILIO_AUTH_TOKEN" validate:"required"`
    
    // Phone numbers
    PhoneNumber string `mapstructure:"phone_number" yaml:"phone_number" env:"TWILIO_PHONE_NUMBER" validate:"required"`
    
    // Webhook configuration
    WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url" env:"TWILIO_WEBHOOK_URL"`
    
    // API configuration
    APIVersion string `mapstructure:"api_version" yaml:"api_version" env:"TWILIO_API_VERSION" default:"2010-04-01"`
    BaseURL    string `mapstructure:"base_url" yaml:"base_url" env:"TWILIO_BASE_URL" default:"https://api.twilio.com"`
    
    // Streaming configuration
    StreamTimeout    time.Duration `mapstructure:"stream_timeout" yaml:"stream_timeout" env:"TWILIO_STREAM_TIMEOUT" default:"30s"`
    StreamBufferSize int           `mapstructure:"stream_buffer_size" yaml:"stream_buffer_size" env:"TWILIO_STREAM_BUFFER_SIZE" default:"4096"`
    
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
- `ErrCodeTwilioCallFailed`: Call creation/update failed
- `ErrCodeTwilioStreamFailed`: Media stream connection failed
- `ErrCodeTwilioInvalidWebhook`: Invalid webhook data or signature

## Observability

### Metrics

- `twilio_backend_operations_total`: Total operations (counter)
- `twilio_backend_operation_duration_seconds`: Operation duration (histogram)
- `twilio_backend_errors_total`: Total errors (counter)
- `twilio_call_create_total`: Calls created (counter)
- `twilio_call_create_duration_seconds`: Call creation duration (histogram)
- `twilio_session_active`: Active sessions (gauge)
- `twilio_stream_active`: Active streams (gauge)

### Tracing

All methods create OTEL spans with attributes:
- `twilio.account_sid`: Twilio account identifier
- `twilio.call_sid`: Call identifier (when applicable)
- `twilio.session_id`: Session identifier (when applicable)
- `twilio.operation`: Operation name

### Logging

Structured logging with OTEL context:
- Trace ID and span ID included
- Log levels: DEBUG (detailed), INFO (operations), WARN (errors), ERROR (failures)
