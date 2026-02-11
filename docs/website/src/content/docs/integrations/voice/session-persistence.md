---
title: Voice Session Persistence
description: Store and restore voice session state for reconnection and restart recovery in Beluga AI.
---

Phone calls drop, WebSocket connections reset, and services restart during rolling deployments. Without persistence, users must start their conversation from scratch after any interruption. Voice session persistence saves session state -- conversation context, form progress, collected fields -- to an external store so sessions resume exactly where they left off. This is essential for multi-turn voice forms, long-running support calls, and any deployment that requires high availability. This guide covers implementing a persistence layer for Beluga AI voice sessions.

## Overview

By persisting session-scoped state keyed by session ID, multi-turn voice flows such as forms and long conversations can resume where they left off. The persistence layer operates independently of the voice session itself, storing application-level state in an external store.

## Prerequisites

- Go 1.23 or later
- A persistence store (Redis, PostgreSQL, or similar)
- Beluga AI voice session configured

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

### Define Stored State

Keep session-scoped state separate from the voice session object:

```go
package main

import (
    "context"
    "time"
)

type SessionState struct {
    SessionID   string            `json:"session_id"`
    FormAnswers map[string]string `json:"form_answers,omitempty"`
    LastTurn    string            `json:"last_turn,omitempty"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type Store interface {
    Get(ctx context.Context, sessionID string) (*SessionState, error)
    Set(ctx context.Context, s *SessionState) error
    Delete(ctx context.Context, sessionID string) error
}
```

## Usage

### Save State on Each Turn

After processing a user turn (in your form orchestrator or agent callback), update and persist state:

```go
func saveState(ctx context.Context, sess session.VoiceSession, store Store, state *SessionState) error {
    state.SessionID = sess.GetSessionID()
    state.UpdatedAt = time.Now()
    return store.Set(ctx, state)
}
```

Call `saveState` whenever you advance the form or update conversation context.

### Restore State on Reconnect

When a client reconnects with the same session ID, load state and resume:

```go
func restoreOrCreate(ctx context.Context, store Store, sessionID string) (*SessionState, error) {
    st, err := store.Get(ctx, sessionID)
    if err != nil || st == nil {
        return &SessionState{
            SessionID:   sessionID,
            FormAnswers: make(map[string]string),
        }, nil
    }
    return st, nil
}
```

Use the restored state to determine the next question, prompt, or agent context.

### Wire Into Session Lifecycle

- **On session start**: Call `restoreOrCreate` with your session ID from transport or backend. Pass restored state to your form orchestrator or agent.
- **On each turn**: Update in-memory state, then call `saveState`.
- **On session stop**: Perform a final `saveState`. Optionally delete state after a TTL.

## Configuration Reference

| Option        | Description                          | Default             |
|---------------|--------------------------------------|---------------------|
| Store backend | Redis, PostgreSQL, DynamoDB, etc.    | Application-defined |
| TTL           | How long to keep state after session | Application-defined |
| Key prefix    | Namespace for keys (e.g., `voice:session:`) | Optional     |

## Troubleshooting

### State not found on reconnect

The client may be sending a new session ID, or the store lost the data. Treat missing state as a new session and start from scratch. Log and emit metrics for monitoring.

### Stale state after long idle

Use the `UpdatedAt` field with a configurable maximum age. Clear or warn-and-restart when state exceeds the age threshold.

### Session ID mismatch

Transport or load balancer may assign different IDs on reconnect. Use a stable identifier (phone number, user ID + call ID) as the persistence key and map it to `GetSessionID()`.

## Advanced Topics

### Production Deployment

- **Error handling**: Retry `Get`/`Set` with backoff to avoid failing session start on transient store errors
- **Monitoring**: Use OpenTelemetry metrics for restore hits/misses, save latency, and store errors
- **Security**: Encrypt PII in stored state and enforce access control by tenant or user

## Related Resources

- [Multi-Provider Session Routing](/integrations/session-routing)
- [Voice Services Overview](/integrations/voice-services)
