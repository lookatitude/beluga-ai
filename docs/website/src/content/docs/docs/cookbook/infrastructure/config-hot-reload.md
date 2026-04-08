---
title: "Config Hot-Reload in Production"
description: "Recipe for updating Go service configuration at runtime without restarts — rotate model settings, toggle features, and adjust limits with zero downtime."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, config hot-reload, Go zero downtime, runtime configuration, feature flags, production recipe"
---

## Problem

You need to update configuration values (model settings, feature flags, resource limits) in a running production service without restarting the application or losing active requests.

## Solution

Use `config.NewFileWatcher` to poll a JSON configuration file for changes and reload atomically using `config.Load[T]`. Validate the new configuration with `config.Validate` before replacing the current one so invalid files never disrupt a running service.

## Why This Matters

Restarting services to apply config changes closes active connections and terminates in-flight requests. For AI systems processing long-running agent workflows or streaming responses, this disruption is costly. Hot-reload lets you adjust rate limits under load, switch model names, or toggle feature flags without a deployment window.

The validate-before-apply pattern ensures that a corrupted or incomplete config file never replaces the working configuration. The old config stays active until a valid replacement arrives.

**Security note:** Never store sensitive credentials (API keys, passwords, tokens) in config files on disk. Use environment variables or a secrets manager for credentials and load them via `config.LoadFromEnv` or `config.MergeEnv`. Hot-reload config files should contain only non-sensitive operational settings.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/config"
)

// AppConfig is the application-specific configuration type.
// Use JSON struct tags; config.Load[T] reads .json files.
// Sensitive values (API keys) must come from environment variables, not this file.
type AppConfig struct {
	Model     string `json:"model"      default:"gpt-4o"`
	MaxTokens int    `json:"max_tokens" default:"2048" min:"1" max:"32768"`
	RateLimit int    `json:"rate_limit" default:"60"   min:"1" max:"10000"`
}

// ConfigReloader manages hot-reloading of typed configuration.
type ConfigReloader[T any] struct {
	mu      sync.RWMutex
	current T
	path    string
}

// NewConfigReloader loads the initial config and starts watching the file.
// The returned cancel function stops the watcher.
func NewConfigReloader[T any](ctx context.Context, path string) (*ConfigReloader[T], context.CancelFunc, error) {
	cfg, err := config.Load[T](path)
	if err != nil {
		return nil, nil, fmt.Errorf("initial config load: %w", err)
	}

	cr := &ConfigReloader[T]{current: cfg, path: path}

	watchCtx, cancel := context.WithCancel(ctx)
	watcher := config.NewFileWatcher(path, 2*time.Second)

	go func() {
		if err := watcher.Watch(watchCtx, func(newRaw any) {
			cr.reload()
		}); err != nil && watchCtx.Err() == nil {
			slog.Error("config watcher stopped unexpectedly", "error", err)
		}
		watcher.Close()
	}()

	return cr, cancel, nil
}

// Get returns the current configuration. Safe for concurrent use.
func (cr *ConfigReloader[T]) Get() T {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.current
}

// reload reads the config file, validates, and atomically replaces the current config.
func (cr *ConfigReloader[T]) reload() {
	newCfg, err := config.Load[T](cr.path)
	if err != nil {
		slog.Warn("config reload failed — keeping previous config", "error", err)
		return
	}

	cr.mu.Lock()
	cr.current = newCfg
	cr.mu.Unlock()

	slog.Info("config reloaded", "path", cr.path)
}

func main() {
	ctx := context.Background()

	reloader, cancel, err := NewConfigReloader[AppConfig](ctx, "./config.json")
	if err != nil {
		slog.Error("config reloader failed to start", "error", err)
		return
	}
	defer cancel()

	cfg := reloader.Get()
	fmt.Printf("Loaded config: model=%s max_tokens=%d\n", cfg.Model, cfg.MaxTokens)

	// Config is automatically reloaded when config.json changes on disk.
	// Call reloader.Get() on every request to pick up the latest values.
	select {}
}
```

## Explanation

1. **`config.Load[T](path)`** — Generic loader that reads a `.json` file, applies `default` struct tag values for zero-value fields, checks `required:"true"` tags, and enforces `min`/`max` constraints on numeric fields. Returns a typed value, not a map or `any`.

2. **`config.NewFileWatcher(path, interval)`** — Returns a `config.Watcher` that polls the file every `interval` using SHA-256 hashing to detect changes without re-reading identical files on every tick.

3. **`watcher.Watch(ctx, callback)`** — Blocks until `ctx` is cancelled. The callback receives the raw file bytes as `any`. Because `config.Load[T]` reads from disk by path, the callback simply triggers a reload rather than parsing the bytes directly.

4. **Validate-before-apply** — `config.Load[T]` always validates before returning. An invalid config file returns an error; the reloader logs a warning and keeps the previous config active.

5. **`sync.RWMutex` for atomic updates** — Multiple concurrent readers can call `Get()` without blocking each other. The reload goroutine acquires an exclusive write lock only during the pointer swap, keeping the critical section minimal.

## Testing

```go
import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigReloader_HotReload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	initial := AppConfig{Model: "gpt-4o", MaxTokens: 1024, RateLimit: 60}
	writeJSON(t, cfgPath, initial)

	ctx := context.Background()
	reloader, cancel, err := NewConfigReloader[AppConfig](ctx, cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cancel()

	if reloader.Get().Model != "gpt-4o" {
		t.Fatal("initial config not loaded")
	}

	updated := AppConfig{Model: "gpt-4o-mini", MaxTokens: 2048, RateLimit: 120}
	writeJSON(t, cfgPath, updated)

	// Wait for the watcher to detect and apply the change.
	time.Sleep(3 * time.Second)

	if got := reloader.Get().Model; got != "gpt-4o-mini" {
		t.Errorf("expected gpt-4o-mini after reload, got %s", got)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
```

## Variations

### Watch Multiple Files

Run one `ConfigReloader` per file and merge results at the call site:

```go
type MultiSourceConfig struct {
	Base      AppConfig
	Overrides AppConfig
}
```

### Config Versioning

Track which config version is active for audit logs:

```go
type VersionedConfig[T any] struct {
	Version int
	Loaded  time.Time
	Config  T
}
```

## Related Recipes

- **[Masking Secrets in Logs](./config-secret-masking)** — Log configuration values without leaking sensitive data
