package devloop

import (
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// ChangeFilter decides whether a filesystem event merits a rebuild. The
// supervisor consults it on every event it receives from the watcher; the
// debounce timer only starts once an event is accepted.
type ChangeFilter interface {
	Accept(path string, op fsnotify.Op) bool
}

// GoSourceFilter accepts only writes/creates/renames of .go files,
// excluding editor scratch files (files beginning with ".#" used by
// Emacs, "~" suffixes used by many editors, and any path component
// beginning with "." such as .git, .idea, vendor/ caches).
type GoSourceFilter struct{}

// Accept reports whether the given path+op should trigger a rebuild.
func (GoSourceFilter) Accept(path string, op fsnotify.Op) bool {
	if op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
		return false
	}
	if filepath.Ext(path) != ".go" {
		return false
	}
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".#") || strings.HasSuffix(base, "~") {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "" || part == "." || part == ".." {
			continue
		}
		if strings.HasPrefix(part, ".") {
			return false
		}
		if part == "vendor" || part == "node_modules" {
			return false
		}
	}
	return true
}
