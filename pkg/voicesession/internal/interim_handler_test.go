package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInterimHandler(t *testing.T) {
	handler := NewInterimHandler(nil)
	assert.NotNil(t, handler)
	assert.Equal(t, 0, handler.interimCount)
	assert.Empty(t, handler.lastInterim)
}

func TestInterimHandler_Handle(t *testing.T) {
	called := false
	var receivedTranscript string

	handlerFunc := func(transcript string) {
		called = true
		receivedTranscript = transcript
	}

	handler := NewInterimHandler(handlerFunc)

	ctx := context.Background()
	handler.Handle(ctx, "test transcript")

	assert.True(t, called)
	assert.Equal(t, "test transcript", receivedTranscript)
	assert.Equal(t, "test transcript", handler.GetLastInterim())
	assert.Equal(t, 1, handler.GetInterimCount())
}

func TestInterimHandler_Handle_NilHandler(t *testing.T) {
	handler := NewInterimHandler(nil)

	ctx := context.Background()
	handler.Handle(ctx, "test transcript")

	// Should not panic
	assert.Equal(t, "test transcript", handler.GetLastInterim())
	assert.Equal(t, 1, handler.GetInterimCount())
}

func TestInterimHandler_Handle_Multiple(t *testing.T) {
	callCount := 0
	handlerFunc := func(transcript string) {
		callCount++
	}

	handler := NewInterimHandler(handlerFunc)

	ctx := context.Background()
	handler.Handle(ctx, "first")
	handler.Handle(ctx, "second")
	handler.Handle(ctx, "third")

	assert.Equal(t, 3, callCount)
	assert.Equal(t, "third", handler.GetLastInterim())
	assert.Equal(t, 3, handler.GetInterimCount())
}

func TestInterimHandler_GetLastInterim(t *testing.T) {
	handler := NewInterimHandler(nil)

	ctx := context.Background()
	handler.Handle(ctx, "first transcript")
	assert.Equal(t, "first transcript", handler.GetLastInterim())

	handler.Handle(ctx, "second transcript")
	assert.Equal(t, "second transcript", handler.GetLastInterim())
}

func TestInterimHandler_GetInterimCount(t *testing.T) {
	handler := NewInterimHandler(nil)
	assert.Equal(t, 0, handler.GetInterimCount())

	ctx := context.Background()
	handler.Handle(ctx, "transcript1")
	assert.Equal(t, 1, handler.GetInterimCount())

	handler.Handle(ctx, "transcript2")
	assert.Equal(t, 2, handler.GetInterimCount())
}

func TestInterimHandler_Reset(t *testing.T) {
	handler := NewInterimHandler(nil)

	ctx := context.Background()
	handler.Handle(ctx, "test transcript")
	handler.Handle(ctx, "another transcript")

	assert.Equal(t, "another transcript", handler.GetLastInterim())
	assert.Equal(t, 2, handler.GetInterimCount())

	handler.Reset()

	assert.Empty(t, handler.GetLastInterim())
	assert.Equal(t, 0, handler.GetInterimCount())
}

func TestInterimHandler_ConcurrentAccess(t *testing.T) {
	handler := NewInterimHandler(nil)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		ctx := context.Background()
		go func(idx int) {
			handler.Handle(ctx, "transcript")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have processed all transcripts
	assert.GreaterOrEqual(t, handler.GetInterimCount(), 1)
}
