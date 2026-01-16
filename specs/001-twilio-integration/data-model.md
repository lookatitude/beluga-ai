# Data Model: Twilio API Integration

**Date**: 2025-01-07  
**Feature**: Twilio API Integration (001-twilio-integration)  
**Version**: 1.0.0

## Overview

This document defines the data models for Twilio API integration, including entities for voice calls, messaging conversations, transcriptions, webhook events, and sessions. All entities follow Beluga AI Framework patterns and integrate with existing schema types.

## Core Entities

### Call

Represents a phone call session managed by Twilio Voice API.

**Attributes**:
- `CallSID` (string): Unique Twilio call identifier (e.g., "CA1234567890abcdef")
- `AccountSID` (string): Twilio account identifier
- `From` (string): Phone number that initiated the call (E.164 format, e.g., "+15551234567")
- `To` (string): Phone number receiving the call (E.164 format)
- `Status` (enum): Call status (initiated, ringing, in-progress, completed, failed, busy, no-answer, canceled)
- `Direction` (enum): Call direction (inbound, outbound-api, outbound-dial)
- `StartTime` (time.Time): When the call was initiated
- `EndTime` (time.Time): When the call ended (null if in progress)
- `Duration` (int): Call duration in seconds (0 if in progress)
- `Price` (decimal): Call cost (if available)
- `PriceUnit` (string): Currency unit (e.g., "USD")
- `CallerName` (string): Caller ID name (if available)
- `MediaStreamURL` (string): WebSocket URL for media streaming (if active)
- `TranscriptionSID` (string): Associated transcription identifier (if available)
- `Metadata` (map[string]any): Additional call metadata (custom parameters)

**State Transitions**:
```
initiated → ringing → in-progress → completed
                ↓
            failed/busy/no-answer/canceled
```

**Validation Rules**:
- CallSID must be non-empty and match Twilio SID format
- From and To must be valid E.164 phone numbers
- Status must be a valid enum value
- StartTime must be set when call is created
- EndTime must be null or after StartTime
- Duration must be >= 0

**Relationships**:
- One Call has zero or one Transcription
- One Call has one VoiceSession
- One Call has zero or more WebhookEvents

### Message

Represents a message in a Twilio Conversations API conversation (SMS, WhatsApp, etc.).

**Attributes**:
- `MessageSID` (string): Unique Twilio message identifier (e.g., "IM1234567890abcdef")
- `ConversationSID` (string): Parent conversation identifier
- `AccountSID` (string): Twilio account identifier
- `Channel` (enum): Message channel (sms, whatsapp, chat, email)
- `From` (string): Sender identifier (phone number, WhatsApp number, etc.)
- `To` (string): Recipient identifier
- `Body` (string): Message text content
- `MediaURLs` ([]string): URLs to media attachments (images, videos, audio)
- `Index` (int): Message index within conversation (0-based)
- `Author` (string): Author identifier (participant SID or identity)
- `DateCreated` (time.Time): When message was created
- `DateUpdated` (time.Time): When message was last updated
- `DeliveryStatus` (enum): Delivery status (sent, delivered, read, failed, undelivered)
- `Attributes` (string): JSON string with additional message attributes
- `Metadata` (map[string]any): Additional message metadata

**State Transitions**:
```
created → sent → delivered → read
              ↓
          failed/undelivered
```

**Validation Rules**:
- MessageSID must be non-empty and match Twilio SID format
- ConversationSID must be non-empty
- Channel must be a valid enum value
- From and To must be non-empty
- Body or MediaURLs must be present (at least one)
- Index must be >= 0
- DateCreated must be set when message is created
- DateUpdated must be >= DateCreated

**Relationships**:
- One Message belongs to one Conversation
- One Message has one Author (Participant)
- One Message has zero or more WebhookEvents (delivery updates)

### Conversation

Represents a multi-channel conversation thread in Twilio Conversations API.

**Attributes**:
- `ConversationSID` (string): Unique Twilio conversation identifier (e.g., "CH1234567890abcdef")
- `AccountSID` (string): Twilio account identifier
- `FriendlyName` (string): Human-readable conversation name
- `UniqueName` (string): Unique identifier for the conversation (optional)
- `State` (enum): Conversation state (active, closed, inactive)
- `DateCreated` (time.Time): When conversation was created
- `DateUpdated` (time.Time): When conversation was last updated
- `Timers` (map[string]time.Time): Timer-based state transitions (inactive, closed)
- `Attributes` (string): JSON string with additional conversation attributes
- `Metadata` (map[string]any): Additional conversation metadata
- `Bindings` ([]Binding): Messaging bindings (SMS, WhatsApp, etc.)

**State Transitions**:
```
created → active → closed
              ↓
          inactive (timer-based)
```

**Validation Rules**:
- ConversationSID must be non-empty and match Twilio SID format
- State must be a valid enum value
- DateCreated must be set when conversation is created
- DateUpdated must be >= DateCreated
- UniqueName must be unique if provided

**Relationships**:
- One Conversation has zero or more Messages
- One Conversation has one or more Participants
- One Conversation has one MessagingSession
- One Conversation has zero or more WebhookEvents

### Participant

Represents a participant in a Twilio Conversations API conversation.

**Attributes**:
- `ParticipantSID` (string): Unique Twilio participant identifier (e.g., "MB1234567890abcdef")
- `ConversationSID` (string): Parent conversation identifier
- `AccountSID` (string): Twilio account identifier
- `Identity` (string): Participant identity (phone number, email, etc.)
- `Attributes` (string): JSON string with additional participant attributes
- `DateCreated` (time.Time): When participant was added
- `DateUpdated` (time.Time): When participant was last updated
- `RoleSID` (string): Role identifier (if using roles)
- `MessagingBinding` (Binding): Messaging binding (SMS, WhatsApp, etc.)

**Validation Rules**:
- ParticipantSID must be non-empty and match Twilio SID format
- ConversationSID must be non-empty
- Identity must be non-empty
- DateCreated must be set when participant is created
- DateUpdated must be >= DateCreated

**Relationships**:
- One Participant belongs to one Conversation
- One Participant can be author of zero or more Messages

### Transcription

Represents a text transcription of a Twilio Voice API call.

**Attributes**:
- `TranscriptionSID` (string): Unique Twilio transcription identifier (e.g., "TR1234567890abcdef")
- `CallSID` (string): Associated call identifier
- `AccountSID` (string): Twilio account identifier
- `Status` (enum): Transcription status (in-progress, completed, failed)
- `Text` (string): Transcribed text content
- `Language` (string): Detected language code (e.g., "en-US")
- `Confidence` (float): Confidence score (0.0 to 1.0)
- `DateCreated` (time.Time): When transcription was created
- `DateUpdated` (time.Time): When transcription was last updated
- `Duration` (int): Audio duration in seconds
- `Price` (decimal): Transcription cost (if available)
- `PriceUnit` (string): Currency unit (e.g., "USD")
- `Metadata` (map[string]any): Additional transcription metadata

**State Transitions**:
```
in-progress → completed
           ↓
         failed
```

**Validation Rules**:
- TranscriptionSID must be non-empty and match Twilio SID format
- CallSID must be non-empty
- Status must be a valid enum value
- Text must be non-empty when status is "completed"
- Confidence must be between 0.0 and 1.0
- DateCreated must be set when transcription is created
- DateUpdated must be >= DateCreated
- Duration must be >= 0

**Relationships**:
- One Transcription belongs to one Call
- One Transcription can be stored in VectorStore for RAG

### Webhook Event

Represents an event received from Twilio via webhook.

**Attributes**:
- `EventID` (string): Unique event identifier (generated)
- `EventType` (enum): Event type (call.status, call.stream, message.added, message.delivery, conversation.created, etc.)
- `EventData` (map[string]any): Event payload (varies by event type)
- `AccountSID` (string): Twilio account identifier
- `Timestamp` (time.Time): When event was received
- `Signature` (string): Twilio webhook signature (for validation)
- `Source` (string): Event source (voice-api, conversations-api)
- `ResourceSID` (string): Related resource identifier (CallSID, MessageSID, etc.)
- `Metadata` (map[string]any): Additional event metadata

**Event Types**:
- **Voice API**: `call.initiated`, `call.ringing`, `call.answered`, `call.completed`, `call.failed`, `stream.started`, `stream.ended`, `transcription.completed`
- **Conversations API**: `conversation.created`, `conversation.updated`, `message.added`, `message.updated`, `message.delivery.updated`, `participant.added`, `participant.removed`, `typing.started`

**Validation Rules**:
- EventType must be a valid enum value
- EventData must be non-empty
- AccountSID must be non-empty
- Timestamp must be set when event is received
- Signature must be present for validation
- ResourceSID must match event type (CallSID for call events, MessageSID for message events, etc.)

**Relationships**:
- One WebhookEvent relates to one resource (Call, Message, Conversation, etc.)
- One WebhookEvent can trigger one or more Orchestration workflows

### Voice Session

Represents an active voice interaction session managed by Beluga AI.

**Attributes**:
- `SessionID` (string): Unique session identifier (generated)
- `CallSID` (string): Associated Twilio call identifier
- `AccountSID` (string): Twilio account identifier
- `AgentInstance` (Agent): Associated AI agent instance
- `ConversationState` (map[string]any): Current conversation state
- `StreamingStatus` (enum): Streaming status (idle, connecting, connected, streaming, disconnected, error)
- `StreamURL` (string): WebSocket stream URL (if active)
- `MemoryInstance` (Memory): Associated memory instance for conversation history
- `StartTime` (time.Time): When session was started
- `LastActivity` (time.Time): Last activity timestamp
- `Metadata` (map[string]any): Additional session metadata

**State Transitions**:
```
created → connecting → connected → streaming → disconnected
                                    ↓
                                error
```

**Validation Rules**:
- SessionID must be non-empty and unique
- CallSID must be non-empty
- AgentInstance must be set when session is active
- StreamingStatus must be a valid enum value
- StartTime must be set when session is created
- LastActivity must be >= StartTime

**Relationships**:
- One VoiceSession belongs to one Call
- One VoiceSession has one AgentInstance
- One VoiceSession has one MemoryInstance
- One VoiceSession manages one WebSocket Stream

### Messaging Session

Represents an active messaging interaction session managed by Beluga AI.

**Attributes**:
- `SessionID` (string): Unique session identifier (generated)
- `ConversationSID` (string): Associated Twilio conversation identifier
- `AccountSID` (string): Twilio account identifier
- `AgentInstance` (Agent): Associated AI agent instance
- `ConversationHistory` ([]Message): Message history for the conversation
- `MemoryState` (map[string]any): Current memory state
- `MemoryInstance` (Memory): Associated memory instance for conversation persistence
- `Channels` ([]string): Active channels (SMS, WhatsApp, etc.)
- `Participants` ([]Participant): Conversation participants
- `StartTime` (time.Time): When session was started
- `LastActivity` (time.Time): Last activity timestamp
- `Metadata` (map[string]any): Additional session metadata

**Validation Rules**:
- SessionID must be non-empty and unique
- ConversationSID must be non-empty
- AgentInstance must be set when session is active
- StartTime must be set when session is created
- LastActivity must be >= StartTime
- Channels must contain at least one channel

**Relationships**:
- One MessagingSession belongs to one Conversation
- One MessagingSession has one AgentInstance
- One MessagingSession has one MemoryInstance
- One MessagingSession has zero or more Messages
- One MessagingSession has one or more Participants

## Supporting Types

### Binding

Represents a messaging binding for a participant (SMS, WhatsApp, etc.).

**Attributes**:
- `Type` (enum): Binding type (sms, whatsapp, chat, email)
- `Address` (string): Binding address (phone number, email, etc.)
- `ProxyAddress` (string): Proxy address (if applicable)

### HealthStatus

Represents the health status of a Twilio provider instance.

**Attributes**:
- `Status` (enum): Health status (healthy, degraded, unhealthy)
- `LastCheck` (time.Time): Last health check timestamp
- `Details` (map[string]any): Health check details
- `Errors` ([]string): Health check errors (if any)

## Integration with Beluga Schema

All entities integrate with existing Beluga AI Framework schema types:

- **Messages**: Use `pkg/schema.Message` for agent message handling
- **Documents**: Use `pkg/schema.Document` for transcription storage
- **History**: Use `pkg/schema.History` for conversation history
- **Memory**: Use `pkg/memory/iface.Memory` for memory integration

## Data Persistence

- **Active Sessions**: In-memory storage for active voice/messaging sessions
- **Conversation History**: Persistent storage via `pkg/memory/VectorStoreMemory`
- **Transcriptions**: Persistent storage in `pkg/vectorstores` for RAG
- **Webhook Events**: Ephemeral (processed immediately, optionally logged)

## Validation

All entities follow Beluga AI Framework validation patterns:
- Use `go-playground/validator/v10` for struct validation
- Validate at creation time
- Return custom error types with error codes
- Support context cancellation

## Error Handling

All entity operations return Beluga error types:
- `ErrCodeInvalidCallSID`: Invalid call identifier
- `ErrCodeInvalidMessageSID`: Invalid message identifier
- `ErrCodeInvalidConversationSID`: Invalid conversation identifier
- `ErrCodeInvalidState`: Invalid state transition
- `ErrCodeMissingRequiredField`: Required field missing
