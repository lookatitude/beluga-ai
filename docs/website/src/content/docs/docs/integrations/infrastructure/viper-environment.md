---
title: Viper Configuration Integration
description: "Use Viper for layered configuration in Beluga AI with file-based defaults, environment variable overrides, and hot-reload in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Viper config, environment variables, Beluga AI, layered configuration, config override Go, hot-reload, application config"
---

Production applications need configuration that adapts to the deployment environment without code changes. Viper provides a layered configuration model where file-based defaults can be overridden by environment variables at runtime. This guide shows how to integrate Viper with Beluga AI for flexible, environment-aware configuration.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- The Viper configuration library

## Installation

```bash
go get github.com/spf13/viper
```

## Basic Configuration with Viper

Set up a configuration loader that reads from files and environment variables:

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

	// Enable environment variable overrides
	v.SetEnvPrefix("BELUGA")
	v.AutomaticEnv()

	// Read from config file (optional)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.beluga")

	if err := v.ReadInConfig(); err != nil {
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

### Config File

Create a `config.yaml` alongside the binary:

```yaml
app:
  name: beluga-ai
  port: 8080
llm:
  provider: openai
```

### Overriding with Environment Variables

Any key can be overridden at runtime using the `BELUGA_` prefix:

```bash
export BELUGA_APP_PORT=9090
go run main.go
# Output: Port: 9090
```

## Integration with Beluga AI

Wrap Viper in a typed loader that Beluga AI components can consume:

```go
package main

import (
	"fmt"
	"log"

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

	fmt.Printf("Provider: %s\n", loader.GetString("llm.provider"))
}
```

## Environment-Specific Configuration

Load a base configuration file and merge environment-specific overrides on top:

```go
func LoadConfigForEnvironment(env string) (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvPrefix("BELUGA")
	v.AutomaticEnv()

	// Load base config
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Merge environment-specific overrides (config.production.yaml, etc.)
	if env != "" {
		v.SetConfigName(fmt.Sprintf("config.%s", env))
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, err
			}
		}
	}

	return v, nil
}
```

This approach supports a file layout like:

```
config.yaml                # base defaults
config.development.yaml    # development overrides
config.staging.yaml        # staging overrides
config.production.yaml     # production overrides
```

## Configuration Reference

| Option | Description | Default |
|--------|-------------|---------|
| `EnvPrefix` | Prefix for environment variables (`BELUGA_`) | `BELUGA` |
| `ConfigName` | Base config file name (without extension) | `config` |
| `ConfigType` | Config file format | `yaml` |
| `ConfigPath` | Directories to search for config files | `.` |

## Production-Ready Example

A complete loader that selects environment-specific config, applies environment variable overrides, and validates required keys at startup:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

type ProductionConfigLoader struct {
	viper *viper.Viper
}

func NewProductionConfigLoader() (*ProductionConfigLoader, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("app.name", "beluga-ai")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.env", "development")

	// Environment variable overrides
	v.SetEnvPrefix("BELUGA")
	v.AutomaticEnv()

	// Load environment-specific config file
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

	// Validate required keys
	if v.GetString("llm.api_key") == "" {
		return nil, fmt.Errorf("llm.api_key is required (set via config file or BELUGA_LLM_API_KEY)")
	}

	return &ProductionConfigLoader{viper: v}, nil
}

func (l *ProductionConfigLoader) GetString(key string) string {
	return l.viper.GetString(key)
}

func (l *ProductionConfigLoader) GetInt(key string) int {
	return l.viper.GetInt(key)
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

## Troubleshooting

### Config file not found

Config files are optional when environment variables provide all required values. Handle the missing-file case explicitly:

```go
if err := v.ReadInConfig(); err != nil {
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		// Continue with defaults and environment variables
	}
}
```

### Environment variable not overriding

Viper maps nested keys to environment variables by replacing dots with underscores and applying the prefix. For the key `app.port`:

```bash
BELUGA_APP_PORT=9090
```

## Production Considerations

- **Prefer environment variables for secrets** -- never commit API keys to config files.
- **Validate required values at startup** -- fail fast if critical configuration is missing.
- **Provide sensible defaults** -- reduce the number of required overrides per environment.
- **Document all variables** -- maintain a table of supported environment variables in your deployment documentation.

## Related Resources

- [HashiCorp Vault Connector](/docs/integrations/vault-connector) -- Secure secret management with Vault
- [Infrastructure Integrations](/docs/integrations/infrastructure) -- Deployment and infrastructure overview
