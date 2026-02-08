package guard

import (
	"context"
	"testing"
)

func TestGuardInput_Fields(t *testing.T) {
	input := GuardInput{
		Content:  "test content",
		Role:     "input",
		Metadata: map[string]any{"key": "value"},
	}

	if input.Content != "test content" {
		t.Errorf("Content = %q, want %q", input.Content, "test content")
	}
	if input.Role != "input" {
		t.Errorf("Role = %q, want %q", input.Role, "input")
	}
	if input.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %v, want %q", input.Metadata["key"], "value")
	}
}

func TestGuardInput_EmptyFields(t *testing.T) {
	input := GuardInput{}

	if input.Content != "" {
		t.Errorf("Content = %q, want empty", input.Content)
	}
	if input.Role != "" {
		t.Errorf("Role = %q, want empty", input.Role)
	}
	if input.Metadata != nil {
		t.Errorf("Metadata = %v, want nil", input.Metadata)
	}
}

func TestGuardResult_Fields(t *testing.T) {
	result := GuardResult{
		Allowed:   false,
		Reason:    "blocked for testing",
		Modified:  "sanitized content",
		GuardName: "test_guard",
	}

	if result.Allowed {
		t.Error("Allowed = true, want false")
	}
	if result.Reason != "blocked for testing" {
		t.Errorf("Reason = %q, want %q", result.Reason, "blocked for testing")
	}
	if result.Modified != "sanitized content" {
		t.Errorf("Modified = %q, want %q", result.Modified, "sanitized content")
	}
	if result.GuardName != "test_guard" {
		t.Errorf("GuardName = %q, want %q", result.GuardName, "test_guard")
	}
}

func TestGuardResult_Defaults(t *testing.T) {
	result := GuardResult{}

	if result.Allowed {
		t.Error("zero-value Allowed should be false")
	}
	if result.Reason != "" {
		t.Errorf("Reason = %q, want empty", result.Reason)
	}
	if result.Modified != "" {
		t.Errorf("Modified = %q, want empty", result.Modified)
	}
	if result.GuardName != "" {
		t.Errorf("GuardName = %q, want empty", result.GuardName)
	}
}

// Verify Guard interface is satisfied by built-in types.
func TestGuardInterface_Compliance(t *testing.T) {
	tests := []struct {
		name  string
		guard Guard
	}{
		{"ContentFilter", NewContentFilter()},
		{"Spotlighting", NewSpotlighting("")},
		{"PromptInjectionDetector", NewPromptInjectionDetector()},
		{"PIIRedactor", NewPIIRedactor()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.guard.Name() == "" {
				t.Error("Name() returned empty string")
			}
			result, err := tt.guard.Validate(context.Background(), GuardInput{Content: "test"})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			// Just verify it returns a valid result (no panic).
			_ = result.Allowed
		})
	}
}

func TestGuardInput_MetadataNilSafe(t *testing.T) {
	// Verify that a guard can handle nil Metadata without panicking.
	f := NewContentFilter(WithKeywords("blocked"))

	result, err := f.Validate(context.Background(), GuardInput{
		Content:  "this is blocked",
		Metadata: nil,
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.Allowed {
		t.Error("should block despite nil metadata")
	}
}
