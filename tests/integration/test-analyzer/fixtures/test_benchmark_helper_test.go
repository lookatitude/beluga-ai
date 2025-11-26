package fixtures

import "testing"

func TestBenchmarkHelperUsage(t *testing.T) {
	// Regular test using benchmark helper methods
	b := &testing.B{}
	b.ResetTimer()
	b.StopTimer()
	b.StartTimer()
}
