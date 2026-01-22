# Environment & Secret Management

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we will master the art of configuration. We'll start with file-based defaults and move to production-grade secret management using environment variables.

## Learning Objectives
By the end of this tutorial, you will:
1.  Load configuration from YAML files.
2.  Override settings using Environment Variables (without changing code).
3.  Manage API keys securely.
4.  Understand how to structure your configuration for different environments (Dev vs. Prod).

## Introduction
Welcome, colleague! Hardcoding strings like `"sk-proj-12345..."` or `"localhost:5432"` is the quickest way to accidentally leak credentials or break your production build. In professional software development, we follow the **12-Factor App** methodology: **Store config in the environment**.

Beluga AI's `pkg/config` makes this effortless. It acts as a bridge between your code and the outside world, allowing you to define default values in a YAML file while overriding sensitive secrets with environment variables at runtime.

## Why This Matters

*   **Security**: Never commit secrets to Git. Environment variables keep them ephemeral.
*   **Flexibility**: Change your LLM model or database host without recompiling your Go binary.
*   **Standardization**: `pkg/config` provides built-in validation, ensuring you don't start your app with missing required fields.

## Prerequisites

*   A working Go environment.
*   The `pkg/config` package.
*   Basic understanding of your OS terminal (exporting variables).

## Concepts

### The Hierarchy of Precedence
When Beluga AI loads your configuration, it looks in multiple places. If the same setting exists in multiple spots, it follows this order of precedence (highest wins):

1.  **Environment Variables** (Runtime overrides)
2.  **Configuration File** (`config.yaml`, `config.json`, etc.)
3.  **Default Values** (Set in Go code)

This allows you to have a "sensible default" in `config.yaml` but override it instantly in production.

## Step-by-Step Implementation

### Step 1: The Configuration File

Let's start by defining a base configuration. By default, Beluga AI looks for a file named `config.yaml` in the current directory or a `./config` subdirectory.

Create a file named `config.yaml`:
# config.yaml (Safe to commit to Git)
app_name: "My Beluga Agent"
log_level: "info"

# LLM Provider Defaults
llm_providers:
  - name: "default-gpt"
```
    provider: "openai"
    model_name: "gpt-4o"
    # Note: We do NOT put the api_key here!
    default_call_options:
      temperature: 0.7

Now, let's write Go code to load it.
```go
package main

import (
    "fmt"
    "log"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
    // 1. Initialize the loader
    // DefaultLoaderOptions looks for "config" in "./config" or "."
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }


    fmt.Printf("App: %s\n", cfg.AppName) // Assuming AppName is mapped
    fmt.Printf("LLM Model: %s\n", cfg.LLMProviders[0].ModelName)
}
```

### Step 2: Overriding with Environment Variables

Now, suppose you want to enable debug logging locally, but you don't want to edit the YAML file (which might be shared with your team). You can use an **Environment Variable**.

The Beluga AI default prefix is `BELUGA`. The naming convention is `PREFIX_SECTION_FIELD` (uppercase).

Run your application like this:
bash
```bash
export BELUGA_LOG_LEVEL="debug"
go run main.go
```

The loader detects `BELUGA_LOG_LEVEL`, sees that it matches the `log_level` key in your config struct, and overrides the value from the YAML file.

### Step 3: Managing Secrets (API Keys)

This is the most critical part. **Secrets should strictly come from the environment.**

In your `config.yaml`, you referenced valid providers but left the API keys blank (or used placeholders).
llm_providers:
  - name: "development"
```
    provider: "openai"
    # api_key is omitted here

To provide the key securely at runtime, we map the list index. Since `llm_providers` is a list, we use the index `0`.
# Set the API Key for the first provider (index 0)
bash
```bash
export BELUGA_LLM_PROVIDERS_0_API_KEY="sk-proj-12345..."
```

When you define your `LLMProviderConfig` struct in Go, ensure it has the `mapstructure` tags so the config loader knows how to map these variables.
```go
type LLMProviderConfig struct {
    Name    string `mapstructure:"name"`
    APIKey  string `mapstructure:"api_key"` // secure mapping
    // ...
}
```

### Step 4: Loading from Custom Locations

Sometimes `config.yaml` isn't enough. Maybe you have `config.dev.yaml` and `config.prod.yaml`.

You can programmatically tell the loader where to look.
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

func main() {
    // Configure custom options
    opts := iface.LoaderOptions{
        ConfigName:  "config.prod",     // Look for config.prod.yaml
        ConfigPaths: []string{"/etc/beluga", "."}, 
        EnvPrefix:   "MYAPP",           // Custom prefix MYAPP_LOG_LEVEL
        Validate:    true,
    }

    loader, _ := config.NewLoader(opts)
    cfg, _ := loader.LoadConfig()
    
    // Now you can use MYAPP_API_KEY instead of BELUGA_API_KEY
}
```

## Pro-Tips

*   **Variable Expansion**: `pkg/config` supports standard shell expansion inside your YAML files!
```yaml
    # config.yaml
    database_url: "postgres://${DB_USER}:${DB_PASSWORD}@localhost:5432/db"
```
    This allows you to structure your config file readability while still keeping the actual secret values in the environment.

*   **Validation**: Always use `config.ValidateConfig(cfg)`. It uses strict struct tags (like `validate:"required"`) to ensure you didn't forget to set that `BELUGA_API_KEY` before starting up your expensive GPU cluster.

## Troubleshooting

### "My Env Var isn't being picked up"
1.  **Check the Prefix**: Are you using `BELUGA_` (default) or a custom one?
2.  **Check Case**: It must be UPPERCASE. `beluga_log_level` will be ignored.
3.  **Check Structure**: Nested fields need underscores. `server.port` becomes `BELUGA_SERVER_PORT`. Lists use indices: `providers[0].key` becomes `BELUGA_PROVIDERS_0_KEY`.

### "Config file not found"
The loader does not recursively search directories. It only checks the paths specified in `ConfigPaths`. Ensure you are running the `go run` command from the root directory where `./config/` is visible, or use an absolute path.

## Conclusion

You now have a production-ready configuration strategy.
1.  **Defaults** live in `config.yaml` for developer ease.
2.  **Overrides** happen via `ENV_VARS` for local debugging.
3.  **Secrets** are injected via `ENV_VARS` in CI/CD pipeline, keeping your repo clean.
