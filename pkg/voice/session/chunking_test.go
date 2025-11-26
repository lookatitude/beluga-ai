package session

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"github.com/stretchr/testify/assert"
)

func TestChunking_Chunk(t *testing.T) {
	config := internal.DefaultChunkingConfig()
	config.ChunkSize = 10
	config.OverlapSize = 2

	chunking := internal.NewChunking(config)

	data := make([]byte, 25)
	for i := range data {
		data[i] = byte(i)
	}

	chunks := chunking.Chunk(data)

	assert.Greater(t, len(chunks), 1, "Should create multiple chunks")
	assert.Len(t, chunks[0], 10, "First chunk should be chunk size")
}

func TestChunking_SmallData(t *testing.T) {
	config := internal.DefaultChunkingConfig()
	config.ChunkSize = 100

	chunking := internal.NewChunking(config)

	data := make([]byte, 50)
	chunks := chunking.Chunk(data)

	assert.Len(t, chunks, 1, "Small data should not be chunked")
}

func TestChunking_GetSetChunkSize(t *testing.T) {
	chunking := internal.NewChunking(nil)

	assert.Equal(t, 8192, chunking.GetChunkSize())

	chunking.SetChunkSize(4096)
	assert.Equal(t, 4096, chunking.GetChunkSize())
}
