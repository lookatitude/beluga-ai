package messaging

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
)

// NewBackend creates a new messaging backend instance using the global registry.
// It uses the provider name and configuration to create the appropriate backend.
// Messaging backends handle multi-channel conversations (SMS, WhatsApp, etc.)
// and integrate with AI agents for automated responses.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the backend provider to use (e.g., "twilio")
//   - config: Backend configuration containing connection details, API keys, etc.
//
// Returns:
//   - iface.ConversationalBackend: A new messaging backend instance ready to use
//   - error: Configuration validation errors or backend creation errors
//
// Example:
//
//	config := &Config{
//	    Provider: "twilio",
//	    // Provider-specific config
//	}
//	backend, err := messaging.NewBackend(ctx, "twilio", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	conversation, err := backend.CreateConversation(ctx, &iface.ConversationConfig{
//	    FriendlyName: "Customer Support",
//	})
func NewBackend(ctx context.Context, providerName string, config *Config) (iface.ConversationalBackend, error) {
	// Apply default config if nil
	if config == nil {
		config = DefaultConfig()
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get backend from registry
	registry := GetRegistry()
	backend, err := registry.Create(ctx, config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create messaging backend '%s': %w", config.Provider, err)
	}

	return backend, nil
}
