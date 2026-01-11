// Package backend provides interfaces and implementations for Voice Backend operations.
// Voice backends manage the infrastructure layer for real-time voice interactions,
// including WebRTC connections, room management, and audio streaming.
package backend

import (
	"context"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// NewBackend creates a new voice backend instance using the global registry.
// It uses the provider name and configuration to create the appropriate backend.
// Voice backends handle the low-level infrastructure for voice interactions,
// such as WebRTC connections, room management, and audio track handling.
// Supported providers include: livekit, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the backend provider to use (e.g., "livekit")
//   - config: Backend configuration containing connection details, API keys, etc.
//
// Returns:
//   - vbiface.VoiceBackend: A new voice backend instance ready to use
//   - error: Configuration validation errors or backend creation errors
//
// Example:
//
//	config := &vbiface.Config{
//	    APIKey:    "your-api-key",
//	    APISecret: "your-api-secret",
//	    URL:       "wss://your-livekit-server.com",
//	}
//	backend, err := backend.NewBackend(ctx, "livekit", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	session, err := backend.CreateSession(ctx, roomName)
//
// Example usage can be found in examples/voice/backend/main.go
func NewBackend(ctx context.Context, providerName string, config *vbiface.Config) (vbiface.VoiceBackend, error) {
	registry := GetRegistry()
	return registry.Create(ctx, providerName, config)
}

// GetRegistry returns the global voice backend registry instance.
// This is a convenience wrapper around the registry.GetRegistry() function.
// Note: This function is already defined in registry.go, so this file
// primarily provides the NewBackend factory function.
