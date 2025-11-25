package fixtures

import "testing"

func TestLargeIteration(t *testing.T) {
	// Large iteration count
	for i := 0; i < 100; i++ {
		_ = i * 2
	}
}

