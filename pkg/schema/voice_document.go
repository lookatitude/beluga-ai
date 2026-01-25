// Package schema provides multimodal document types for voice/audio.
package schema

import (
	"context"
	"log/slog"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// VoiceDocument represents a document containing audio/voice data.
// It extends Document with audio-specific fields while maintaining compatibility.
type VoiceDocument struct {
	AudioURL    string `json:"audio_url,omitempty"`
	AudioFormat string `json:"audio_format,omitempty"`
	Transcript  string `json:"transcript,omitempty"`
	AudioData   []byte `json:"audio_data,omitempty"`
	internal.Document
	Duration   float64 `json:"duration,omitempty"`
	SampleRate int     `json:"sample_rate,omitempty"`
	Channels   int     `json:"channels,omitempty"`
}

// GetContent returns the transcript if available, otherwise the page content.
func (d *VoiceDocument) GetContent() string {
	if d.Transcript != "" {
		return d.Transcript
	}
	return d.Document.GetContent()
}

// HasAudioData returns true if the document contains audio data.
func (d *VoiceDocument) HasAudioData() bool {
	return len(d.AudioData) > 0 || d.AudioURL != ""
}

// GetAudioURL returns the audio URL if present.
func (d *VoiceDocument) GetAudioURL() string {
	return d.AudioURL
}

// GetAudioData returns the audio data if present.
func (d *VoiceDocument) GetAudioData() []byte {
	return d.AudioData
}

// GetAudioFormat returns the audio format.
func (d *VoiceDocument) GetAudioFormat() string {
	return d.AudioFormat
}

// GetDuration returns the audio duration in seconds.
func (d *VoiceDocument) GetDuration() float64 {
	return d.Duration
}

// GetTranscript returns the transcript if available.
func (d *VoiceDocument) GetTranscript() string {
	return d.Transcript
}

// GetSampleRate returns the audio sample rate.
func (d *VoiceDocument) GetSampleRate() int {
	return d.SampleRate
}

// GetChannels returns the number of audio channels.
func (d *VoiceDocument) GetChannels() int {
	return d.Channels
}

// NewVoiceDocument creates a new VoiceDocument with a URL.
// Note: VoiceDocument embeds Document but is a different type.
// Use AsDocument() to convert to Document if needed, or use directly as Message.
func NewVoiceDocument(audioURL, transcript string, metadata map[string]string) *VoiceDocument {
	return &VoiceDocument{
		Document: internal.Document{
			PageContent: transcript,
			Metadata:    metadata,
		},
		AudioURL:    audioURL,
		AudioFormat: "auto", // Will be detected from URL if possible
		Transcript:  transcript,
	}
}

// NewVoiceDocumentWithData creates a new VoiceDocument with audio data.
// Note: VoiceDocument embeds Document but is a different type.
// Use AsDocument() to convert to Document if needed, or use directly as Message.
func NewVoiceDocumentWithData(audioData []byte, format, transcript string, metadata map[string]string) *VoiceDocument {
	return &VoiceDocument{
		Document: internal.Document{
			PageContent: transcript,
			Metadata:    metadata,
		},
		AudioData:   audioData,
		AudioFormat: format,
		Transcript:  transcript,
	}
}

// NewVoiceDocumentWithContext creates a new VoiceDocument with OTEL tracing context.
// Note: VoiceDocument embeds Document but is a different type.
// Use AsDocument() to convert to Document if needed, or use directly as Message.
func NewVoiceDocumentWithContext(ctx context.Context, audioURL, transcript string, metadata map[string]string) *VoiceDocument {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/schema")
	ctx, span := tracer.Start(ctx, "schema.NewVoiceDocument",
		trace.WithAttributes(
			attribute.String("document.type", "voice"),
			attribute.String("audio_url", audioURL),
			attribute.Int("transcript.length", len(transcript)),
		))
	defer span.End()

	// Record metrics
	RecordDocumentCreated(ctx)

	// Structured logging with OTEL context
	logWithOTELContext(ctx, slog.LevelInfo, "Creating voice document",
		"document_type", "voice",
		"audio_url", audioURL,
		"transcript_length", len(transcript))

	vd := NewVoiceDocument(audioURL, transcript, metadata)
	span.SetStatus(codes.Ok, "")
	return vd
}

// AsDocument converts VoiceDocument to Document by extracting the embedded Document.
// This allows VoiceDocument to be used where Document is expected.
func (vd *VoiceDocument) AsDocument() Document {
	return vd.Document
}

// Ensure VoiceDocument implements the Message interface (via embedded Document).
var _ iface.Message = (*VoiceDocument)(nil)
