# LiveKit Webhooks Integration

Welcome, colleague! In this guide we'll integrate **LiveKit webhooks** with Beluga AI's voice backend. You'll handle room and participant events (e.g. join, leave, track published) via webhooks and wire them to your backend sessions for visibility, analytics, and lifecycle management.

## What you will build

You will set up an HTTP endpoint that receives LiveKit webhooks, verify signatures, parse events (e.g. `room_finished`, `participant_joined`), and correlate them with `pkg/voice/backend` sessions. This allows you to track session lifecycle, clean up state, and trigger downstream logic (e.g. disposition, metrics) when calls end.

## Learning Objectives

- ✅ Expose an HTTP endpoint for LiveKit webhooks
- ✅ Verify webhook signatures (LiveKit API secret)
- ✅ Parse room and participant events
- ✅ Correlate events with backend sessions (e.g. by room name or session ID)
- ✅ Handle `room_finished` for cleanup and metrics

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- LiveKit project (URL, API key, API secret)
- Voice backend using the LiveKit provider

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

Use LiveKit's webhook format and signing (see [LiveKit webhooks](https://docs.livekit.io/realtime-api/webhooks/)).

## Step 2: Webhook Endpoint and Verification

Create an HTTP handler that accepts POSTs from LiveKit, verifies the signature, and parses the JSON body:
```go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

const livekitWebhookPath = "/voice/livekit/webhooks"

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
		if !verifyLiveKitSignature(apiSecret, body, sig) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var evt struct {
			Event string `json:"event"`
			Room  struct {
				Name string `json:"name"`
				SID  string `json:"sid"`
			} `json:"room"`
		}
		if err := json.Unmarshal(body, &evt); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		switch evt.Event {
		case "room_finished":
			onRoomFinished(evt.Room.Name, evt.Room.SID)
		case "participant_joined", "participant_left":
			// optional: track participants
		}
		w.WriteHeader(http.StatusOK)
	}
}

func verifyLiveKitSignature(secret string, body []byte, sig string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func onRoomFinished(roomName, roomSID string) {
	// Correlate room with your backend session (e.g. room name = session ID).
	// Close session, update disposition, emit metrics.
}
```

## Step 3: Register Webhook with LiveKit

Configure your LiveKit project to send webhooks to your endpoint (e.g. `https://your-app.example.com/voice/livekit/webhooks`). Subscribe to `room_finished`, `participant_joined`, `participant_left` as needed.

## Step 4: Correlate with Backend Sessions

When creating a session, use the LiveKit room name (or a mapping) as your logical session ID. In `onRoomFinished`, look up that session, call `CloseSession` if your backend holds references, and persist disposition or metrics.

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Webhook path | HTTP path for LiveKit | `/voice/livekit/webhooks` |
| API secret | LiveKit API secret for verification | Env `LIVEKIT_API_SECRET` |
| Events | Subscribed events | `room_finished`, participant events |

## Common Issues

### "Unauthorized" / signature mismatch

**Problem**: LiveKit signs with API secret; verification fails.

**Solution**: Use the same API secret as your LiveKit project. Ensure the raw request body is used for verification (before JSON parsing).

### "Room finished but session not found"

**Problem**: Room name or mapping doesn't match your session IDs.

**Solution**: Use a stable mapping (e.g. room name = `sessionID`) when creating LiveKit rooms and backend sessions.

### "Webhook never fires"

**Problem**: LiveKit can't reach your URL or subscription is wrong.

**Solution**: Use a public HTTPS URL; check LiveKit project webhook config and logs.

## Production Considerations

- **Idempotency**: Handle duplicate webhooks (e.g. `room_finished` retries).
- **Timeouts**: Respond quickly; do minimal work in handler; enqueue heavy work.
- **Monitoring**: Log and metric webhook receipt, verification failures, and `onRoomFinished` processing.

## Next Steps

- **[Vapi Custom Tools](./vapi-custom-tools.md)** — Custom tools with Vapi.
- **[Voice Backends Tutorial](../../../tutorials/voice/voice-backends-livekit-vapi.md)** — Backend setup.
- **[Scaling Concurrent Streams](../../../cookbook/voice-backend-scaling-concurrent-streams.md)** — Production scaling.
