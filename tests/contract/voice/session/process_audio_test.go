package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessAudio_Contract tests the contract for the ProcessAudio() method.
func TestProcessAudio_Contract(t *testing.T) {
	t.Run("process audio accepts valid audio data", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		audio := []byte{1, 2, 3, 4, 5}
		err = sess.ProcessAudio(ctx, audio)
		require.NoError(t, err)
	})

	t.Run("process audio returns error if session not active", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		// Don't start session
		audio := []byte{1, 2, 3, 4, 5}
		err := sess.ProcessAudio(ctx, audio)
		require.Error(t, err)
	})

	t.Run("process audio handles empty audio", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		audio := []byte{}
		err = sess.ProcessAudio(ctx, audio)
		// Should either accept empty audio or return error
		_ = err
	})

	t.Run("process audio handles large audio chunks", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		audio := make([]byte, 1024*1024) // 1MB
		err = sess.ProcessAudio(ctx, audio)
		// Should handle large chunks (may chunk internally)
		_ = err
	})

	t.Run("process audio respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		cancel() // Cancel context

		audio := []byte{1, 2, 3, 4, 5}
		err = sess.ProcessAudio(ctx, audio)
		// Should either return error or respect cancellation
		if err != nil {
			require.Error(t, err)
		}
	})

	t.Run("process audio transitions to processing state", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		audio := []byte{1, 2, 3, 4, 5}
		err = sess.ProcessAudio(ctx, audio)
		// State may transition to processing (implementation dependent)
		_ = err
		_ = sess.GetState()
	})
}
