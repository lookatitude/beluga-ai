package devloop

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TestAddRecursive_SkipsHiddenAndVendor asserts the watcher walk skips
// the three expensive-to-watch classes of directories (hidden dirs,
// vendor/, node_modules/) while still registering normal subtrees.
func TestAddRecursive_SkipsHiddenAndVendor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mkdirs(t, root, []string{
		"pkg/a",
		"pkg/b/nested",
		".git/objects",
		".idea/inspectionProfiles",
		"vendor/dep1",
		"node_modules/foo",
	})

	w := newWatcher(t)
	if err := addRecursive(w, root); err != nil {
		t.Fatalf("addRecursive: %v", err)
	}
	watched := watchedRelPaths(w, root)

	// Expected: root, pkg, pkg/a, pkg/b, pkg/b/nested.
	wantAll := []string{".", "pkg", filepath.Join("pkg", "a"), filepath.Join("pkg", "b"), filepath.Join("pkg", "b", "nested")}
	for _, want := range wantAll {
		if !watched[want] {
			t.Errorf("addRecursive: missing watched dir %q (got %v)", want, keysOf(watched))
		}
	}
	// Disallowed: anything under .git, .idea, vendor, node_modules.
	for got := range watched {
		if bad := underForbidden(got, []string{".git", ".idea", "vendor", "node_modules"}); bad != "" {
			t.Errorf("addRecursive: unexpectedly watched %q (under %q)", got, bad)
		}
	}
}

// mkdirs creates each relative path under root or fatals the test.
func mkdirs(t *testing.T, root string, dirs []string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
}

// newWatcher returns a fresh fsnotify.Watcher registered for cleanup.
func newWatcher(t *testing.T) *fsnotify.Watcher {
	t.Helper()
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// watchedRelPaths materialises w.WatchList() as a set of paths relative
// to root so test assertions can be path-independent.
func watchedRelPaths(w *fsnotify.Watcher, root string) map[string]bool {
	out := map[string]bool{}
	for _, p := range w.WatchList() {
		if rel, err := filepath.Rel(root, p); err == nil {
			out[rel] = true
		}
	}
	return out
}

// underForbidden returns the first forbidden prefix that path lies under,
// or "" if path is not under any of them. Keeping the "under?" check in
// one helper flattens the nested for/for/if structure in the caller.
func underForbidden(path string, forbidden []string) string {
	for _, bad := range forbidden {
		if path == bad || strings.HasPrefix(path, bad+string(os.PathSeparator)) {
			return bad
		}
	}
	return ""
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestRun_Watch_RebuildsOnSave drives runWithWatcher / startChild /
// restart end-to-end by appending a marker to main.go and asserting the
// fresh child prints a new token. Exercises the heart of `beluga dev`.
func TestRun_Watch_RebuildsOnSave(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain required")
	}
	if testing.Short() {
		t.Skip("watch test is a slow integration path")
	}

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	writeFile(t, filepath.Join(dir, "go.mod"), "module devloop_fixture_watch\n\ngo 1.25\n")
	writeFile(t, mainPath, `package main

import "fmt"

func main() { fmt.Println("FIRST_RUN") }
`)

	var (
		bufMu sync.Mutex
		buf   bytes.Buffer
	)
	stdout := &lockedBuf{mu: &bufMu, buf: &buf}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	seqSeen := make(chan int, 8)
	onRestart := func(seq int) {
		select {
		case seqSeen <- seq:
		default:
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{
			ProjectRoot: dir,
			Stdout:      stdout,
			Stderr:      io.Discard,
			Watch:       true,
			Filter:      GoSourceFilter{},
			Debounce:    50 * time.Millisecond,
			OnRestart:   onRestart,
		})
	}()

	// Wait for the first child to actually print before editing.
	if !waitForSubstring(&bufMu, &buf, "FIRST_RUN", 90*time.Second) {
		cancel()
		<-done
		t.Fatalf("first child never printed: stdout so far = %q", buf.String())
	}
	select {
	case <-seqSeen:
	case <-time.After(5 * time.Second):
		t.Fatalf("OnRestart(1) never fired")
	}

	// Rewrite main.go to trigger a rebuild.
	writeFile(t, mainPath, `package main

import "fmt"

func main() { fmt.Println("SECOND_RUN") }
`)

	if !waitForSubstring(&bufMu, &buf, "SECOND_RUN", 90*time.Second) {
		cancel()
		<-done
		t.Fatalf("rebuilt child never printed: stdout so far = %q", buf.String())
	}
	// At least one additional OnRestart must have fired.
	select {
	case <-seqSeen:
	case <-time.After(5 * time.Second):
		t.Fatalf("OnRestart never fired on rebuild")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && !isContextCanceled(err) {
			t.Errorf("Run returned %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestRun_Watch_MissingProjectRoot is a fast-failure path for
// runWithWatcher: a nonexistent ProjectRoot must surface as an error
// from the watcher-walk step, not a hang.
func TestRun_Watch_MissingProjectRoot(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := Run(ctx, Config{
		ProjectRoot: filepath.Join(t.TempDir(), "does-not-exist"),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
		Watch:       true,
		Filter:      GoSourceFilter{},
	})
	if err == nil {
		t.Fatal("want error for missing project root, got nil")
	}
}

// --- helpers ---

type lockedBuf struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func waitForSubstring(mu *sync.Mutex, buf *bytes.Buffer, want string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := buf.String()
		mu.Unlock()
		if strings.Contains(got, want) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func isContextCanceled(err error) bool {
	if err == nil {
		return false
	}
	// Accept wrapped context-canceled / killed-child variants as clean shutdown.
	s := err.Error()
	return strings.Contains(s, "context canceled") ||
		strings.Contains(s, "signal: killed") ||
		strings.Contains(s, "signal: terminated")
}
