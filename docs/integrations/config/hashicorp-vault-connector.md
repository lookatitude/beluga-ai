# HashiCorp Vault Connector

Welcome, colleague! In this integration guide, we're going to integrate HashiCorp Vault for secure secret management with Beluga AI. This enables secure storage and retrieval of API keys, credentials, and other sensitive configuration.

## What you will build

You will create a Vault connector that securely retrieves secrets for Beluga AI configuration, enabling secure secret management without hardcoding credentials in your application.

## Learning Objectives

- ✅ Connect to HashiCorp Vault
- ✅ Retrieve secrets from Vault
- ✅ Integrate Vault with Beluga AI config
- ✅ Understand secret management best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- HashiCorp Vault server (local or remote)
- Vault client library

## Step 1: Setup and Installation

Install Vault client:
bash
```bash
go get github.com/hashicorp/vault/api
```

Start Vault (for local testing):
vault server -dev
bash
```bash
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='dev-root-token'
```

## Step 2: Basic Vault Connection

Create a Vault client:
```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/hashicorp/vault/api"
)

func NewVaultClient() (*api.Client, error) {
    config := api.DefaultConfig()
    config.Address = os.Getenv("VAULT_ADDR")
    if config.Address == "" {
        config.Address = "http://127.0.0.1:8200"
    }
    
    client, err := api.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }
    
    // Set token
    token := os.Getenv("VAULT_TOKEN")
    if token == "" {
        return nil, fmt.Errorf("VAULT_TOKEN environment variable is required")
    }
    client.SetToken(token)
    
    return client, nil
}

func main() {
    client, err := NewVaultClient()
    if err != nil {
        log.Fatalf("Failed to create vault client: %v", err)
    }
    
    // Test connection
    health, err := client.Sys().Health()
    if err != nil {
        log.Fatalf("Failed to connect to vault: %v", err)
    }
    
    fmt.Printf("Vault is %s\n", health.Status)
}
```

## Step 3: Secret Retrieval

Retrieve secrets from Vault:
```go
type VaultSecretLoader struct {
    client *api.Client
}

func NewVaultSecretLoader() (*VaultSecretLoader, error) {
    client, err := NewVaultClient()
    if err != nil {
        return nil, err
    }
    
    return &VaultSecretLoader{client: client}, nil
}

func (l *VaultSecretLoader) GetSecret(path string) (map[string]interface{}, error) {
    secret, err := l.client.Logical().Read(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read secret: %w", err)
    }
    
    if secret == nil {
        return nil, fmt.Errorf("secret not found at path: %s", path)
    }
    
    return secret.Data, nil
}

func (l *VaultSecretLoader) GetString(path, key string) (string, error) {
    data, err := l.GetSecret(path)
    if err != nil {
        return "", err
    }
    
    value, ok := data[key].(string)
    if !ok {
        return "", fmt.Errorf("key %s not found or not a string", key)
    }

    
    return value, nil
}
```

## Step 4: Integration with Beluga AI

Integrate Vault with Beluga AI config:
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/hashicorp/vault/api"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

type VaultConfigProvider struct {
    vault *VaultSecretLoader
    cache map[string]interface{}
}

func NewVaultConfigProvider() (*VaultConfigProvider, error) {
    vault, err := NewVaultSecretLoader()
    if err != nil {
        return nil, err
    }
    
    return &VaultConfigProvider{
        vault: vault,
        cache: make(map[string]interface{}),
    }, nil
}

func (p *VaultConfigProvider) GetLLMAPIKey(ctx context.Context, provider string) (string, error) {
    cacheKey := fmt.Sprintf("llm.%s.api_key", provider)
    
    // Check cache
    if cached, ok := p.cache[cacheKey]; ok {
        return cached.(string), nil
    }
    
    // Retrieve from Vault
    path := fmt.Sprintf("secret/data/beluga/llm/%s", provider)
    apiKey, err := p.vault.GetString(path, "api_key")
    if err != nil {
        return "", fmt.Errorf("failed to get API key: %w", err)
    }
    
    // Cache it
    p.cache[cacheKey] = apiKey
    
    return apiKey, nil
}

func main() {
    provider, err := NewVaultConfigProvider()
    if err != nil {
        log.Fatalf("Failed to create vault provider: %v", err)
    }
    
    ctx := context.Background()
    apiKey, err := provider.GetLLMAPIKey(ctx, "openai")
    if err != nil {
        log.Fatalf("Failed to get API key: %v", err)
    }

    
    fmt.Printf("API Key: %s\n", maskAPIKey(apiKey))
}
```

## Step 5: Secret Management Setup

Store secrets in Vault:
# Write a secret
vault kv put secret/beluga/llm/openai api_key="sk-..."

# Read a secret
vault kv get secret/beluga/llm/openai
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `VaultAddr` | Vault server address | `http://127.0.0.1:8200` | No |
| `VaultToken` | Authentication token | - | Yes |
| `SecretPath` | Base path for secrets | `secret/data/beluga` | No |
| `CacheTTL` | Cache time-to-live | `5m` | No |

## Common Issues

### "Vault connection refused"

**Problem**: Vault server not running or wrong address.

**Solution**: Check Vault server status:vault status
bash
```bash
export VAULT_ADDR='http://127.0.0.1:8200'
```

### "Permission denied"

**Problem**: Token doesn't have permission to read secrets.

**Solution**: Check token permissions:vault token capabilities secret/beluga/llm/openai
```

## Production Considerations

When using Vault in production:

- **Use AppRole authentication**: More secure than tokens
- **Implement caching**: Reduce Vault API calls
- **Handle failures gracefully**: Fallback to environment variables
- **Rotate secrets**: Implement secret rotation
- **Monitor access**: Log all secret access

## Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "sync"
    "time"

    "github.com/hashicorp/vault/api"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionVaultProvider struct {
    client *api.Client
    cache  map[string]cachedSecret
    mu     sync.RWMutex
    tracer trace.Tracer
}

type cachedSecret struct {
    value     string
    expiresAt time.Time
}

func NewProductionVaultProvider() (*ProductionVaultProvider, error) {
    config := api.DefaultConfig()
    config.Address = os.Getenv("VAULT_ADDR")
    if config.Address == "" {
        return nil, fmt.Errorf("VAULT_ADDR environment variable is required")
    }
    
    client, err := api.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }
    
    // Use AppRole or token
    token := os.Getenv("VAULT_TOKEN")
    if token != "" {
        client.SetToken(token)
    }
    
    return &ProductionVaultProvider{
        client: client,
        cache:  make(map[string]cachedSecret),
        tracer: otel.Tracer("beluga.config.vault"),
    }, nil
}

func (p *ProductionVaultProvider) GetSecret(ctx context.Context, path, key string) (string, error) {
    ctx, span := p.tracer.Start(ctx, "vault.GetSecret",
        trace.WithAttributes(
            attribute.String("vault.path", path),
            attribute.String("vault.key", key),
        ),
    )
    defer span.End()
    
    cacheKey := fmt.Sprintf("%s:%s", path, key)
    
    // Check cache
    p.mu.RLock()
    if cached, ok := p.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
        p.mu.RUnlock()
        return cached.value, nil
    }
    p.mu.RUnlock()
    
    // Retrieve from Vault
    secret, err := p.client.Logical().Read(path)
    if err != nil {
        span.RecordError(err)
        return "", fmt.Errorf("failed to read secret: %w", err)
    }
    
    if secret == nil {
        err := fmt.Errorf("secret not found")
        span.RecordError(err)
        return "", err
    }
    
    value, ok := secret.Data["data"].(map[string]interface{})[key].(string)
    if !ok {
        err := fmt.Errorf("key %s not found", key)
        span.RecordError(err)
        return "", err
    }
    
    // Cache it
    p.mu.Lock()
    p.cache[cacheKey] = cachedSecret{
        value:     value,
        expiresAt: time.Now().Add(5 * time.Minute),
    }
    p.mu.Unlock()
    
    span.SetAttributes(attribute.Bool("vault.cached", false))
    return value, nil
}

func main() {
    provider, err := NewProductionVaultProvider()
    if err != nil {
        log.Fatalf("Failed to create vault provider: %v", err)
    }
    
    ctx := context.Background()
    apiKey, err := provider.GetSecret(ctx, "secret/data/beluga/llm/openai", "api_key")
    if err != nil {
        log.Fatalf("Failed to get secret: %v", err)
    }

    
    fmt.Printf("Retrieved API key: %s\n", maskAPIKey(apiKey))
}
```

## Next Steps

Congratulations! You've integrated HashiCorp Vault with Beluga AI. Next, learn how to:

- **[Viper & Environment Overrides](./viper-environment-overrides.md)** - Configuration management
- **[Config Package Documentation](../../api-docs/packages/config.md)** - Deep dive into config package
- **[Security Best Practices](../../best-practices.md)** - Security patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
