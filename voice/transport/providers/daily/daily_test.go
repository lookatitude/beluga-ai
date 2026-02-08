package daily

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
			URL:   "https://myapp.daily.co/room",
			Token: "token",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://myapp.daily.co/room", tr.url)
		assert.Equal(t, 16000, tr.sampleRate)
	})

	t.Run("custom sample rate", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:        "https://myapp.daily.co/room",
			SampleRate: 48000,
		})
		require.NoError(t, err)
		assert.Equal(t, 48000, tr.sampleRate)
	})
}

func TestRecv(t *testing.T) {
	t.Run("returns channel", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "https://test.daily.co/room"})
		ch, err := tr.Recv(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, ch)
	})

	t.Run("error when closed", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "https://test.daily.co/room"})
		tr.Close()
		_, err := tr.Recv(context.Background())
		require.Error(t, err)
	})
}

func TestSend(t *testing.T) {
	t.Run("send frame", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "https://test.daily.co/room"})
		err := tr.Send(context.Background(), voice.NewAudioFrame([]byte("audio"), 16000))
		require.NoError(t, err)
	})

	t.Run("error when closed", func(t *testing.T) {
		tr, _ := New(transport.Config{URL: "https://test.daily.co/room"})
		tr.Close()
		err := tr.Send(context.Background(), voice.NewAudioFrame([]byte("audio"), 16000))
		require.Error(t, err)
	})
}

func TestClose(t *testing.T) {
	tr, _ := New(transport.Config{URL: "https://test.daily.co/room"})
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
		if name == "daily" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'daily' in registered transports: %v", names)
}
