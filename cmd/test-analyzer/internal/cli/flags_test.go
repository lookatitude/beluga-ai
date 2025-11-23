package cli

import (
	"testing"
)

func TestFlags(t *testing.T) {
	t.Run("DefaultFlags", func(t *testing.T) {
		config := &Config{}
		// Test default values
		if config.DryRun != false {
			t.Error("Expected DryRun to default to false")
		}
		if config.AutoFix != false {
			t.Error("Expected AutoFix to default to false")
		}
	})
}
