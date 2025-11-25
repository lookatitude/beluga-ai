package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockCancellableHandle struct {
	cancelled bool
	err       error
}

func (m *mockCancellableHandle) Cancel() error {
	m.cancelled = true
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
	assert.NoError(t, err)
	assert.True(t, rc.IsCancelled())
	assert.True(t, handle.cancelled)

	// Cancel again (should be idempotent)
	err = rc.Cancel()
	assert.NoError(t, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_Cancel_WithError(t *testing.T) {
	rc := NewResponseCancellation()
	expectedErr := errors.New("cancel error")
	handle := &mockCancellableHandle{err: expectedErr}

	rc.SetActiveHandle(handle)

	err := rc.Cancel()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_Cancel_NonCancellableHandle(t *testing.T) {
	rc := NewResponseCancellation()
	// Set a handle that doesn't implement Cancel() method
	rc.SetActiveHandle("not cancellable")

	err := rc.Cancel()
	assert.NoError(t, err)
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_IsCancelled(t *testing.T) {
	rc := NewResponseCancellation()
	assert.False(t, rc.IsCancelled())

	rc.Cancel()
	assert.True(t, rc.IsCancelled())
}

func TestResponseCancellation_Clear(t *testing.T) {
	rc := NewResponseCancellation()
	handle := &mockCancellableHandle{}

	rc.SetActiveHandle(handle)
	rc.Cancel()
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
	assert.NoError(t, err)
	assert.False(t, rc.IsCancelled())

	// Trigger interruption (need both word count and duration thresholds)
	detector.CheckInterruption(3, 500*time.Millisecond)
	assert.True(t, detector.IsInterrupted())

	err = rc.CancelOnInterruption(context.Background(), detector)
	assert.NoError(t, err)
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
			rc.Cancel()
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
