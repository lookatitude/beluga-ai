package session

import (
	"context"
	"testing"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestState_Contract tests the contract for state management
func TestState_Contract(t *testing.T) {
	t.Run("initial state is initial", func(t *testing.T) {
		sess := createTestSession(t)
		state := sess.GetState()
		assert.Equal(t, sessioniface.SessionState("initial"), state)
	})

	t.Run("state transitions from initial to listening on start", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		assert.Equal(t, sessioniface.SessionState("initial"), sess.GetState())

		err := sess.Start(ctx)
		require.NoError(t, err)

		assert.Equal(t, sessioniface.SessionState("listening"), sess.GetState())
	})

	t.Run("state transitions to ended on stop", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		err = sess.Stop(ctx)
		require.NoError(t, err)

		assert.Equal(t, sessioniface.SessionState("ended"), sess.GetState())
	})

	t.Run("state callback is called on state changes", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		stateChanges := []sessioniface.SessionState{}
		sess.OnStateChanged(func(state sessioniface.SessionState) {
			stateChanges = append(stateChanges, state)
		})

		err := sess.Start(ctx)
		require.NoError(t, err)

		// Should have received listening state
		assert.Contains(t, stateChanges, sessioniface.SessionState("listening"))

		err = sess.Stop(ctx)
		require.NoError(t, err)

		// Should have received ended state
		assert.Contains(t, stateChanges, sessioniface.SessionState("ended"))
	})

	t.Run("state is thread-safe", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		// Concurrent state reads should not panic
		done := make(chan bool)
		go func() {
			for i := 0; i < 100; i++ {
				_ = sess.GetState()
			}
			done <- true
		}()

		err := sess.Start(ctx)
		require.NoError(t, err)

		<-done
		assert.Equal(t, sessioniface.SessionState("listening"), sess.GetState())
	})

	t.Run("state transitions are valid", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		// Initial -> Listening
		err := sess.Start(ctx)
		require.NoError(t, err)
		assert.Equal(t, sessioniface.SessionState("listening"), sess.GetState())

		// Listening -> Ended
		err = sess.Stop(ctx)
		require.NoError(t, err)
		assert.Equal(t, sessioniface.SessionState("ended"), sess.GetState())
	})
}
