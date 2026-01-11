// Package schema provides tests for VoiceDocument type.
package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVoiceDocument(t *testing.T) {
	tests := []struct {
		name      string
		audioURL  string
		transcript string
		metadata  map[string]string
		validate  func(t *testing.T, doc *VoiceDocument)
	}{
		{
			name:      "basic_voice_document",
			audioURL:  "https://example.com/audio.wav",
			transcript: "This is a test transcript",
			metadata:  map[string]string{"speaker": "Alice"},
			validate: func(t *testing.T, doc *VoiceDocument) {
				assert.Equal(t, "https://example.com/audio.wav", doc.AudioURL)
				assert.Equal(t, "This is a test transcript", doc.Transcript)
				assert.Equal(t, "This is a test transcript", doc.GetContent())
				assert.Equal(t, "Alice", doc.Metadata["speaker"])
				assert.True(t, doc.HasAudioData())
			},
		},
		{
			name:      "voice_document_no_metadata",
			audioURL:  "https://example.com/audio.mp3",
			transcript: "Test transcript",
			metadata:  nil,
			validate: func(t *testing.T, doc *VoiceDocument) {
				assert.Equal(t, "https://example.com/audio.mp3", doc.AudioURL)
				assert.Equal(t, "Test transcript", doc.Transcript)
				assert.True(t, doc.HasAudioData())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewVoiceDocument(tt.audioURL, tt.transcript, tt.metadata)
			tt.validate(t, doc)
		})
	}
}

func TestNewVoiceDocumentWithData(t *testing.T) {
	audioData := []byte{0x52, 0x49, 0x46, 0x46} // WAV header
	format := "wav"
	transcript := "WAV audio transcript"
	metadata := map[string]string{"format": "wav"}

	doc := NewVoiceDocumentWithData(audioData, format, transcript, metadata)
	
	assert.Equal(t, audioData, doc.AudioData)
	assert.Equal(t, format, doc.AudioFormat)
	assert.Equal(t, transcript, doc.Transcript)
	assert.Equal(t, transcript, doc.GetContent())
	assert.True(t, doc.HasAudioData())
}

func TestNewVoiceDocumentWithContext(t *testing.T) {
	ctx := context.Background()
	audioURL := "https://example.com/audio.wav"
	transcript := "Test transcript"
	metadata := map[string]string{"test": "value"}

	doc := NewVoiceDocumentWithContext(ctx, audioURL, transcript, metadata)
	
	assert.Equal(t, audioURL, doc.AudioURL)
	assert.Equal(t, transcript, doc.Transcript)
	assert.Equal(t, "value", doc.Metadata["test"])
}

func TestVoiceDocumentAsDocument(t *testing.T) {
	audioURL := "https://example.com/audio.wav"
	transcript := "Test transcript"
	metadata := map[string]string{"key": "value"}

	voiceDoc := NewVoiceDocument(audioURL, transcript, metadata)
	doc := voiceDoc.AsDocument()
	
	assert.Equal(t, transcript, doc.PageContent)
	assert.Equal(t, metadata, doc.Metadata)
	assert.Equal(t, transcript, doc.GetContent())
}

func TestVoiceDocumentMethods(t *testing.T) {
	audioURL := "https://example.com/audio.wav"
	transcript := "Test transcript"
	duration := 30.5
	sampleRate := 44100
	channels := 2
	metadata := map[string]string{"test": "value"}

	doc := NewVoiceDocument(audioURL, transcript, metadata)
	doc.Duration = duration
	doc.SampleRate = sampleRate
	doc.Channels = channels
	
	assert.Equal(t, audioURL, doc.GetAudioURL())
	assert.Equal(t, "auto", doc.GetAudioFormat())
	assert.Equal(t, duration, doc.GetDuration())
	assert.Equal(t, transcript, doc.GetTranscript())
	assert.Equal(t, sampleRate, doc.GetSampleRate())
	assert.Equal(t, channels, doc.GetChannels())
	assert.True(t, doc.HasAudioData())
}

func TestVoiceDocumentImplementsMessage(t *testing.T) {
	doc := NewVoiceDocument("https://example.com/audio.wav", "Test", nil)
	
	// VoiceDocument should implement Message interface
	var msg Message = doc
	assert.Equal(t, RoleSystem, msg.GetType())
	assert.Equal(t, "Test", msg.GetContent())
	assert.Empty(t, msg.ToolCalls())
}
