package guard

import (
	"context"
	"strings"
	"testing"
)

func TestPIIRedactor_Name(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)
	if got := r.Name(); got != "pii_redactor" {
		t.Errorf("Name() = %q, want %q", got, "pii_redactor")
	}
}

func TestPIIRedactor_Email(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple_email",
			input: "Contact me at john@example.com for details.",
			want:  "Contact me at [EMAIL] for details.",
		},
		{
			name:  "email_with_plus",
			input: "Send to user+tag@domain.org",
			want:  "Send to [EMAIL]",
		},
		{
			name:  "multiple_emails",
			input: "Email a@b.com or c@d.com",
			want:  "Email [EMAIL] or [EMAIL]",
		},
		{
			name:  "no_email",
			input: "No email here.",
			want:  "No email here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Validate(context.Background(), GuardInput{Content: tt.input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !result.Allowed {
				t.Error("PII redactor should always allow")
			}
			if tt.input != tt.want {
				// Expect modification.
				if result.Modified != tt.want {
					t.Errorf("Modified = %q, want %q", result.Modified, tt.want)
				}
			}
		})
	}
}

func TestPIIRedactor_Phone(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "us_phone_dashes",
			input: "Call me at 555-123-4567.",
			want:  "Call me at [PHONE].",
		},
		{
			name:  "us_phone_parens",
			input: "Phone: (555) 123-4567",
			want:  "Phone: [PHONE]",
		},
		{
			name:  "us_phone_with_country",
			input: "Dial +1-555-123-4567",
			want:  "Dial [PHONE]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Validate(context.Background(), GuardInput{Content: tt.input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if result.Modified != tt.want {
				t.Errorf("Modified = %q, want %q", result.Modified, tt.want)
			}
		})
	}
}

func TestPIIRedactor_SSN(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	result, err := r.Validate(context.Background(), GuardInput{
		Content: "SSN: 123-45-6789",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !strings.Contains(result.Modified, "[SSN]") {
		t.Errorf("Modified = %q, want to contain [SSN]", result.Modified)
	}
}

func TestPIIRedactor_CreditCard(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	tests := []struct {
		name  string
		input string
	}{
		{"spaces", "Card: 4111 1111 1111 1111"},
		{"dashes", "Card: 4111-1111-1111-1111"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Validate(context.Background(), GuardInput{Content: tt.input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !strings.Contains(result.Modified, "[CREDIT_CARD]") {
				t.Errorf("Modified = %q, want to contain [CREDIT_CARD]", result.Modified)
			}
		})
	}
}

func TestPIIRedactor_IPAddress(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	result, err := r.Validate(context.Background(), GuardInput{
		Content: "Server at 192.168.1.100",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !strings.Contains(result.Modified, "[IP_ADDRESS]") {
		t.Errorf("Modified = %q, want to contain [IP_ADDRESS]", result.Modified)
	}
}

func TestPIIRedactor_MultiplePIITypes(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	input := "Email: test@example.com, Phone: 555-123-4567, SSN: 123-45-6789"
	result, err := r.Validate(context.Background(), GuardInput{Content: input})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !strings.Contains(result.Modified, "[EMAIL]") {
		t.Error("should contain [EMAIL]")
	}
	if !strings.Contains(result.Modified, "[PHONE]") {
		t.Error("should contain [PHONE]")
	}
	if !strings.Contains(result.Modified, "[SSN]") {
		t.Error("should contain [SSN]")
	}

	if result.Reason != "PII redacted" {
		t.Errorf("Reason = %q, want %q", result.Reason, "PII redacted")
	}
	if result.GuardName != "pii_redactor" {
		t.Errorf("GuardName = %q, want %q", result.GuardName, "pii_redactor")
	}
}

func TestPIIRedactor_NoPII(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	result, err := r.Validate(context.Background(), GuardInput{
		Content: "This is a safe message with no PII.",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true")
	}
	if result.Modified != "" {
		t.Errorf("Modified = %q, want empty (no redaction)", result.Modified)
	}
	if result.Reason != "" {
		t.Errorf("Reason = %q, want empty", result.Reason)
	}
}

func TestPIIRedactor_EmptyContent(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	result, err := r.Validate(context.Background(), GuardInput{Content: ""})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("empty content should be allowed")
	}
	if result.Modified != "" {
		t.Errorf("Modified = %q, want empty", result.Modified)
	}
}

func TestPIIRedactor_NoPatterns(t *testing.T) {
	r := NewPIIRedactor() // No patterns.

	result, err := r.Validate(context.Background(), GuardInput{
		Content: "test@example.com 555-123-4567",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true")
	}
	if result.Modified != "" {
		t.Errorf("no patterns should not modify: Modified = %q", result.Modified)
	}
}

func TestPIIRedactor_CustomPattern(t *testing.T) {
	r := NewPIIRedactor(PIIPattern{
		Name:        "custom_id",
		Pattern:     DefaultPIIPatterns[0].Pattern, // reuse email pattern for simplicity
		Placeholder: "[CUSTOM]",
	})

	result, err := r.Validate(context.Background(), GuardInput{
		Content: "Contact: foo@bar.com",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !strings.Contains(result.Modified, "[CUSTOM]") {
		t.Errorf("Modified = %q, want to contain [CUSTOM]", result.Modified)
	}
}

func TestPIIRedactor_AlwaysAllows(t *testing.T) {
	r := NewPIIRedactor(DefaultPIIPatterns...)

	// Even with PII present, Allowed should be true (content is redacted, not blocked).
	result, err := r.Validate(context.Background(), GuardInput{
		Content: "SSN: 111-22-3333",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("PIIRedactor should always allow (redact, not block)")
	}
}
