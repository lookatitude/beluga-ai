package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"time"
)

// Watcher watches for configuration changes and invokes a callback when
// the configuration is updated. Implementations may poll files, watch
// key-value stores, or subscribe to change notifications.
type Watcher interface {
	// Watch starts watching for changes and calls callback whenever the
	// configuration changes. It blocks until ctx is cancelled or an
	// unrecoverable error occurs.
	Watch(ctx context.Context, callback func(newConfig any)) error

	// Close releases resources held by the watcher.
	Close() error
}

// WatchConfig holds configuration for watchers.
type WatchConfig struct {
	// Path is the configuration file to watch.
	Path string

	// Interval is the polling interval for file-based watchers.
	Interval time.Duration
}

// FileWatcher polls a file at a regular interval and invokes a callback
// when the file content changes. Change detection uses SHA-256 hashing
// of file contents.
type FileWatcher struct {
	path     string
	interval time.Duration

	mu       sync.Mutex
	lastHash [sha256.Size]byte
	closed   bool
	done     chan struct{}
}

// NewFileWatcher creates a FileWatcher that polls path every interval for
// changes. The minimum interval is 100ms; smaller values are clamped.
func NewFileWatcher(path string, interval time.Duration) Watcher {
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	return &FileWatcher{
		path:     path,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Watch polls the file for changes until ctx is cancelled. When a change
// is detected, callback is invoked with the raw file content as a []byte.
// The caller can unmarshal the data as needed.
func (w *FileWatcher) Watch(ctx context.Context, callback func(newConfig any)) error {
	// Compute initial hash so we only fire on actual changes.
	data, err := os.ReadFile(w.path)
	if err != nil {
		return fmt.Errorf("config: watch initial read %s: %w", w.path, err)
	}

	w.mu.Lock()
	w.lastHash = sha256.Sum256(data)
	w.mu.Unlock()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.done:
			return nil
		case <-ticker.C:
			w.mu.Lock()
			if w.closed {
				w.mu.Unlock()
				return nil
			}
			w.mu.Unlock()

			data, err := os.ReadFile(w.path)
			if err != nil {
				// File temporarily unavailable — skip this tick.
				continue
			}

			hash := sha256.Sum256(data)
			w.mu.Lock()
			changed := hash != w.lastHash
			if changed {
				w.lastHash = hash
			}
			w.mu.Unlock()

			if changed {
				callback(data)
			}
		}
	}
}

// Close stops the watcher. It is safe to call Close concurrently with Watch.
func (w *FileWatcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.closed = true
		close(w.done)
	}
	return nil
}
