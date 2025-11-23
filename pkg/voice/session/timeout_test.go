package session

import (
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"github.com/stretchr/testify/assert"
)

func TestSessionTimeout_StartStop(t *testing.T) {
	timeoutCalled := false
	var mu sync.Mutex

	onTimeout := func() {
		mu.Lock()
		timeoutCalled = true
		mu.Unlock()
	}

	timeout := internal.NewSessionTimeout(100*time.Millisecond, onTimeout)
	timeout.Start()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	called := timeoutCalled
	mu.Unlock()

	assert.True(t, called)

	timeout.Stop()
}

func TestSessionTimeout_UpdateActivity(t *testing.T) {
	timeoutCalled := false
	var mu sync.Mutex

	onTimeout := func() {
		mu.Lock()
		timeoutCalled = true
		mu.Unlock()
	}

	timeout := internal.NewSessionTimeout(100*time.Millisecond, onTimeout)
	timeout.Start()

	// Update activity before timeout
	time.Sleep(50 * time.Millisecond)
	timeout.UpdateActivity()

	// Wait - should not timeout because activity was updated
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := timeoutCalled
	mu.Unlock()

	assert.False(t, called)

	timeout.Stop()
}
