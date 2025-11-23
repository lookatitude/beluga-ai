package fixtures

import "testing"

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestLargeIteration(t *testing.T) {
	// Large iteration count
	for i := 0; i < 100; i++ {
		_ = i * 2
	}
}

