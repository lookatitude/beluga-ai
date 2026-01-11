package textsplitters

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegistryRegistration tests registry registration and retrieval.
func TestRegistryRegistration(t *testing.T) {
	registry := GetRegistry()

	// Test that built-in splitters are registered
	assert.True(t, registry.IsRegistered("recursive"), "recursive splitter should be registered")
	assert.True(t, registry.IsRegistered("markdown"), "markdown splitter should be registered")

	// Test listing registered splitters
	names := registry.List()
	assert.Contains(t, names, "recursive", "recursive should be in list")
	assert.Contains(t, names, "markdown", "markdown should be in list")
}

// TestRegistryCreate tests creating splitters via registry.
func TestRegistryCreate(t *testing.T) {
	registry := GetRegistry()
	ctx := context.Background()

	// Test creating recursive splitter
	splitter, err := registry.Create("recursive", map[string]any{
		"chunk_size":    100,
		"chunk_overlap": 20,
	})
	require.NoError(t, err)
	assert.NotNil(t, splitter)

	// Test that it works
	chunks, err := splitter.SplitText(ctx, "This is a test document that should be split into multiple chunks.")
	require.NoError(t, err)
	assert.NotNil(t, chunks)

	// Test creating markdown splitter
	markdownSplitter, err := registry.Create("markdown", map[string]any{
		"chunk_size":    50,
		"chunk_overlap": 10, // Must be less than chunk_size
	})
	require.NoError(t, err)
	assert.NotNil(t, markdownSplitter)
}

// TestRegistryUnregisteredSplitter tests error handling for unregistered splitters.
func TestRegistryUnregisteredSplitter(t *testing.T) {
	registry := GetRegistry()

	_, err := registry.Create("nonexistent", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

// TestConfigValidation tests configuration validation.
func TestConfigValidation(t *testing.T) {
	// Test valid recursive config
	recursiveCfg := DefaultRecursiveConfig()
	err := recursiveCfg.Validate()
	assert.NoError(t, err)

	// Test invalid recursive config (overlap > chunk size)
	recursiveCfg.ChunkOverlap = recursiveCfg.ChunkSize + 1
	err = recursiveCfg.Validate()
	assert.Error(t, err)

	// Test invalid recursive config (zero chunk size)
	recursiveCfg = DefaultRecursiveConfig()
	recursiveCfg.ChunkSize = 0
	err = recursiveCfg.Validate()
	assert.Error(t, err)

	// Test valid markdown config
	markdownCfg := DefaultMarkdownConfig()
	err = markdownCfg.Validate()
	assert.NoError(t, err)

	// Test invalid markdown config (overlap > chunk size)
	markdownCfg.ChunkOverlap = markdownCfg.ChunkSize + 1
	err = markdownCfg.Validate()
	assert.Error(t, err)
}

// TestErrorHelpers tests error helper functions.
func TestErrorHelpers(t *testing.T) {
	// Test NewSplitterError
	err := NewSplitterError("TestOp", ErrCodeInvalidConfig, "test message", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, ErrCodeInvalidConfig, err.Code)

	// Test IsSplitterError
	assert.True(t, IsSplitterError(err))
	assert.False(t, IsSplitterError(fmt.Errorf("regular error")))

	// Test GetSplitterError
	extracted := GetSplitterError(err)
	assert.NotNil(t, extracted)
	assert.Equal(t, err, extracted)

	// Test Unwrap
	assert.Nil(t, err.Unwrap())
	wrappedErr := fmt.Errorf("wrapped")
	errWithWrap := NewSplitterError("TestOp", ErrCodeInvalidConfig, "", wrappedErr)
	assert.Equal(t, wrappedErr, errWithWrap.Unwrap())
}

// TestRegistryCustomSplitter tests custom splitter registration and usage.
func TestRegistryCustomSplitter(t *testing.T) {
	registry := GetRegistry()
	ctx := context.Background()

	// Create a custom factory
	customFactory := func(config map[string]any) (iface.TextSplitter, error) {
		return &mockSplitter{
			chunks: []string{"chunk1", "chunk2"},
		}, nil
	}

	// Register custom splitter
	func() {
		defer func() {
			if r := recover(); r == nil {
				// Should not panic on first registration
			}
		}()
		registry.Register("custom_test", customFactory)
	}()

	// Verify it's registered
	assert.True(t, registry.IsRegistered("custom_test"))

	// Create and use the custom splitter
	splitter, err := registry.Create("custom_test", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, splitter)

	chunks, err := splitter.SplitText(ctx, "test")
	require.NoError(t, err)
	assert.Len(t, chunks, 2)
}

// mockSplitter is a simple mock splitter for testing.
type mockSplitter struct {
	chunks []string
}

func (m *mockSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
	return m.chunks, nil
}

func (m *mockSplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
	var result []schema.Document
	for i, chunk := range m.chunks {
		result = append(result, schema.Document{
			PageContent: chunk,
			Metadata:    map[string]string{"chunk_index": string(rune(i))},
		})
	}
	return result, nil
}

func (m *mockSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
	var result []schema.Document
	for i, text := range texts {
		doc := schema.Document{
			PageContent: text,
			Metadata:    make(map[string]string),
		}
		if i < len(metadatas) && metadatas[i] != nil {
			for k, v := range metadatas[i] {
				if str, ok := v.(string); ok {
					doc.Metadata[k] = str
				}
			}
		}
		result = append(result, doc)
	}
	return m.SplitDocuments(ctx, result)
}

// Ensure mockSplitter implements iface.TextSplitter
var _ iface.TextSplitter = (*mockSplitter)(nil)
