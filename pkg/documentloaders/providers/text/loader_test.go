package text

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextLoader_Load(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "test*.txt")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name()) //nolint:errcheck // Best effort cleanup
	}()

	_, err = tmpfile.WriteString("Test content")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	config := &LoaderConfig{
		MaxFileSize: 1024 * 1024,
	}

	loader, err := NewTextLoader(tmpfile.Name(), config)
	require.NoError(t, err)

	ctx := context.Background()
	docs, err := loader.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Test content", docs[0].PageContent)
}

func TestTextLoader_LazyLoad(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.txt")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name()) //nolint:errcheck // Best effort cleanup
	}()

	_, err = tmpfile.WriteString("Test content")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	config := &LoaderConfig{
		MaxFileSize: 1024 * 1024,
	}

	loader, err := NewTextLoader(tmpfile.Name(), config)
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := loader.LazyLoad(ctx)
	require.NoError(t, err)

	count := 0
	for item := range ch {
		switch item.(type) {
		case error:
			// Ignore errors
		default:
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestTextLoader_NonExistentFile(t *testing.T) {
	config := &LoaderConfig{
		MaxFileSize: 1024 * 1024,
	}

	loader, err := NewTextLoader("/nonexistent/file.txt", config)
	require.NoError(t, err) // Creation succeeds

	ctx := context.Background()
	_, err = loader.Load(ctx)
	assert.Error(t, err)
}

func TestTextLoader_FileTooLarge(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.txt")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name()) //nolint:errcheck // Best effort cleanup
	}()

	// Write content larger than limit
	largeContent := make([]byte, 2048)
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	_, err = tmpfile.Write(largeContent)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	config := &LoaderConfig{
		MaxFileSize: 1024, // 1KB limit
	}

	loader, err := NewTextLoader(tmpfile.Name(), config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = loader.Load(ctx)
	assert.Error(t, err)
}
