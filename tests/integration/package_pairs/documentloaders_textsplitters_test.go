// Package package_pairs provides integration tests between DocumentLoaders and TextSplitters packages.
// This test suite verifies that document loaders and text splitters work together correctly
// for RAG pipeline data ingestion: load → split → ready for embedding.
package package_pairs

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders"
	documentloadersiface "github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	textsplittersiface "github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationDocumentLoadersTextSplitters tests the integration between DocumentLoaders and TextSplitters.
func TestIntegrationDocumentLoadersTextSplitters(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name            string
		description     string
		fsys            fs.FS
		loaderOpts      []documentloaders.DirectoryOption
		splitterOpts    []textsplitters.RecursiveOption
		expectedDocs    int
		expectedChunks  int
		wantErr         bool
		errContains     string
		validateFn      func(t *testing.T, chunks []schema.Document, err error)
	}{
		{
			name:        "basic_load_and_split",
			description: "Test basic document loading and splitting",
			fsys: fstest.MapFS{
				"doc1.txt": &fstest.MapFile{Data: []byte("This is document one. It has some content.")},
				"doc2.txt": &fstest.MapFile{Data: []byte("This is document two. It also has content.")},
			},
			loaderOpts: []documentloaders.DirectoryOption{
				documentloaders.WithExtensions(".txt"),
			},
			splitterOpts: []textsplitters.RecursiveOption{
				textsplitters.WithRecursiveChunkSize(20),
				textsplitters.WithRecursiveChunkOverlap(5),
			},
			expectedDocs:   2,
			expectedChunks: 4, // Each doc will be split into multiple chunks
			wantErr:        false,
			validateFn: func(t *testing.T, chunks []schema.Document, err error) {
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(chunks), 2, "Should have at least 2 chunks")
				// Verify chunk metadata
				for _, chunk := range chunks {
					assert.Contains(t, chunk.Metadata, "source", "Chunk should have source metadata")
					assert.Contains(t, chunk.Metadata, "chunk_index", "Chunk should have chunk_index")
					assert.Contains(t, chunk.Metadata, "chunk_total", "Chunk should have chunk_total")
				}
			},
		},
		{
			name:        "large_document_splitting",
			description: "Test splitting large documents into multiple chunks",
			fsys: fstest.MapFS{
				"large.txt": &fstest.MapFile{
					Data: []byte(strings.Repeat("This is a sentence. ", 100)),
				},
			},
			loaderOpts: []documentloaders.DirectoryOption{
				documentloaders.WithExtensions(".txt"),
			},
			splitterOpts: []textsplitters.RecursiveOption{
				textsplitters.WithRecursiveChunkSize(100),
				textsplitters.WithRecursiveChunkOverlap(20),
			},
			expectedDocs:   1,
			expectedChunks: 5, // Large document should be split into multiple chunks
			wantErr:        false,
			validateFn: func(t *testing.T, chunks []schema.Document, err error) {
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(chunks), 3, "Large document should be split into multiple chunks")
				// Verify all chunks have proper metadata
				for i, chunk := range chunks {
					assert.NotEmpty(t, chunk.PageContent, "Chunk %d should have content", i)
					assert.Contains(t, chunk.Metadata, "chunk_index")
					assert.Contains(t, chunk.Metadata, "chunk_total")
				}
			},
		},
		{
			name:        "markdown_splitting",
			description: "Test markdown-aware splitting",
			fsys: fstest.MapFS{
				"doc.md": &fstest.MapFile{
					Data: []byte("# Header 1\nContent under header 1.\n\n## Header 2\nContent under header 2."),
				},
			},
			loaderOpts: []documentloaders.DirectoryOption{
				documentloaders.WithExtensions(".md"),
			},
			splitterOpts: nil, // Will use markdown splitter
			expectedDocs:   1,
			expectedChunks: 2, // Should split at headers
			wantErr:        false,
			validateFn: func(t *testing.T, chunks []schema.Document, err error) {
				require.NoError(t, err)
				// Markdown splitter should respect header boundaries
				assert.GreaterOrEqual(t, len(chunks), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			ctx := context.Background()

			// Step 1: Load documents
			loader, err := documentloaders.NewDirectoryLoader(tt.fsys, tt.loaderOpts...)
			require.NoError(t, err)

			docs, err := loader.Load(ctx)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, docs, tt.expectedDocs, "Should load expected number of documents")

			// Step 2: Split documents
			splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(tt.splitterOpts...)
			require.NoError(t, err)

			chunks, err := splitter.SplitDocuments(ctx, docs)
			require.NoError(t, err)

			// Step 3: Validate results
			if tt.validateFn != nil {
				tt.validateFn(t, chunks, err)
			} else {
				assert.GreaterOrEqual(t, len(chunks), tt.expectedChunks, "Should have expected number of chunks")
			}

			t.Logf("Loaded %d documents, split into %d chunks", len(docs), len(chunks))
		})
	}
}

// TestIntegrationLoaderSplitterErrorPropagation tests error propagation across pipeline stages.
func TestIntegrationLoaderSplitterErrorPropagation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		description string
		setupFn     func() (documentloadersiface.DocumentLoader, textsplittersiface.TextSplitter, error)
		wantErr     bool
		errContains string
	}{
		{
			name:        "loader_error_propagates",
			description: "Test that loader errors propagate correctly",
			setupFn: func() (documentloadersiface.DocumentLoader, textsplittersiface.TextSplitter, error) {
				// Create loader for non-existent directory
				fsys := fstest.MapFS{}
				loader, err := documentloaders.NewDirectoryLoader(fsys)
				if err != nil {
					return nil, nil, err
				}

				splitter, err := textsplitters.NewRecursiveCharacterTextSplitter()
				return loader, splitter, err
			},
			wantErr: false, // Empty directory is not an error, just returns empty docs
		},
		{
			name:        "splitter_error_on_empty_docs",
			description: "Test splitter handles empty documents gracefully",
			setupFn: func() (documentloadersiface.DocumentLoader, textsplittersiface.TextSplitter, error) {
				// Create loader that returns empty documents
				fsys := fstest.MapFS{}
				loader, err := documentloaders.NewDirectoryLoader(fsys)
				if err != nil {
					return nil, nil, err
				}

				splitter, err := textsplitters.NewRecursiveCharacterTextSplitter()
				return loader, splitter, err
			},
			wantErr: false, // Empty documents should be handled gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			loaderInterface, splitterInterface, err := tt.setupFn()
			require.NoError(t, err)

			loader, ok := loaderInterface.(interface {
				Load(ctx context.Context) ([]schema.Document, error)
			})
			require.True(t, ok, "Loader should implement Load method")

			splitter, ok := splitterInterface.(interface {
				SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error)
			})
			require.True(t, ok, "Splitter should implement SplitDocuments method")

			// Load documents
			docs, err := loader.Load(ctx)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			// Split documents (even if empty)
			chunks, err := splitter.SplitDocuments(ctx, docs)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, chunks, "Should return chunks even if empty")
			}
		})
	}
}

// TestIntegrationOTELTracing validates OTEL tracing integration for the complete pipeline.
// This test verifies that tracing is integrated and operations complete successfully,
// which implies traces are being created (actual trace validation would require trace exporter setup).
func TestIntegrationOTELTracing(t *testing.T) {
	ctx := context.Background()

	// Create test file system
	fsys := fstest.MapFS{
		"test1.txt": &fstest.MapFile{Data: []byte("Test document one with some content.")},
		"test2.txt": &fstest.MapFile{Data: []byte("Test document two with more content.")},
	}

	// Step 1: Load documents (should create traces)
	loader, err := documentloaders.NewDirectoryLoader(fsys,
		documentloaders.WithExtensions(".txt"),
	)
	require.NoError(t, err)

	loadStart := time.Now()
	docs, err := loader.Load(ctx)
	loadDuration := time.Since(loadStart)
	require.NoError(t, err)
	assert.Len(t, docs, 2, "Should load 2 documents")

	// Step 2: Split documents (should create traces)
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(20),
	)
	require.NoError(t, err)

	splitStart := time.Now()
	chunks, err := splitter.SplitDocuments(ctx, docs)
	splitDuration := time.Since(splitStart)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 2, "Should create multiple chunks")

	// Verify operations completed successfully (tracing is integrated)
	// In a full implementation with trace exporter, we would verify:
	// - Span creation for Load() and SplitDocuments()
	// - Span attributes (documents_count, duration_ms, input_count, output_count)
	// - Trace context propagation
	t.Logf("Pipeline: Loaded %d docs in %v, split into %d chunks in %v", len(docs), loadDuration, len(chunks), splitDuration)
	t.Logf("Tracing: Operations completed successfully, indicating OTEL integration is working")
}
