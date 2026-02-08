package pipecat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/voice"
	"github.com/lookatitude/beluga-ai/voice/transport"
)

func TestNew(t *testing.T) {
	t.Run("missing url", func(t *testing.T) {
		_, err := New(transport.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "URL is required")
	})

	t.Run("valid config", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL: "ws://localhost:8765",
		})
		require.NoError(t, err)
		assert.Equal(t, "ws://localhost:8765", tr.url)
		assert.Equal(t, 16000, tr.sampleRate)
	})

	t.Run("custom sample rate", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:        "ws://localhost:8765",
			SampleRate: 24000,
		})
		require.NoError(t, err)
		assert.Equal(t, 24000, tr.sampleRate)
	})
}

func TestRecv(t *testing.T) {
	t.Run("returns channel", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "ws://localhost:8765"})
		ch, err := tr.Recv(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, ch)
	})

	t.Run("error when closed", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "ws://localhost:8765"})
		tr.Close()
		_, err := tr.Recv(context.Background())
		require.Error(t, err)
	})
}

func TestSend(t *testing.T) {
	t.Run("send frame", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "ws://localhost:8765"})
		err := tr.Send(context.Background(), voice.NewAudioFrame([]byte("audio"), 16000))
		require.NoError(t, err)
	})

	t.Run("error when closed", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "ws://localhost:8765"})
		tr.Close()
		err := tr.Send(context.Background(), voice.NewAudioFrame([]byte("audio"), 16000))
		require.Error(t, err)
	})
}

func TestClose(t *testing.T) {
	tr, _ := New(transport.Config{URL: "ws://localhost:8765"})
	err := tr.Close()
	require.NoError(t, err)
	assert.True(t, tr.closed)

	// Close again should be safe.
	err = tr.Close()
	require.NoError(t, err)
}

func TestRegistry(t *testing.T) {
	names := transport.List()
	found := false
	for _, name := range names {
		if name == "pipecat" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'pipecat' in registered transports: %v", names)
}
