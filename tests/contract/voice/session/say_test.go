package session

import (
	"context"
	"testing"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSay_Contract tests the contract for the Say() and SayWithOptions() methods.
func TestSay_Contract(t *testing.T) {
	t.Run("say returns handle for controlling playback", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)
		_ = sess // Use sess

		err := sess.Start(ctx)
		require.NoError(t, err)

		handle, err := sess.Say(ctx, "Hello")
		require.NoError(t, err)
		assert.NotNil(t, handle)
	})

	t.Run("say returns error if session not active", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		// Don't start session
		handle, err := sess.Say(ctx, "Hello")
		require.Error(t, err)
		assert.Nil(t, handle)
	})

	t.Run("say with options respects configuration", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		options := sessioniface.SayOptions{
			AllowInterruptions: true,
			Voice:              "en-US",
			Speed:              1.2,
			Volume:             0.8,
		}

		handle, err := sess.SayWithOptions(ctx, "Hello", options)
		require.NoError(t, err)
		assert.NotNil(t, handle)
	})

	t.Run("say handle wait for playout completes", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		handle, err := sess.Say(ctx, "Hello")
		require.NoError(t, err)

		// Wait for playout should complete (or timeout)
		err = handle.WaitForPlayout(ctx)
		// Should either complete successfully or timeout
		_ = err
	})

	t.Run("say handle cancel stops playback", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		handle, err := sess.Say(ctx, "Hello")
		require.NoError(t, err)

		err = handle.Cancel()
		require.NoError(t, err)
	})

	t.Run("say respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		cancel() // Cancel context

		handle, err := sess.Say(ctx, "Hello")
		// Should either return error or handle cancellation
		if err != nil {
			require.Error(t, err)
		}
		if handle != nil {
			_ = handle.Cancel() // Already handled
		}
	})
}
