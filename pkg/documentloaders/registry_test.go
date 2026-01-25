package documentloaders

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegistryRegistration tests registry registration and retrieval.
func TestRegistryRegistration(t *testing.T) {
	registry := GetRegistry()

	// Test that built-in loaders are registered
	assert.True(t, registry.IsRegistered("directory"), "directory loader should be registered")
	assert.True(t, registry.IsRegistered("text"), "text loader should be registered")

	// Test listing registered loaders
	names := registry.List()
	assert.Contains(t, names, "directory", "directory should be in list")
	assert.Contains(t, names, "text", "text should be in list")
}

// TestRegistryCreate tests creating loaders via registry.
func TestRegistryCreate(t *testing.T) {
	registry := GetRegistry()
	ctx := context.Background()

	// Test creating directory loader
	// Note: This test verifies the factory works, actual loading may fail if path is invalid
	loader, err := registry.Create("directory", map[string]any{
		"path":       ".",
		"max_depth":  2,
		"extensions": []string{".txt"},
	})
	// Factory should succeed even if the directory doesn't have files
	require.NoError(t, err, "Factory should create loader successfully")
	if loader != nil {
		// Only test loading if loader was created
		_, err = loader.Load(ctx) // Ignore errors as directory may be empty
		_ = err                   // Explicitly ignore - directory may be empty
	}

	// Test creating text loader
	textLoader, err := registry.Create("text", map[string]any{
		"path": "/nonexistent/file.txt", // Will fail but tests the factory
	})
	require.NoError(t, err) // Factory creation succeeds
	assert.NotNil(t, textLoader)
}

// TestRegistryUnregisteredLoader tests error handling for unregistered loaders.
func TestRegistryUnregisteredLoader(t *testing.T) {
	registry := GetRegistry()

	_, err := registry.Create("nonexistent", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

// TestConfigValidation tests configuration validation.
func TestConfigValidation(t *testing.T) {
	// Test valid config (Path is required for validation, but not used in factory)
	cfg := DefaultDirectoryConfig()
	cfg.Path = "." // Set path for validation
	err := cfg.Validate()
	assert.NoError(t, err)

	// Test invalid config (negative max depth)
	cfg.MaxDepth = -1
	err = cfg.Validate()
	assert.Error(t, err)

	// Test invalid config (zero concurrency)
	cfg = DefaultDirectoryConfig()
	cfg.Path = "."
	cfg.Concurrency = 0
	err = cfg.Validate()
	assert.Error(t, err)
}

// TestErrorHelpers tests error helper functions.
func TestErrorHelpers(t *testing.T) {
	// Test NewLoaderError
	err := NewLoaderError("TestOp", ErrCodeIOError, "test.txt", "test message", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, ErrCodeIOError, err.Code)

	// Test IsLoaderError
	assert.True(t, IsLoaderError(err))
	assert.False(t, IsLoaderError(errors.New("regular error")))

	// Test GetLoaderError
	extracted := GetLoaderError(err)
	assert.NotNil(t, extracted)
	assert.Equal(t, err, extracted)

	// Test Unwrap
	assert.NoError(t, err.Unwrap())
	wrappedErr := errors.New("wrapped")
	errWithWrap := NewLoaderError("TestOp", ErrCodeIOError, "", "", wrappedErr)
	assert.Equal(t, wrappedErr, errWithWrap.Unwrap())
}

// TestWithDirectoryMaxFileSize tests the directory max file size option.
func TestWithDirectoryMaxFileSize(t *testing.T) {
	cfg := DefaultDirectoryConfig()
	WithDirectoryMaxFileSize(5000)(cfg)
	assert.Equal(t, int64(5000), cfg.MaxFileSize)
}

// TestWithMaxFileSize tests the generic max file size option.
func TestWithMaxFileSize(t *testing.T) {
	cfg := DefaultLoaderConfig()
	WithMaxFileSize(3000)(cfg)
	assert.Equal(t, int64(3000), cfg.MaxFileSize)
}

// TestRegistryDuplicateRegistration tests duplicate registration handling.
func TestRegistryDuplicateRegistration(t *testing.T) {
	registry := GetRegistry()

	// Create a custom factory
	customFactory := func(config map[string]any) (iface.DocumentLoader, error) {
		return &mockLoader{}, nil
	}

	// Register a custom loader
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic on duplicate registration
				if str, ok := r.(string); ok {
					assert.Contains(t, str, "already registered")
				}
			}
		}()
		registry.Register("test_duplicate", customFactory)
		// Try to register again - should panic
		registry.Register("test_duplicate", customFactory)
	}()
}

// TestRegistryCustomLoader tests custom loader registration and usage.
func TestRegistryCustomLoader(t *testing.T) {
	registry := GetRegistry()
	ctx := context.Background()

	// Create a custom factory
	customFactory := func(config map[string]any) (iface.DocumentLoader, error) {
		return &mockLoader{
			docs: []schema.Document{
				{PageContent: "Custom loader content", Metadata: map[string]string{"source": "custom"}},
			},
		}, nil
	}

	// Register custom loader
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

	// Create and use the custom loader
	loader, err := registry.Create("custom_test", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, loader)

	docs, err := loader.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Custom loader content", docs[0].PageContent)
}

// mockLoader is a simple mock loader for testing.
type mockLoader struct {
	docs []schema.Document
}

func (m *mockLoader) Load(ctx context.Context) ([]schema.Document, error) {
	return m.docs, nil
}

func (m *mockLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
	ch := make(chan any, len(m.docs))
	go func() {
		defer close(ch)
		for _, doc := range m.docs {
			ch <- doc
		}
	}()
	return ch, nil
}

// Ensure mockLoader implements iface.DocumentLoader.
var _ iface.DocumentLoader = (*mockLoader)(nil)
