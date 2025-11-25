package fixtures

import (
	"testing"
	"time"
)

func TestSleepDelay(t *testing.T) {
	// Multiple sleep calls that accumulate to exceed threshold
	time.Sleep(50 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	time.Sleep(40 * time.Millisecond)
	// Total: 120ms > 100ms threshold
}

