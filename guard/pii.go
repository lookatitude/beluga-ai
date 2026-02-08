package guard

import (
	"context"
	"regexp"
)

// PIIPattern defines a named PII detection pattern with its replacement
// placeholder. For example, an email pattern would use the placeholder
// "[EMAIL]".
type PIIPattern struct {
	// Name identifies the PII type, e.g. "email" or "phone".
	Name string

	// Pattern is the compiled regexp that matches the PII in text.
	Pattern *regexp.Regexp

	// Placeholder is the replacement string, e.g. "[EMAIL]".
	Placeholder string
}

// DefaultPIIPatterns contains the built-in PII detection patterns for common
// data types: email addresses, US phone numbers, US Social Security numbers,
// credit card numbers, and IPv4 addresses.
var DefaultPIIPatterns = []PIIPattern{
	{
		Name:        "email",
		Pattern:     regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		Placeholder: "[EMAIL]",
	},
	{
		Name:        "credit_card",
		Pattern:     regexp.MustCompile(`\b(?:[0-9]{4}[-\s]?){3}[0-9]{4}\b`),
		Placeholder: "[CREDIT_CARD]",
	},
	{
		Name:        "ssn",
		Pattern:     regexp.MustCompile(`\b[0-9]{3}-[0-9]{2}-[0-9]{4}\b`),
		Placeholder: "[SSN]",
	},
	{
		Name:        "phone",
		Pattern:     regexp.MustCompile(`(\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s][0-9]{3}[-.\s]?[0-9]{4}`),
		Placeholder: "[PHONE]",
	},
	{
		Name:        "ip_address",
		Pattern:     regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`),
		Placeholder: "[IP_ADDRESS]",
	},
}

// PIIRedactor is a Guard that detects and redacts personally identifiable
// information from content. It replaces matched patterns with configurable
// placeholders and returns the sanitized content as a modified result.
type PIIRedactor struct {
	patterns []PIIPattern
}

// NewPIIRedactor creates a PIIRedactor with the given patterns. If no
// patterns are provided, the result is a no-op guard that allows all content.
//
// Usage:
//
//	redactor := guard.NewPIIRedactor(guard.DefaultPIIPatterns...)
func NewPIIRedactor(patterns ...PIIPattern) *PIIRedactor {
	return &PIIRedactor{
		patterns: patterns,
	}
}

// Name returns "pii_redactor".
func (r *PIIRedactor) Name() string {
	return "pii_redactor"
}

// Validate scans the input content for PII and returns a result with the
// sanitized content in Modified. The result is always Allowed because
// redaction makes the content safe to pass through.
func (r *PIIRedactor) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	modified := input.Content
	redacted := false

	for _, p := range r.patterns {
		if p.Pattern.MatchString(modified) {
			modified = p.Pattern.ReplaceAllString(modified, p.Placeholder)
			redacted = true
		}
	}

	result := GuardResult{Allowed: true}
	if redacted {
		result.Modified = modified
		result.Reason = "PII redacted"
		result.GuardName = r.Name()
	}
	return result, nil
}

func init() {
	Register("pii_redactor", func(cfg map[string]any) (Guard, error) {
		return NewPIIRedactor(DefaultPIIPatterns...), nil
	})
}
