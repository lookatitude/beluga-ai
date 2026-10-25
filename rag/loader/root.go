package loader

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// rootedLoader confines a file-based loader to one directory tree.
type rootedLoader struct {
	inner DocumentLoader
	root  string
}

// WithinRoot wraps a file-based DocumentLoader (text, markdown, JSON, CSV)
// so that it only reads files inside root. A relative source is resolved
// against root; an absolute source must already lie inside it. Sources that
// escape root, lexically or through a symbolic link, are rejected with
// core.ErrInvalidInput before the inner loader runs.
//
// Use it whenever load sources are supplied by users or LLM tools: the
// file loaders on their own accept any absolute path.
func WithinRoot(inner DocumentLoader, root string) (DocumentLoader, error) {
	if inner == nil {
		return nil, core.Errorf(core.ErrInvalidInput, "loader: WithinRoot requires a loader")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "loader: root %q: %w", root, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "loader: root %q: %w", root, err)
	}
	return &rootedLoader{inner: inner, root: resolved}, nil
}

// Load resolves source inside the root and delegates to the inner loader.
func (l *rootedLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	path, err := l.resolve(source)
	if err != nil {
		return nil, err
	}
	return l.inner.Load(ctx, path)
}

// resolve returns the absolute, symlink-resolved path for source, or an
// error when it does not stay inside the root.
func (l *rootedLoader) resolve(source string) (string, error) {
	if source == "" {
		return "", core.Errorf(core.ErrInvalidInput, "loader: empty path")
	}
	path := source
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.root, path)
	}
	path = filepath.Clean(path)
	if !l.contains(path) {
		return "", escapeErr(source)
	}
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		if !l.contains(resolved) {
			return "", escapeErr(source)
		}
		return resolved, nil
	case errors.Is(err, fs.ErrNotExist):
		// Let the inner loader report the missing file.
		return path, nil
	default:
		return "", core.Errorf(core.ErrInvalidInput, "loader: resolve %q: %w", source, err)
	}
}

func (l *rootedLoader) contains(path string) bool {
	rel, err := filepath.Rel(l.root, path)
	return err == nil && (rel == "." || filepath.IsLocal(rel))
}

func escapeErr(source string) error {
	return core.Errorf(core.ErrInvalidInput, "loader: path %q is outside the allowed root", source)
}
