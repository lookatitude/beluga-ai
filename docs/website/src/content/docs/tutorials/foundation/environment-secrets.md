---
title: Environment and Secret Management
description: Load configuration from files and environment variables using Beluga AI's config package.
---

The `config` package provides type-safe configuration loading from JSON files, environment variable overrides, and struct tag-based validation. This approach follows the 12-Factor App methodology — defaults in files, secrets in environment variables.

## What You Will Build

A configuration system that loads defaults from a JSON file, overrides values with environment variables, and validates the result using struct tags.

## Prerequisites

- Go 1.23+
- Understanding of Go struct tags

## Configuration Precedence

When loading configuration, Beluga AI follows this hierarchy (highest wins):

1. **Environment variables** — runtime overrides
2. **Configuration file** — JSON defaults
3. **Struct tag defaults** — Go code defaults

## Step 1: Define a Configuration Struct

Use struct tags to specify JSON keys, defaults, and validation rules:

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
)

type AppConfig struct {
    Host     string `json:"host" default:"localhost"`
    Port     int    `json:"port" default:"8080" min:"1" max:"65535"`
    LogLevel string `json:"log_level" default:"info"`
    LLM      LLMConfig `json:"llm"`
}

type LLMConfig struct {
    Provider string `json:"provider" default:"openai" required:"true"`
    Model    string `json:"model" default:"gpt-4o"`
    APIKey   string `json:"api_key" required:"true"`
}
```

Supported struct tags:
- `default:"value"` — applied when the field is zero-valued and not provided in JSON
- `required:"true"` — validation fails if the field is missing and zero-valued
- `min:"N"` / `max:"N"` — numeric range constraints

## Step 2: Load from a JSON File

Create a `config.json` file with non-sensitive defaults:

```json
{
    "host": "0.0.0.0",
    "port": 8080,
    "log_level": "info",
    "llm": {
        "provider": "openai",
        "model": "gpt-4o"
    }
}
```

Load it with `config.Load`:

```go
func main() {
    cfg, err := config.Load[AppConfig]("config.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    fmt.Printf("Host: %s\n", cfg.Host)
    fmt.Printf("Port: %d\n", cfg.Port)
    fmt.Printf("LLM Provider: %s\n", cfg.LLM.Provider)
}
```

`Load` handles the full pipeline: read file, unmarshal JSON, check required fields, apply defaults for unset fields, and validate constraints.

## Step 3: Override with Environment Variables

Use `config.MergeEnv` to overlay environment variable values onto an existing configuration:

```go
func main() {
    cfg, err := config.Load[AppConfig]("config.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Override with environment variables prefixed with BELUGA_
    if err := config.MergeEnv(&cfg, "BELUGA"); err != nil {
        log.Fatalf("Failed to merge env: %v", err)
    }

    fmt.Printf("Log Level: %s\n", cfg.LogLevel)
    fmt.Printf("LLM API Key set: %v\n", cfg.LLM.APIKey != "")
}
```

Environment variable naming follows the convention `PREFIX_FIELDNAME`:

```bash
export BELUGA_LOG_LEVEL="debug"
export BELUGA_LLM_API_KEY="sk-..."
export BELUGA_PORT="9090"
```

Nested structs use underscores for the path: the `LLM.APIKey` field maps to `BELUGA_LLM_API_KEY`.

## Step 4: Load Entirely from Environment

For container deployments where no config file exists, load the entire configuration from environment variables:

```go
func main() {
    cfg, err := config.LoadFromEnv[AppConfig]("BELUGA")
    if err != nil {
        log.Fatalf("Failed to load from env: %v", err)
    }

    fmt.Printf("Provider: %s\n", cfg.LLM.Provider)
}
```

This applies struct tag defaults first, then overlays any set environment variables, and finally validates the result.

## Step 5: Validation

Validation runs automatically during `Load` and `LoadFromEnv`. You can also validate manually:

```go
cfg := AppConfig{
    Port: 99999, // exceeds max
}

if err := config.Validate(&cfg); err != nil {
    fmt.Printf("Validation error: %v\n", err)
    // Output: config: validation failed for "port": value 99999 is greater than maximum 65535
}
```

## Security Guidelines

1. **Never commit API keys** — keep `api_key` fields empty in JSON files and provide them via environment variables.
2. **Use `required:"true"`** on secret fields to catch missing values at startup rather than at runtime.
3. **Separate config files per environment** — use `config.dev.json` and `config.prod.json` with different defaults.

## Troubleshooting

**Environment variable not picked up**: Verify the prefix and casing. Environment variables must be uppercase. The field name is converted using Go's naming convention — `APIKey` becomes `API_KEY`, `BaseURL` becomes `BASE_URL`.

**Required field error despite having a default**: The `required:"true"` check runs before defaults are applied. If a required field must also have a default, provide it in the JSON file rather than only in the struct tag.

## Next Steps

- [Vault Integration](/tutorials/foundation/vault-integration) — Dynamic secret loading from HashiCorp Vault
- [Health Checks](/tutorials/foundation/health-checks) — Monitor service and provider health
