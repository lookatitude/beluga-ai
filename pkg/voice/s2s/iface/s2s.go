// Package iface defines the core interfaces for the S2S package.
// It follows the Interface Segregation Principle by providing small, focused interfaces
// that serve specific purposes within the speech-to-speech system.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// S2SProvider defines the interface for speech-to-speech providers.
// Implementations of this interface will provide access to different
// S2S services (e.g., Amazon Nova 2 Sonic, Grok Voice Agent, Gemini 2.5 Flash Native Audio, GPT Realtime).
//
// S2SProvider follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to speech-to-speech operations.
type S2SProvider interface {
	// Process converts audio input to audio output directly.
	// It takes a context for cancellation and deadline propagation, audio input,
	// conversation context, and options, and returns audio output or an error if the process fails.
	Process(ctx context.Context, input *internal.AudioInput, context *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error)

	// Name returns the provider name.
	Name() string
}
