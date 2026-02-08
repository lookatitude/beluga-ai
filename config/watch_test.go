package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewFileWatcher_MinimumInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Interval below 100ms should be clamped.
	w := NewFileWatcher(path, 10*time.Millisecond)
	fw, ok := w.(*FileWatcher)
	if !ok {
		t.Fatal("expected *FileWatcher")
	}
	if fw.interval != 100*time.Millisecond {
		t.Errorf("interval = %v, want %v", fw.interval, 100*time.Millisecond)
	}
}

func TestNewFileWatcher_NormalInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 500*time.Millisecond)
	fw, ok := w.(*FileWatcher)
	if !ok {
		t.Fatal("expected *FileWatcher")
	}
	if fw.interval != 500*time.Millisecond {
		t.Errorf("interval = %v, want %v", fw.interval, 500*time.Millisecond)
	}
}

func TestFileWatcher_Watch_DetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"version": 1}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)
	defer w.Close()

	var mu sync.Mutex
	var received []byte

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- w.Watch(ctx, func(newConfig any) {
			mu.Lock()
			defer mu.Unlock()
			if data, ok := newConfig.([]byte); ok {
				received = make([]byte, len(data))
				copy(received, data)
			}
		})
	}()

	// Wait for initial hash computation.
	time.Sleep(150 * time.Millisecond)

	// Update the file.
	if err := os.WriteFile(path, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to detect the change.
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	data := received
	mu.Unlock()

	if data == nil {
		t.Fatal("expected callback to be invoked with new config")
	}
	if string(data) != `{"version": 2}` {
		t.Errorf("received data = %q, want %q", string(data), `{"version": 2}`)
	}

	cancel()
	<-watchDone
}

func TestFileWatcher_Watch_NoChangeNoCallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"stable": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)
	defer w.Close()

	callCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- w.Watch(ctx, func(newConfig any) {
			callCount++
		})
	}()

	<-watchDone

	if callCount != 0 {
		t.Errorf("callback called %d times, expected 0 (no changes)", callCount)
	}
}

func TestFileWatcher_Watch_FileNotFound(t *testing.T) {
	w := NewFileWatcher("/nonexistent/path/config.json", 100*time.Millisecond)
	err := w.Watch(context.Background(), func(newConfig any) {})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileWatcher_Watch_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- w.Watch(ctx, func(newConfig any) {})
	}()

	// Let Watch start up.
	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case err := <-watchDone:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not return after context cancellation")
	}
}

func TestFileWatcher_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- w.Watch(context.Background(), func(newConfig any) {})
	}()

	// Let Watch start.
	time.Sleep(50 * time.Millisecond)

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	select {
	case err := <-watchDone:
		if err != nil {
			t.Errorf("Watch returned unexpected error after Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not return after Close")
	}
}

func TestFileWatcher_Close_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)

	// Close multiple times should not panic.
	if err := w.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close() error: %v", err)
	}
}

func TestFileWatcher_Watch_MultipleChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"v": 0}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)
	defer w.Close()

	var mu sync.Mutex
	var callCount int

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- w.Watch(ctx, func(newConfig any) {
			mu.Lock()
			callCount++
			mu.Unlock()
		})
	}()

	// Wait for initial read.
	time.Sleep(150 * time.Millisecond)

	// Write multiple changes.
	for i := 1; i <= 3; i++ {
		if err := os.WriteFile(path, []byte(`{"v": `+string(rune('0'+i))+`}`), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count < 2 {
		t.Errorf("expected at least 2 callbacks for 3 changes, got %d", count)
	}

	cancel()
	<-watchDone
}

func TestFileWatcher_Watch_FileTemporarilyUnavailable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"initial": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(path, 100*time.Millisecond)
	defer w.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var mu sync.Mutex
	var lastData []byte

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- w.Watch(ctx, func(newConfig any) {
			mu.Lock()
			defer mu.Unlock()
			if data, ok := newConfig.([]byte); ok {
				lastData = make([]byte, len(data))
				copy(lastData, data)
			}
		})
	}()

	// Wait for initial read.
	time.Sleep(150 * time.Millisecond)

	// Remove file temporarily.
	os.Remove(path)
	time.Sleep(200 * time.Millisecond)

	// Restore with new content.
	if err := os.WriteFile(path, []byte(`{"restored": true}`), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	data := lastData
	mu.Unlock()

	if data == nil {
		t.Fatal("expected callback after file restoration")
	}
	if string(data) != `{"restored": true}` {
		t.Errorf("received data = %q, want %q", string(data), `{"restored": true}`)
	}

	cancel()
	<-watchDone
}

func TestWatcher_InterfaceCompliance(t *testing.T) {
	// Verify FileWatcher implements Watcher at compile time.
	var _ Watcher = (*FileWatcher)(nil)
}
