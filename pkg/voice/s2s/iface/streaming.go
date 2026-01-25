package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// StreamingS2SProvider defines the interface for streaming speech-to-speech providers.
// Implementations of this interface provide real-time bidirectional streaming capabilities.
//
// StreamingS2SProvider follows the Interface Segregation Principle (ISP) by segregating
// streaming functionality from basic S2S processing.
type StreamingS2SProvider interface {
	// Embed S2SProvider interface
	S2SProvider

	// StartStreaming begins a bidirectional streaming S2S session.
	// It takes a context, conversation context, and options, and returns a StreamingSession
	// for real-time audio input/output streaming.
	StartStreaming(ctx context.Context, context *internal.ConversationContext, opts ...internal.STSOption) (StreamingSession, error)
}

// StreamingSession defines the interface for streaming S2S sessions.
type StreamingSession interface {
	// SendAudio sends audio data to the streaming session.
	SendAudio(ctx context.Context, audio []byte) error

	// ReceiveAudio receives audio output from the streaming session.
	// Returns a channel that receives audio output chunks.
	ReceiveAudio() <-chan AudioOutputChunk

	// Close closes the streaming session gracefully.
	Close() error
}

// AudioOutputChunk represents an audio output chunk from a streaming session.
type AudioOutputChunk struct {
	Error     error
	Audio     []byte
	Timestamp int64
	IsFinal   bool
}
