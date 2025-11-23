package internal

import (
	"sync"
)

// ChunkingConfig configures chunking for long utterances
type ChunkingConfig struct {
	ChunkSize          int // Maximum chunk size in bytes
	ProcessingStrategy ChunkProcessingStrategy
	OverlapSize        int // Overlap between chunks
}

// ChunkProcessingStrategy defines how chunks are processed
type ChunkProcessingStrategy int

const (
	ChunkProcessingSequential ChunkProcessingStrategy = iota
	ChunkProcessingParallel
	ChunkProcessingStreaming
)

// DefaultChunkingConfig returns default chunking configuration
func DefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		ChunkSize:          8192, // 8KB default
		ProcessingStrategy: ChunkProcessingSequential,
		OverlapSize:        1024, // 1KB overlap
	}
}

// Chunking manages chunking of long utterances
type Chunking struct {
	mu     sync.RWMutex
	config *ChunkingConfig
}

// NewChunking creates a new chunking manager
func NewChunking(config *ChunkingConfig) *Chunking {
	if config == nil {
		config = DefaultChunkingConfig()
	}

	return &Chunking{
		config: config,
	}
}

// Chunk splits data into chunks
func (c *Chunking) Chunk(data []byte) [][]byte {
	c.mu.RLock()
	chunkSize := c.config.ChunkSize
	overlapSize := c.config.OverlapSize
	c.mu.RUnlock()

	if len(data) <= chunkSize {
		return [][]byte{data}
	}

	chunks := [][]byte{}
	offset := 0

	for offset < len(data) {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := make([]byte, end-offset)
		copy(chunk, data[offset:end])
		chunks = append(chunks, chunk)

		// Move offset with overlap
		offset = end - overlapSize
		if offset >= len(data) {
			break
		}
		if offset < 0 {
			offset = end // Prevent negative offset
		}
		if len(chunks) > 1000 {
			break // Safety limit to prevent infinite loops
		}
	}

	return chunks
}

// GetChunkSize returns the configured chunk size
func (c *Chunking) GetChunkSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.ChunkSize
}

// SetChunkSize sets the chunk size
func (c *Chunking) SetChunkSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.ChunkSize = size
}
