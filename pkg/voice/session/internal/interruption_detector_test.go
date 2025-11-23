package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWordCountDetector(t *testing.T) {
	wcd := NewWordCountDetector(5)
	assert.NotNil(t, wcd)
	assert.False(t, wcd.CheckThreshold())
}

func TestWordCountDetector_AddWords(t *testing.T) {
	wcd := NewWordCountDetector(5)

	// Add words below threshold
	wcd.AddWords(3)
	assert.False(t, wcd.CheckThreshold())

	// Add more words to reach threshold
	wcd.AddWords(2)
	assert.True(t, wcd.CheckThreshold())
}

func TestWordCountDetector_CheckThreshold(t *testing.T) {
	wcd := NewWordCountDetector(10)

	assert.False(t, wcd.CheckThreshold())

	wcd.AddWords(10)
	assert.True(t, wcd.CheckThreshold())

	wcd.AddWords(1)
	assert.True(t, wcd.CheckThreshold())
}

func TestWordCountDetector_Reset(t *testing.T) {
	wcd := NewWordCountDetector(5)

	wcd.AddWords(5)
	assert.True(t, wcd.CheckThreshold())

	wcd.Reset()
	assert.False(t, wcd.CheckThreshold())
}

func TestNewDurationDetector(t *testing.T) {
	dd := NewDurationDetector(100 * time.Millisecond)
	assert.NotNil(t, dd)
	assert.False(t, dd.CheckThreshold())
}

func TestDurationDetector_StartStop(t *testing.T) {
	dd := NewDurationDetector(50 * time.Millisecond)

	// Not active initially
	assert.False(t, dd.CheckThreshold())

	// Start
	dd.Start()
	assert.False(t, dd.CheckThreshold()) // Not yet at threshold

	// Wait for threshold
	time.Sleep(60 * time.Millisecond)
	assert.True(t, dd.CheckThreshold())

	// Stop
	dd.Stop()
	assert.False(t, dd.CheckThreshold())
}

func TestDurationDetector_GetElapsed(t *testing.T) {
	dd := NewDurationDetector(100 * time.Millisecond)

	// Not active
	assert.Equal(t, time.Duration(0), dd.GetElapsed())

	// Start and check elapsed
	dd.Start()
	time.Sleep(20 * time.Millisecond)
	elapsed := dd.GetElapsed()
	assert.Greater(t, elapsed, time.Duration(0))
	assert.Less(t, elapsed, 100*time.Millisecond)

	dd.Stop()
	assert.Equal(t, time.Duration(0), dd.GetElapsed())
}

func TestDurationDetector_CheckThreshold_NotActive(t *testing.T) {
	dd := NewDurationDetector(50 * time.Millisecond)

	// Should return false when not active
	assert.False(t, dd.CheckThreshold())

	// Start and stop
	dd.Start()
	dd.Stop()

	// Should return false after stop
	assert.False(t, dd.CheckThreshold())
}

