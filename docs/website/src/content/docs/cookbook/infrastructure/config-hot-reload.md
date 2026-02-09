---
title: "Config Hot-reloading in Production"
description: "Update configuration values in a running production service without restarting the application or losing active requests."
---

## Problem

You need to update configuration values (API keys, model settings, feature flags) in a running production service without restarting the application or losing active requests.

## Solution

Implement a file watcher that monitors configuration files for changes and reloads them atomically, notifying registered listeners. This works because Beluga AI's config package supports loading from files, and Go's file system events allow detecting changes without polling.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "sync"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/config"
)

var tracer = otel.Tracer("beluga.config.hotreload")

// ConfigReloader manages hot-reloading of configuration
type ConfigReloader struct {
    configPath    string
    currentConfig *config.Config
    listeners     []ConfigChangeListener
    mu            sync.RWMutex
    watcher       *FileWatcher
}

// ConfigChangeListener is notified when config changes
type ConfigChangeListener interface {
    OnConfigChange(oldConfig, newConfig *config.Config) error
}

// NewConfigReloader creates a new config reloader
func NewConfigReloader(configPath string) (*ConfigReloader, error) {
    cfg, err := config.LoadFromFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load initial config: %w", err)
    }

    reloader := &ConfigReloader{
        configPath:    configPath,
        currentConfig: cfg,
        listeners:     []ConfigChangeListener{},
    }

    watcher, err := NewFileWatcher(configPath, reloader.onConfigFileChanged)
    if err != nil {
        return nil, fmt.Errorf("failed to create file watcher: %w", err)
    }
    reloader.watcher = watcher

    return reloader, nil
}

// RegisterListener registers a listener for config changes
func (cr *ConfigReloader) RegisterListener(listener ConfigChangeListener) {
    cr.mu.Lock()
    defer cr.mu.Unlock()
    cr.listeners = append(cr.listeners, listener)
}

// GetConfig returns the current configuration (thread-safe)
func (cr *ConfigReloader) GetConfig() *config.Config {
    cr.mu.RLock()
    defer cr.mu.RUnlock()
    return cr.currentConfig
}

// onConfigFileChanged handles config file changes
func (cr *ConfigReloader) onConfigFileChanged(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "reloader.on_config_changed")
    defer span.End()

    span.SetAttributes(attribute.String("config.path", cr.configPath))

    newConfig, err := config.LoadFromFile(cr.configPath)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, "failed to reload config")
        log.Printf("Failed to reload config: %v", err)
        return err
    }

    if err := config.ValidateConfig(newConfig); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, "invalid config")
        log.Printf("Reloaded config is invalid: %v", err)
        return err
    }

    cr.mu.Lock()
    oldConfig := cr.currentConfig
    cr.mu.Unlock()

    for _, listener := range cr.listeners {
        if err := listener.OnConfigChange(oldConfig, newConfig); err != nil {
            span.RecordError(err)
            log.Printf("Listener error on config change: %v", err)
        }
    }

    cr.mu.Lock()
    cr.currentConfig = newConfig
    cr.mu.Unlock()

    span.SetStatus(trace.StatusOK, "config reloaded successfully")
    log.Printf("Config reloaded successfully from %s", cr.configPath)

    return nil
}

// Stop stops the config reloader
func (cr *ConfigReloader) Stop() {
    if cr.watcher != nil {
        cr.watcher.Stop()
    }
}

// FileWatcher watches a file for changes
type FileWatcher struct {
    filePath string
    callback func(context.Context) error
    stopCh   chan struct{}
    doneCh   chan struct{}
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(filePath string, callback func(context.Context) error) (*FileWatcher, error) {
    watcher := &FileWatcher{
        filePath: filePath,
        callback: callback,
        stopCh:   make(chan struct{}),
        doneCh:   make(chan struct{}),
    }

    go watcher.watch()
    return watcher, nil
}

// watch monitors the file for changes
func (fw *FileWatcher) watch() {
    defer close(fw.doneCh)

    ctx := context.Background()
    lastModTime := time.Time{}

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-fw.stopCh:
            return
        case <-ticker.C:
            info, err := os.Stat(fw.filePath)
            if err != nil {
                continue
            }

            modTime := info.ModTime()
            if modTime.After(lastModTime) {
                lastModTime = modTime

                // Debounce: wait to ensure file write is complete
                time.Sleep(100 * time.Millisecond)

                if err := fw.callback(ctx); err != nil {
                    log.Printf("Error in config change callback: %v", err)
                }
            }
        }
    }
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() {
    close(fw.stopCh)
    <-fw.doneCh
}

// LLMConfigListener updates LLM providers when config changes
type LLMConfigListener struct {
    llmProvider interface{}
}

func (l *LLMConfigListener) OnConfigChange(oldConfig, newConfig *config.Config) error {
    log.Printf("LLM config changed, updating provider...")
    return nil
}

func main() {
    reloader, err := NewConfigReloader("./config.yaml")
    if err != nil {
        log.Fatalf("Failed to create config reloader: %v", err)
    }
    defer reloader.Stop()

    reloader.RegisterListener(&LLMConfigListener{})

    cfg := reloader.GetConfig()
    fmt.Printf("Config loaded successfully\n")

    // Config will automatically reload when file changes
    select {}
}
```

## Explanation

1. **Atomic config updates** — `sync.RWMutex` ensures thread-safe access to the current config. Readers can access concurrently, but updates are exclusive. This prevents race conditions when multiple goroutines read config while it's being updated.

2. **Validation before reload** — The new config is validated before applying it. If validation fails, the old config is kept and an error is logged. This prevents invalid configs from breaking the running service.

3. **Listener pattern** — Components that depend on config can register as listeners. When config changes, they're notified and can update themselves. This decouples config reloading from component logic.

Always validate new configs before applying them. Invalid configs should never replace valid ones, even if the file changes.

## Testing

```go
func TestConfigReloader_HotReload(t *testing.T) {
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "config.yaml")

    initialConfig := `
llm_providers:
  - name: "test"
    provider: "openai"
    model_name: "gpt-3.5-turbo"
`
    os.WriteFile(configFile, []byte(initialConfig), 0644)

    reloader, err := NewConfigReloader(configFile)
    require.NoError(t, err)
    defer reloader.Stop()

    cfg := reloader.GetConfig()
    require.Len(t, cfg.LLMProviders, 1)

    updatedConfig := `
llm_providers:
  - name: "test"
    provider: "openai"
    model_name: "gpt-4"
`
    os.WriteFile(configFile, []byte(updatedConfig), 0644)

    // Wait for reload
    time.Sleep(2 * time.Second)

    cfg = reloader.GetConfig()
    require.Equal(t, "gpt-4", cfg.LLMProviders[0].ModelName)
}
```

## Variations

### Watch Multiple Files

Watch multiple config files and merge them:

```go
type MultiFileReloader struct {
    files []string
    // ...
}
```

### Config Versioning

Track config versions and rollback on errors:

```go
type VersionedConfig struct {
    Version int
    Config  *config.Config
}
```

## Related Recipes

- **[Masking Secrets in Logs](./config-secret-masking)** — Secure config logging
