# Viper & Environment Overrides

Welcome, colleague! In this integration guide, we're going to integrate Viper for configuration management with environment variable overrides in Beluga AI applications. This enables flexible, environment-aware configuration.

## What you will build

You will create a configuration system using Viper that loads settings from files and allows environment variable overrides, enabling different configurations for development, staging, and production environments.

## Learning Objectives

- ✅ Configure Viper with Beluga AI
- ✅ Load configuration from files and environment
- ✅ Implement environment variable overrides
- ✅ Understand configuration best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Viper library

## Step 1: Setup and Installation

Install Viper:
bash
```bash
go get github.com/spf13/viper
```

## Step 2: Basic Viper Configuration

Create a basic configuration loader:
```go
package main

import (
    "fmt"
    "log"

    "github.com/spf13/viper"
)

func LoadConfig() (*viper.Viper, error) {
    v := viper.New()
    
    // Set default values
    v.SetDefault("app.name", "beluga-ai")
    v.SetDefault("app.port", 8080)
    v.SetDefault("llm.provider", "openai")
    
    // Enable environment variables
    v.SetEnvPrefix("BELUGA")
    v.AutomaticEnv()
    
    // Read from config file (optional)
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    v.AddConfigPath("$HOME/.beluga")
    
    if err := v.ReadInConfig(); err != nil {
        // Config file not found; use defaults and env vars
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("error reading config: %w", err)
        }
    }
    
    return v, nil
}

func main() {
    config, err := LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    
    fmt.Printf("App Name: %s\n", config.GetString("app.name"))
    fmt.Printf("Port: %d\n", config.GetInt("app.port"))
    fmt.Printf("LLM Provider: %s\n", config.GetString("llm.provider"))
}
```

### Verification

Create a `config.yaml` file:

```yaml
app:
  name: beluga-ai
  port: 8080
llm:
  provider: openai
  api_key: "${OPENAI_API_KEY}"
```

Run with environment override:
bash
```bash
export BELUGA_APP_PORT=9090
go run main.go
```

You should see the port overridden to 9090.

## Step 3: Integration with Beluga AI Config

Integrate with Beluga AI's config package:
```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/spf13/viper"
)

type ViperConfigLoader struct {
    viper *viper.Viper
}

func NewViperConfigLoader() (*ViperConfigLoader, error) {
    v := viper.New()
    v.SetEnvPrefix("BELUGA")
    v.AutomaticEnv()
    
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }
    
    return &ViperConfigLoader{viper: v}, nil
}

func (l *ViperConfigLoader) GetString(key string) string {
    return l.viper.GetString(key)
}

func (l *ViperConfigLoader) GetInt(key string) int {
    return l.viper.GetInt(key)
}

func (l *ViperConfigLoader) GetBool(key string) bool {
    return l.viper.GetBool(key)
}

func main() {
    loader, err := NewViperConfigLoader()
    if err != nil {
        log.Fatalf("Failed to create config loader: %v", err)
    }
    
    // Use with Beluga AI
    llmProvider := loader.GetString("llm.provider")
    apiKey := loader.GetString("llm.api_key")

    
    fmt.Printf("Provider: %s\n", llmProvider)
    fmt.Printf("API Key: %s\n", maskAPIKey(apiKey))
}
```

## Step 4: Environment-Specific Configuration

Load different configs per environment:
```go
func LoadConfigForEnvironment(env string) (*viper.Viper, error) {
    v := viper.New()
    v.SetEnvPrefix("BELUGA")
    v.AutomaticEnv()
    
    // Load base config
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    v.ReadInConfig()
    
    // Load environment-specific overrides
    if env != "" {
        v.SetConfigName(fmt.Sprintf("config.%s", env))
        if err := v.MergeInConfig(); err != nil {
            // Environment config is optional
            if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
                return nil, err
            }
        }
    }

    
    return v, nil
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `EnvPrefix` | Environment variable prefix | `BELUGA` | No |
| `ConfigName` | Config file name | `config` | No |
| `ConfigType` | Config file type | `yaml` | No |
| `ConfigPath` | Config file search paths | `.` | No |

## Common Issues

### "Config file not found"

**Problem**: Config file doesn't exist.

**Solution**: Config files are optional when using environment variables:

```go
if err := v.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
        // Use defaults and env vars
    }
}
```

### "Environment variable not overriding"

**Problem**: Environment variable naming mismatch.

**Solution**: Ensure correct prefix and naming:

```bash
# For key "app.port", use:
BELUGA_APP_PORT=9090
```

## Production Considerations

When using Viper in production:

- **Use environment variables**: For secrets and environment-specific values
- **Validate configuration**: Check required values at startup
- **Use defaults**: Provide sensible defaults
- **Document variables**: Document all environment variables

## Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/spf13/viper"
)

type ProductionConfigLoader struct {
    viper *viper.Viper
}

func NewProductionConfigLoader() (*ProductionConfigLoader, error) {
    v := viper.New()
    
    // Set defaults
    v.SetDefault("app.name", "beluga-ai")
    v.SetDefault("app.port", 8080)
    v.SetDefault("app.env", "development")
    
    // Environment variables
    v.SetEnvPrefix("BELUGA")
    v.AutomaticEnv()
    
    // Config file
    env := os.Getenv("BELUGA_ENV")
    if env == "" {
        env = "development"
    }
    
    v.SetConfigName(fmt.Sprintf("config.%s", env))
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    v.AddConfigPath("/etc/beluga")
    
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
    }
    
    // Validate required values
    if v.GetString("llm.api_key") == "" {
        return nil, fmt.Errorf("llm.api_key is required")
    }
    
    return &ProductionConfigLoader{viper: v}, nil
}

func (l *ProductionConfigLoader) GetString(key string) string {
    return l.viper.GetString(key)
}

func main() {
    loader, err := NewProductionConfigLoader()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    
    fmt.Printf("App: %s\n", loader.GetString("app.name"))
    fmt.Printf("Port: %d\n", loader.GetInt("app.port"))
}
```

## Next Steps

Congratulations! You've integrated Viper with Beluga AI. Next, learn how to:

- **[HashiCorp Vault Connector](./hashicorp-vault-connector.md)** - Secure secret management
- **[Config Package Documentation](../../api/packages/config.md)** - Deep dive into config package
- **[Best Practices Guide](../../best-practices.md)** - Configuration best practices

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
