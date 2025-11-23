package internal

import (
	"context"
	"sync"
)

// StreamingIncremental manages streaming incremental processing
type StreamingIncremental struct {
	mu             sync.RWMutex
	enabled        bool
	chunkProcessor func(ctx context.Context, chunk []byte) error
	results        []interface{}
}

// NewStreamingIncremental creates a new streaming incremental processor
func NewStreamingIncremental(enabled bool, chunkProcessor func(ctx context.Context, chunk []byte) error) *StreamingIncremental {
	return &StreamingIncremental{
		enabled:        enabled,
		chunkProcessor: chunkProcessor,
		results:        make([]interface{}, 0),
	}
}

// ProcessChunk processes a single chunk incrementally
func (si *StreamingIncremental) ProcessChunk(ctx context.Context, chunk []byte) error {
	if !si.enabled {
		return nil
	}

	si.mu.RLock()
	processor := si.chunkProcessor
	si.mu.RUnlock()

	if processor != nil {
		return processor(ctx, chunk)
	}

	return nil
}

// AddResult adds a processing result
func (si *StreamingIncremental) AddResult(result interface{}) {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.results = append(si.results, result)
}

// GetResults returns all accumulated results
func (si *StreamingIncremental) GetResults() []interface{} {
	si.mu.RLock()
	defer si.mu.RUnlock()

	result := make([]interface{}, len(si.results))
	copy(result, si.results)
	return result
}

// ClearResults clears all accumulated results
func (si *StreamingIncremental) ClearResults() {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.results = si.results[:0]
}

// IsEnabled returns whether streaming incremental processing is enabled
func (si *StreamingIncremental) IsEnabled() bool {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return si.enabled
}
