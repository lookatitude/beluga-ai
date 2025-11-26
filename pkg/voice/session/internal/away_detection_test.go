package internal

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAwayDetection(t *testing.T) {
	ad := NewAwayDetection(30*time.Second, nil)
	assert.NotNil(t, ad)
	assert.False(t, ad.IsAway())
}

func TestAwayDetection_UpdateActivity(t *testing.T) {
	ad := NewAwayDetection(100*time.Millisecond, func(isAway bool) {
		// Callback for state change
		_ = isAway
	})

	// Initially not away
	assert.False(t, ad.IsAway())

	// Update activity
	ad.UpdateActivity()
	assert.False(t, ad.IsAway())

	// Wait for away threshold
	time.Sleep(150 * time.Millisecond)
	ad.CheckAwayStatus()
	assert.True(t, ad.IsAway())

	// Update activity should mark as not away
	ad.UpdateActivity()
	assert.False(t, ad.IsAway())
}

func TestAwayDetection_CheckAwayStatus(t *testing.T) {
	ad := NewAwayDetection(50*time.Millisecond, nil)

	// Initially not away
	assert.False(t, ad.CheckAwayStatus())

	// Wait for threshold
	time.Sleep(60 * time.Millisecond)
	assert.True(t, ad.CheckAwayStatus())
	assert.True(t, ad.IsAway())
}

func TestAwayDetection_IsAway(t *testing.T) {
	ad := NewAwayDetection(50*time.Millisecond, nil)
	assert.False(t, ad.IsAway())

	time.Sleep(60 * time.Millisecond)
	ad.CheckAwayStatus()
	assert.True(t, ad.IsAway())
}

func TestAwayDetection_StartMonitoring(t *testing.T) {
	ad := NewAwayDetection(50*time.Millisecond, nil)
	ad.StartMonitoring(20 * time.Millisecond)

	// Wait for a few monitoring cycles
	time.Sleep(100 * time.Millisecond)

	// Should eventually detect away status
	time.Sleep(100 * time.Millisecond)
	ad.CheckAwayStatus()
	assert.True(t, ad.IsAway())
}

func TestAwayDetection_CallbackOnStateChange(t *testing.T) {
	var awayStates []bool
	var mu sync.Mutex

	ad := NewAwayDetection(50*time.Millisecond, func(isAway bool) {
		mu.Lock()
		awayStates = append(awayStates, isAway)
		mu.Unlock()
	})

	// Wait for away
	time.Sleep(60 * time.Millisecond)
	ad.CheckAwayStatus()

	// Update activity (should trigger callback)
	ad.UpdateActivity()

	mu.Lock()
	assert.NotEmpty(t, awayStates)
	mu.Unlock()
}
