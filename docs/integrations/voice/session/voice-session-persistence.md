# Voice Session Persistence

Welcome, colleague! In this guide we'll integrate **session persistence** with Beluga AI's `pkg/voice/session`: storing session and form state (e.g. conversation context, collected fields) so you can restore after reconnect or deploy restarts.

## What you will build

You will add a persistence layer that saves session state keyed by `GetSessionID()`, and restores it when a session reconnects or restarts. This allows multi-turn flows (e.g. voice forms, long conversations) to survive drops and restarts.

## Learning Objectives

- ✅ Use `VoiceSession.GetSessionID()` as a persistence key
- ✅ Store and restore session-related state (context, form progress)
- ✅ Reattach restored state when creating or resuming a session
- ✅ Handle missing or stale state gracefully

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- A store (e.g. Redis, DB) for state

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

## Step 2: Define Stored State

Keep session-scoped state separate from the voice session itself. Example:
```go
package main

import (
	"context"
	"encoding/json"
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

## Step 3: Save State on Each Turn

After processing a user turn (e.g. in your form orchestrator or agent callback), update and persist state:
```go
	func saveState(ctx context.Context, sess session.VoiceSession, state *SessionState) error {
		state.SessionID = sess.GetSessionID()
		state.UpdatedAt = time.Now()
		return store.Set(ctx, state)
	}
```

Call `saveState` whenever you advance the form or update conversation context.

## Step 4: Restore State on Reconnect

When a client reconnects (e.g. same `sessionID` from your front end or telephony layer), load state and resume:
```go
	func restoreOrCreate(ctx context.Context, sessionID string) (*SessionState, error) {
		st, err := store.Get(ctx, sessionID)
		if err != nil || st == nil {
			return &SessionState{SessionID: sessionID, FormAnswers: make(map[string]string)}, nil
		}
		return st, nil
	}
```

Use restored state to decide the next question, prompt, or agent context.

## Step 5: Wire Into Session Lifecycle

- **On session start**: Call `restoreOrCreate` with your session ID (e.g. from transport or backend-assigned ID). Pass restored state into your form orchestrator or agent.
- **On each turn**: Update in-memory state, then `saveState`.
- **On session stop**: Optional final `saveState`; optionally `Delete` after some TTL.

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Store backend | Redis, DB, etc. | Application-defined |
| TTL | How long to keep state after session end | Application-defined |
| Key prefix | e.g. `voice:session:` | Optional |

## Common Issues

### "State not found on reconnect"

**Problem**: Client sends a new session ID or store lost data.

**Solution**: Treat missing state as a new session; start form or conversation from scratch. Log and metric for analytics.

### "Stale state after long idle"

**Problem**: User returns after hours; form or context outdated.

**Solution**: Use `UpdatedAt` and a max age; optionally clear or warn and restart.

### "Session ID mismatch"

**Problem**: Transport or load balancer assigns a different ID on reconnect.

**Solution**: Use a stable identifier (e.g. phone number, user id + call id) as the persistence key and map it to `GetSessionID()` if needed.

## Production Considerations

- **Error handling**: Retry `Get`/`Set` with backoff; avoid failing session start on transient store errors.
- **Monitoring**: OTEL metrics for restore hits/misses, save latency, store errors.
- **Security**: Encrypt PII in stored state; enforce access control by tenant/user.

## Next Steps

- **[Multi-Provider Session Routing](./multi-provider-session-routing.md)** — Route sessions by provider or backend.
- **[Voice Sessions](../../../../use-cases/voice-sessions.md)** — Session architecture.
- **[Multi-Turn Voice Forms](../../../../use-cases/voice-session-multi-turn-forms.md)** — Form state and resumption.
