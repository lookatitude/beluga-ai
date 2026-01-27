package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultChunkingConfig(t *testing.T) {
	config := DefaultChunkingConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 8192, config.ChunkSize)
	assert.Equal(t, ChunkProcessingSequential, config.ProcessingStrategy)
	assert.Equal(t, 1024, config.OverlapSize)
}

func TestNewChunking(t *testing.T) {
	config := &ChunkingConfig{
		ChunkSize:          4096,
		ProcessingStrategy: ChunkProcessingParallel,
		OverlapSize:        512,
	}
	c := NewChunking(config)
	assert.NotNil(t, c)
	assert.Equal(t, 4096, c.GetChunkSize())
}

func TestNewChunking_NilConfig(t *testing.T) {
	c := NewChunking(nil)
	assert.NotNil(t, c)
	assert.Equal(t, 8192, c.GetChunkSize()) // Should use defaults
}

func TestChunking_Chunk_SmallData(t *testing.T) {
	c := NewChunking(&ChunkingConfig{
		ChunkSize:   100,
		OverlapSize: 10,
	})

	data := []byte{1, 2, 3, 4, 5}
	chunks := c.Chunk(data)
	assert.Len(t, chunks, 1)
	assert.Equal(t, data, chunks[0])
}

func TestChunking_Chunk_LargeData(t *testing.T) {
	c := NewChunking(&ChunkingConfig{
		ChunkSize:   10,
		OverlapSize: 2,
	})

	// Create data larger than chunk size
	data := make([]byte, 25)
	for i := range data {
		data[i] = byte(i)
	}

	chunks := c.Chunk(data)
	assert.Greater(t, len(chunks), 1)

	// Verify total size (accounting for overlaps)
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}
	assert.GreaterOrEqual(t, totalSize, len(data))
}

func TestChunking_Chunk_ExactChunkSize(t *testing.T) {
	c := NewChunking(&ChunkingConfig{
		ChunkSize:   10,
		OverlapSize: 0,
	})

	data := make([]byte, 10)
	chunks := c.Chunk(data)
	assert.Len(t, chunks, 1)
	assert.Equal(t, data, chunks[0])
}

func TestChunking_GetChunkSize(t *testing.T) {
	c := NewChunking(&ChunkingConfig{
		ChunkSize: 2048,
	})
	assert.Equal(t, 2048, c.GetChunkSize())
}

func TestChunking_SetChunkSize(t *testing.T) {
	c := NewChunking(&ChunkingConfig{
		ChunkSize: 1024,
	})
	assert.Equal(t, 1024, c.GetChunkSize())

	c.SetChunkSize(2048)
	assert.Equal(t, 2048, c.GetChunkSize())
}
