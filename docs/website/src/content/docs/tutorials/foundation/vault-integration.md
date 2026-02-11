---
title: Vault and Secrets Manager Integration
description: Integrate HashiCorp Vault or other secrets managers with Beluga AI for production-grade secret management.
---

Environment variables are better than hardcoded secrets, but they have limitations: they are visible in process lists, difficult to rotate dynamically, and lack fine-grained access control. In AI applications, these limitations are amplified because a single service may consume API keys from multiple LLM providers, embedding services, and vector databases — each with different rotation schedules and access policies. Secrets managers like HashiCorp Vault solve these issues with dynamic secrets, audit logging, and automated rotation.

## What You Will Build

A secrets provider interface and Vault implementation that loads API keys and credentials dynamically, integrating with Beluga AI's configuration system.

## Prerequisites

- Understanding of [Environment and Secret Management](/tutorials/foundation/environment-secrets)
- HashiCorp Vault installed or a development server available

## Step 1: Define a Secret Provider Interface

Create an abstraction that decouples your application from any specific secrets backend. This interface follows Beluga AI's interface-first design principle — define the contract, then implement it for specific backends. The two-parameter design (`path` and `key`) maps naturally to Vault's path-based secret organization but is general enough to work with AWS Secrets Manager, GCP Secret Manager, or Azure Key Vault.

```go
package main

import (
    "context"
    "fmt"
)

// SecretProvider retrieves secrets from an external store.
type SecretProvider interface {
    // GetSecret fetches a secret value by path and key.
    GetSecret(ctx context.Context, path string, key string) (string, error)
}
```

## Step 2: Implement the Vault Provider

Use the Vault API client to fetch secrets from a KV v2 secrets engine. The KV v2 engine is the recommended choice for static secrets like API keys because it provides versioning and soft-delete capabilities. The implementation handles the KV v2 data nesting (`secret.Data["data"]`) which is a common source of confusion when working with Vault directly.

```go
import "github.com/hashicorp/vault/api"

type VaultProvider struct {
    client *api.Client
}

func NewVaultProvider(address, token string) (*VaultProvider, error) {
    vaultCfg := api.DefaultConfig()
    vaultCfg.Address = address

    client, err := api.NewClient(vaultCfg)
    if err != nil {
        return nil, fmt.Errorf("vault: create client: %w", err)
    }

    client.SetToken(token)
    return &VaultProvider{client: client}, nil
}

func (v *VaultProvider) GetSecret(ctx context.Context, path string, key string) (string, error) {
    secret, err := v.client.Logical().ReadWithContext(ctx, path)
    if err != nil {
        return "", fmt.Errorf("vault: read %s: %w", path, err)
    }
    if secret == nil {
        return "", fmt.Errorf("vault: secret not found at %s", path)
    }

    // KV v2 nests data under a "data" key
    data, ok := secret.Data["data"].(map[string]interface{})
    if !ok {
        data = secret.Data
    }

    val, ok := data[key].(string)
    if !ok {
        return "", fmt.Errorf("vault: key %q not found in secret %s", key, path)
    }

    return val, nil
}
```

## Step 3: Populate Configuration from Vault

Fetch secrets at startup and populate the configuration struct. This approach combines file-based configuration for non-sensitive values with Vault for secrets, which keeps the separation of concerns clean: the config file describes the application's shape, while Vault provides the sensitive values. The final `config.Validate` call ensures the complete configuration is valid after all sources have been merged.

```go
import "github.com/lookatitude/beluga-ai/config"

type AppConfig struct {
    Host    string `json:"host" default:"localhost"`
    Port    int    `json:"port" default:"8080"`
    APIKey  string `json:"api_key" required:"true"`
    DBUrl   string `json:"db_url" required:"true"`
}

func loadConfigWithVault(ctx context.Context, vault SecretProvider) (AppConfig, error) {
    // Load base config from file
    cfg, err := config.Load[AppConfig]("config.json")
    if err != nil {
        return cfg, fmt.Errorf("load config: %w", err)
    }

    // Fetch secrets from Vault
    apiKey, err := vault.GetSecret(ctx, "secret/data/myapp/llm", "api_key")
    if err != nil {
        return cfg, fmt.Errorf("fetch api_key: %w", err)
    }
    cfg.APIKey = apiKey

    dbURL, err := vault.GetSecret(ctx, "secret/data/myapp/db", "url")
    if err != nil {
        return cfg, fmt.Errorf("fetch db_url: %w", err)
    }
    cfg.DBUrl = dbURL

    // Validate the populated config
    if err := config.Validate(&cfg); err != nil {
        return cfg, fmt.Errorf("validate config: %w", err)
    }

    return cfg, nil
}
```

## Step 4: Dynamic Secret Refresh

For secrets that rotate (database credentials, temporary tokens), fetch them on demand rather than caching them at startup. This per-request approach ensures your application always uses current credentials, which is critical when Vault's database secret engine issues short-lived credentials with automatic rotation. The trade-off is additional latency per request, which can be mitigated with short-lived local caching if needed.

```go
type DynamicConfig struct {
    vault     SecretProvider
    basePath  string
}

func NewDynamicConfig(vault SecretProvider, basePath string) *DynamicConfig {
    return &DynamicConfig{vault: vault, basePath: basePath}
}

// GetAPIKey fetches the current API key. Call this per-request to handle rotation.
func (d *DynamicConfig) GetAPIKey(ctx context.Context) (string, error) {
    return d.vault.GetSecret(ctx, d.basePath+"/llm", "api_key")
}

// GetDBCredentials fetches database credentials that may have been rotated.
func (d *DynamicConfig) GetDBCredentials(ctx context.Context) (string, string, error) {
    user, err := d.vault.GetSecret(ctx, d.basePath+"/db", "username")
    if err != nil {
        return "", "", err
    }
    pass, err := d.vault.GetSecret(ctx, d.basePath+"/db", "password")
    if err != nil {
        return "", "", err
    }
    return user, pass, nil
}
```

## Step 5: Full Example

```go
func main() {
    ctx := context.Background()

    // In production, use Kubernetes auth, AppRole, or similar — not a root token
    vault, err := NewVaultProvider("http://localhost:8200", "dev-root-token")
    if err != nil {
        fmt.Printf("Vault init failed: %v\n", err)
        return
    }

    cfg, err := loadConfigWithVault(ctx, vault)
    if err != nil {
        fmt.Printf("Config load failed: %v\n", err)
        return
    }

    fmt.Printf("Config loaded: host=%s, api_key set=%v\n", cfg.Host, cfg.APIKey != "")
}
```

## Verification

1. Start a local Vault dev server: `vault server -dev -dev-root-token-id="dev-root-token"`
2. Store a secret: `vault kv put secret/myapp/llm api_key="sk-test-key"`
3. Run the Go program and verify it retrieves the secret.

## Next Steps

- [Environment and Secret Management](/tutorials/foundation/environment-secrets) — Base configuration patterns
- [Health Checks](/tutorials/foundation/health-checks) — Monitor service health
