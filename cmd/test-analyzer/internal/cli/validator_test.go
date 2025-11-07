package cli

import (
	"testing"
)

func TestValidateFlags(t *testing.T) {
	t.Run("ValidateValidFlags", func(t *testing.T) {
		config := &Config{
			DryRun:  true,
			AutoFix: false,
			Output:  "json",
		}

		err := ValidateFlags(config)
		if err != nil {
			t.Errorf("ValidateFlags() error = %v", err)
		}
	})

	t.Run("ValidateConflictingFlags", func(t *testing.T) {
		config := &Config{
			DryRun:  true,
			AutoFix: true, // Conflict
			Output:  "json",
		}

		err := ValidateFlags(config)
		if err == nil {
			t.Error("Expected error for conflicting flags, got nil")
		}
	})

	t.Run("ValidateInvalidOutput", func(t *testing.T) {
		config := &Config{
			Output: "invalid",
		}

		err := ValidateFlags(config)
		if err == nil {
			t.Error("Expected error for invalid output format, got nil")
		}
	})

	t.Run("ValidateInvalidSeverity", func(t *testing.T) {
		config := &Config{
			Severity: "invalid",
		}

		err := ValidateFlags(config)
		if err == nil {
			t.Error("Expected error for invalid severity, got nil")
		}
	})
}

