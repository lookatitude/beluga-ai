# Phase 1: Quick Start Guide

**Feature**: Config Package Full Compliance  
**Date**: October 5, 2025  

This quickstart guide demonstrates the enhanced config package with full constitutional compliance, focusing on provider registry, health checks, enhanced validation, and comprehensive observability.

## Prerequisites

- Go 1.21+ installed
- Basic understanding of Go interfaces and struct tags
- Familiarity with configuration management concepts

## Basic Usage

### 1. Simple Configuration Loading

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Load configuration with enhanced options
    options := config.DefaultLoaderOptions()
    options.ConfigName = "app"
    options.ConfigPaths = []string{"./config", "/etc/app"}
    options.EnvPrefix = "APP"
    options.EnableHotReload = true
    options.EnableHealthChecks = true
    
    loader, err := config.NewLoader(options)
    if err != nil {
        log.Fatalf("Failed to create loader: %v", err)
    }
    
    // Load configuration with context
    cfg, err := loader.LoadConfig(ctx)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    fmt.Printf("Configuration loaded successfully\n")
    fmt.Printf("Version: %s\n", cfg.GetVersion())
    fmt.Printf("Sources: %v\n", cfg.GetSources())
}
```

### 2. Provider Registry Usage

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

func main() {
    ctx := context.Background()
    
    // Get global registry
    registry := config.GetGlobalRegistry()
    
    // Register a custom provider
    err := registry.RegisterProvider("custom", func(ctx context.Context, cfg interface{}) (iface.Provider, error) {
        return NewCustomProvider(cfg), nil
    })
    if err != nil {
        log.Fatalf("Failed to register provider: %v", err)
    }
    
    // Create provider instance
    customConfig := CustomProviderConfig{
        Source: "database",
        ConnectionString: "postgres://localhost/config",
    }
    
    provider, err := registry.CreateProvider(ctx, "custom", customConfig)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }
    
    // Check provider health
    health := provider.HealthCheck(ctx)
    fmt.Printf("Provider health: %s\n", health.Status)
    
    // Get system health
    systemHealth := registry.GetSystemHealth(ctx)
    fmt.Printf("System health: %s\n", systemHealth.OverallStatus)
    fmt.Printf("Healthy providers: %d\n", systemHealth.Components["healthy_count"])
}
```

### 3. Enhanced Validation

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

type AppConfig struct {
    DatabaseURL string `mapstructure:"database_url" validate:"required,url" env:"DATABASE_URL"`
    APIKey      string `mapstructure:"api_key" validate:"required,min=32" env:"API_KEY"`
    Port        int    `mapstructure:"port" validate:"min=1,max=65535" env:"PORT" default:"8080"`
    Debug       bool   `mapstructure:"debug" env:"DEBUG" default:"false"`
    
    // Enhanced validation with custom tags
    Timeout     time.Duration `mapstructure:"timeout" validate:"min=1s,max=5m" env:"TIMEOUT" default:"30s"`
    MaxRetries  int          `mapstructure:"max_retries" validate:"min=0,max=10" env:"MAX_RETRIES" default:"3"`
}

func main() {
    ctx := context.Background()
    
    // Create enhanced validator with custom rules
    validator, err := config.NewValidator(config.ValidatorOptions{
        StrictMode:        true,
        FailOnWarnings:    false,
        EnableSuggestions: true,
        EnableCrossField:  true,
    })
    if err != nil {
        log.Fatalf("Failed to create validator: %v", err)
    }
    
    // Register custom validation function
    validator.RegisterCustomValidator("positive_duration", func(ctx context.Context, field reflect.Value) error {
        if duration, ok := field.Interface().(time.Duration); ok {
            if duration <= 0 {
                return fmt.Errorf("duration must be positive")
            }
        }
        return nil
    })
    
    // Register cross-field validation rule
    validator.RegisterCrossFieldRule("timeout_retry_consistency", iface.CrossFieldRule{
        Name:        "timeout_retry_consistency",
        Description: "Timeout should be reasonable for the number of retries",
        Fields:      []string{"timeout", "max_retries"},
        Validator: func(ctx context.Context, config interface{}) error {
            cfg := config.(*AppConfig)
            if cfg.Timeout < time.Duration(cfg.MaxRetries)*time.Second {
                return fmt.Errorf("timeout too short for %d retries", cfg.MaxRetries)
            }
            return nil
        },
        Severity: iface.ValidationSeverityWarning,
        Message:  "Consider increasing timeout or reducing max retries",
    })
    
    // Load and validate configuration
    var appConfig AppConfig
    
    loader, _ := config.NewLoader(config.DefaultLoaderOptions())
    rawConfig, err := loader.LoadConfig(ctx)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Unmarshal to app config
    if err := rawConfig.Unmarshal(&appConfig); err != nil {
        log.Fatalf("Failed to unmarshal config: %v", err)
    }
    
    // Validate with detailed reporting
    report := validator.GetValidationReport(ctx, &appConfig)
    if !report.Valid {
        fmt.Printf("Configuration validation failed:\n")
        for _, err := range report.Errors {
            fmt.Printf("  Field %s: %s (suggestion: %s)\n", 
                err.Field, err.Message, err.Suggestion)
        }
        log.Fatal("Fix configuration errors before continuing")
    }
    
    fmt.Printf("Configuration validation passed\n")
    fmt.Printf("Compliance score: %.2f%%\n", report.Summary.ComplianceScore*100)
}
```

### 4. Hot-Reload with Health Monitoring

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create loader with hot-reload enabled
    options := config.DefaultLoaderOptions()
    options.EnableHotReload = true
    options.EnableHealthChecks = true
    options.ConfigPaths = []string{"./config"}
    
    loader, err := config.NewEnhancedLoader(options)
    if err != nil {
        log.Fatalf("Failed to create loader: %v", err)
    }
    
    // Load initial configuration
    cfg, err := loader.LoadConfig(ctx)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Register reload callback
    reloadCallback := func(oldConfig, newConfig iface.Config, source string) error {
        fmt.Printf("Configuration reloaded from %s\n", source)
        fmt.Printf("Old version: %s\n", oldConfig.GetVersion())
        fmt.Printf("New version: %s\n", newConfig.GetVersion())
        
        // Validate new configuration before applying
        if err := newConfig.Validate(ctx); err != nil {
            fmt.Printf("New configuration validation failed: %v\n", err)
            return fmt.Errorf("refusing invalid configuration: %w", err)
        }
        
        // Apply configuration changes here
        fmt.Printf("Configuration successfully updated\n")
        return nil
    }
    
    // Start hot-reload watching
    if err := loader.StartWatching(ctx, reloadCallback); err != nil {
        log.Fatalf("Failed to start watching: %v", err)
    }
    
    // Start health checking
    healthChecker := config.NewHealthChecker()
    healthCallback := func(component string, oldStatus, newStatus iface.HealthStatus) {
        fmt.Printf("Health status changed for %s: %s -> %s\n", 
            component, oldStatus.Status, newStatus.Status)
        
        if newStatus.Status == iface.HealthStatusUnhealthy {
            fmt.Printf("ALERT: Component %s is unhealthy: %s\n", 
                component, newStatus.LastError)
        }
    }
    
    if err := healthChecker.RegisterHealthCallback(healthCallback); err != nil {
        log.Fatalf("Failed to register health callback: %v", err)
    }
    
    if err := healthChecker.StartHealthChecks(ctx, 30*time.Second); err != nil {
        log.Fatalf("Failed to start health checks: %v", err)
    }
    
    // Graceful shutdown handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    fmt.Printf("Application started with hot-reload and health monitoring\n")
    fmt.Printf("Press Ctrl+C to shutdown gracefully\n")
    
    // Main application loop
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-sigChan:
            fmt.Printf("Shutdown signal received\n")
            
            // Stop health checks
            if err := healthChecker.StopHealthChecks(); err != nil {
                fmt.Printf("Error stopping health checks: %v\n", err)
            }
            
            // Stop watching
            if err := loader.StopWatching(ctx); err != nil {
                fmt.Printf("Error stopping watcher: %v\n", err)
            }
            
            fmt.Printf("Graceful shutdown completed\n")
            return
            
        case <-ticker.C:
            // Periodic health status reporting
            systemHealth := config.GetGlobalRegistry().GetSystemHealth(ctx)
            fmt.Printf("System health: %s (healthy: %d, total: %d)\n",
                systemHealth.OverallStatus,
                len(systemHealth.Providers),
                systemHealth.TotalChecks)
        }
    }
}
```

### 5. Performance Testing and Benchmarking

```go
package main

import (
    "context"
    "fmt"
    "testing"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func BenchmarkConfigLoad(b *testing.B) {
    ctx := context.Background()
    
    options := config.DefaultLoaderOptions()
    loader, err := config.NewLoader(options)
    if err != nil {
        b.Fatalf("Failed to create loader: %v", err)
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        cfg, err := loader.LoadConfig(ctx)
        if err != nil {
            b.Fatalf("Failed to load config: %v", err)
        }
        _ = cfg
    }
}

func BenchmarkValidation(b *testing.B) {
    ctx := context.Background()
    
    validator, err := config.NewValidator(config.DefaultValidatorOptions())
    if err != nil {
        b.Fatalf("Failed to create validator: %v", err)
    }
    
    testConfig := &TestConfig{
        Name:    "test",
        Value:   42,
        Enabled: true,
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        err := validator.ValidateStruct(ctx, testConfig)
        if err != nil {
            b.Fatalf("Validation failed: %v", err)
        }
    }
}

func TestPerformanceGoals(t *testing.T) {
    ctx := context.Background()
    
    // Test load time under 10ms
    start := time.Now()
    loader, _ := config.NewLoader(config.DefaultLoaderOptions())
    cfg, err := loader.LoadConfig(ctx)
    loadTime := time.Since(start)
    
    if err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    
    if loadTime > 10*time.Millisecond {
        t.Errorf("Load time %v exceeds 10ms goal", loadTime)
    }
    
    // Test validation time under 1ms
    start = time.Now()
    validator, _ := config.NewValidator(config.DefaultValidatorOptions())
    err = validator.ValidateStruct(ctx, cfg)
    validationTime := time.Since(start)
    
    if err != nil {
        t.Fatalf("Validation failed: %v", err)
    }
    
    if validationTime > 1*time.Millisecond {
        t.Errorf("Validation time %v exceeds 1ms goal", validationTime)
    }
    
    t.Logf("Performance goals met: load=%v, validation=%v", loadTime, validationTime)
}
```

## Configuration Examples

### Enhanced Configuration File (config.yaml)
```yaml
# Application configuration with enhanced validation
app:
  name: "beluga-ai-app"
  version: "1.0.0"
  debug: false
  
database:
  url: "${DATABASE_URL}"
  max_connections: 20
  timeout: "30s"
  
api:
  key: "${API_KEY}"
  base_url: "https://api.example.com"
  timeout: "10s"
  retries: 3
  
# Health check configuration
health_checks:
  enabled: true
  interval: "30s"
  timeout: "5s"
  failure_threshold: 3
  enable_recovery: true
  
# Observability configuration  
observability:
  metrics:
    enabled: true
    provider: "otel"
    export_interval: "30s"
  tracing:
    enabled: true
    sample_rate: 0.1
  logging:
    level: "info"
    format: "json"
    enable_trace_id: true
    
# Migration configuration
migration:
  enabled: false
  current_version: "1.0.0"
  backup_before_migration: true
  
# Provider configurations
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    api_key: "${OPENAI_API_KEY}"
    model_name: "gpt-4"
    default_call_options:
      temperature: 0.7
      max_tokens: 1000
      
embedding_providers:
  - name: "openai-embeddings"
    provider: "openai"
    api_key: "${OPENAI_API_KEY}"
    model_name: "text-embedding-ada-002"
```

## Testing Integration

The quickstart includes comprehensive test scenarios that validate:

1. **Configuration Loading**: Multiple sources, fallbacks, error handling
2. **Provider Registry**: Registration, creation, health checks, lifecycle
3. **Validation**: Schema validation, custom rules, cross-field validation
4. **Hot-reload**: File watching, change detection, callback execution
5. **Health Monitoring**: Status tracking, failure detection, recovery
6. **Performance**: Load times, validation speed, memory usage
7. **Observability**: Metrics collection, tracing, structured logging

## Next Steps

1. Run the examples to validate functionality
2. Customize configuration for your use case
3. Add custom providers and validation rules
4. Set up monitoring and alerting based on health checks
5. Configure hot-reload for your deployment environment

This quickstart demonstrates all key features required for constitutional compliance while maintaining the flexibility and ease of use that makes the config package powerful for various scenarios.
