package fixtures

import (
	"context"
	"testing"
	"time"
)

func TestSleepDelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	// Multiple sleep calls that accumulate to exceed threshold
	time.Sleep(50 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	time.Sleep(40 * time.Millisecond)
	// Total: 120ms > 100ms threshold
}
