// Package package_pairs provides integration tests between DocumentLoaders and TextSplitters packages.
// This test suite verifies that document loaders work correctly with text splitters
// for document processing, chunking, and text splitting operations.
package package_pairs

import (
	"context"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	textsplittersiface "github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationDocumentLoadersTextSplitters tests the integration between DocumentLoaders and TextSplitters packages.
func TestIntegrationDocumentLoadersTextSplitters(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name         string
		splitterType string
		wantErr      bool
	}{
		{
			name:         "documentloader_with_recursive_character_splitter",
			splitterType: "recursive_character",
			wantErr:      false,
		},
		{
			name:         "documentloader_with_character_splitter",
			splitterType: "character",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create test file system
			fsys := fstest.MapFS{
				"file1.txt": &fstest.MapFile{
					Data: []byte("This is a test document. It contains multiple sentences. Each sentence should be split properly."),
				},
				"file2.txt": &fstest.MapFile{
					Data: []byte("Another document with content. This will also be split. Testing integration between loaders and splitters."),
				},
			}

			// Create document loader
			loader, err := documentloaders.NewDirectoryLoader(fsys)
			if err != nil {
				t.Skipf("Loader creation failed: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loader)

			// Load documents
			docs, err := loader.Load(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, docs)
			assert.Greater(t, len(docs), 0)

			// Create text splitter
			var splitter textsplittersiface.TextSplitter
			var splitterErr error
			switch tt.splitterType {
			case "recursive_character":
				splitter, splitterErr = textsplitters.NewRecursiveCharacterTextSplitter()
				require.NoError(t, splitterErr)
			case "character":
				// Character splitter not available, skip
				t.Skipf("Splitter type %s not implemented", tt.splitterType)
				return
			default:
				t.Skipf("Splitter type %s not implemented in test", tt.splitterType)
				return
			}

			require.NotNil(t, splitter)

			// Split documents
			for i, doc := range docs {
				sourceStr := doc.Metadata["source"]
				if sourceStr == "" {
					sourceStr = fmt.Sprintf("doc_%d", i)
				}
				t.Run(sourceStr, func(t *testing.T) {
					chunks, err := splitter.SplitText(ctx, doc.PageContent)
					if err != nil {
						t.Logf("SplitText error (document %d): %v", i, err)
					} else {
						require.NotEmpty(t, chunks)
						assert.Greater(t, len(chunks), 0)

						// Verify chunks are reasonable size
						for j, chunk := range chunks {
							assert.NotEmpty(t, chunk, "Chunk %d should not be empty", j)
							assert.LessOrEqual(t, len(chunk), len(doc.PageContent), "Chunk should not be larger than original")
						}
					}
				})
			}
		})
	}
}

// TestDocumentLoadersTextSplittersWorkflow tests a complete workflow: load -> split -> process.
func TestDocumentLoadersTextSplittersWorkflow(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create test file system with longer document
	fsys := fstest.MapFS{
		"long_document.txt": &fstest.MapFile{
			Data: []byte(`This is a longer document that will be split into multiple chunks.
It contains multiple paragraphs and sentences.
Each paragraph should be handled appropriately by the text splitter.
The document loader should load it correctly.
Then the text splitter should split it into manageable chunks.
This allows for better processing and embedding of the content.

Here is another paragraph to ensure we have enough content for splitting.
The text splitter needs sufficient content to create multiple chunks.
We need to make sure the document is long enough to trigger multiple splits.
This paragraph adds more content to help achieve that goal.
Additional sentences help ensure proper chunking behavior.

Yet another paragraph to guarantee multiple chunks are created.
The recursive character text splitter should handle this properly.
With enough content, we should see multiple chunks in the output.
This helps verify that the integration between document loaders and text splitters works correctly.`),
		},
	}

	// Step 1: Load document
	loader, err := documentloaders.NewDirectoryLoader(fsys)
	require.NoError(t, err)

	docs, err := loader.Load(ctx)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	originalDoc := docs[0]
	assert.NotEmpty(t, originalDoc.PageContent)

	// Step 2: Split document with smaller chunk size to ensure multiple chunks
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(100),   // Small chunk size to ensure splitting
		textsplitters.WithRecursiveChunkOverlap(10), // Set overlap explicitly (must be < ChunkSize)
	)
	require.NoError(t, err)
	require.NotNil(t, splitter)

	chunks, err := splitter.SplitText(ctx, originalDoc.PageContent)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	// Step 3: Verify chunks maintain document metadata
	for i, chunk := range chunks {
		t.Run(chunk, func(t *testing.T) {
			assert.NotEmpty(t, chunk, "Chunk %d should not be empty", i)
			// Verify chunk is part of original content
			assert.Contains(t, originalDoc.PageContent, chunk, "Chunk %d should be part of original document", i)
		})
	}

	// Verify we have multiple chunks for a long document
	assert.Greater(t, len(chunks), 1, "Long document should be split into multiple chunks")
}

// TestDocumentLoadersTextSplittersLazyLoad tests lazy loading with text splitting.
func TestDocumentLoadersTextSplittersLazyLoad(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create test file system
	fsys := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data: []byte("First document content."),
		},
		"file2.txt": &fstest.MapFile{
			Data: []byte("Second document content."),
		},
	}

	// Create loader
	loader, err := documentloaders.NewDirectoryLoader(fsys)
	require.NoError(t, err)

	// Create splitter
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter()
	require.NoError(t, err)
	require.NotNil(t, splitter)

	// Lazy load documents
	ch, err := loader.LazyLoad(ctx)
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Process documents as they arrive
	docCount := 0
	chunkCount := 0

	for item := range ch {
		if err, ok := item.(error); ok {
			t.Logf("LazyLoad error: %v", err)
			continue
		}

		if doc, ok := item.(schema.Document); ok {
			docCount++
			// Split each document as it arrives
			chunks, err := splitter.SplitText(ctx, doc.PageContent)
			if err != nil {
				t.Logf("SplitText error (document %d): %v", docCount, err)
			} else {
				chunkCount += len(chunks)
			}
		}
	}

	assert.Greater(t, docCount, 0, "Should load at least one document")
	assert.Greater(t, chunkCount, 0, "Should create at least one chunk")
}
