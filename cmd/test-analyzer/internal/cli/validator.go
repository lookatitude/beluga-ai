package cli

import (
	"fmt"
	"strings"
)

// ValidateFlags validates the configuration flags.
func ValidateFlags(config *Config) error {
	// Check for conflicting flags
	if config.DryRun && config.AutoFix {
		return fmt.Errorf("--dry-run and --auto-fix cannot be used together")
	}

	if config.SkipValidation && config.AutoFix {
		return fmt.Errorf("--skip-validation and --auto-fix cannot be used together (validation is required for auto-fix)")
	}

	// Validate output format
	validFormats := map[string]bool{
		"stdout":   true,
		"json":     true,
		"html":     true,
		"markdown": true,
		"plain":    true,
	}
	if !validFormats[config.Output] {
		return fmt.Errorf("invalid output format: %s (must be one of: stdout, json, html, markdown, plain)", config.Output)
	}

	// Validate severity
	if config.Severity != "" {
		validSeverities := map[string]bool{
			"low":      true,
			"medium":   true,
			"high":     true,
			"critical": true,
		}
		if !validSeverities[strings.ToLower(config.Severity)] {
			return fmt.Errorf("invalid severity: %s (must be one of: low, medium, high, critical)", config.Severity)
		}
	}

	// Validate output file if specified
	if config.OutputFile != "" && config.Output == "stdout" {
		return fmt.Errorf("--output-file cannot be used with --output stdout")
	}

	return nil
}

