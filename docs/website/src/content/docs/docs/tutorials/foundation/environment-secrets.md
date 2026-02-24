---
title: Environment and Secret Management
description: "Load type-safe configuration from JSON files and environment variables in Go using Beluga AI's config package with struct tag validation and 12-factor patterns."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, configuration, environment variables, secrets, config package, 12-factor"
---

The `config` package provides type-safe configuration loading from JSON files, environment variable overrides, and struct tag-based validation. This approach follows the [12-Factor App methodology](https://12factor.net/config) — defaults in files, secrets in environment variables. The 12-Factor pattern matters for AI applications because LLM providers, vector databases, and embedding services each require API keys and connection strings that vary across environments. Hardcoding these values creates security risks and makes deployment inflexible.

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

This ordering is intentional. Environment variables take highest priority because they are the standard mechanism for injecting secrets in container orchestrators (Kubernetes, ECS, Cloud Run). Configuration files provide sensible defaults that work across most environments. Struct tag defaults serve as the final fallback, ensuring the application can start even without a configuration file — useful for local development and testing.

## Step 1: Define a Configuration Struct

Use struct tags to specify JSON keys, defaults, and validation rules. Beluga AI's config package uses struct tags rather than external schema files because Go's type system can then enforce constraints at compile time while the tags handle runtime validation. This keeps configuration and code co-located, reducing the risk of drift between schema definitions and actual usage.

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

Create a `config.json` file with non-sensitive defaults. The JSON file should contain values that are safe to commit to version control — host addresses, model names, log levels — but never API keys or database passwords.

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

Load it with `config.Load`. The generic type parameter `[AppConfig]` enables the loader to apply validation and defaults using the struct tag metadata specific to your configuration type.

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

Use `config.MergeEnv` to overlay environment variable values onto an existing configuration. This two-phase approach — load from file first, then merge environment variables — allows you to see which values came from the file and which were overridden by the environment, making debugging easier in production.

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

Nested structs use underscores for the path: the `LLM.APIKey` field maps to `BELUGA_LLM_API_KEY`. The prefix requirement prevents collisions with other applications sharing the same environment.

## Step 4: Load Entirely from Environment

For container deployments where no config file exists, load the entire configuration from environment variables. This is the preferred approach for Kubernetes deployments where ConfigMaps and Secrets inject all values as environment variables and mounting a JSON file adds unnecessary complexity.

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

Validation runs automatically during `Load` and `LoadFromEnv`. You can also validate manually, which is useful when constructing configuration programmatically in tests or when receiving configuration from an external source like a secrets manager.

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
