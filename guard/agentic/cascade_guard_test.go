package agentic

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCascadeGuard_Name(t *testing.T) {
	g := NewCascadeGuard()
	assert.Equal(t, "cascade_guard", g.Name())
}

func TestCascadeGuard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    []CascadeOption
		input   guard.GuardInput
		allowed bool
		reason  string
	}{
		{
			name:    "no chain_id allows",
			input:   guard.GuardInput{Content: "hello", Metadata: map[string]any{}},
			allowed: true,
		},
		{
			name: "depth within limit allows",
			opts: []CascadeOption{WithMaxRecursionDepth(5)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"chain_id": "chain-1",
					"depth":    3,
				},
			},
			allowed: true,
		},
		{
			name: "depth exceeding limit blocks",
			opts: []CascadeOption{WithMaxRecursionDepth(3)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"chain_id": "chain-1",
					"depth":    4,
				},
			},
			allowed: false,
			reason:  "recursion depth 4 exceeds maximum of 3",
		},
		{
			name: "token budget exceeded blocks",
			opts: []CascadeOption{WithMaxTokenBudget(1000)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"chain_id":    "chain-2",
					"tokens_used": int64(1500),
				},
			},
			allowed: false,
			reason:  "token usage 1500 exceeds budget of 1000",
		},
		{
			name: "token budget within limit allows",
			opts: []CascadeOption{WithMaxTokenBudget(2000)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"chain_id":    "chain-3",
					"tokens_used": int64(500),
				},
			},
			allowed: true,
		},
		{
			name: "float64 depth from JSON works",
			opts: []CascadeOption{WithMaxRecursionDepth(3)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"chain_id": "chain-4",
					"depth":    float64(4),
				},
			},
			allowed: false,
			reason:  "recursion depth 4 exceeds maximum of 3",
		},
		{
			name: "explicit iteration exceeding limit blocks",
			opts: []CascadeOption{WithMaxIterations(5)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"chain_id":  "chain-5",
					"iteration": 6,
				},
			},
			allowed: false,
			reason:  "iteration count 6 exceeds maximum of 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewCascadeGuard(tt.opts...)
			result, err := g.Validate(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, result.Allowed)
			if tt.reason != "" {
				assert.Contains(t, result.Reason, tt.reason)
			}
		})
	}
}

func TestCascadeGuard_AutoIncrement(t *testing.T) {
	g := NewCascadeGuard(WithMaxIterations(3))
	input := guard.GuardInput{
		Content:  "",
		Metadata: map[string]any{"chain_id": "auto-chain"},
	}

	// First 3 calls should pass (iterations 1, 2, 3).
	for i := 1; i <= 3; i++ {
		result, err := g.Validate(context.Background(), input)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "iteration %d should be allowed", i)
	}

	// Fourth call should be blocked (iteration 4 > max 3).
	result, err := g.Validate(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "iteration count")
}

func TestCascadeGuard_CircuitBreaker(t *testing.T) {
	g := NewCascadeGuard(
		WithMaxRecursionDepth(2),
		WithFailureThreshold(2),
	)
	chainID := "cb-chain"
	input := guard.GuardInput{
		Content: "",
		Metadata: map[string]any{
			"chain_id": chainID,
			"depth":    5, // exceeds max of 2
		},
	}

	// Two failures should trip the circuit breaker.
	for i := 0; i < 2; i++ {
		result, err := g.Validate(context.Background(), input)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
	}

	// Now even a valid request should be blocked by the circuit breaker.
	validInput := guard.GuardInput{
		Content: "",
		Metadata: map[string]any{
			"chain_id": chainID,
			"depth":    1,
		},
	}
	result, err := g.Validate(context.Background(), validInput)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "circuit breaker open")

	// Reset should clear the circuit breaker.
	g.ResetChain(chainID)
	result, err = g.Validate(context.Background(), validInput)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestCascadeGuard_ContextCancellation(t *testing.T) {
	g := NewCascadeGuard()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.Validate(ctx, guard.GuardInput{
		Content:  "",
		Metadata: map[string]any{"chain_id": "x"},
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestCascadeGuard_CompileTimeCheck(t *testing.T) {
	var _ guard.Guard = (*CascadeGuard)(nil)
}
