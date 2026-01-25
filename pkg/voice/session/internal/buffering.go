package internal

import (
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice"
)

// BufferingStrategy defines buffering strategies for long utterances.
type BufferingStrategy int

const (
	BufferingStrategyNone BufferingStrategy = iota
	BufferingStrategyFixed
	BufferingStrategyAdaptive
)

// Buffering manages buffering strategy for long utterances.
type Buffering struct {
	pool     *voice.BufferPool
	buffer   []byte
	strategy BufferingStrategy
	maxSize  int
	mu       sync.RWMutex
}

// NewBuffering creates a new buffering manager.
func NewBuffering(strategy BufferingStrategy, maxSize int) *Buffering {
	pool := voice.GetGlobalBufferPool()
	buffer := pool.Get(maxSize)

	return &Buffering{
		strategy: strategy,
		buffer:   buffer,
		maxSize:  maxSize,
		pool:     pool,
	}
}

// Add adds data to the buffer.
func (b *Buffering) Add(data []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.strategy {
	case BufferingStrategyNone:
		return false // Don't buffer
	case BufferingStrategyFixed:
		if len(b.buffer)+len(data) > b.maxSize {
			return false // Buffer full
		}
		b.buffer = append(b.buffer, data...)
		return true
	case BufferingStrategyAdaptive:
		// Adaptive buffering - adjust based on available space
		available := b.maxSize - len(b.buffer)
		if available < len(data) {
			// Would need to flush or resize in production
			return false
		}
		b.buffer = append(b.buffer, data...)
		return true
	default:
		return false
	}
}

// Flush returns and clears the buffer.
func (b *Buffering) Flush() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]byte, len(b.buffer))
	copy(result, b.buffer)
	b.buffer = b.buffer[:0]
	return result
}

// GetSize returns the current buffer size.
func (b *Buffering) GetSize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.buffer)
}

// IsFull returns whether the buffer is full.
func (b *Buffering) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.buffer) >= b.maxSize
}

// Clear clears the buffer.
func (b *Buffering) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer = b.buffer[:0]
}

// Close releases the buffer back to the pool.
func (b *Buffering) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pool != nil && b.buffer != nil {
		b.pool.Put(b.buffer)
		b.buffer = nil
	}
}
