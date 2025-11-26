package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCancellableHandle struct {
	err      error
	canceled bool
}

func (m *mockCancellableHandle) Cancel() error {
	m.canceled = true
	return m.err
}

func TestNewResponseCancellation(t *testing.T) {
	rc := NewResponseCancellation()
	assert.NotNil(t, rc)
	assert.False(t, rc.IsCancelled())
}

func TestResponseCancellation_SetActiveHandle(t *testing.T) {
	rc := NewResponseCancellation()
	handle := &mockCancellableHandle{}

	rc.SetActiveHandle(handle)
	assert.False(t, rc.IsCancelled())
}

func TestResponseCancellation_Cancel(t *testing.T) {
	rc := NewResponseCancellation()
	handle := &mockCancellableHandle{}

	rc.SetActiveHandle(handle)

	// Cancel
	err := rc.Cancel()
	require.NoError(t, err)
	assert.True(t, rc.IsCancelled())
	assert.True(t, handle.canceled)

	// Cancel again (should be idempotent)
	err = rc.Cancel()
	require.NoError(t, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_Cancel_WithError(t *testing.T) {
	rc := NewResponseCancellation()
	expectedErr := errors.New("cancel error")
	handle := &mockCancellableHandle{err: expectedErr}

	rc.SetActiveHandle(handle)

	err := rc.Cancel()
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_Cancel_NonCancellableHandle(t *testing.T) {
	rc := NewResponseCancellation()
	// Set a handle that doesn't implement Cancel() method
	rc.SetActiveHandle("not cancellable")

	err := rc.Cancel()
	require.NoError(t, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_IsCancelled(t *testing.T) {
	rc := NewResponseCancellation()
	assert.False(t, rc.IsCancelled())

	_ = rc.Cancel()
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_Clear(t *testing.T) {
	rc := NewResponseCancellation()
	handle := &mockCancellableHandle{}

	rc.SetActiveHandle(handle)
	_ = rc.Cancel()
	assert.True(t, rc.IsCancelled())

	// Clear
	rc.Clear()
	assert.False(t, rc.IsCancelled())
}

func TestResponseCancellation_CancelOnInterruption(t *testing.T) {
	rc := NewResponseCancellation()
	handle := &mockCancellableHandle{}
	rc.SetActiveHandle(handle)

	config := &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            true,
	}
	detector := NewInterruptionDetector(config)

	// No interruption
	err := rc.CancelOnInterruption(context.Background(), detector)
	require.NoError(t, err)
	assert.False(t, rc.IsCancelled())

	// Trigger interruption (need both word count and duration thresholds)
	detector.CheckInterruption(3, 500*time.Millisecond)
	assert.True(t, detector.IsInterrupted())

	err = rc.CancelOnInterruption(context.Background(), detector)
	require.NoError(t, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_ConcurrentAccess(t *testing.T) {
	rc := NewResponseCancellation()
	handle := &mockCancellableHandle{}

	rc.SetActiveHandle(handle)

	// Concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = rc.Cancel()
			rc.IsCancelled()
			rc.Clear()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
