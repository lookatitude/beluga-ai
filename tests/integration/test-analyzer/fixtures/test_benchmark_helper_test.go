package fixtures

import "testing"

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestBenchmarkHelperUsage(t *testing.T) {
	// Regular test using benchmark helper methods
	b := &testing.B{}
	b.ResetTimer()
	b.StopTimer()
	b.StartTimer()
}

