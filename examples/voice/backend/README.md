# Voice Backend Example

This example demonstrates how to use the Voice Backend package for managing infrastructure-level voice interactions.

## Prerequisites

- Go 1.21+
- LiveKit server (for LiveKit backend)
- API keys and secrets

## Running the Example

```bash
# Set environment variables
export LIVEKIT_API_KEY="your-api-key"
export LIVEKIT_API_SECRET="your-api-secret"
export LIVEKIT_URL="wss://your-livekit-server.com"

go run main.go
```

## What This Example Shows

1. Creating backend configuration with API credentials
2. Creating a backend instance using the factory pattern
3. Creating sessions for voice interactions
4. Managing session lifecycle

## Configuration Options

- `APIKey`: Backend API key (required)
- `APISecret`: Backend API secret (required)
- `URL`: Backend server URL (required)
- Additional provider-specific options

## Using LiveKit Backend

To use the LiveKit backend:

```go
config := &vbiface.Config{
    APIKey:    os.Getenv("LIVEKIT_API_KEY"),
    APISecret: os.Getenv("LIVEKIT_API_SECRET"),
    URL:       "wss://your-livekit-server.com",
}

backend, err := backend.NewBackend(ctx, "livekit", config)
```

## Use Cases

- WebRTC-based voice applications
- Real-time voice agent infrastructure
- Multi-participant voice rooms
- Audio track management

## See Also

- [Backend Package Documentation](../../../pkg/voice/backend/README.md)
- [Voice Session Example](../simple/main.go)
