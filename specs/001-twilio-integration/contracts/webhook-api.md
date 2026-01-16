# Webhook API Contract: Twilio Integration

**Date**: 2025-01-07  
**Feature**: Twilio API Integration (001-twilio-integration)  
**Version**: 1.0.0

## Overview

This document defines the webhook handling contract for Twilio Voice API and Conversations API integration. Webhooks enable event-driven workflows and real-time event processing.

## Webhook Endpoint Structure

### Voice API Webhooks

**Base Path**: `/webhooks/twilio/voice`

**Endpoints**:
- `POST /webhooks/twilio/voice/status` - Call status callbacks
- `POST /webhooks/twilio/voice/stream` - Media stream events
- `POST /webhooks/twilio/voice/transcription` - Transcription events

### Conversations API Webhooks

**Base Path**: `/webhooks/twilio/conversations`

**Endpoints**:
- `POST /webhooks/twilio/conversations/events` - Conversation and message events

## Webhook Signature Validation

All webhooks MUST validate Twilio signatures to ensure authenticity.

**Validation Process**:
1. Extract signature from `X-Twilio-Signature` header
2. Construct signature string from request URL and POST parameters
3. Compute HMAC-SHA1 hash using auth token
4. Compare computed signature with provided signature
5. Reject request if signatures don't match

**Implementation**:
```go
func ValidateSignature(url string, params map[string]string, authToken string, signature string) bool {
    // Construct signature string
    sigString := url
    for key, value := range params {
        sigString += key + value
    }
    
    // Compute HMAC-SHA1
    mac := hmac.New(sha1.New, []byte(authToken))
    mac.Write([]byte(sigString))
    computedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
    
    // Compare signatures
    return hmac.Equal([]byte(computedSig), []byte(signature))
}
```

## Voice API Webhook Events

### Call Status Callback

**Endpoint**: `POST /webhooks/twilio/voice/status`

**Event Type**: `call.status`

**Payload**:
```
CallSid=CA1234567890abcdef
AccountSid=AC1234567890abcdef
From=%2B15551234567
To=%2B15559876543
CallStatus=answered
Direction=inbound
CallDuration=120
ApiVersion=2010-04-01
```

**Event Data Structure**:
```go
type CallStatusEvent struct {
    CallSID     string
    AccountSID  string
    From        string
    To          string
    Status      string  // initiated, ringing, in-progress, completed, failed, busy, no-answer, canceled
    Direction   string  // inbound, outbound-api, outbound-dial
    Duration    int     // seconds (if completed)
    APIVersion  string
    Timestamp   time.Time
}
```

**Status Values**:
- `initiated`: Call created
- `ringing`: Call is ringing
- `in-progress`: Call answered and active
- `completed`: Call completed normally
- `failed`: Call failed
- `busy`: Called party busy
- `no-answer`: No answer
- `canceled`: Call canceled

### Media Stream Event

**Endpoint**: `POST /webhooks/twilio/voice/stream`

**Event Type**: `stream.event`

**Payload**:
```
Event=start
SequenceNumber=1
StreamSid=MZ1234567890abcdef
AccountSid=AC1234567890abcdef
CallSid=CA1234567890abcdef
```

**Event Data Structure**:
```go
type StreamEvent struct {
    Event          string  // start, stop, media
    SequenceNumber int
    StreamSID      string
    AccountSID     string
    CallSID        string
    Timestamp      time.Time
}
```

### Transcription Event

**Endpoint**: `POST /webhooks/twilio/voice/transcription`

**Event Type**: `transcription.completed`

**Payload**:
```
TranscriptionSid=TR1234567890abcdef
AccountSid=AC1234567890abcdef
CallSid=CA1234567890abcdef
TranscriptionText=Hello%20world
TranscriptionStatus=completed
TranscriptionUrl=https://api.twilio.com/2010-04-01/Accounts/.../Transcriptions/TR...
```

**Event Data Structure**:
```go
type TranscriptionEvent struct {
    TranscriptionSID string
    AccountSID        string
    CallSID           string
    Text              string
    Status            string  // in-progress, completed, failed
    URL               string
    Language          string
    Confidence        float64
    Timestamp         time.Time
}
```

## Conversations API Webhook Events

### Conversation Events

**Endpoint**: `POST /webhooks/twilio/conversations/events`

**Event Types**:
- `conversation.created`
- `conversation.updated`
- `conversation.state.updated`

**Payload Example**:
```
EventType=conversation.created
AccountSid=AC1234567890abcdef
ConversationSid=CH1234567890abcdef
FriendlyName=Customer%20Support
State=active
```

**Event Data Structure**:
```go
type ConversationEvent struct {
    EventType      string
    AccountSID     string
    ConversationSID string
    FriendlyName    string
    State           string  // active, closed, inactive
    Timestamp       time.Time
}
```

### Message Events

**Event Types**:
- `message.added`
- `message.updated`
- `message.delivery.updated`

**Payload Example**:
```
EventType=message.added
AccountSid=AC1234567890abcdef
ConversationSid=CH1234567890abcdef
MessageSid=IM1234567890abcdef
Index=0
Author=MB1234567890abcdef
Body=Hello%20world
Attributes={}
```

**Event Data Structure**:
```go
type MessageEvent struct {
    EventType      string
    AccountSID     string
    ConversationSID string
    MessageSID     string
    Index          int
    Author         string
    Body           string
    MediaURLs      []string
    Attributes     map[string]any
    Timestamp      time.Time
}
```

### Participant Events

**Event Types**:
- `participant.added`
- `participant.removed`
- `participant.updated`

**Payload Example**:
```
EventType=participant.added
AccountSid=AC1234567890abcdef
ConversationSid=CH1234567890abcdef
ParticipantSid=MB1234567890abcdef
Identity=%2B15551234567
```

**Event Data Structure**:
```go
type ParticipantEvent struct {
    EventType      string
    AccountSID     string
    ConversationSID string
    ParticipantSID string
    Identity       string
    Attributes     map[string]any
    Timestamp      time.Time
}
```

### Typing Events

**Event Types**:
- `typing.started`
- `typing.ended`

**Payload Example**:
```
EventType=typing.started
AccountSid=AC1234567890abcdef
ConversationSid=CH1234567890abcdef
ParticipantSid=MB1234567890abcdef
```

## Webhook Handler Interface

### Handler Function

```go
type WebhookHandler func(ctx context.Context, event *WebhookEvent) error

type WebhookEvent struct {
    Type      string
    Source    string  // voice-api, conversations-api
    Data      map[string]any
    Timestamp time.Time
    Signature string
}
```

### Handler Registration

```go
// Register webhook handler
func (p *TwilioProvider) RegisterWebhookHandler(eventType string, handler WebhookHandler) error

// Handle webhook request
func (p *TwilioProvider) HandleWebhookRequest(ctx context.Context, r *http.Request) error
```

## Orchestration Integration

Webhook events trigger orchestration workflows via `pkg/orchestration`.

**Integration Pattern**:
```go
func (p *TwilioProvider) HandleWebhook(ctx context.Context, event *WebhookEvent) error {
    // Validate signature
    if err := p.validateSignature(event); err != nil {
        return err
    }
    
    // Parse event
    parsedEvent := p.parseEvent(event)
    
    // Trigger orchestration workflow
    if p.orchestrator != nil {
        workflow := p.getWorkflowForEvent(parsedEvent.Type)
        return p.orchestrator.Invoke(ctx, workflow, parsedEvent.Data)
    }
    
    // Handle event directly
    return p.handleEvent(ctx, parsedEvent)
}
```

**Workflow Mapping**:
- `call.answered` → Call flow workflow (Inbound → Agent → Stream)
- `message.added` → Message processing workflow
- `transcription.completed` → RAG workflow

## Error Handling

**Error Responses**:
- `400 Bad Request`: Invalid webhook data or signature
- `500 Internal Server Error`: Processing failure

**Retry Logic**:
- Twilio retries failed webhooks with exponential backoff
- Handler should be idempotent
- Log errors for monitoring

## Observability

### Metrics

- `twilio_webhook_received_total`: Webhooks received (counter)
- `twilio_webhook_processed_total`: Webhooks processed (counter)
- `twilio_webhook_errors_total`: Webhook processing errors (counter)
- `twilio_webhook_duration_seconds`: Webhook processing duration (histogram)

### Tracing

All webhook handlers create OTEL spans:
- `twilio.webhook.handle`
- Attributes: `webhook.type`, `webhook.source`, `webhook.event_type`

### Logging

Structured logging with OTEL context:
- Log all webhook events (INFO level)
- Log validation failures (WARN level)
- Log processing errors (ERROR level)
