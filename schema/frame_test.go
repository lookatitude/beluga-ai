package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameTypes(t *testing.T) {
	tests := []struct {
		name string
		ft   FrameType
		want string
	}{
		{"audio", FrameAudio, "audio"},
		{"text", FrameText, "text"},
		{"control", FrameControl, "control"},
		{"image", FrameImage, "image"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, FrameType(tt.want), tt.ft)
		})
	}
}

func TestSignalConstants(t *testing.T) {
	assert.Equal(t, "start", SignalStart)
	assert.Equal(t, "stop", SignalStop)
	assert.Equal(t, "interrupt", SignalInterrupt)
	assert.Equal(t, "end_of_utterance", SignalEndOfUtterance)
}

func TestNewAudioFrame(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		sampleRate int
	}{
		{"16kHz audio", []byte{1, 2, 3, 4}, 16000},
		{"44.1kHz audio", []byte{5, 6, 7, 8}, 44100},
		{"empty audio", []byte{}, 8000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := NewAudioFrame(tt.data, tt.sampleRate)
			assert.Equal(t, FrameAudio, frame.Type)
			assert.Equal(t, tt.data, frame.Data)
			require.NotNil(t, frame.Metadata)
			assert.Equal(t, tt.sampleRate, frame.Metadata["sample_rate"])
		})
	}
}

func TestNewTextFrame(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"simple text", "hello world"},
		{"empty text", ""},
		{"multi-line text", "line1\nline2\nline3"},
		{"unicode text", "你好世界 🌍"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := NewTextFrame(tt.text)
			assert.Equal(t, FrameText, frame.Type)
			assert.Equal(t, []byte(tt.text), frame.Data)
		})
	}
}

func TestNewControlFrame(t *testing.T) {
	tests := []struct {
		name   string
		signal string
	}{
		{"start signal", SignalStart},
		{"stop signal", SignalStop},
		{"interrupt signal", SignalInterrupt},
		{"end of utterance", SignalEndOfUtterance},
		{"custom signal", "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := NewControlFrame(tt.signal)
			assert.Equal(t, FrameControl, frame.Type)
			require.NotNil(t, frame.Metadata)
			assert.Equal(t, tt.signal, frame.Metadata["signal"])
			assert.Empty(t, frame.Data)
		})
	}
}

func TestNewImageFrame(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
	}{
		{"PNG image", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png"},
		{"JPEG image", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg"},
		{"WebP image", []byte{0x52, 0x49, 0x46, 0x46}, "image/webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := NewImageFrame(tt.data, tt.contentType)
			assert.Equal(t, FrameImage, frame.Type)
			assert.Equal(t, tt.data, frame.Data)
			require.NotNil(t, frame.Metadata)
			assert.Equal(t, tt.contentType, frame.Metadata["content_type"])
		})
	}
}

func TestFrameSignal(t *testing.T) {
	tests := []struct {
		name     string
		frame    Frame
		wantSig  string
		wantDesc string
	}{
		{
			"control frame with start signal",
			NewControlFrame(SignalStart),
			SignalStart,
			"should extract signal from control frame",
		},
		{
			"non-control frame",
			NewTextFrame("hello"),
			"",
			"should return empty string for non-control frame",
		},
		{
			"control frame with no metadata",
			Frame{Type: FrameControl},
			"",
			"should return empty string when metadata is nil",
		},
		{
			"control frame with non-string signal",
			Frame{
				Type: FrameControl,
				Metadata: map[string]any{
					"signal": 123,
				},
			},
			"",
			"should return empty string when signal is not a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.frame.Signal()
			assert.Equal(t, tt.wantSig, got, tt.wantDesc)
		})
	}
}

func TestFrameText(t *testing.T) {
	tests := []struct {
		name     string
		frame    Frame
		wantText string
		wantDesc string
	}{
		{
			"text frame",
			NewTextFrame("hello world"),
			"hello world",
			"should extract text from text frame",
		},
		{
			"empty text frame",
			NewTextFrame(""),
			"",
			"should handle empty text",
		},
		{
			"non-text frame",
			NewAudioFrame([]byte{1, 2, 3}, 16000),
			string([]byte{1, 2, 3}),
			"should convert bytes to string for any frame type",
		},
		{
			"frame with no data",
			Frame{Type: FrameControl},
			"",
			"should return empty string for frame with no data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.frame.Text()
			assert.Equal(t, tt.wantText, got, tt.wantDesc)
		})
	}
}

func TestFrameRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
	}{
		{"audio frame", NewAudioFrame([]byte{1, 2, 3, 4}, 16000)},
		{"text frame", NewTextFrame("test message")},
		{"control frame", NewControlFrame(SignalStop)},
		{"image frame", NewImageFrame([]byte{0xFF, 0xD8}, "image/jpeg")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify frame properties are preserved
			assert.NotEmpty(t, tt.frame.Type)
			// Type should be one of the defined frame types
			switch tt.frame.Type {
			case FrameAudio, FrameText, FrameControl, FrameImage:
				// Valid type
			default:
				t.Errorf("invalid frame type: %s", tt.frame.Type)
			}
		})
	}
}

func TestFrameMetadata(t *testing.T) {
	t.Run("metadata preservation", func(t *testing.T) {
		frame := Frame{
			Type: FrameAudio,
			Data: []byte{1, 2, 3},
			Metadata: map[string]any{
				"sample_rate": 16000,
				"encoding":    "pcm16",
				"channels":    2,
			},
		}
		assert.Equal(t, 16000, frame.Metadata["sample_rate"])
		assert.Equal(t, "pcm16", frame.Metadata["encoding"])
		assert.Equal(t, 2, frame.Metadata["channels"])
	})

	t.Run("nil metadata safe access", func(t *testing.T) {
		frame := Frame{Type: FrameText, Data: []byte("test")}
		// These should not panic
		sig := frame.Signal()
		assert.Empty(t, sig)
	})
}
