package livekit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/voice"
	"github.com/lookatitude/beluga-ai/v2/voice/transport"
)

func TestNew(t *testing.T) {
	t.Run("missing url", func(t *testing.T) {
		_, err := New(transport.Config{Token: "token"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "URL is required")
	})

	t.Run("missing token", func(t *testing.T) {
		_, err := New(transport.Config{URL: "wss://test.livekit.cloud"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Token is required")
	})

	t.Run("valid config", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "test-token",
			Extra: map[string]any{"room": "test-room"},
		})
		require.NoError(t, err)
		assert.Equal(t, "wss://test.livekit.cloud", tr.url)
		assert.Equal(t, "test-token", tr.token)
		assert.Equal(t, "test-room", tr.room)
		assert.Equal(t, 16000, tr.sampleRate)
	})

	t.Run("custom sample rate", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:        "wss://test.livekit.cloud",
			Token:      "token",
			SampleRate: 48000,
			Channels:   2,
		})
		require.NoError(t, err)
		assert.Equal(t, 48000, tr.sampleRate)
		assert.Equal(t, 2, tr.channels)
	})
}

func TestRecv(t *testing.T) {
	t.Run("returns frame iterator", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "token",
		})
		require.NoError(t, err)

		// Iterator should not yield an error when the transport is healthy.
		// Drain in a goroutine with a cancellable context.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			var lastErr error
			for _, err := range tr.Recv(ctx) {
				if err != nil {
					lastErr = err
					break
				}
			}
			done <- lastErr
		}()
		cancel()
		select {
		case err := <-done:
			assert.NoError(t, err)
		}
	})

	t.Run("error when closed", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "token",
		})
		require.NoError(t, err)

		tr.Close()
		var gotErr error
		for _, err := range tr.Recv(context.Background()) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
		assert.Contains(t, gotErr.Error(), "closed")
	})
}

func TestSend(t *testing.T) {
	t.Run("send frame", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "token",
		})
		require.NoError(t, err)

		err = tr.Send(context.Background(), voice.NewAudioFrame([]byte("audio"), 16000))
		require.NoError(t, err)
	})

	t.Run("error when closed", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "token",
		})
		require.NoError(t, err)

		tr.Close()
		err = tr.Send(context.Background(), voice.NewAudioFrame([]byte("audio"), 16000))
		require.Error(t, err)
	})
}

func TestAudioOut(t *testing.T) {
	tr, err := New(transport.Config{
		URL:   "wss://test.livekit.cloud",
		Token: "token",
	})
	require.NoError(t, err)

	w := tr.AudioOut()
	assert.NotNil(t, w)

	n, err := w.Write([]byte("audio"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
}

func TestClose(t *testing.T) {
	t.Run("close once", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "token",
		})
		require.NoError(t, err)

		err = tr.Close()
		require.NoError(t, err)
		assert.True(t, tr.closed)
	})

	t.Run("close idempotent", func(t *testing.T) {
		tr, err := New(transport.Config{
			URL:   "wss://test.livekit.cloud",
			Token: "token",
		})
		require.NoError(t, err)

		tr.Close()
		err = tr.Close()
		require.NoError(t, err)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as livekit", func(t *testing.T) {
		names := transport.List()
		found := false
		for _, name := range names {
			if name == "livekit" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'livekit' in registered transports: %v", names)
	})
}
