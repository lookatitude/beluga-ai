package loader

import (
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
)

// cleanPath validates and normalises a filesystem path supplied to a loader.
// It rejects paths that, after cleaning, still contain a ".." segment — a
// common indicator of a path-traversal attempt. The cleaned path is returned
// for use by the caller.
//
// Loaders are explicitly file-I/O shaped by design (their whole purpose is
// to read a user-supplied path), so gosec G304 is acceptable at the callsite
// as long as the input is cleaned. Callers should add a
// `// #nosec G304 -- path validated by cleanPath` comment next to the
// os.ReadFile / os.Open call.
func cleanPath(source string) (string, error) {
	if source == "" {
		return "", core.Errorf(core.ErrInvalidInput, "loader: empty path")
	}
	cleaned := filepath.Clean(source)
	// After Clean, any remaining ".." element means the caller tried to
	// escape an ancestor directory. Reject these outright.
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return "", core.Errorf(core.ErrInvalidInput, "loader: path %q contains parent-directory traversal", source)
		}
	}
	return cleaned, nil
}
