package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamingIncremental(t *testing.T) {
	processor := func(ctx context.Context, chunk []byte) error {
		return nil
	}

	si := NewStreamingIncremental(true, processor)
	assert.NotNil(t, si)
	assert.True(t, si.IsEnabled())

	si = NewStreamingIncremental(false, processor)
	assert.NotNil(t, si)
	assert.False(t, si.IsEnabled())
}

func TestStreamingIncremental_ProcessChunk_Enabled(t *testing.T) {
	called := false
	var receivedChunk []byte
	processor := func(ctx context.Context, chunk []byte) error {
		called = true
		receivedChunk = chunk
		return nil
	}

	si := NewStreamingIncremental(true, processor)

	ctx := context.Background()
	chunk := []byte{1, 2, 3, 4, 5}
	err := si.ProcessChunk(ctx, chunk)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, chunk, receivedChunk)
}

func TestStreamingIncremental_ProcessChunk_Disabled(t *testing.T) {
	processor := func(ctx context.Context, chunk []byte) error {
		t.Fatal("Processor should not be called when disabled")
		return nil
	}

	si := NewStreamingIncremental(false, processor)

	ctx := context.Background()
	chunk := []byte{1, 2, 3, 4, 5}
	err := si.ProcessChunk(ctx, chunk)
	require.NoError(t, err)
}

func TestStreamingIncremental_ProcessChunk_NoProcessor(t *testing.T) {
	si := NewStreamingIncremental(true, nil)

	ctx := context.Background()
	chunk := []byte{1, 2, 3, 4, 5}
	err := si.ProcessChunk(ctx, chunk)
	require.NoError(t, err)
}

func TestStreamingIncremental_ProcessChunk_Error(t *testing.T) {
	expectedErr := errors.New("processing error")
	processor := func(ctx context.Context, chunk []byte) error {
		return expectedErr
	}

	si := NewStreamingIncremental(true, processor)

	ctx := context.Background()
	chunk := []byte{1, 2, 3, 4, 5}
	err := si.ProcessChunk(ctx, chunk)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestStreamingIncremental_AddResult(t *testing.T) {
	si := NewStreamingIncremental(true, nil)

	result1 := "result1"
	result2 := 42

	si.AddResult(result1)
	si.AddResult(result2)

	results := si.GetResults()
	assert.Len(t, results, 2)
	assert.Equal(t, result1, results[0])
	assert.Equal(t, result2, results[1])
}

func TestStreamingIncremental_GetResults(t *testing.T) {
	si := NewStreamingIncremental(true, nil)

	// Initially empty
	results := si.GetResults()
	assert.Empty(t, results)

	// Add results
	si.AddResult("result1")
	si.AddResult("result2")

	results = si.GetResults()
	assert.Len(t, results, 2)

	// Results should be a copy
	results[0] = "modified"
	results2 := si.GetResults()
	assert.Equal(t, "result1", results2[0])
}

func TestStreamingIncremental_ClearResults(t *testing.T) {
	si := NewStreamingIncremental(true, nil)

	si.AddResult("result1")
	si.AddResult("result2")

	results := si.GetResults()
	assert.Len(t, results, 2)

	si.ClearResults()

	results = si.GetResults()
	assert.Empty(t, results)
}

func TestStreamingIncremental_IsEnabled(t *testing.T) {
	si := NewStreamingIncremental(true, nil)
	assert.True(t, si.IsEnabled())

	si = NewStreamingIncremental(false, nil)
	assert.False(t, si.IsEnabled())
}

func TestStreamingIncremental_ConcurrentAccess(t *testing.T) {
	si := NewStreamingIncremental(true, nil)

	// Concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			si.AddResult(idx)
			si.GetResults()
			si.IsEnabled()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	results := si.GetResults()
	assert.Len(t, results, 10)
}
