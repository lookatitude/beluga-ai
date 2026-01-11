package livekit

import (
	"context"
	"fmt"

	"github.com/livekit/protocol/livekit"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
	"github.com/lookatitude/beluga-ai/pkg/voice/internal/audio"
)

// TrackHandler manages WebRTC audio tracks for LiveKit sessions.
type TrackHandler struct {
	session            *LiveKitSession
	pipelineOrchestrator *internal.PipelineOrchestrator
	userAudioTrack     *livekit.TrackInfo
	agentAudioTrack    *livekit.TrackInfo
}

// NewTrackHandler creates a new track handler.
func NewTrackHandler(session *LiveKitSession, orchestrator *internal.PipelineOrchestrator) *TrackHandler {
	return &TrackHandler{
		session:              session,
		pipelineOrchestrator: orchestrator,
	}
}

// SubscribeToUserAudio subscribes to the user's audio track.
func (th *TrackHandler) SubscribeToUserAudio(ctx context.Context, trackInfo *livekit.TrackInfo) error {
	th.userAudioTrack = trackInfo

	// TODO: In a full implementation, this would:
	// 1. Subscribe to the user's audio track via WebRTC
	// 2. Set up audio data callback
	// 3. Route audio to pipeline orchestrator

	return nil
}

// PublishAgentAudio publishes the agent's audio track.
func (th *TrackHandler) PublishAgentAudio(ctx context.Context) error {
	// TODO: In a full implementation, this would:
	// 1. Create a local audio track for the agent
	// 2. Publish it to the LiveKit room
	// 3. Set up audio data source from pipeline orchestrator output

	return nil
}

// HandleTrackEvent handles track events (muted, unmuted, disconnected).
// Note: TrackEvent type may not be available in protocol package, using string-based approach.
func (th *TrackHandler) HandleTrackEvent(eventType string) error {
	switch eventType {
	case "muted":
		// Handle muted event
		return nil
	case "unmuted":
		// Handle unmuted event
		return nil
	case "disconnected":
		// Handle disconnected event
		return backend.NewBackendError("HandleTrackEvent", backend.ErrCodeConnectionFailed,
			fmt.Errorf("track disconnected"))
	default:
		return nil
	}
}

// ConvertAudioFormat converts audio between formats (T290, FR-019).
// Handles audio format mismatch with automatic conversion.
func (th *TrackHandler) ConvertAudioFormat(ctx context.Context, audioData []byte, fromFormat, toFormat string) ([]byte, error) {
	// If formats are the same, return audio as-is
	if fromFormat == toFormat {
		return audioData, nil
	}

	// Use audio converter from voice package
	converter := audio.NewConverter()

	// Parse format strings to AudioFormat structs
	fromAudioFormat, err := parseFormatString(fromFormat)
	if err != nil {
		return nil, backend.NewBackendError("ConvertAudioFormat", backend.ErrCodeInvalidFormat,
			fmt.Errorf("invalid source format '%s': %w", fromFormat, err))
	}

	toAudioFormat, err := parseFormatString(toFormat)
	if err != nil {
		return nil, backend.NewBackendError("ConvertAudioFormat", backend.ErrCodeInvalidFormat,
			fmt.Errorf("invalid target format '%s': %w", toFormat, err))
	}

	// Validate formats
	if err = fromAudioFormat.Validate(); err != nil {
		return nil, backend.NewBackendError("ConvertAudioFormat", backend.ErrCodeInvalidFormat,
			fmt.Errorf("invalid source format: %w", err))
	}

	if err = toAudioFormat.Validate(); err != nil {
		return nil, backend.NewBackendError("ConvertAudioFormat", backend.ErrCodeInvalidFormat,
			fmt.Errorf("invalid target format: %w", err))
	}

	// Perform conversion
	convertedAudio, err := converter.Convert(audioData, fromAudioFormat, toAudioFormat)
	if err != nil {
		return nil, backend.NewBackendError("ConvertAudioFormat", backend.ErrCodeConversionFailed, err)
	}

	return convertedAudio, nil
}

// parseFormatString parses a format string (e.g., "pcm_16k_16bit_mono", "opus_48k") into AudioFormat.
func parseFormatString(format string) (*audio.AudioFormat, error) {
	// Default format if parsing fails
	defaultFormat := audio.DefaultAudioFormat()

	// Simple format parsing - in a full implementation, this would be more sophisticated
	// For now, support common formats:
	// - "pcm" or "pcm_16k_16bit_mono" -> 16kHz, 16-bit, mono PCM
	// - "opus" or "opus_48k" -> 48kHz Opus
	// - "g722" -> G.722 codec

	switch format {
	case "pcm", "pcm_16k_16bit_mono":
		return &audio.AudioFormat{
			SampleRate: 16000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "pcm",
		}, nil
	case "opus", "opus_48k":
		return &audio.AudioFormat{
			SampleRate: 48000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "opus",
		}, nil
	case "g722":
		return &audio.AudioFormat{
			SampleRate: 16000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "g722",
		}, nil
	default:
		// Return default format for unknown formats
		return defaultFormat, nil
	}
}

// BridgeToPipeline bridges LiveKit audio tracks to Beluga AI pipeline.
func (th *TrackHandler) BridgeToPipeline(ctx context.Context, audio []byte) error {
	// Convert audio format if needed
	convertedAudio, err := th.ConvertAudioFormat(ctx, audio, "opus", "pcm")
	if err != nil {
		return backend.WrapError("BridgeToPipeline", err)
	}

	// Process through pipeline orchestrator
	_, err = th.pipelineOrchestrator.ProcessAudio(ctx, convertedAudio,
		th.session.sessionConfig.AgentCallback,
		th.session.sessionConfig.AgentInstance)
	if err != nil {
		return backend.WrapError("BridgeToPipeline", err)
	}

	return nil
}
