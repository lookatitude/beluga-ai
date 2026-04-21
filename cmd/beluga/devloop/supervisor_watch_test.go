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
	dirs := []string{
		"pkg/a",
		"pkg/b/nested",
		".git/objects",
		".idea/inspectionProfiles",
		"vendor/dep1",
		"node_modules/foo",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if err := addRecursive(w, root); err != nil {
		t.Fatalf("addRecursive: %v", err)
	}

	watched := map[string]bool{}
	for _, p := range w.WatchList() {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			continue
		}
		watched[rel] = true
	}
	// Expected: root, pkg, pkg/a, pkg/b, pkg/b/nested.
	for _, want := range []string{".", "pkg", filepath.Join("pkg", "a"), filepath.Join("pkg", "b"), filepath.Join("pkg", "b", "nested")} {
		if !watched[want] {
			t.Errorf("addRecursive: missing watched dir %q (got %v)", want, keysOf(watched))
		}
	}
	// Disallowed: anything under .git, .idea, vendor, node_modules.
	for got := range watched {
		for _, bad := range []string{".git", ".idea", "vendor", "node_modules"} {
			if got == bad || strings.HasPrefix(got, bad+string(os.PathSeparator)) {
				t.Errorf("addRecursive: unexpectedly watched %q (under %q)", got, bad)
			}
		}
	}
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
