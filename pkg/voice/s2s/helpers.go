package s2s

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// NewAudioInput creates a new AudioInput with default format settings.
// This helper allows creating internal types from external packages.
func NewAudioInput(data []byte, sessionID string) *internal.AudioInput {
	return &internal.AudioInput{
		Data: data,
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}
}

// NewConversationContext creates a new ConversationContext from session ID.
// This helper allows creating internal types from external packages.
func NewConversationContext(sessionID string) *internal.ConversationContext {
	return &internal.ConversationContext{
		ConversationID: sessionID,
		SessionID:      sessionID,
	}
}

// ExtractAudioData extracts audio data from AudioOutput.
// This helper allows accessing internal types from external packages.
func ExtractAudioData(output *internal.AudioOutput) []byte {
	if output == nil {
		return nil
	}
	return output.Data
}

// NewAudioOutput creates a new AudioOutput with default format settings.
// This helper allows creating internal types from external packages.
func NewAudioOutput(data []byte, provider string, latency time.Duration) *internal.AudioOutput {
	return &internal.AudioOutput{
		Data: data,
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
		Provider:  provider,
		Latency:   latency,
		VoiceCharacteristics: internal.VoiceCharacteristics{
			Language: "en-US",
		},
	}
}
