# Vault & Secrets Manager Integration

In this tutorial, you'll learn how to integrate HashiCorp Vault (or other secrets managers) with Beluga AI's configuration system for secure, production-grade secret management.

## Learning Objectives

- ✅ Understand the security risks of plain-text secrets
- ✅ Integrate HashiCorp Vault with Beluga AI
- ✅ Implement dynamic secret loading
- ✅ Handle secret rotation

## Prerequisites

- Basic understanding of configuration (see [Environment & Secret Management](./config-environment-secrets.md))
- HashiCorp Vault installed and running (or a mock)
- Go 1.24+

## Why Use a Secrets Manager?

Environment variables are better than hardcoded secrets, but they have limitations:
- They are often visible in process lists.
- They are hard to rotate dynamically.
- They lack fine-grained access control.
- They don't support audit logging for access.

Secrets managers like Vault solve these issues.

## Step 1: Define a Secret Provider Interface

Beluga AI's config system is extensible. Let's define an interface for fetching secrets.
```go
package main

import (
    "context"
    "fmt"
)

type SecretProvider interface {
    GetSecret(ctx context.Context, path string, key string) (string, error)
}
```

## Step 2: Implement Vault Provider

Using the Vault API client to fetch secrets.
```go
import (
    "github.com/hashicorp/vault/api"
)
go
type VaultProvider struct {
    client *api.Client
}

func NewVaultProvider(address, token string) (*VaultProvider, error) {
    config := api.DefaultConfig()
    config.Address = address
    
    client, err := api.NewClient(config)
    if err != nil {
        return nil, err
    }
    
    client.SetToken(token)
    return &VaultProvider{client: client}, nil
}

func (v *VaultProvider) GetSecret(ctx context.Context, path string, key string) (string, error) {
    secret, err := v.client.Logical().Read(path)
    if err != nil {
        return "", err
    }
    if secret == nil {
        return "", fmt.Errorf("secret not found at %s", path)
    }
    
    data, ok := secret.Data["data"].(map[string]interface{})
    if !ok {
        // Handle different Vault KV versions
        data = secret.Data
    }
    
    val, ok := data[key].(string)
    if !ok {
        return "", fmt.Errorf("key %s not found in secret %s", key, path)
    }

    
    return val, nil
}
```

## Step 3: Integrate with Configuration Loading

Now we use this provider to populate our configuration struct.
```go
type AppConfig struct \{
    OpenAIKey    string
    DatabaseURL  string
}
go
func LoadConfigWithVault(ctx context.Context, vault *VaultProvider) (*AppConfig, error) {
    // Fetch secrets
    openAIKey, err := vault.GetSecret(ctx, "secret/data/myapp/llm", "openai_key")
    if err != nil {
        return nil, err
    }
    
    dbURL, err := vault.GetSecret(ctx, "secret/data/myapp/db", "url")
    if err != nil {
        return nil, err
    }

    
    return &AppConfig\{
        OpenAIKey:   openAIKey,
        DatabaseURL: dbURL,
    }, nil
}
```

## Step 4: Dynamic Secret Loading (Advanced)

For secrets that rotate (like database credentials), you might want to fetch them just-in-time or use a callback.
```go
type DynamicConfig struct {
    vault *VaultProvider
}

func (d *DynamicConfig) GetOpenAIKey(ctx context.Context) (string, error) {
    return d.vault.GetSecret(ctx, "secret/data/myapp/llm", "openai_key")
}

## Step 5: Full Example
func main() {
    ctx := context.Background()
    
    // Initialize Vault Provider
    // In production, use Kubernetes Auth or similar instead of token
    vault, err := NewVaultProvider("http://localhost:8200", "my-root-token")
    if err != nil {
        panic(err)
    }
    
    // Load Config
    config, err := LoadConfigWithVault(ctx, vault)
    if err != nil {
        panic(err)
    }

    
    fmt.Printf("Successfully loaded configuration from Vault\n")
    // Use config.OpenAIKey...
}
```

## Verification

1. Start a local Vault dev server: `vault server -dev -dev-root-token-id="my-root-token"`
2. Write a secret: `vault kv put secret/myapp/llm openai_key="sk-..."`
3. Run the Go program and verify it retrieves the secret.

## Next Steps

- **[Environment & Secret Management](./config-environment-secrets.md)** - Learn basic config patterns
- **[Production Deployment](../../getting-started/07-production-deployment.md)** - Deploy securely
