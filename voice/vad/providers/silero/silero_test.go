//go:build cgo

package silero

import (
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/voice"
)

func generatePCM(numSamples int, amplitude float64) []byte {
	buf := make([]byte, numSamples*2)
	for i := range numSamples {
		sample := int16(amplitude * math.Sin(2*math.Pi*float64(i)/float64(numSamples)*10))
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(sample))
	}
	return buf
}

func generateSilence(numSamples int) []byte {
	return make([]byte, numSamples*2)
}

func TestNew(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		v, err := New(Config{})
		require.NoError(t, err)
		assert.Equal(t, defaultThreshold, v.threshold)
		assert.Equal(t, defaultSampleRate, v.sampleRate)
	})

	t.Run("custom config", func(t *testing.T) {
		v, err := New(Config{
			Threshold:  0.7,
			SampleRate: 8000,
		})
		require.NoError(t, err)
		assert.Equal(t, 0.7, v.threshold)
		assert.Equal(t, 8000, v.sampleRate)
	})

	t.Run("invalid threshold defaults", func(t *testing.T) {
		v, err := New(Config{Threshold: -1})
		require.NoError(t, err)
		assert.Equal(t, defaultThreshold, v.threshold)
	})
}

func TestDetectActivity(t *testing.T) {
	t.Run("silence detection", func(t *testing.T) {
		v, err := New(Config{Threshold: 0.5})
		require.NoError(t, err)

		audio := generateSilence(160)
		result, err := v.DetectActivity(context.Background(), audio)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSilence, result.EventType)
	})

	t.Run("speech detection", func(t *testing.T) {
		v, err := New(Config{Threshold: 0.1})
		require.NoError(t, err)

		audio := generatePCM(160, 20000)
		result, err := v.DetectActivity(context.Background(), audio)
		require.NoError(t, err)
		assert.True(t, result.IsSpeech)
		assert.Equal(t, voice.VADSpeechStart, result.EventType)
		assert.Greater(t, result.Confidence, 0.0)
	})

	t.Run("speech start transition", func(t *testing.T) {
		v, err := New(Config{Threshold: 0.1})
		require.NoError(t, err)

		// Silence first.
		silence := generateSilence(160)
		result, err := v.DetectActivity(context.Background(), silence)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)

		// Then speech.
		speech := generatePCM(160, 20000)
		result, err = v.DetectActivity(context.Background(), speech)
		require.NoError(t, err)
		assert.True(t, result.IsSpeech)
		assert.Equal(t, voice.VADSpeechStart, result.EventType)
	})

	t.Run("speech end transition", func(t *testing.T) {
		v, err := New(Config{Threshold: 0.1})
		require.NoError(t, err)

		// Speech first.
		speech := generatePCM(160, 20000)
		_, err = v.DetectActivity(context.Background(), speech)
		require.NoError(t, err)

		// Then silence.
		silence := generateSilence(160)
		result, err := v.DetectActivity(context.Background(), silence)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSpeechEnd, result.EventType)
	})

	t.Run("empty audio", func(t *testing.T) {
		v, err := New(Config{})
		require.NoError(t, err)

		result, err := v.DetectActivity(context.Background(), nil)
		require.NoError(t, err)
		assert.False(t, result.IsSpeech)
		assert.Equal(t, voice.VADSilence, result.EventType)
	})

	t.Run("confidence clamped to 1", func(t *testing.T) {
		v, err := New(Config{Threshold: 0.01})
		require.NoError(t, err)

		audio := generatePCM(160, 30000)
		result, err := v.DetectActivity(context.Background(), audio)
		require.NoError(t, err)
		assert.LessOrEqual(t, result.Confidence, 1.0)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as silero", func(t *testing.T) {
		names := voice.ListVAD()
		found := false
		for _, name := range names {
			if name == "silero" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'silero' in registered VADs: %v", names)
	})

	t.Run("factory with config", func(t *testing.T) {
		vad, err := voice.NewVAD("silero", map[string]any{
			"threshold":   0.7,
			"sample_rate": 8000,
		})
		require.NoError(t, err)
		assert.NotNil(t, vad)
	})
}
