package agentic

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/guard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolMisuseGuard_Name(t *testing.T) {
	g := NewToolMisuseGuard()
	assert.Equal(t, "tool_misuse_guard", g.Name())
}

func TestToolMisuseGuard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ToolMisuseOption
		input   guard.GuardInput
		allowed bool
		reason  string
	}{
		{
			name:    "no tool metadata allows",
			input:   guard.GuardInput{Content: "hello", Metadata: map[string]any{}},
			allowed: true,
		},
		{
			name: "allowed tool passes",
			opts: []ToolMisuseOption{WithAllowedTools("search", "read")},
			input: guard.GuardInput{
				Content:  `{"query": "test"}`,
				Metadata: map[string]any{"tool_name": "search"},
			},
			allowed: true,
		},
		{
			name: "disallowed tool blocks",
			opts: []ToolMisuseOption{WithAllowedTools("search")},
			input: guard.GuardInput{
				Content:  `{"cmd": "rm -rf /"}`,
				Metadata: map[string]any{"tool_name": "exec"},
			},
			allowed: false,
			reason:  `tool "exec" is not in the allowed set`,
		},
		{
			name: "schema required fields missing",
			opts: []ToolMisuseOption{
				WithToolSchema("search", ToolSchema{RequiredFields: []string{"query"}}),
			},
			input: guard.GuardInput{
				Content:  `{"limit": 10}`,
				Metadata: map[string]any{"tool_name": "search"},
			},
			allowed: false,
			reason:  "tool arguments missing required fields: [query]",
		},
		{
			name: "schema required fields present",
			opts: []ToolMisuseOption{
				WithToolSchema("search", ToolSchema{RequiredFields: []string{"query"}}),
			},
			input: guard.GuardInput{
				Content:  `{"query": "hello"}`,
				Metadata: map[string]any{"tool_name": "search"},
			},
			allowed: true,
		},
		{
			name: "schema forbidden fields detected",
			opts: []ToolMisuseOption{
				WithToolSchema("api", ToolSchema{ForbiddenFields: []string{"__proto__", "constructor"}}),
			},
			input: guard.GuardInput{
				Content:  `{"data": "ok", "__proto__": "bad"}`,
				Metadata: map[string]any{"tool_name": "api"},
			},
			allowed: false,
			reason:  "tool arguments contain forbidden fields: [__proto__]",
		},
		{
			name: "schema max field count exceeded",
			opts: []ToolMisuseOption{
				WithToolSchema("api", ToolSchema{MaxFieldCount: 2}),
			},
			input: guard.GuardInput{
				Content:  `{"a": 1, "b": 2, "c": 3}`,
				Metadata: map[string]any{"tool_name": "api"},
			},
			allowed: false,
			reason:  "tool arguments have 3 fields, maximum is 2",
		},
		{
			name: "invalid JSON blocks",
			opts: []ToolMisuseOption{
				WithToolSchema("api", ToolSchema{RequiredFields: []string{"x"}}),
			},
			input: guard.GuardInput{
				Content:  `not json`,
				Metadata: map[string]any{"tool_name": "api"},
			},
			allowed: false,
			reason:  "tool arguments are not valid JSON",
		},
		{
			name: "empty content with required fields blocks",
			opts: []ToolMisuseOption{
				WithToolSchema("api", ToolSchema{RequiredFields: []string{"x"}}),
			},
			input: guard.GuardInput{
				Content:  "",
				Metadata: map[string]any{"tool_name": "api"},
			},
			allowed: false,
			reason:  "tool arguments missing required fields: [x]",
		},
		{
			name: "unknown tool with no schema passes",
			input: guard.GuardInput{
				Content:  `{"anything": "goes"}`,
				Metadata: map[string]any{"tool_name": "unknown"},
			},
			allowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewToolMisuseGuard(tt.opts...)
			result, err := g.Validate(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, result.Allowed)
			if tt.reason != "" {
				assert.Contains(t, result.Reason, tt.reason)
			}
		})
	}
}

func TestToolMisuseGuard_RateLimit(t *testing.T) {
	g := NewToolMisuseGuard(
		WithToolRateLimit(2, 1*time.Second),
	)
	input := guard.GuardInput{
		Content:  `{"q": "x"}`,
		Metadata: map[string]any{"tool_name": "search"},
	}

	// First two calls should pass.
	for i := 0; i < 2; i++ {
		result, err := g.Validate(context.Background(), input)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "call %d should be allowed", i+1)
	}

	// Third call should be rate limited.
	result, err := g.Validate(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "exceeded rate limit")
}

func TestToolMisuseGuard_ContextCancellation(t *testing.T) {
	g := NewToolMisuseGuard()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.Validate(ctx, guard.GuardInput{
		Content:  "test",
		Metadata: map[string]any{"tool_name": "x"},
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestToolMisuseGuard_CompileTimeCheck(t *testing.T) {
	var _ guard.Guard = (*ToolMisuseGuard)(nil)
}
