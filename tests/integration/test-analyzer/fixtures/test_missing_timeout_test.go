package fixtures

import "testing"

func TestMissingTimeout(t *testing.T) {
	// This test is missing a timeout mechanism
	// It could potentially hang
	select {}
}

