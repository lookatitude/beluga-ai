package cli

import (
	"testing"
)

func TestParseFlags(t *testing.T) {
	t.Run("ParseEmptyFlags", func(t *testing.T) {
		config, err := ParseFlags([]string{})
		if err != nil {
			t.Fatalf("ParseFlags() error = %v", err)
		}
		if config == nil {
			t.Fatal("ParseFlags() returned nil")
		}
	})

	t.Run("ParseDryRunFlag", func(t *testing.T) {
		config, err := ParseFlags([]string{"--dry-run"})
		if err != nil {
			t.Fatalf("ParseFlags() error = %v", err)
		}
		if !config.DryRun {
			t.Error("Expected DryRun to be true")
		}
	})

	t.Run("ParseAutoFixFlag", func(t *testing.T) {
		config, err := ParseFlags([]string{"--auto-fix"})
		if err != nil {
			t.Fatalf("ParseFlags() error = %v", err)
		}
		if !config.AutoFix {
			t.Error("Expected AutoFix to be true")
		}
	})

	t.Run("ParseOutputFlag", func(t *testing.T) {
		config, err := ParseFlags([]string{"--output", "json"})
		if err != nil {
			t.Fatalf("ParseFlags() error = %v", err)
		}
		if config.Output != "json" {
			t.Errorf("Expected Output to be 'json', got %q", config.Output)
		}
	})
}

