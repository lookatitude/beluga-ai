---
title: Config Package API
description: API documentation for configuration loading and validation.
---

```go
import "github.com/lookatitude/beluga-ai/config"
```

Package config provides configuration loading, validation, environment variable merging, and hot-reload watchers.

## Quick Start

### Load from File

```go
type AppConfig struct {
    Port    int    `json:"port" default:"8080" min:"1" max:"65535"`
    Host    string `json:"host" default:"localhost" required:"true"`
    Debug   bool   `json:"debug" default:"false"`
    Timeout int    `json:"timeout" default:"30" min:"1"`
}

cfg, err := config.Load[AppConfig]("config.json")
```

### Load from Environment

```go
cfg, err := config.LoadFromEnv[AppConfig]("BELUGA")
// Reads: BELUGA_PORT, BELUGA_HOST, BELUGA_DEBUG, BELUGA_TIMEOUT
```

### Merge Environment Overrides

```go
cfg, _ := config.Load[AppConfig]("config.json")
config.MergeEnv(&cfg, "BELUGA") // Override with env vars
```

## ProviderConfig

Standard configuration for providers:

```go
type ProviderConfig struct {
    Provider string                 `json:"provider" required:"true"`
    APIKey   string                 `json:"api_key"`
    Model    string                 `json:"model"`
    BaseURL  string                 `json:"base_url"`
    Timeout  time.Duration          `json:"timeout" default:"30s"`
    Options  map[string]any         `json:"options"`
}
```

Usage:

```go
cfg := config.ProviderConfig{
    Provider: "openai",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Model:    "gpt-4o",
    Options: map[string]any{
        "temperature": 0.7,
        "max_tokens":  1000,
    },
}

// Type-safe option access
temp, ok := config.GetOption[float64](cfg, "temperature")
```

## Validation

### Struct Tags

```go
type Config struct {
    Name  string  `json:"name" required:"true"`
    Age   int     `json:"age" min:"0" max:"120"`
    Score float64 `json:"score" min:"0.0" max:"1.0"`
}

if err := config.Validate(&cfg); err != nil {
    var valErr *config.ValidationError
    if errors.As(err, &valErr) {
        fmt.Printf("Field %s: %s\n", valErr.Field, valErr.Message)
    }
}
```

### Supported Tags

- `required:"true"` — Field must not be zero-valued
- `min:"N"` — Numeric minimum
- `max:"N"` — Numeric maximum
- `default:"value"` — Default value if unset

## Hot-Reload

Watch for configuration changes:

```go
watcher := config.NewFileWatcher("config.json", 5*time.Second)
defer watcher.Close()

err := watcher.Watch(ctx, func(newConfig any) {
    data := newConfig.([]byte)
    var cfg AppConfig
    json.Unmarshal(data, &cfg)
    // apply new config
})
```

## Example JSON Config

```json
{
  "provider": "openai",
  "api_key": "${OPENAI_API_KEY}",
  "model": "gpt-4o",
  "base_url": "https://api.openai.com/v1",
  "timeout": 30000000000,
  "options": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
```

## See Also

- [Core Package](./core.md) for runtime configuration
- [LLM Package](./llm.md) for provider configuration
