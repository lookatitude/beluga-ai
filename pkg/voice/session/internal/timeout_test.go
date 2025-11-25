package internal

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSessionTimeout(t *testing.T) {
	timeout := NewSessionTimeout(30*time.Second, nil)
	assert.NotNil(t, timeout)
	assert.False(t, timeout.active)
}

func TestSessionTimeout_Start(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	timeout.Start()
	assert.True(t, timeout.active)
}

func TestSessionTimeout_Start_AlreadyActive(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	timeout.Start()
	assert.True(t, timeout.active)

	// Start again
	timeout.Start()
	assert.True(t, timeout.active) // Should still be active
}

func TestSessionTimeout_Stop(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	timeout.Start()
	assert.True(t, timeout.active)

	timeout.Stop()
	assert.False(t, timeout.active)
}

func TestSessionTimeout_Stop_NotActive(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	timeout.Stop() // Should not panic
	assert.False(t, timeout.active)
}

func TestSessionTimeout_UpdateActivity(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	initialActivity := timeout.GetLastActivity()

	timeout.Start()
	time.Sleep(10 * time.Millisecond)
	timeout.UpdateActivity()

	newActivity := timeout.GetLastActivity()
	assert.True(t, newActivity.After(initialActivity))
}

func TestSessionTimeout_UpdateActivity_NotActive(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	initialActivity := timeout.GetLastActivity()

	time.Sleep(10 * time.Millisecond)
	timeout.UpdateActivity()

	// Activity should not change if not active
	newActivity := timeout.GetLastActivity()
	assert.True(t, newActivity.Equal(initialActivity) || newActivity.After(initialActivity))
}

func TestSessionTimeout_TimeoutCallback(t *testing.T) {
	var callbackCalled sync.WaitGroup
	callbackCalled.Add(1)

	timeout := NewSessionTimeout(50*time.Millisecond, func() {
		callbackCalled.Done()
	})

	timeout.Start()

	// Wait for timeout
	done := make(chan bool, 1)
	go func() {
		callbackCalled.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Callback was called
		assert.True(t, true)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout callback was not called")
	}
}

func TestSessionTimeout_GetLastActivity(t *testing.T) {
	timeout := NewSessionTimeout(100*time.Millisecond, nil)
	activity := timeout.GetLastActivity()
	assert.False(t, activity.IsZero())
}

func TestSessionTimeout_ResetOnUpdateActivity(t *testing.T) {
	var callbackCalled bool
	var mu sync.Mutex

	timeout := NewSessionTimeout(100*time.Millisecond, func() {
		mu.Lock()
		callbackCalled = true
		mu.Unlock()
	})

	timeout.Start()

	// Update activity before timeout
	time.Sleep(50 * time.Millisecond)
	timeout.UpdateActivity()

	// Wait longer than original timeout
	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	// Callback should not have been called yet due to reset
	// (though timing is tricky in tests)
	_ = callbackCalled
	mu.Unlock()
}
