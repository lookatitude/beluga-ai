package devloop

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/fsnotify/fsnotify"
)

// filterFunc adapts a function to ChangeFilter for tests.
type filterFunc func(path string, op fsnotify.Op) bool

func (f filterFunc) Accept(path string, op fsnotify.Op) bool { return f(path, op) }

func newTestWatcher(t *testing.T) *fsnotify.Watcher {
	t.Helper()
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// TestAcceptEvent_RejectedByFilter asserts a filtered-out event is ignored
// and never registers anything with the watcher.
func TestAcceptEvent_RejectedByFilter(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t)
	s := &supervisor{cfg: Config{Filter: filterFunc(func(string, fsnotify.Op) bool { return false })}}

	if s.acceptEvent(w, fsnotify.Event{Name: dir, Op: fsnotify.Create}) {
		t.Fatal("acceptEvent = true for an event the filter rejects")
	}
	if got := w.WatchList(); len(got) != 0 {
		t.Fatalf("watch list = %v, want empty", got)
	}
}

// TestAcceptEvent_NewDirectoryIsWatched asserts an accepted Create event
// for a directory adds that directory to the watcher.
func TestAcceptEvent_NewDirectoryIsWatched(t *testing.T) {
	sub := filepath.Join(t.TempDir(), "pkg")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	w := newTestWatcher(t)
	s := &supervisor{cfg: Config{Filter: filterFunc(func(string, fsnotify.Op) bool { return true })}}

	if !s.acceptEvent(w, fsnotify.Event{Name: sub, Op: fsnotify.Create}) {
		t.Fatal("acceptEvent = false for an accepted event")
	}
	if got := w.WatchList(); !slices.Contains(got, sub) {
		t.Fatalf("watch list = %v, want it to contain %q", got, sub)
	}
}

// TestAcceptEvent_FileWriteNotWatched asserts an accepted event for a
// regular file triggers a rebuild without adding the file to the watcher.
func TestAcceptEvent_FileWriteNotWatched(t *testing.T) {
	f := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	w := newTestWatcher(t)
	s := &supervisor{cfg: Config{Filter: GoSourceFilter{}}}

	for _, op := range []fsnotify.Op{fsnotify.Create, fsnotify.Write} {
		if !s.acceptEvent(w, fsnotify.Event{Name: f, Op: op}) {
			t.Fatalf("acceptEvent(%v) = false for a .go file", op)
		}
	}
	if got := w.WatchList(); len(got) != 0 {
		t.Fatalf("watch list = %v, want empty", got)
	}
}
