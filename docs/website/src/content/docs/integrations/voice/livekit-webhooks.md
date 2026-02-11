---
title: LiveKit Webhooks
description: Handle LiveKit room and participant events via webhooks for session lifecycle management, analytics, and cleanup.
---

When using LiveKit as your voice transport, the audio pipeline runs inside LiveKit rooms -- but your backend needs visibility into what is happening. LiveKit webhooks deliver real-time server-side notifications about room and participant lifecycle events, enabling session cleanup, usage tracking, billing triggers, and transcript archival without polling. This guide covers setting up an HTTP endpoint that receives LiveKit webhooks, verifies signatures, and correlates events with Beluga AI voice backend sessions.

## Overview

When using Beluga AI's voice pipeline with the LiveKit transport, webhooks provide server-side visibility into session lifecycle events. By handling events like `room_finished` and `participant_joined`, you can:

- Track active sessions and participant counts
- Clean up backend state when rooms close
- Record session disposition and duration metrics
- Trigger downstream workflows (e.g., transcript archival, billing)

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- LiveKit project with URL, API key, and API secret
- A publicly accessible HTTPS endpoint for webhook delivery

## Installation

Install the Beluga AI module:

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

| Option | Description | Default |
|--------|-------------|---------|
| Webhook path | HTTP path for the webhook endpoint | `/voice/livekit/webhooks` |
| API secret | LiveKit API secret for HMAC signature verification | Env: `LIVEKIT_API_SECRET` |
| Subscribed events | Events to handle | `room_finished`, `participant_joined`, `participant_left` |

Set the required environment variable:

```bash
export LIVEKIT_API_SECRET="your-livekit-api-secret"
```

## Usage

### Webhook Endpoint with Signature Verification

Create an HTTP handler that accepts POST requests from LiveKit, verifies the HMAC-SHA256 signature, and dispatches events:

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
)

func main() {
    apiSecret := os.Getenv("LIVEKIT_API_SECRET")
    if apiSecret == "" {
        log.Fatal("LIVEKIT_API_SECRET is required")
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/voice/livekit/webhooks", livekitWebhookHandler(apiSecret))

    log.Println("Listening on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}

func livekitWebhookHandler(apiSecret string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "bad request", http.StatusBadRequest)
            return
        }

        sig := r.Header.Get("X-LiveKit-Signature")
        if !verifySignature(apiSecret, body, sig) {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        var evt struct {
            Event string `json:"event"`
            Room  struct {
                Name string `json:"name"`
                SID  string `json:"sid"`
            } `json:"room"`
            Participant struct {
                Identity string `json:"identity"`
                SID      string `json:"sid"`
            } `json:"participant"`
        }
        if err := json.Unmarshal(body, &evt); err != nil {
            http.Error(w, "bad request", http.StatusBadRequest)
            return
        }

        switch evt.Event {
        case "room_finished":
            handleRoomFinished(evt.Room.Name, evt.Room.SID)
        case "participant_joined":
            handleParticipantJoined(evt.Room.Name, evt.Participant.Identity)
        case "participant_left":
            handleParticipantLeft(evt.Room.Name, evt.Participant.Identity)
        default:
            log.Printf("Unhandled event: %s", evt.Event)
        }

        w.WriteHeader(http.StatusOK)
    }
}

func verifySignature(secret string, body []byte, sig string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(sig), []byte(expected))
}

func handleRoomFinished(roomName, roomSID string) {
    log.Printf("Room finished: name=%s sid=%s", roomName, roomSID)
    // Look up the backend session by room name.
    // Close the session, persist disposition, and emit metrics.
}

func handleParticipantJoined(roomName, identity string) {
    log.Printf("Participant joined: room=%s identity=%s", roomName, identity)
}

func handleParticipantLeft(roomName, identity string) {
    log.Printf("Participant left: room=%s identity=%s", roomName, identity)
}
```

### Registering the Webhook with LiveKit

Configure your LiveKit project to send webhooks to your endpoint:

1. Open your LiveKit project settings
2. Add a webhook URL (e.g., `https://your-app.example.com/voice/livekit/webhooks`)
3. Subscribe to the events you need: `room_finished`, `participant_joined`, `participant_left`

See the [LiveKit webhooks documentation](https://docs.livekit.io/realtime-api/webhooks/) for the full event catalog.

### Correlating Events with Backend Sessions

When creating a voice session with the LiveKit transport, use a deterministic room name that maps to your logical session ID. This enables correlation in the webhook handler:

```go
// When creating a LiveKit room, use the session ID as the room name.
// In handleRoomFinished, look up the session by room name:

func handleRoomFinished(roomName, roomSID string) {
    sessionID := roomName // Direct mapping: room name == session ID

    // Close the backend session and record metrics.
    log.Printf("Closing session %s (room SID: %s)", sessionID, roomSID)
}
```

## Advanced Topics

### Idempotent Event Handling

LiveKit may retry webhook delivery on failure. Use the room SID or a deduplication key to ensure handlers are idempotent:

```go
var processed sync.Map // In production, use a persistent store.

func handleRoomFinished(roomName, roomSID string) {
    if _, loaded := processed.LoadOrStore(roomSID, true); loaded {
        log.Printf("Duplicate room_finished for SID %s, skipping", roomSID)
        return
    }
    // Process the event.
}
```

### Asynchronous Processing

Webhook handlers should respond quickly. Enqueue heavy work (e.g., transcript archival, billing) to a background worker:

```go
func handleRoomFinished(roomName, roomSID string) {
    // Respond immediately; process asynchronously.
    go func() {
        // Archive transcripts, update billing, emit metrics.
        log.Printf("Processing room_finished asynchronously: %s", roomName)
    }()
}
```

In production, replace the goroutine with a proper job queue (e.g., NATS, Redis streams) for reliability.

### Production Considerations

- **HTTPS required**: LiveKit delivers webhooks over HTTPS. Use a reverse proxy or TLS-terminated load balancer.
- **Response time**: Return HTTP 200 within a few seconds. Defer heavy processing to background workers.
- **Monitoring**: Log webhook receipt, verification failures, and processing errors. Instrument with OpenTelemetry for end-to-end tracing.
- **Error handling**: Return non-2xx status codes only for genuine failures. LiveKit retries on non-2xx responses.

## Troubleshooting

### Signature Mismatch (Unauthorized)

The API secret used for verification does not match the LiveKit project secret. Ensure that `LIVEKIT_API_SECRET` matches exactly. Verify the signature against the raw request body before any JSON parsing.

### Room Finished but Session Not Found

The room name does not map to a known session ID. Use a stable, deterministic mapping (e.g., room name equals session ID) when creating LiveKit rooms and backend sessions.

### Webhook Never Fires

LiveKit cannot reach the webhook URL. Confirm that:
- The URL is publicly accessible over HTTPS
- The webhook subscription is configured in the LiveKit project settings
- Firewall rules allow inbound traffic on the webhook port

## Related Resources

- [Voice Services Overview](/integrations/voice-services) -- All supported voice providers
- [Vapi Custom Tools Integration](/integrations/vapi-custom-tools) -- Custom tools with Vapi
