package session

import (
	"context"
	"testing"
	"time"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformance_Contract tests performance contracts for session operations
func TestPerformance_Contract(t *testing.T) {
	t.Run("start completes within reasonable time", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		start := time.Now()
		err := sess.Start(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		// Should complete within 100ms (contract requirement)
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("stop completes within reasonable time", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		start := time.Now()
		err = sess.Stop(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		// Should complete within 100ms (contract requirement)
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("process audio handles high throughput", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		// Process multiple audio chunks quickly
		start := time.Now()
		for i := 0; i < 100; i++ {
			audio := []byte{byte(i)}
			_ = sess.ProcessAudio(ctx, audio)
		}
		duration := time.Since(start)

		// Should handle 100 chunks within reasonable time
		assert.Less(t, duration, 1*time.Second)
	})

	t.Run("say operation is non-blocking", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		start := time.Now()
		handle, err := sess.Say(ctx, "Hello")
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, handle)
		// Say should return quickly (non-blocking)
		assert.Less(t, duration, 50*time.Millisecond)
	})

	t.Run("concurrent operations are safe", func(t *testing.T) {
		ctx := context.Background()
		sess := createTestSession(t)

		err := sess.Start(ctx)
		require.NoError(t, err)

		// Concurrent operations should not panic
		done := make(chan bool, 3)
		go func() {
			_ = sess.ProcessAudio(ctx, []byte{1, 2, 3})
			done <- true
		}()
		go func() {
			_ = sess.GetState()
			done <- true
		}()
		go func() {
			_, _ = sess.Say(ctx, "test")
			done <- true
		}()

		// Wait for all operations
		for i := 0; i < 3; i++ {
			<-done
		}

		// Session should still be in valid state
		state := sess.GetState()
		assert.NotEqual(t, sessioniface.SessionState("ended"), state)
	})
}
