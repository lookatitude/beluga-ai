package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFinalHandler(t *testing.T) {
	handler := NewFinalHandler(nil, false, false)
	assert.NotNil(t, handler)
	assert.Equal(t, 0, handler.finalCount)
}

func TestFinalHandler_Handle_AlwaysUse(t *testing.T) {
	var calledWith string

	handler := NewFinalHandler(func(transcript string) {
		calledWith = transcript
	}, false, true)

	ctx := context.Background()
	handler.Handle(ctx, "final transcript", "preemptive response")

	assert.Equal(t, "preemptive response", calledWith)
	assert.Equal(t, 1, handler.finalCount)
}

func TestFinalHandler_Handle_UseIfSimilar(t *testing.T) {
	var calledWith string

	handler := NewFinalHandler(func(transcript string) {
		calledWith = transcript
	}, true, false)

	ctx := context.Background()
	// Use similar transcript (should use preemptive)
	handler.Handle(ctx, "Hello world", "Hello world")

	// Should use preemptive if similar
	assert.Equal(t, "Hello world", calledWith)
}

func TestFinalHandler_Handle_UseIfSimilar_NotSimilar(t *testing.T) {
	var calledWith string

	handler := NewFinalHandler(func(transcript string) {
		calledWith = transcript
	}, true, false)

	ctx := context.Background()
	// Use different transcript (should use final)
	handler.Handle(ctx, "Hello world", "Goodbye")

	// Should use final transcript if not similar
	assert.Equal(t, "Hello world", calledWith)
}

func TestFinalHandler_Handle_NoPreemptive(t *testing.T) {
	var calledWith string

	handler := NewFinalHandler(func(transcript string) {
		calledWith = transcript
	}, false, false)

	ctx := context.Background()
	handler.Handle(ctx, "final transcript", "")

	assert.Equal(t, "final transcript", calledWith)
}

func TestFinalHandler_Handle_NilHandler(t *testing.T) {
	handler := NewFinalHandler(nil, false, false)

	ctx := context.Background()
	// Should not panic
	handler.Handle(ctx, "transcript", "preemptive")
	assert.Equal(t, 1, handler.finalCount)
}

func TestFinalHandler_FinalCount(t *testing.T) {
	handler := NewFinalHandler(nil, false, false)
	assert.Equal(t, 0, handler.finalCount)

	ctx := context.Background()
	handler.Handle(ctx, "transcript1", "")
	assert.Equal(t, 1, handler.finalCount)

	handler.Handle(ctx, "transcript2", "")
	assert.Equal(t, 2, handler.finalCount)
}

func TestFinalHandler_LastFinal(t *testing.T) {
	handler := NewFinalHandler(nil, false, false)

	ctx := context.Background()
	handler.Handle(ctx, "first", "")
	assert.Equal(t, "first", handler.lastFinal)

	handler.Handle(ctx, "second", "")
	assert.Equal(t, "second", handler.lastFinal)
}

func TestFinalHandler_GetLastFinal(t *testing.T) {
	handler := NewFinalHandler(nil, false, false)

	ctx := context.Background()
	handler.Handle(ctx, "test transcript", "")
	
	lastFinal := handler.GetLastFinal()
	assert.Equal(t, "test transcript", lastFinal)
}

func TestFinalHandler_GetFinalCount(t *testing.T) {
	handler := NewFinalHandler(nil, false, false)
	assert.Equal(t, 0, handler.GetFinalCount())

	ctx := context.Background()
	handler.Handle(ctx, "transcript1", "")
	assert.Equal(t, 1, handler.GetFinalCount())

	handler.Handle(ctx, "transcript2", "")
	assert.Equal(t, 2, handler.GetFinalCount())
}

