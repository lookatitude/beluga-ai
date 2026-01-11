package markdown

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownTextSplitter_SplitText(t *testing.T) {
	config := &MarkdownConfig{
		ChunkSize:        100,
		ChunkOverlap:     20,
		HeadersToSplitOn: []string{"#", "##", "###"},
	}

	splitter, err := NewMarkdownTextSplitter(config)
	require.NoError(t, err)

	markdown := "# Header 1\n\nContent here.\n\n## Header 2\n\nMore content."
	ctx := context.Background()
	chunks, err := splitter.SplitText(ctx, markdown)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
}

func TestMarkdownTextSplitter_SplitDocuments(t *testing.T) {
	config := &MarkdownConfig{
		ChunkSize:        50,
		ChunkOverlap:     10,
		HeadersToSplitOn: []string{"#", "##"},
	}

	splitter, err := NewMarkdownTextSplitter(config)
	require.NoError(t, err)

	docs := []schema.Document{
		{PageContent: "# Title\n\nContent here.", Metadata: map[string]string{"source": "doc1.md"}},
	}

	ctx := context.Background()
	chunks, err := splitter.SplitDocuments(ctx, docs)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
}

func TestMarkdownTextSplitter_CreateDocuments(t *testing.T) {
	config := &MarkdownConfig{
		ChunkSize:        50,
		ChunkOverlap:     10,
		HeadersToSplitOn: []string{"#"},
	}

	splitter, err := NewMarkdownTextSplitter(config)
	require.NoError(t, err)

	texts := []string{
		"# Document 1\n\nContent here.",
	}
	metadatas := []map[string]any{
		{"source": "doc1.md"},
	}

	ctx := context.Background()
	docs, err := splitter.CreateDocuments(ctx, texts, metadatas)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)
}

func TestMarkdownTextSplitter_CodeBlocks(t *testing.T) {
	config := &MarkdownConfig{
		ChunkSize:        100,
		ChunkOverlap:     20,
		HeadersToSplitOn: []string{"#"},
	}

	splitter, err := NewMarkdownTextSplitter(config)
	require.NoError(t, err)

	markdown := "```go\nfunc test() {}\n```\n\n# Header\n\nContent"
	ctx := context.Background()
	chunks, err := splitter.SplitText(ctx, markdown)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
}
