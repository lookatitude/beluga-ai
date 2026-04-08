package memory

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterAgentCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewInterAgentCircuitBreaker()
	assert.Equal(t, 3, cb.failureThreshold)
	assert.Equal(t, 5*time.Minute, cb.resetTimeout)
}

func TestInterAgentCircuitBreaker_AllowWhenClosed(t *testing.T) {
	cb := NewInterAgentCircuitBreaker()
	err := cb.Allow(context.Background(), "writer", "reader")
	assert.NoError(t, err)
}

func TestInterAgentCircuitBreaker_TripsAfterThreshold(t *testing.T) {
	var tripped bool
	cb := NewInterAgentCircuitBreaker(
		WithFailureThreshold(2),
		WithResetTimeout(time.Hour),
		WithCircuitHooks(Hooks{
			OnCircuitTripped: func(_ context.Context, writer, reader string) {
				tripped = true
				assert.Equal(t, "bad-agent", writer)
				assert.Equal(t, "victim", reader)
			},
		}),
	)

	ctx := context.Background()

	// Record poisoning events.
	cb.RecordPoisoning(ctx, "bad-agent", "victim")
	assert.Equal(t, resilience.StateClosed, cb.State("bad-agent", "victim"))
	assert.False(t, tripped)

	cb.RecordPoisoning(ctx, "bad-agent", "victim")
	assert.Equal(t, resilience.StateOpen, cb.State("bad-agent", "victim"))
	assert.True(t, tripped)

	// Allow should fail.
	err := cb.Allow(ctx, "bad-agent", "victim")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit open")
}

func TestInterAgentCircuitBreaker_IndependentPairs(t *testing.T) {
	cb := NewInterAgentCircuitBreaker(
		WithFailureThreshold(1),
		WithResetTimeout(time.Hour),
	)

	ctx := context.Background()

	// Trip one pair.
	cb.RecordPoisoning(ctx, "bad", "reader1")
	assert.Equal(t, resilience.StateOpen, cb.State("bad", "reader1"))

	// Other pair unaffected.
	assert.Equal(t, resilience.StateClosed, cb.State("bad", "reader2"))
	err := cb.Allow(ctx, "bad", "reader2")
	assert.NoError(t, err)
}

func TestInterAgentCircuitBreaker_Reset(t *testing.T) {
	cb := NewInterAgentCircuitBreaker(
		WithFailureThreshold(1),
		WithResetTimeout(time.Hour),
	)

	ctx := context.Background()
	cb.RecordPoisoning(ctx, "agent", "reader")
	assert.Equal(t, resilience.StateOpen, cb.State("agent", "reader"))

	cb.Reset("agent", "reader")
	assert.Equal(t, resilience.StateClosed, cb.State("agent", "reader"))

	// Reset non-existent pair is a no-op.
	cb.Reset("nonexistent", "pair")
}

func TestInterAgentCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewInterAgentCircuitBreaker(
		WithFailureThreshold(2),
		WithResetTimeout(10*time.Millisecond),
	)

	ctx := context.Background()

	// Trip the circuit.
	cb.RecordPoisoning(ctx, "w", "r")
	cb.RecordPoisoning(ctx, "w", "r")
	assert.Equal(t, resilience.StateOpen, cb.State("w", "r"))

	// Wait for reset timeout.
	time.Sleep(15 * time.Millisecond)

	// Should be half-open now; record success.
	cb.RecordSuccess(ctx, "w", "r")
	assert.Equal(t, resilience.StateClosed, cb.State("w", "r"))
}

func TestInterAgentCircuitBreaker_ListTripped(t *testing.T) {
	cb := NewInterAgentCircuitBreaker(
		WithFailureThreshold(1),
		WithResetTimeout(time.Hour),
	)

	ctx := context.Background()
	assert.Empty(t, cb.ListTripped())

	cb.RecordPoisoning(ctx, "a", "b")
	cb.RecordPoisoning(ctx, "c", "d")

	tripped := cb.ListTripped()
	assert.Len(t, tripped, 2)
	assert.Contains(t, tripped, "a->b")
	assert.Contains(t, tripped, "c->d")
}

func TestInterAgentCircuitBreaker_StateNonExistent(t *testing.T) {
	cb := NewInterAgentCircuitBreaker()
	assert.Equal(t, resilience.StateClosed, cb.State("x", "y"))
}
