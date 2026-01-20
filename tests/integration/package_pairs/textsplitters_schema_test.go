// Package package_pairs provides integration tests between TextSplitters and Schema packages.
// This test suite verifies that text splitters work correctly with schema.Document types
// for document splitting, metadata preservation, and chunk creation.
package package_pairs

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	textsplittersiface "github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationTextSplittersSchema tests the integration between TextSplitters and Schema packages.
func TestIntegrationTextSplittersSchema(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name          string
		splitterType  string
		setupSplitter func(t *testing.T) (textsplittersiface.TextSplitter, error)
		documents     []schema.Document
		wantErr       bool
	}{
		{
			name:         "recursive_splitter_with_documents",
			splitterType: "recursive",
			setupSplitter: func(t *testing.T) (textsplittersiface.TextSplitter, error) {
				return textsplitters.NewRecursiveCharacterTextSplitter(
					textsplitters.WithRecursiveChunkSize(500),
					textsplitters.WithRecursiveChunkOverlap(50),
				)
			},
			documents: []schema.Document{
				{
					PageContent: "This is a test document. " + repeatString("Content ", 100),
					Metadata: map[string]string{
						"source": "test1.txt",
						"author": "test-author",
					},
				},
				{
					PageContent: "Another document. " + repeatString("More content ", 100),
					Metadata: map[string]string{
						"source": "test2.txt",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "markdown_splitter_with_documents",
			splitterType: "markdown",
			setupSplitter: func(t *testing.T) (textsplittersiface.TextSplitter, error) {
				return textsplitters.NewMarkdownTextSplitter(
					textsplitters.WithMarkdownChunkSize(500),
					textsplitters.WithMarkdownChunkOverlap(50),
				)
			},
			documents: []schema.Document{
				{
					PageContent: "# Header 1\n\nContent under header 1.\n\n## Header 2\n\nContent under header 2.",
					Metadata: map[string]string{
						"source": "markdown.md",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			splitter, err := tt.setupSplitter(t)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, splitter)

			// Test SplitDocuments
			chunks, err := splitter.SplitDocuments(ctx, tt.documents)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, chunks)
			assert.Greater(t, len(chunks), 0, "Should produce at least one chunk")

			// Verify metadata preservation
			for i, chunk := range chunks {
				assert.NotNil(t, chunk.Metadata, "Chunk %d should have metadata", i)
				// Verify chunk_index and chunk_total are added
				if len(tt.documents) > 0 {
					// At least some chunks should have chunk_index
					hasChunkIndex := false
					for _, c := range chunks {
						if _, ok := c.Metadata["chunk_index"]; ok {
							hasChunkIndex = true
							break
						}
					}
					assert.True(t, hasChunkIndex, "Chunks should have chunk_index metadata")
				}
			}

			// Test CreateDocuments
			texts := make([]string, len(tt.documents))
			metadatas := make([]map[string]any, len(tt.documents))
			for i, doc := range tt.documents {
				texts[i] = doc.PageContent
				// Convert map[string]string to map[string]any
				metaAny := make(map[string]any)
				for k, v := range doc.Metadata {
					metaAny[k] = v
				}
				metadatas[i] = metaAny
			}

			createdDocs, err := splitter.CreateDocuments(ctx, texts, metadatas)
			require.NoError(t, err)
			assert.NotNil(t, createdDocs)
			assert.Greater(t, len(createdDocs), 0, "Should create at least one document")
		})
	}
}

// TestTextSplittersSchemaMetadataPreservation tests metadata preservation during splitting.
func TestTextSplittersSchemaMetadataPreservation(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(100),
		textsplitters.WithRecursiveChunkOverlap(20),
	)
	require.NoError(t, err)

	doc := schema.Document{
		PageContent: repeatString("Test content ", 50),
		Metadata: map[string]string{
			"source":   "test.txt",
			"author":   "test-author",
			"category": "test",
		},
	}

	chunks, err := splitter.SplitDocuments(ctx, []schema.Document{doc})
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1, "Should split into multiple chunks")

	// Verify all chunks preserve original metadata
	for i, chunk := range chunks {
		assert.Equal(t, "test.txt", chunk.Metadata["source"], "Chunk %d should preserve source", i)
		assert.Equal(t, "test-author", chunk.Metadata["author"], "Chunk %d should preserve author", i)
		assert.Equal(t, "test", chunk.Metadata["category"], "Chunk %d should preserve category", i)
		assert.Contains(t, chunk.Metadata, "chunk_index", "Chunk %d should have chunk_index", i)
		assert.Contains(t, chunk.Metadata, "chunk_total", "Chunk %d should have chunk_total", i)
	}
}

// TestTextSplittersSchemaEmptyDocuments tests handling of empty documents.
func TestTextSplittersSchemaEmptyDocuments(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter()
	require.NoError(t, err)

	// Test with empty document
	emptyDoc := schema.Document{
		PageContent: "",
		Metadata:    map[string]string{"source": "empty.txt"},
	}

	chunks, err := splitter.SplitDocuments(ctx, []schema.Document{emptyDoc})
	// Empty documents may return error or empty chunks depending on implementation
	if err != nil {
		assert.Contains(t, err.Error(), "empty", "Error should mention empty")
	} else {
		assert.Len(t, chunks, 0, "Empty document should produce no chunks")
	}
}

// Helper function to repeat a string.
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
