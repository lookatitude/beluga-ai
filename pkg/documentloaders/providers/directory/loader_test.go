package directory

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecursiveDirectoryLoader_Load(t *testing.T) {
	fsys := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data:    []byte("Content 1"),
			Mode:    0644,
			ModTime: time.Now(),
		},
		"file2.txt": &fstest.MapFile{
			Data:    []byte("Content 2"),
			Mode:    0644,
			ModTime: time.Now(),
		},
	}

	config := &DirectoryConfig{
		MaxDepth:    10,
		Concurrency: 2,
		MaxFileSize: 1024 * 1024,
	}

	loader, err := NewRecursiveDirectoryLoader(fsys, config)
	require.NoError(t, err)

	ctx := context.Background()
	docs, err := loader.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestRecursiveDirectoryLoader_LazyLoad(t *testing.T) {
	fsys := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data: []byte("Content 1"),
		},
	}

	config := &DirectoryConfig{
		MaxDepth:    10,
		Concurrency: 1,
		MaxFileSize: 1024 * 1024,
	}

	loader, err := NewRecursiveDirectoryLoader(fsys, config)
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := loader.LazyLoad(ctx)
	require.NoError(t, err)

	count := 0
	for item := range ch {
		switch item.(type) {
		case error:
			// Ignore errors for this test
		default:
			count++
		}
	}
	assert.GreaterOrEqual(t, count, 1)
}

func TestRecursiveDirectoryLoader_InvalidConfig(t *testing.T) {
	fsys := fstest.MapFS{}

	// Test negative MaxDepth
	config := &DirectoryConfig{
		MaxDepth:    -1,
		Concurrency: 1,
		MaxFileSize: 1024,
	}
	_, err := NewRecursiveDirectoryLoader(fsys, config)
	assert.Error(t, err)

	// Test zero Concurrency
	config = &DirectoryConfig{
		MaxDepth:    10,
		Concurrency: 0,
		MaxFileSize: 1024,
	}
	_, err = NewRecursiveDirectoryLoader(fsys, config)
	assert.Error(t, err)

	// Test zero MaxFileSize
	config = &DirectoryConfig{
		MaxDepth:    10,
		Concurrency: 1,
		MaxFileSize: 0,
	}
	_, err = NewRecursiveDirectoryLoader(fsys, config)
	assert.Error(t, err)
}

func TestRecursiveDirectoryLoader_GetDepth(t *testing.T) {
	fsys := fstest.MapFS{}
	config := &DirectoryConfig{
		MaxDepth:    10,
		Concurrency: 1,
		MaxFileSize: 1024,
	}

	loader, err := NewRecursiveDirectoryLoader(fsys, config)
	require.NoError(t, err)

	// Test getDepth method (via reflection or by testing behavior)
	// Since getDepth is private, we test it indirectly through MaxDepth behavior
	fsysWithNested := fstest.MapFS{
		"file.txt":        &fstest.MapFile{Data: []byte("content")},
		"dir/file.txt":    &fstest.MapFile{Data: []byte("content")},
		"dir/sub/file.txt": &fstest.MapFile{Data: []byte("content")},
	}

	config.MaxDepth = 1
	loader, err = NewRecursiveDirectoryLoader(fsysWithNested, config)
	require.NoError(t, err)

	ctx := context.Background()
	docs, err := loader.Load(ctx)
	require.NoError(t, err)
	// Should load files at depth 0 and 1, but not depth 2
	assert.LessOrEqual(t, len(docs), 2)
}
