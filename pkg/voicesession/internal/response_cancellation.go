package internal

import (
	"context"
	"sync"
)

// ResponseCancellation manages cancellation of responses on interruption.
type ResponseCancellation struct {
	activeHandle any
	mu           sync.RWMutex
	canceled     bool
}

// NewResponseCancellation creates a new response cancellation manager.
func NewResponseCancellation() *ResponseCancellation {
	return &ResponseCancellation{
		canceled: false,
	}
}

// SetActiveHandle sets the active response handle.
func (rc *ResponseCancellation) SetActiveHandle(handle any) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.activeHandle = handle
	rc.canceled = false
}

// Cancel cancels the active response.
func (rc *ResponseCancellation) Cancel() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.canceled {
		return nil
	}

	rc.canceled = true

	// Try to cancel the handle if it supports cancellation
	if handle, ok := rc.activeHandle.(interface{ Cancel() error }); ok {
		return handle.Cancel()
	}

	return nil
}

// IsCancelled returns whether the response is canceled.
func (rc *ResponseCancellation) IsCancelled() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.canceled
}

// Clear clears the active handle.
func (rc *ResponseCancellation) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.activeHandle = nil
	rc.canceled = false
}

// CancelOnInterruption cancels the response if an interruption is detected.
func (rc *ResponseCancellation) CancelOnInterruption(ctx context.Context, detector *InterruptionDetector) error {
	if detector.IsInterrupted() {
		return rc.Cancel()
	}
	return nil
}
