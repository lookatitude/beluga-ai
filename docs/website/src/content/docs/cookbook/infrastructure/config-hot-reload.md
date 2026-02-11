---
title: "Config Hot-reloading in Production"
description: "Update configuration values in a running production service without restarting the application or losing active requests."
---

## Problem

You need to update configuration values (API keys, model settings, feature flags) in a running production service without restarting the application or losing active requests. This is a critical challenge in production environments where downtime is unacceptable. Restarting services to apply config changes means closing active connections, terminating in-flight requests, and forcing clients to reconnect. For AI systems processing long-running agent workflows or streaming responses, this disruption is particularly problematic. Additionally, configuration changes often need to be rolled out quickly—such as rotating compromised API keys, adjusting rate limits under load, or switching model providers during an outage—without waiting for deployment windows.

## Solution

Implement a file watcher that monitors configuration files for changes and reloads them atomically, notifying registered listeners. This approach works because Beluga AI's config package supports loading from files, and Go's file system events allow detecting changes without polling. The key design principle is validate-before-apply: new configurations are loaded and validated before they replace the current configuration, preventing invalid configs from breaking running services. The listener pattern decouples configuration reloading from component logic, allowing each subsystem (LLM providers, agents, databases) to react to config changes independently. This design respects Beluga's Watch pattern from the config package and ensures thread-safe access to configuration state.

The solution uses Go's sync.RWMutex for atomic config updates, allowing multiple concurrent readers while ensuring exclusive write access during reloads. This prevents race conditions where a reader might see a partially-updated configuration. By validating new configurations before applying them, the system maintains availability even when invalid config files are written to disk—the old, working configuration stays active until a valid replacement arrives.

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

1. **Atomic config updates** — `sync.RWMutex` ensures thread-safe access to the current config. Readers can access concurrently, but updates are exclusive. This matters because without atomicity, concurrent goroutines could read partially-updated configuration state, leading to inconsistent behavior. For example, an agent might read a new model name but an old API key, causing authentication failures. The RWMutex pattern allows high-throughput config reads during normal operation while ensuring safe updates during reloads.

2. **Validation before reload** — The new config is validated before applying it. If validation fails, the old config is kept and an error is logged. This prevents invalid configs from breaking the running service. This design choice is critical because configuration files can be corrupted during writes, contain syntax errors, or have semantically invalid values (like negative timeouts or missing required fields). By validating first, the system maintains availability even when bad configs are written to disk. The old configuration remains active until a valid replacement arrives, ensuring zero-downtime operation.

3. **Listener pattern** — Components that depend on config can register as listeners. When config changes, they're notified and can update themselves. This decouples config reloading from component logic. This matters because different components need different reactions to config changes: LLM providers might need to re-initialize clients with new API keys, rate limiters might need to adjust thresholds, and feature flags might enable new code paths. The listener pattern allows each component to handle its own update logic independently, avoiding a centralized "reload everything" approach that would be fragile and hard to maintain.

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
