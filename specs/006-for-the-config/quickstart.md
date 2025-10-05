# Config Package Constitutional Compliance - Quickstart Guide

## Overview

This guide demonstrates how to use the enhanced Config package with full constitutional compliance including global registry pattern, complete OTEL integration, and structured error handling, while preserving the existing excellent multi-provider architecture and configuration management capabilities.

## Prerequisites

- Beluga AI Framework installed and configured
- Config package with constitutional compliance enhancements
- Configuration files (YAML, JSON, TOML) or environment variables for testing
- Go 1.21+ for development

## Quick Start Examples

### 1. Using the Global Registry (New Constitutional Feature)

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    ctx := context.Background()
    
    // Register providers globally (typically done at application startup)
    err := config.RegisterGlobal("viper", config.NewViperCreator())
    if err != nil {
        log.Fatal("Failed to register Viper provider:", err)
    }
    
    err = config.RegisterGlobal("composite", config.NewCompositeCreator())
    if err != nil {
        log.Fatal("Failed to register Composite provider:", err)
    }
    
    // Create provider using registry
    options := config.ProviderOptions{
        ProviderType:     "viper",
        ConfigName:       "config",
        ConfigPaths:      []string{"./config", "."},
        EnvPrefix:        "BELUGA",
        Format:           "yaml",
        EnableValidation: true,
        EnableMetrics:    true,
        EnableTracing:    true,
    }
    
    provider, err := config.NewProvider(ctx, "viper", options)
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    
    // Use exactly the same loading interface as before - no breaking changes!
    var cfg config.Config
    err = provider.Load(&cfg)
    if err != nil {
        // Enhanced error handling with structured errors
        if configErr := config.AsConfigError(err); configErr != nil {
            log.Printf("Operation: %s, Code: %s, Provider: %s", 
                configErr.GetOperation(), configErr.GetCode(), configErr.GetProvider())
            
            if configErr.IsRetryable() {
                log.Printf("Error is retryable, retry after: %v", configErr.GetRetryAfter())
            }
        }
        return
    }
    
    log.Printf("Configuration loaded successfully with %d LLM providers", len(cfg.LLMProviders))
}
```

### 2. Provider Discovery and Capabilities (Registry Features)

```go
func discoverProviders() {
    // List all registered providers
    providers := config.ListProviders()
    log.Printf("Available providers: %v", providers)
    
    // Get detailed provider information
    for _, providerName := range providers {
        metadata, err := config.GetProviderMetadata(providerName)
        if err != nil {
            continue
        }
        
        log.Printf("Provider: %s", metadata.Name)
        log.Printf("  Description: %s", metadata.Description)
        log.Printf("  Supported Formats: %v", metadata.SupportedFormats)
        log.Printf("  Capabilities: %v", metadata.Capabilities)
        log.Printf("  Supports Watching: %t", metadata.SupportsWatch)
        log.Printf("  Supports Environment Variables: %t", metadata.SupportsEnvVars)
    }
    
    // Find providers with specific capabilities
    yamlProviders, err := config.GetProvidersForFormat("yaml")
    if err == nil {
        log.Printf("Providers supporting YAML: %v", yamlProviders)
    }
    
    // Find providers supporting specific capabilities
    watchProviders, err := config.GetProviderByCapability("file_watching")
    if err == nil {
        log.Printf("Providers supporting file watching: %v", watchProviders)
    }
}
```

### 3. Enhanced OTEL Observability (New Feature)

```go
func demonstrateObservability() {
    ctx := context.Background()
    
    // Initialize OTEL (typically done in main)
    meter := otel.Meter("config-example")
    tracer := otel.Tracer("config-example")
    
    // Create metrics-enabled configuration
    options := config.ProviderOptions{
        ProviderType:     "viper",
        ConfigName:       "config", 
        ConfigPaths:      []string{"./config"},
        Format:           "yaml",
        EnableValidation: true,
        EnableMetrics:    true,
        EnableTracing:    true,
        EnableLogging:    true,
    }
    
    provider, err := config.NewProvider(ctx, "viper", options)
    if err != nil {
        log.Fatal(err)
    }
    
    // All configuration loading operations are automatically instrumented
    var cfg config.Config
    
    // This operation will generate:
    // - Metrics: load count, duration, success/failure, validation time
    // - Tracing: distributed trace spans with configuration context
    // - Logging: structured logs with operation details
    err = provider.Load(&cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Configuration loaded with full observability: %d providers configured", 
        len(cfg.LLMProviders)+len(cfg.EmbeddingProviders))
    
    // Access health metrics
    if metricsProvider, ok := provider.(config.MetricsProvider); ok {
        healthMetrics := metricsProvider.GetHealthMetrics()
        log.Printf("Load Success Rate: %.2f%%", healthMetrics.SuccessRate*100)
        log.Printf("Average Load Time: %v", healthMetrics.AverageLoadTime)
        log.Printf("Validation Success Rate: %.2f%%", healthMetrics.ValidationSuccessRate*100)
    }
}
```

### 4. Structured Error Handling (Enhanced Feature)

```go
func handleConfigErrors() {
    ctx := context.Background()
    
    // Create provider with configuration that might fail
    options := config.ProviderOptions{
        ProviderType: "viper",
        ConfigName:   "nonexistent",
        ConfigPaths:  []string{"./invalid/path"},
        Format:       "yaml",
        EnableValidation: true,
    }
    
    provider, err := config.NewProvider(ctx, "viper", options)
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    
    var cfg config.Config
    err = provider.Load(&cfg)
    if err != nil {
        // Demonstrate structured error handling
        configErr := config.AsConfigError(err)
        if configErr != nil {
            log.Printf("Configuration Error Details:")
            log.Printf("  Operation: %s", configErr.GetOperation())
            log.Printf("  Code: %s", configErr.GetCode())
            log.Printf("  Provider: %s", configErr.GetProvider())
            log.Printf("  Format: %s", configErr.GetFormat())
            log.Printf("  Config Path: %s", configErr.GetConfigPath())
            log.Printf("  Retryable: %t", configErr.IsRetryable())
            log.Printf("  Timestamp: %v", configErr.GetTimestamp())
            
            // Handle specific error codes
            switch configErr.GetCode() {
            case config.ErrCodeFileNotFound:
                log.Println("Check configuration file path and permissions")
            case config.ErrCodeValidationFailed:
                log.Println("Review configuration format and required fields")
            case config.ErrCodeProviderNotFound:
                log.Println("Register the provider or use a different provider type")
            case config.ErrCodeFormatNotSupported:
                log.Println("Use a supported format (yaml, json, toml) or different provider")
            }
            
            // Access additional context
            if context := configErr.GetContext(); len(context) > 0 {
                log.Printf("Additional Context: %+v", context)
            }
        }
        return
    }
    
    // Example of retry logic with structured errors
    maxRetries := 3
    for attempt := 1; attempt <= maxRetries; attempt++ {
        var retryConfig config.Config
        err := provider.Load(&retryConfig)
        if err == nil {
            log.Printf("Success on attempt %d", attempt)
            return
        }
        
        configErr := config.AsConfigError(err)
        if configErr == nil || !configErr.IsRetryable() {
            log.Printf("Non-retryable error: %v", err)
            return
        }
        
        if attempt < maxRetries {
            delay := configErr.GetRetryAfter()
            if delay == 0 {
                delay = time.Duration(attempt) * time.Second // exponential backoff
            }
            log.Printf("Retrying in %v (attempt %d/%d)", delay, attempt, maxRetries)
            time.Sleep(delay)
        }
    }
    
    log.Printf("All retry attempts exhausted")
}
```

### 5. Backward Compatibility (Existing Code Works Unchanged)

```go
func demonstrateBackwardCompatibility() {
    // This is exactly how developers used Config package before compliance enhancements
    // All existing code continues to work without any changes!
    
    // Original loader functions still work (now use registry internally)
    loader, err := config.NewLoader(config.DefaultLoaderOptions())
    if err != nil {
        log.Fatal(err)
    }
    
    // Same loading methods work exactly the same
    cfg, err := loader.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }
    
    // Provider creation unchanged
    provider, err := config.NewYAMLProvider("config", []string{"./config"}, "BELUGA")
    if err != nil {
        log.Fatal(err)
    }
    
    // Loading unchanged
    var yamlConfig config.Config
    err = provider.Load(&yamlConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Composite providers unchanged
    composite := config.NewCompositeProvider(provider, envProvider)
    err = composite.Load(&yamlConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Loaded configuration with %d LLM providers using backward-compatible API", 
        len(yamlConfig.LLMProviders))
}
```

### 6. Advanced Registry Usage and Health Monitoring

```go
func advancedRegistryUsage() {
    ctx := context.Background()
    
    // Register custom provider with enhanced metadata
    metadata := config.ProviderMetadata{
        Name:             "custom-provider",
        Description:      "Custom configuration provider implementation",
        SupportedFormats: []string{"yaml", "json", "custom"},
        Capabilities:     []string{"file_loading", "env_vars", "validation", "health_check"},
        RequiredOptions:  []string{"config_name", "config_paths"},
        OptionalOptions:  []string{"env_prefix", "format", "enable_watching"},
        SupportsWatch:    true,
        SupportsEnvVars:  true,
        HealthCheckSupported: true,
        DefaultTimeout:   time.Second * 30,
    }
    
    creator := func(options config.ProviderOptions) (config.Provider, error) {
        return &CustomConfigProvider{options: options}, nil
    }
    
    err := config.RegisterGlobalWithMetadata("custom", creator, metadata)
    if err != nil {
        log.Fatal("Failed to register custom provider:", err)
    }
    
    // Use registry validation
    options := config.ProviderOptions{
        ProviderType:     "custom",
        ConfigName:       "custom-config",
        ConfigPaths:      []string{"./custom-config"},
        Format:           "yaml",
        EnableValidation: true,
        EnableMetrics:    true,
        ProviderSpecific: map[string]interface{}{
            "custom_param": "value",
        },
    }
    
    // Options are validated against provider requirements
    err = config.ValidateProviderOptions("custom", options)
    if err != nil {
        log.Printf("Options validation failed: %v", err)
        return
    }
    
    // Create provider with validated options
    provider, err := config.NewProvider(ctx, "custom", options)
    if err != nil {
        log.Fatal("Failed to create custom provider:", err)
    }
    
    // Monitor provider health
    healthStatus := config.CheckProviderHealth(ctx, "custom")
    log.Printf("Custom provider health: %s (success rate: %.2f%%)", 
        healthStatus.Status, healthStatus.SuccessRate*100)
    
    // Use custom provider with same interface
    var customConfig config.Config
    err = provider.Load(&customConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Custom provider loaded successfully")
}
```

## Migration Guide

### For Existing Code

**No changes required!** All existing Config package code continues to work exactly as before. The constitutional compliance enhancements are additive and don't break any existing APIs.

### To Adopt New Features Gradually

1. **Start using registry for new code:**
   ```go
   // Old way (still works)
   provider, err := config.NewYAMLProvider("config", paths, "BELUGA")
   
   // New way with registry
   provider, err := config.NewProvider(ctx, "viper", options)
   ```

2. **Enable observability:**
   ```go
   options.EnableMetrics = true
   options.EnableTracing = true
   options.EnableLogging = true
   ```

3. **Improve error handling:**
   ```go
   if configErr := config.AsConfigError(err); configErr != nil {
       // Use structured error information
       log.Printf("Error code: %s", configErr.GetCode())
   }
   ```

### Best Practices

1. **Registry Management:**
   ```go
   // Register providers at application startup
   func init() {
       config.RegisterGlobal("viper", config.NewViperCreator())
       config.RegisterGlobal("composite", config.NewCompositeCreator())
   }
   ```

2. **Configuration Validation:**
   ```go
   // Always validate provider options
   if err := config.ValidateProviderOptions(providerName, options); err != nil {
       return fmt.Errorf("invalid provider options: %w", err)
   }
   ```

3. **Error Handling:**
   ```go
   // Use structured error handling for better debugging
   if configErr := config.AsConfigError(err); configErr != nil {
       if configErr.IsRetryable() {
           // Implement retry logic
       }
       // Log with proper context
       log.WithFields(log.Fields{
           "operation": configErr.GetOperation(),
           "provider":  configErr.GetProvider(),
           "error_code": configErr.GetCode(),
       }).Error("Configuration operation failed")
   }
   ```

4. **Health Monitoring:**
   ```go
   // Monitor configuration system health
   healthStatus := config.GetConfigHealthStatus(ctx)
   if healthStatus.Status != "healthy" {
       log.Printf("Configuration system health: %s", healthStatus.Status)
   }
   ```

## Testing with Enhanced Features

### Using Advanced Mocks

```go
func TestWithAdvancedMocks(t *testing.T) {
    // Create advanced mock with registry support
    mock := config.NewAdvancedMockProvider("test-provider",
        config.WithMockLoadResults(map[string]interface{}{
            "llm_providers": []map[string]interface{}{
                {"name": "mock-llm", "provider": "mock"},
            },
        }),
        config.WithMockLoadTime(time.Millisecond*10),
        config.WithMockErrorRate(0.0), // No errors for successful test
    )
    
    // Register mock provider
    err := config.RegisterGlobal("mock", func(options config.ProviderOptions) (config.Provider, error) {
        return mock, nil
    })
    require.NoError(t, err)
    
    // Use mock through registry
    provider, err := config.NewProvider(context.Background(), "mock", config.ProviderOptions{
        ProviderType: "mock",
        ConfigName:   "test",
        ConfigPaths:  []string{"./test"},
    })
    require.NoError(t, err)
    
    // Test with realistic mock behavior
    var cfg config.Config
    err = provider.Load(&cfg)
    require.NoError(t, err)
    assert.Len(t, cfg.LLMProviders, 1)
    assert.Equal(t, "mock-llm", cfg.LLMProviders[0].Name)
}
```

### Performance Testing

```go
func TestConfigPerformance(t *testing.T) {
    // Performance benchmarks included in advanced_test.go
    provider, err := config.NewProvider(ctx, "viper", options)
    require.NoError(t, err)
    
    // Measure load performance
    start := time.Now()
    var cfg config.Config
    err = provider.Load(&cfg)
    loadTime := time.Since(start)
    
    require.NoError(t, err)
    assert.Less(t, loadTime, 10*time.Millisecond, "Configuration loading should be under 10ms")
    
    // Health monitoring validation
    healthMetrics := config.GetConfigHealthMetrics()
    assert.Greater(t, healthMetrics.SuccessRate, 0.95, "Success rate should be above 95%")
}
```

This quickstart guide demonstrates how the Config package maintains complete backward compatibility while providing powerful new constitutional compliance features that enhance observability, error handling, and extensibility through the global registry pattern.
