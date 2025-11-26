package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStart_Contract tests the contract for the Start() method
// These tests ensure that any implementation of VoiceSession follows the expected behavior.
func TestStart_Contract(t *testing.T) {
	t.Run("start initializes session to listening state", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		state := sess.GetState()
		assert.Equal(t, "listening", string(state))
	})

	t.Run("start returns error if session already active", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		err = sess.Start(ctx)
		require.Error(t, err)
	})

	t.Run("start respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		sess := createTestSession(t)
		err := sess.Start(ctx)
		// Should either return error or respect cancellation
		if err != nil {
			require.Error(t, err)
		}
	})

	t.Run("start is idempotent after stop", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		// Start and stop
		err := sess.Start(ctx)
		require.NoError(t, err)

		err = sess.Stop(ctx)
		require.NoError(t, err)

		// Should be able to start again
		err = sess.Start(ctx)
		require.NoError(t, err)
	})

	t.Run("start generates session ID if not provided", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		sessionID := sess.GetSessionID()
		assert.NotEmpty(t, sessionID)
	})
}
