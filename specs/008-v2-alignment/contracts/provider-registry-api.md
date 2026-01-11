# Provider Registry API Contract

**Feature**: V2 Framework Alignment  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This contract defines the provider registry operations for adding new providers to multi-provider packages. These operations follow the standard global registry pattern used across the framework.

---

## Provider Registry Operations

### 1. Register Provider

**Operation**: `RegisterProvider(packageName string, providerName string, factory ProviderFactory) error`

**Purpose**: Register a new provider in a package's global registry.

**Input**:
- `packageName` (string): Name of the package (e.g., "llms", "embeddings")
- `providerName` (string): Name of the provider (e.g., "grok", "gemini")
- `factory` (ProviderFactory): Factory function to create provider instances

**Output**:
- `error`: Error if registration fails

**Behavior**:
- Registers provider in package's global registry
- Provider becomes discoverable via configuration
- Provider can be instantiated using standard factory pattern
- Maintains thread-safe registration

**Errors**:
- `ErrPackageNotFound`: Package does not exist
- `ErrProviderExists`: Provider already registered
- `ErrInvalidFactory`: Factory function is invalid
- `ErrRegistrationFailed`: Registration process failed

---

### 2. Create Provider Instance

**Operation**: `CreateProvider(packageName string, providerName string, config ProviderConfig) (Provider, error)`

**Purpose**: Create a provider instance using the global registry.

**Input**:
- `packageName` (string): Name of the package
- `providerName` (string): Name of the provider to create
- `config` (ProviderConfig): Provider configuration

**Output**:
- `Provider` (interface): Provider instance
- `error`: Error if creation fails

**Behavior**:
- Looks up provider in global registry
- Calls provider factory function with configuration
- Returns provider instance
- Validates configuration before creation

**Errors**:
- `ErrPackageNotFound`: Package does not exist
- `ErrProviderNotFound`: Provider not found in registry
- `ErrInvalidConfig`: Configuration is invalid
- `ErrProviderCreationFailed`: Provider creation failed

---

### 3. List Providers

**Operation**: `ListProviders(packageName string) ([]string, error)`

**Purpose**: List all registered providers in a package.

**Input**:
- `packageName` (string): Name of the package

**Output**:
- `[]string`: List of provider names
- `error`: Error if listing fails

**Behavior**:
- Returns all provider names registered in package
- Returns empty list if no providers registered
- Thread-safe operation

**Errors**:
- `ErrPackageNotFound`: Package does not exist
- `ErrRegistryAccessFailed`: Registry access failed

---

### 4. Get Provider Info

**Operation**: `GetProviderInfo(packageName string, providerName string) (ProviderInfo, error)`

**Purpose**: Get information about a registered provider.

**Input**:
- `packageName` (string): Name of the package
- `providerName` (string): Name of the provider

**Output**:
- `ProviderInfo` (struct): Provider information
- `error`: Error if info retrieval fails

**Behavior**:
- Returns provider metadata (name, type, capabilities, etc.)
- Returns configuration schema
- Returns supported features

**Errors**:
- `ErrPackageNotFound`: Package does not exist
- `ErrProviderNotFound`: Provider not found

---

## Provider Factory Pattern

### Standard Factory Function Signature

```go
type ProviderFactory func(ctx context.Context, config ProviderConfig) (Provider, error)
```

**Parameters**:
- `ctx` (context.Context): Context for cancellation and timeouts
- `config` (ProviderConfig): Provider-specific configuration

**Returns**:
- `Provider` (interface): Provider instance implementing package interface
- `error`: Error if factory fails

**Behavior**:
- Validates configuration
- Creates provider instance
- Initializes provider with configuration
- Returns provider ready for use

---

## Provider Registration Pattern

### Auto-Registration via init()

```go
// pkg/{package}/providers/{provider}/init.go
package {provider}

import "github.com/lookatitude/beluga-ai/pkg/{package}"

func init() {
    {package}.RegisterGlobal("{provider_name}", New{Provider}Factory)
}
```

**Behavior**:
- Auto-registers provider when package is imported
- Uses global registry from package
- Provider becomes available immediately

---

## Provider Configuration

### Standard Configuration Structure

```go
type ProviderConfig struct {
    ProviderName string            `mapstructure:"provider_name" validate:"required"`
    APIKey       string            `mapstructure:"api_key" validate:"required"`
    Endpoint     string            `mapstructure:"endpoint,omitempty"`
    Timeout      time.Duration     `mapstructure:"timeout,omitempty"`
    // Provider-specific fields
    CustomFields map[string]interface{} `mapstructure:",remain"`
}
```

**Validation**:
- Required fields must be present
- Field types must match expected types
- Custom fields validated by provider

---

## Data Types

### ProviderInfo

```go
type ProviderInfo struct {
    Name            string
    Type            ProviderType
    Capabilities    []string
    ConfigSchema    ConfigSchema
    SupportedFeatures []string
    Documentation   string
}
```

### ProviderConfig

```go
type ProviderConfig struct {
    ProviderName string
    // Package-specific configuration fields
    // Validated using go-playground/validator
}
```

---

## Error Codes

- `ErrPackageNotFound`: Package does not exist in framework
- `ErrProviderExists`: Provider already registered
- `ErrProviderNotFound`: Provider not found in registry
- `ErrInvalidFactory`: Factory function is invalid
- `ErrInvalidConfig`: Provider configuration is invalid
- `ErrRegistrationFailed`: Provider registration failed
- `ErrProviderCreationFailed`: Provider instance creation failed
- `ErrRegistryAccessFailed`: Registry access failed

---

## Validation Rules

1. **Provider Name Uniqueness**: Provider names must be unique within a package
2. **Factory Validity**: Factory functions must be valid and callable
3. **Configuration Validation**: All provider configurations must be validated
4. **Thread Safety**: Registry operations must be thread-safe
5. **Backward Compatibility**: Adding providers must not break existing functionality

---

## Example Usage

### Registering a New Provider

```go
// pkg/llms/providers/grok/init.go
package grok

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
    llms.RegisterGlobal("grok", NewGrokFactory)
}

func NewGrokFactory(ctx context.Context, config llms.Config) (llms.LLMCaller, error) {
    // Validate config
    // Create Grok provider instance
    // Return provider
}
```

### Using a Registered Provider

```go
// User code
import (
    "github.com/lookatitude/beluga-ai/pkg/llms"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/grok" // Auto-register
)

config := llms.Config{
    ProviderName: "grok",
    APIKey: "your-api-key",
}

llm, err := llms.NewProvider(ctx, "grok", config)
if err != nil {
    // Handle error
}
```

---

**Status**: Contract complete, ready for implementation.
