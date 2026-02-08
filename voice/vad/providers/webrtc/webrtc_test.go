package webrtc

import (
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/voice"
)

// generatePCM creates 16-bit PCM audio with a sine wave at the given amplitude.
func generatePCM(numSamples int, amplitude float64) []byte {
	buf := make([]byte, numSamples*2)
	for i := range numSamples {
		sample := int16(amplitude * math.Sin(2*math.Pi*float64(i)/float64(numSamples)*10))
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(sample))
	}
	return buf
}

// generateSilence creates silent PCM audio (all zeros).
func generateSilence(numSamples int) []byte {
	return make([]byte, numSamples*2)
}

func TestNew(t *testing.T) {
	t.Run("default thresholds", func(t *testing.T) {
		v := New(0, 0)
		assert.Equal(t, defaultEnergyThreshold, v.energyThreshold)
		assert.Equal(t, defaultZCRThreshold, v.zcrThreshold)
	})

	t.Run("custom thresholds", func(t *testing.T) {
		v := New(2000.0, 0.2)
		assert.Equal(t, 2000.0, v.energyThreshold)
		assert.Equal(t, 0.2, v.zcrThreshold)
	})
}

func TestDetectActivity(t *testing.T) {
	t.Run("silence detection", func(t *testing.T) {
		v := New(1000, 0.5)
		audio := generateSilence(160)

		result, err := v.DetectActivity(context.Background(), audio)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSilence, result.EventType)
	})

	t.Run("speech detection", func(t *testing.T) {
		v := New(500, 0.5)
		audio := generatePCM(160, 5000)

		result, err := v.DetectActivity(context.Background(), audio)
		require.NoError(t, err)
		assert.True(t, result.IsSpeech)
		assert.Equal(t, voice.VADSpeechStart, result.EventType)
		assert.Greater(t, result.Confidence, 0.0)
	})

	t.Run("speech start event", func(t *testing.T) {
		v := New(500, 0.5)

		// First call with silence.
		silence := generateSilence(160)
		result, err := v.DetectActivity(context.Background(), silence)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSilence, result.EventType)

		// Second call with speech.
		speech := generatePCM(160, 5000)
		result, err = v.DetectActivity(context.Background(), speech)
		require.NoError(t, err)
		assert.True(t, result.IsSpeech)
		assert.Equal(t, voice.VADSpeechStart, result.EventType)
	})

	t.Run("speech end event", func(t *testing.T) {
		v := New(500, 0.5)

		// Start with speech.
		speech := generatePCM(160, 5000)
		result, err := v.DetectActivity(context.Background(), speech)
		require.NoError(t, err)
		assert.True(t, result.IsSpeech)

		// Then silence.
		silence := generateSilence(160)
		result, err = v.DetectActivity(context.Background(), silence)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSpeechEnd, result.EventType)
	})

	t.Run("too short audio", func(t *testing.T) {
		v := New(1000, 0.5)
		result, err := v.DetectActivity(context.Background(), []byte{0, 1})
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSilence, result.EventType)
	})

	t.Run("empty audio", func(t *testing.T) {
		v := New(1000, 0.5)
		result, err := v.DetectActivity(context.Background(), nil)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
	})

	t.Run("confidence clamped to 1", func(t *testing.T) {
		v := New(100, 0.5)
		audio := generatePCM(160, 10000)

		result, err := v.DetectActivity(context.Background(), audio)
		require.NoError(t, err)
		assert.LessOrEqual(t, result.Confidence, 1.0)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as webrtc", func(t *testing.T) {
		names := voice.ListVAD()
		found := false
		for _, name := range names {
			if name == "webrtc" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'webrtc' in registered VADs: %v", names)
	})

	t.Run("factory with config", func(t *testing.T) {
		vad, err := voice.NewVAD("webrtc", map[string]any{
			"threshold":     2000.0,
			"zcr_threshold": 0.2,
		})
		require.NoError(t, err)
		assert.NotNil(t, vad)
	})
}
