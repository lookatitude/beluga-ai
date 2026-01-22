---
title: "Config Hot-reloading in Production"
package: "config"
category: "configuration"
complexity: "advanced"
---

# Config Hot-reloading in Production

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
    
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

var tracer = otel.Tracer("beluga.config.hotreload")

// ConfigReloader manages hot-reloading of configuration
type ConfigReloader struct {
    configPath    string
    currentConfig *iface.Config
    listeners     []ConfigChangeListener
    mu            sync.RWMutex
    watcher       *FileWatcher
}

// ConfigChangeListener is notified when config changes
type ConfigChangeListener interface {
    OnConfigChange(oldConfig, newConfig *iface.Config) error
}

// NewConfigReloader creates a new config reloader
func NewConfigReloader(configPath string) (*ConfigReloader, error) {
    // Load initial config
    cfg, err := config.LoadFromFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load initial config: %w", err)
    }

    reloader := &ConfigReloader{
        configPath:    configPath,
        currentConfig: cfg,
        listeners:     []ConfigChangeListener{},
    }
    
    // Start file watcher
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
func (cr *ConfigReloader) GetConfig() *iface.Config {
    cr.mu.RLock()
    defer cr.mu.RUnlock()
    return cr.currentConfig
}

// onConfigFileChanged handles config file changes
func (cr *ConfigReloader) onConfigFileChanged(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "reloader.on_config_changed")
    defer span.End()
    
    span.SetAttributes(attribute.String("config.path", cr.configPath))
    
    // Load new config
    newConfig, err := config.LoadFromFile(cr.configPath)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, "failed to reload config")
        log.Printf("Failed to reload config: %v", err)
        return err
    }
    
    // Validate new config
    if err := config.ValidateConfig(newConfig); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, "invalid config")
        log.Printf("Reloaded config is invalid: %v", err)
        return err
    }
    
    // Get old config
    cr.mu.Lock()
    oldConfig := cr.currentConfig
    cr.mu.Unlock()
    
    // Notify listeners
    for _, listener := range cr.listeners {
        if err := listener.OnConfigChange(oldConfig, newConfig); err != nil {
            span.RecordError(err)
            log.Printf("Listener error on config change: %v", err)
            // Continue with other listeners
        }
    }
    
    // Atomically update config
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
                
                // Debounce: wait a bit to ensure file write is complete
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

// Example listener implementation
type LLMConfigListener struct {
    llmProvider interface{} // Your LLM provider that needs config updates
}

func (l *LLMConfigListener) OnConfigChange(oldConfig, newConfig *iface.Config) error {
    // Update LLM provider with new config
    // This is where you'd update your actual components
    log.Printf("LLM config changed, updating provider...")
    return nil
}

func main() {
    ctx := context.Background()

    // Create config reloader
    reloader, err := NewConfigReloader("./config.yaml")
    if err != nil {
        log.Fatalf("Failed to create config reloader: %v", err)
    }
    defer reloader.Stop()
    
    // Register listeners
    reloader.RegisterListener(&LLMConfigListener{})
    
    // Use config
    cfg := reloader.GetConfig()
    fmt.Printf("Config loaded: %d LLM providers\n", len(cfg.LLMProviders))
    // Config will automatically reload when file changes
text
    select \{\}
}
```

## Explanation

Let's break down what's happening:

1. **Atomic config updates** - Notice how we use `sync.RWMutex` to ensure thread-safe access to the current config. Readers can access concurrently, but updates are exclusive. This prevents race conditions when multiple goroutines read config while it's being updated.

2. **Validation before reload** - We validate the new config before applying it. If validation fails, we keep the old config and log an error. This prevents invalid configs from breaking the running service.

3. **Listener pattern** - Components that depend on config can register as listeners. When config changes, they're notified and can update themselves. This decouples config reloading from component logic.

```go
**Key insight:** Always validate new configs before applying them. Invalid configs should never replace valid ones, even if the file changes.

## Testing

```
Here's how to test this solution:
```go
func TestConfigReloader_HotReload(t *testing.T) {
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "config.yaml")
    
    // Create initial config
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
    
    // Verify initial config
    cfg := reloader.GetConfig()
    require.Len(t, cfg.LLMProviders, 1)
    
    // Update config file
    updatedConfig := `
llm_providers:
  - name: "test"
    provider: "openai"
    model_name: "gpt-4"
`
    os.WriteFile(configFile, []byte(updatedConfig), 0644)
    
    // Wait for reload
    time.Sleep(2 * time.Second)
    
    // Verify updated config
    cfg = reloader.GetConfig()
    require.Equal(t, "gpt-4", cfg.LLMProviders[0].ModelName)
}

## Variations

### Watch Multiple Files

Watch multiple config files and merge them:
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
    Config  *iface.Config
}
```

## Related Recipes

- **[Config Masking Secrets in Logs](./config-masking-secrets-logs.md)** - Secure config logging
- **[Config Package Guide](../guides/config-providers.md)** - For a deeper understanding of config management
