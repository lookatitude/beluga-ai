---
title: HashiCorp Vault Connector
description: Integrate HashiCorp Vault for secure secret management with Beluga AI, including API key retrieval, caching, and OpenTelemetry tracing.
---

Beluga AI applications frequently manage sensitive credentials such as LLM API keys, database passwords, and service tokens. HashiCorp Vault provides a centralized secrets engine that keeps these values out of source code and environment files. This guide shows how to build a Vault connector that retrieves secrets on demand and integrates with the Beluga AI configuration layer.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- A running HashiCorp Vault instance (local dev server or remote)
- The Vault Go client library

## Installation

Install the Vault client SDK:

```bash
go get github.com/hashicorp/vault/api
```

For local development, start Vault in dev mode and export the required environment variables:

```bash
vault server -dev
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='dev-root-token'
```

## Basic Vault Connection

Create a Vault client that reads connection details from environment variables:

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

	health, err := client.Sys().Health()
	if err != nil {
		log.Fatalf("Failed to connect to vault: %v", err)
	}

	fmt.Printf("Vault status: %s\n", health.ClusterName)
}
```

## Secret Retrieval

Wrap the Vault client in a loader that reads individual keys from secret paths:

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
		return nil, fmt.Errorf("failed to read secret at %s: %w", path, err)
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
		return "", fmt.Errorf("key %s not found or not a string at path %s", key, path)
	}
	return value, nil
}
```

## Integration with Beluga AI Config

Build a configuration provider that retrieves LLM API keys from Vault and caches them locally:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/vault/api"
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

	if cached, ok := p.cache[cacheKey]; ok {
		return cached.(string), nil
	}

	path := fmt.Sprintf("secret/data/beluga/llm/%s", provider)
	apiKey, err := p.vault.GetString(path, "api_key")
	if err != nil {
		return "", fmt.Errorf("failed to get API key for %s: %w", provider, err)
	}

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

	fmt.Printf("Retrieved API key (length %d)\n", len(apiKey))
}
```

## Storing Secrets in Vault

Use the Vault CLI to write and verify secrets:

```bash
# Write a secret
vault kv put secret/beluga/llm/openai api_key="sk-..."

# Read it back
vault kv get secret/beluga/llm/openai
```

## Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `VAULT_ADDR` | Vault server address | `http://127.0.0.1:8200` | No |
| `VAULT_TOKEN` | Authentication token | -- | Yes |
| `SecretPath` | Base path for Beluga secrets | `secret/data/beluga` | No |
| `CacheTTL` | In-memory cache time-to-live | `5m` | No |

## Production-Ready Example with OTel Tracing

For production deployments, add time-based cache expiration, concurrency-safe access, and OpenTelemetry tracing:

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

	// Check cache under read lock
	p.mu.RLock()
	if cached, ok := p.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		p.mu.RUnlock()
		span.SetAttributes(attribute.Bool("vault.cache_hit", true))
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
		err := fmt.Errorf("secret not found at %s", path)
		span.RecordError(err)
		return "", err
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		err := fmt.Errorf("unexpected data format at %s", path)
		span.RecordError(err)
		return "", err
	}

	value, ok := data[key].(string)
	if !ok {
		err := fmt.Errorf("key %s not found at %s", key, path)
		span.RecordError(err)
		return "", err
	}

	// Update cache under write lock
	p.mu.Lock()
	p.cache[cacheKey] = cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.mu.Unlock()

	span.SetAttributes(attribute.Bool("vault.cache_hit", false))
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

	fmt.Printf("Retrieved API key (length %d)\n", len(apiKey))
}
```

## Troubleshooting

### Connection refused

The Vault server is unreachable. Verify the address and that Vault is running:

```bash
vault status
export VAULT_ADDR='http://127.0.0.1:8200'
```

### Permission denied

The token lacks the required policy. Inspect its capabilities:

```bash
vault token capabilities secret/beluga/llm/openai
```

## Production Considerations

- **Use AppRole authentication** instead of static tokens for automated workloads.
- **Implement TTL-based caching** to reduce Vault API calls under load.
- **Add fallback logic** that reads from environment variables when Vault is temporarily unavailable.
- **Rotate secrets regularly** using Vault's dynamic secrets or lease renewal.
- **Audit secret access** through Vault's built-in audit log.

## Related Resources

- [Viper and Environment Overrides](/integrations/viper-environment) -- Configuration management with Viper
- [Infrastructure Integrations](/integrations/infrastructure) -- Deployment and infrastructure overview
