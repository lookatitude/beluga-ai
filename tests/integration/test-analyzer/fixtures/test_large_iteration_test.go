package fixtures

import (
	"context"
	"testing"
	"time"
)

func TestLargeIteration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	// Large iteration count
	for i := 0; i < 100; i++ {
		_ = i * 2
	}
}
