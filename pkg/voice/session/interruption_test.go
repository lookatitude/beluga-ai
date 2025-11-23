package session

import (
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"github.com/stretchr/testify/assert"
)

func TestInterruptionDetector_CheckInterruption(t *testing.T) {
	config := internal.DefaultInterruptionConfig()
	detector := internal.NewInterruptionDetector(config)

	// Test word count threshold
	assert.False(t, detector.CheckInterruption(2, 600*time.Millisecond))
	assert.True(t, detector.CheckInterruption(3, 600*time.Millisecond))

	// Test duration threshold
	assert.False(t, detector.CheckInterruption(3, 400*time.Millisecond))
	assert.True(t, detector.CheckInterruption(3, 600*time.Millisecond))

	// Test both thresholds
	assert.True(t, detector.CheckInterruption(3, 600*time.Millisecond))
}

func TestInterruptionDetector_Reset(t *testing.T) {
	config := internal.DefaultInterruptionConfig()
	detector := internal.NewInterruptionDetector(config)

	detector.CheckInterruption(3, 600*time.Millisecond)
	assert.True(t, detector.IsInterrupted())

	detector.Reset()
	assert.False(t, detector.IsInterrupted())
}
