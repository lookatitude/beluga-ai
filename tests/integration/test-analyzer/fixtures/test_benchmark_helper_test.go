package fixtures

import (
	"context"
	"testing"
	"time"
)

func TestBenchmarkHelperUsage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	// Regular test using benchmark helper methods
	b := &testing.B{}
	b.ResetTimer()
	b.StopTimer()
	b.StartTimer()
}
