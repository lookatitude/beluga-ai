package fixtures

import "testing"

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestMissingTimeout(t *testing.T) {
	// This test is missing a timeout mechanism
	// It could potentially hang
	select {}
}

