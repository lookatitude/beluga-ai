package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPreemptiveGeneration(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyUseIfSimilar)
	assert.NotNil(t, pg)
	assert.True(t, pg.enabled)
	assert.Equal(t, ResponseStrategyUseIfSimilar, pg.responseStrategy)
}

func TestPreemptiveGeneration_SetInterimHandler(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyDiscard)

	called := false
	handler := func(transcript string) {
		called = true
	}

	pg.SetInterimHandler(handler)

	ctx := context.Background()
	pg.HandleInterim(ctx, "test")

	assert.True(t, called)
}

func TestPreemptiveGeneration_SetFinalHandler(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyDiscard)

	called := false
	handler := func(transcript string) {
		called = true
	}

	pg.SetFinalHandler(handler)

	ctx := context.Background()
	pg.HandleFinal(ctx, "test")

	assert.True(t, called)
}

func TestPreemptiveGeneration_HandleInterim_Disabled(t *testing.T) {
	pg := NewPreemptiveGeneration(false, ResponseStrategyDiscard)

	called := false
	handler := func(transcript string) {
		called = true
	}

	pg.SetInterimHandler(handler)

	ctx := context.Background()
	pg.HandleInterim(ctx, "test")

	assert.False(t, called)
}

func TestPreemptiveGeneration_HandleInterim_NilHandler(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyDiscard)

	ctx := context.Background()
	// Should not panic
	pg.HandleInterim(ctx, "test")
}

func TestPreemptiveGeneration_HandleFinal_ResponseStrategyDiscard(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyDiscard)

	called := false
	handler := func(transcript string) {
		called = true
	}

	pg.SetFinalHandler(handler)

	ctx := context.Background()
	pg.HandleFinal(ctx, "test")

	assert.True(t, called)
}

func TestPreemptiveGeneration_HandleFinal_ResponseStrategyUseIfSimilar(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyUseIfSimilar)

	called := false
	handler := func(transcript string) {
		called = true
	}

	pg.SetFinalHandler(handler)

	ctx := context.Background()
	pg.HandleFinal(ctx, "test")

	assert.True(t, called)
}

func TestPreemptiveGeneration_HandleFinal_ResponseStrategyAlwaysUse(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyAlwaysUse)

	called := false
	handler := func(transcript string) {
		called = true
	}

	pg.SetFinalHandler(handler)

	ctx := context.Background()
	pg.HandleFinal(ctx, "test")

	assert.True(t, called)
}

func TestPreemptiveGeneration_HandleFinal_NilHandler(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyDiscard)

	ctx := context.Background()
	// Should not panic
	pg.HandleFinal(ctx, "test")
}

func TestPreemptiveGeneration_GetResponseStrategy(t *testing.T) {
	pg := NewPreemptiveGeneration(true, ResponseStrategyAlwaysUse)
	assert.Equal(t, ResponseStrategyAlwaysUse, pg.GetResponseStrategy())

	pg = NewPreemptiveGeneration(true, ResponseStrategyUseIfSimilar)
	assert.Equal(t, ResponseStrategyUseIfSimilar, pg.GetResponseStrategy())

	pg = NewPreemptiveGeneration(true, ResponseStrategyDiscard)
	assert.Equal(t, ResponseStrategyDiscard, pg.GetResponseStrategy())
}

