package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInterruptionConfig(t *testing.T) {
	config := DefaultInterruptionConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 3, config.WordCountThreshold)
	assert.Equal(t, 500*time.Millisecond, config.DurationThreshold)
	assert.True(t, config.Enabled)
}

func TestNewInterruptionDetector(t *testing.T) {
	// With nil config
	detector := NewInterruptionDetector(nil)
	assert.NotNil(t, detector)
	assert.NotNil(t, detector.config)
	assert.Equal(t, 3, detector.config.WordCountThreshold)

	// With custom config
	customConfig := &InterruptionConfig{
		WordCountThreshold: 5,
		DurationThreshold:  1 * time.Second,
		Enabled:            true,
	}
	detector = NewInterruptionDetector(customConfig)
	assert.NotNil(t, detector)
	assert.Equal(t, customConfig, detector.config)
}

func TestInterruptionDetector_CheckInterruption(t *testing.T) {
	config := &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            true,
	}
	detector := NewInterruptionDetector(config)

	// Below threshold
	assert.False(t, detector.CheckInterruption(2, 400*time.Millisecond))
	assert.False(t, detector.IsInterrupted())

	// Word count threshold met, but duration not
	assert.False(t, detector.CheckInterruption(3, 400*time.Millisecond))
	assert.False(t, detector.IsInterrupted())

	// Duration threshold met, but word count not
	assert.False(t, detector.CheckInterruption(2, 500*time.Millisecond))
	assert.False(t, detector.IsInterrupted())

	// Both thresholds met
	assert.True(t, detector.CheckInterruption(3, 500*time.Millisecond))
	assert.True(t, detector.IsInterrupted())

	// Above thresholds
	assert.True(t, detector.CheckInterruption(5, 1*time.Second))
	assert.True(t, detector.IsInterrupted())
}

func TestInterruptionDetector_CheckInterruption_Disabled(t *testing.T) {
	config := &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            false,
	}
	detector := NewInterruptionDetector(config)

	// Should always return false when disabled
	assert.False(t, detector.CheckInterruption(10, 1*time.Second))
	assert.False(t, detector.IsInterrupted())
}

func TestInterruptionDetector_IsInterrupted(t *testing.T) {
	config := &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            true,
	}
	detector := NewInterruptionDetector(config)

	// Initially not interrupted
	assert.False(t, detector.IsInterrupted())

	// Check interruption
	detector.CheckInterruption(3, 500*time.Millisecond)
	assert.True(t, detector.IsInterrupted())
}

func TestInterruptionDetector_Reset(t *testing.T) {
	config := &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            true,
	}
	detector := NewInterruptionDetector(config)

	// Trigger interruption
	detector.CheckInterruption(3, 500*time.Millisecond)
	assert.True(t, detector.IsInterrupted())

	// Reset
	detector.Reset()
	assert.False(t, detector.IsInterrupted())

	// Can detect again after reset
	detector.CheckInterruption(3, 500*time.Millisecond)
	assert.True(t, detector.IsInterrupted())
}

func TestInterruptionDetector_ConcurrentAccess(t *testing.T) {
	config := &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            true,
	}
	detector := NewInterruptionDetector(config)

	// Concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			detector.CheckInterruption(3, 500*time.Millisecond)
			detector.IsInterrupted()
			detector.Reset()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
