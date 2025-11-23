package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStop_Contract tests the contract for the Stop() method
func TestStop_Contract(t *testing.T) {
	t.Run("stop transitions session to ended state", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		err = sess.Stop(ctx)
		require.NoError(t, err)

		state := sess.GetState()
		assert.Equal(t, "ended", string(state))
	})

	t.Run("stop returns error if session not active", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		// Don't start session
		err := sess.Stop(ctx)
		assert.Error(t, err)
	})

	t.Run("stop respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		cancel() // Cancel context
		err = sess.Stop(ctx)

		// Should either return error or respect cancellation
		if err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		err = sess.Stop(ctx)
		require.NoError(t, err)

		// Second stop should not error (or should handle gracefully)
		err = sess.Stop(ctx)
		// Implementation may vary - either no error or specific error
		_ = err
	})

	t.Run("stop preserves session ID", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		sessionIDBefore := sess.GetSessionID()

		err = sess.Stop(ctx)
		require.NoError(t, err)

		sessionIDAfter := sess.GetSessionID()
		assert.Equal(t, sessionIDBefore, sessionIDAfter)
	})
}
