package loader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FI-43: cleanPath alone accepts absolute paths; WithinRoot confines reads.
func TestWithinRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "in.txt"), []byte("inside"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")))

	l, err := WithinRoot(NewTextLoader(), root)
	require.NoError(t, err)
	ctx := context.Background()

	docs, err := l.Load(ctx, "in.txt")
	require.NoError(t, err)
	assert.Equal(t, "inside", docs[0].Content)

	docs, err = l.Load(ctx, filepath.Join(root, "in.txt"))
	require.NoError(t, err)
	assert.Equal(t, "inside", docs[0].Content)

	for _, src := range []string{
		filepath.Join(outside, "secret.txt"),
		"../" + filepath.Base(outside) + "/secret.txt",
		"link.txt",
		"/etc/passwd",
		"",
	} {
		_, err := l.Load(ctx, src)
		var ce *core.Error
		require.True(t, errors.As(err, &ce), src)
		assert.Equal(t, core.ErrInvalidInput, ce.Code, src)
	}

	// Missing files inside the root reach the inner loader's error.
	_, err = l.Load(ctx, "missing.txt")
	var ce *core.Error
	require.True(t, errors.As(err, &ce))
	assert.Equal(t, core.ErrProviderDown, ce.Code)
}

func TestWithinRoot_InvalidArgs(t *testing.T) {
	_, err := WithinRoot(nil, t.TempDir())
	require.Error(t, err)
	_, err = WithinRoot(NewTextLoader(), filepath.Join(t.TempDir(), "nope"))
	require.Error(t, err)
}
