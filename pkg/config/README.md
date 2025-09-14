# Config Package

The `config` package provides a flexible and extensible configuration management system for the Beluga AI Framework. It supports multiple configuration sources including files (YAML), environment variables, and programmatic configuration.

## Features

- **Multiple Configuration Sources**: Support for YAML files, environment variables, and programmatic configuration
- **Validation**: Built-in configuration validation with detailed error messages
- **Defaults**: Automatic setting of default values for configuration fields
- **Extensibility**: Easy to extend with new configuration providers
- **Observability**: OpenTelemetry integration for metrics and tracing
- **Type Safety**: Strong typing with Go interfaces and structs

## Package Structure

```
pkg/config/
├── iface/              # Public interfaces and types
│   ├── provider.go     # Provider interface definitions
│   └── types.go        # Configuration type definitions
├── internal/           # Private implementation details
│   └── loader/         # Configuration loading logic
│       ├── loader.go
│       └── validation.go
├── providers/          # Configuration provider implementations
│   └── viper/          # Viper-based provider
│       └── viper_provider.go
├── config.go           # Main package interface with factory functions
├── metrics.go          # Observability and metrics
└── README.md           # This file
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    // Load configuration with default settings
    cfg, err := config.LoadConfig()
    if err != nil {
        panic(fmt.Sprintf("Failed to load config: %v", err))
    }

    // Access configuration
    for _, llm := range cfg.LLMProviders {
        fmt.Printf("LLM Provider: %s\n", llm.Name)
    }
}
```

### Custom Configuration

```go
package main

import (
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    // Create custom loader options
    options := config.DefaultLoaderOptions()
    options.ConfigName = "myapp"
    options.ConfigPaths = []string{"./config", "/etc/myapp"}
    options.EnvPrefix = "MYAPP"

    // Create and use loader
    loader, err := config.NewLoader(options)
    if err != nil {
        panic(fmt.Sprintf("Failed to create loader: %v", err))
    }

    cfg := loader.MustLoadConfig()

    // Use configuration...
    _ = cfg
}
```

### Environment Variables Only

```go
package main

import (
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    // Load configuration from environment variables only
    cfg, err := config.LoadFromEnv("MYAPP")
    if err != nil {
        panic(fmt.Sprintf("Failed to load config from env: %v", err))
    }

    // Use configuration...
    _ = cfg
}
```

### Load from Specific File

```go
package main

import (
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    // Load configuration from a specific file
    cfg, err := config.LoadFromFile("/path/to/config.yaml")
    if err != nil {
        panic(fmt.Sprintf("Failed to load config from file: %v", err))
    }

    // Use configuration...
    _ = cfg
}
```

## Configuration Format

The configuration uses YAML format with the following structure:

```yaml
# config.yaml
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

vector_stores:
  - name: "chroma-db"
    provider: "chroma"
    host: "localhost"
    port: 8000

tools:
  - name: "calculator"
    provider: "calculator"
    description: "Performs mathematical calculations"
    enabled: true
    config:
      precision: 2

agents:
  - name: "assistant"
    description: "General purpose AI assistant"
    llm_provider: "openai-gpt4"
    tools:
      - "calculator"
```

## Environment Variables

Configuration values can be overridden using environment variables. The package supports automatic mapping from environment variables to configuration fields.

### Naming Convention

Environment variables follow the pattern: `{PREFIX}_{CONFIG_KEY}`

- Dots in config keys are replaced with underscores
- All keys are converted to uppercase

### Examples

```bash
# Set LLM provider API key
export BELUGA_LLM_PROVIDERS_0_API_KEY="your-api-key"

# Set embedding provider model
export BELUGA_EMBEDDING_PROVIDERS_0_MODEL_NAME="text-embedding-ada-002"

# Enable/disable a tool
export BELUGA_TOOLS_0_ENABLED="true"
```

## Configuration Providers

### Viper Provider

The default configuration provider uses the [Viper](https://github.com/spf13/viper) library, which supports:

- YAML, JSON, and TOML file formats
- Environment variable overrides
- Configuration file watching for hot reloading
- Multiple configuration paths

### Extending with Custom Providers

You can implement custom configuration providers by implementing the `iface.Provider` interface:

```go
package custom

import (
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

type CustomProvider struct {
    // implementation details
}

func (p *CustomProvider) Load(configStruct interface{}) error {
    // Custom loading logic
    return nil
}

// Implement other required methods...
```

## Validation

The package includes comprehensive configuration validation:

```go
// Validate configuration
cfg, err := config.LoadConfig()
if err != nil {
    // Handle validation errors
    if validationErr, ok := err.(iface.ValidationErrors); ok {
        for _, fieldErr := range validationErr {
            fmt.Printf("Validation error for %s: %s\n", fieldErr.Field, fieldErr.Message)
        }
    }
}
```

### Validation Rules

- **Required Fields**: Fields marked with `validate:"required"` must be present
- **Provider Validation**: Each provider type has specific validation rules
- **Cross-field Validation**: Relationships between fields are validated

## Observability

The package integrates with OpenTelemetry for observability:

### Metrics

```go
// Get global metrics instance
metrics := config.GetGlobalMetrics()

// Metrics are automatically recorded for:
// - Configuration load operations
// - Validation operations
// - Error counts
```

### Tracing

Configuration operations are automatically traced using OpenTelemetry spans.

## Extending the Package

### Adding New Configuration Sections

1. **Define Types**: Add new types to `iface/types.go`

```go
type NewConfigSection struct {
    Name    string `mapstructure:"name" yaml:"name" validate:"required"`
    Enabled bool   `mapstructure:"enabled" yaml:"enabled" default:"true"`
}
```

2. **Update Config Struct**: Add the new section to the main `Config` struct

```go
type Config struct {
    // existing fields...
    NewSections []NewConfigSection `mapstructure:"new_sections" yaml:"new_sections"`
}
```

3. **Add Validation**: Update validation logic in `internal/loader/validation.go`

```go
func validateNewSection(section NewConfigSection) error {
    if section.Name == "" {
        return errors.New("name is required")
    }
    return nil
}
```

### Adding New Providers

1. **Create Provider Package**: Create a new directory under `providers/`

```bash
mkdir providers/custom
```

2. **Implement Provider**: Implement the `iface.Provider` interface

```go
package custom

type CustomProvider struct {
    // fields
}

func (p *CustomProvider) Load(configStruct interface{}) error {
    // implementation
}
```

3. **Add Factory Function**: Add a factory function to the main `config.go`

```go
func NewCustomProvider(options CustomOptions) (iface.Provider, error) {
    // implementation
}
```

## Best Practices

### Configuration File Organization

- Use descriptive names for configuration files
- Group related configuration sections together
- Use environment-specific configuration files

### Environment Variables

- Use consistent naming patterns
- Document required environment variables
- Provide sensible defaults

### Error Handling

```go
cfg, err := config.LoadConfig()
if err != nil {
    // Log detailed error information
    log.Printf("Configuration error: %v", err)

    // Check for validation errors
    if validationErr, ok := err.(iface.ValidationErrors); ok {
        for _, fieldErr := range validationErr {
            log.Printf("Field %s: %s", fieldErr.Field, fieldErr.Message)
        }
    }

    // Exit gracefully
    os.Exit(1)
}
```

### Testing

```go
func TestConfig(t *testing.T) {
    // Create test configuration
    cfg := &iface.Config{
        LLMProviders: []schema.LLMProviderConfig{
            {
                Name:      "test-llm",
                Provider:  "mock",
                APIKey:    "test-key",
                ModelName: "test-model",
            },
        },
    }

    // Validate configuration
    err := config.ValidateConfig(cfg)
    assert.NoError(t, err)
}
```

## Migration Guide

### From v0.x to v1.x

The v1.x release includes breaking changes to align with Beluga AI Framework patterns:

1. **Interface Changes**: Configuration interfaces moved to `iface/` package
2. **Package Structure**: Internal implementations moved to `internal/` package
3. **Provider Pattern**: Configuration providers now follow the standard pattern
4. **Factory Functions**: Use factory functions instead of direct struct creation

### Migration Example

```go
// Before (v0.x)
loader := config.NewLoader(config.DefaultLoaderOptions())
cfg := loader.MustLoadConfig()

// After (v1.x) - No changes needed for basic usage
loader, _ := config.NewLoader(config.DefaultLoaderOptions())
cfg := loader.MustLoadConfig()
```

## Troubleshooting

### Common Issues

1. **Configuration File Not Found**
   - Check file paths and permissions
   - Verify configuration file name matches loader options
   - Use absolute paths for custom locations

2. **Environment Variables Not Working**
   - Verify environment variable prefix
   - Check variable naming (uppercase, underscores)
   - Ensure variables are exported in the shell

3. **Validation Errors**
   - Check required fields are present
   - Verify data types match expected values
   - Review validation error messages for specific issues

### Debugging

Enable debug logging to troubleshoot configuration issues:

```go
import "log"

// Set log level to debug
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Configuration operations will log detailed information
cfg, err := config.LoadConfig()
```

## Contributing

When contributing to the config package:

1. **Follow Package Patterns**: Adhere to the Beluga AI Framework design patterns
2. **Add Tests**: Include comprehensive tests for new functionality
3. **Update Documentation**: Keep README and code documentation current
4. **Maintain Compatibility**: Consider backward compatibility for public APIs

### Development Setup

```bash
# Clone the repository
git clone https://github.com/lookatitude/beluga-ai.git

# Navigate to config package
cd pkg/config

# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

## Related Packages

- [`pkg/schema`](../../../schema/README.md): Core type definitions
- [`pkg/agents`](../../../agents/README.md): Agent configuration and management
- [`pkg/tools`](../../../tools/README.md): Tool configuration and execution

## License

This package is part of the Beluga AI Framework and follows the same license terms.
