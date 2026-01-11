package recursive

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecursiveCharacterTextSplitter_SplitText(t *testing.T) {
	config := &RecursiveConfig{
		ChunkSize:    100,
		ChunkOverlap: 20,
		Separators:   []string{"\n\n", "\n", " ", ""},
	}

	splitter, err := NewRecursiveCharacterTextSplitter(config)
	require.NoError(t, err)

	ctx := context.Background()
	text := "This is a test. " + "This is another sentence. " + "And one more. "
	chunks, err := splitter.SplitText(ctx, text)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
}

func TestRecursiveCharacterTextSplitter_SplitDocuments(t *testing.T) {
	config := &RecursiveConfig{
		ChunkSize:    50,
		ChunkOverlap: 10,
		Separators:   []string{"\n\n", "\n", " ", ""},
	}

	splitter, err := NewRecursiveCharacterTextSplitter(config)
	require.NoError(t, err)

	docs := []schema.Document{
		{PageContent: "Document 1 content here", Metadata: map[string]string{"source": "doc1"}},
		{PageContent: "Document 2 content here", Metadata: map[string]string{"source": "doc2"}},
	}

	ctx := context.Background()
	chunks, err := splitter.SplitDocuments(ctx, docs)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
	assert.GreaterOrEqual(t, len(chunks), len(docs))
}

func TestRecursiveCharacterTextSplitter_CreateDocuments(t *testing.T) {
	config := &RecursiveConfig{
		ChunkSize:    50,
		ChunkOverlap: 10,
		Separators:   []string{"\n\n", "\n", " ", ""},
	}

	splitter, err := NewRecursiveCharacterTextSplitter(config)
	require.NoError(t, err)

	texts := []string{
		"First document content",
		"Second document content",
	}
	metadatas := []map[string]any{
		{"source": "doc1"},
		{"source": "doc2"},
	}

	ctx := context.Background()
	docs, err := splitter.CreateDocuments(ctx, texts, metadatas)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)
}

func TestRecursiveCharacterTextSplitter_InvalidConfig(t *testing.T) {
	// Test overlap > chunk size
	config := &RecursiveConfig{
		ChunkSize:    50,
		ChunkOverlap: 100, // Invalid: overlap > chunk size
		Separators:   []string{"\n\n", "\n", " ", ""},
	}

	_, err := NewRecursiveCharacterTextSplitter(config)
	assert.Error(t, err)
}
