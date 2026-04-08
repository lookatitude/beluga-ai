package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComposeHooks_Empty(t *testing.T) {
	h := ComposeHooks()
	// All fields should be nil.
	assert.Nil(t, h.OnPoisoningDetected)
	assert.Nil(t, h.OnSignatureInvalid)
	assert.Nil(t, h.OnCircuitTripped)
}

func TestComposeHooks_Single(t *testing.T) {
	var called bool
	h := ComposeHooks(Hooks{
		OnPoisoningDetected: func(_ context.Context, _ string, _ []AnomalyResult) {
			called = true
		},
	})

	h.OnPoisoningDetected(context.Background(), "test", nil)
	assert.True(t, called)
}

func TestComposeHooks_Multiple(t *testing.T) {
	var order []int

	h1 := Hooks{
		OnPoisoningDetected: func(_ context.Context, _ string, _ []AnomalyResult) {
			order = append(order, 1)
		},
		OnSignatureInvalid: func(_ context.Context, _ string) {
			order = append(order, 10)
		},
	}

	h2 := Hooks{
		OnPoisoningDetected: func(_ context.Context, _ string, _ []AnomalyResult) {
			order = append(order, 2)
		},
		OnCircuitTripped: func(_ context.Context, _, _ string) {
			order = append(order, 20)
		},
	}

	h3 := Hooks{
		OnPoisoningDetected: func(_ context.Context, _ string, _ []AnomalyResult) {
			order = append(order, 3)
		},
	}

	composed := ComposeHooks(h1, h2, h3)

	// OnPoisoningDetected should call all three in order.
	composed.OnPoisoningDetected(context.Background(), "test", nil)
	assert.Equal(t, []int{1, 2, 3}, order)

	// OnSignatureInvalid should only call h1's.
	order = nil
	composed.OnSignatureInvalid(context.Background(), "reason")
	assert.Equal(t, []int{10}, order)

	// OnCircuitTripped should only call h2's.
	order = nil
	composed.OnCircuitTripped(context.Background(), "w", "r")
	assert.Equal(t, []int{20}, order)
}

func TestComposeHooks_NilHooksSkipped(t *testing.T) {
	var called bool
	h1 := Hooks{} // all nil
	h2 := Hooks{
		OnPoisoningDetected: func(_ context.Context, _ string, _ []AnomalyResult) {
			called = true
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnPoisoningDetected(context.Background(), "test", nil)
	assert.True(t, called)
}
