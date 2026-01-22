---
title: "Masking Secrets in Logs"
package: "config"
category: "security"
complexity: "intermediate"
---

# Masking Secrets in Logs

## Problem

You need to log configuration values for debugging but must prevent sensitive data (API keys, passwords, tokens) from appearing in logs, which could be exposed in log aggregation systems, error reports, or debugging output.

## Solution

Implement a config logger that automatically masks sensitive fields using field name patterns and custom masking rules. This works because Beluga AI's config structures use consistent field naming, allowing you to identify and mask sensitive fields before logging.

## Code Example
```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "reflect"
    "strings"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/config/iface"
)

var tracer = otel.Tracer("beluga.config.masking")

// SecretMasker masks sensitive values in config structures
type SecretMasker struct {
    sensitiveFields map[string]bool
    maskValue       string
}

// NewSecretMasker creates a new secret masker
func NewSecretMasker() *SecretMasker {
    masker := &SecretMasker{
        sensitiveFields: make(map[string]bool),
        maskValue:       "***REDACTED***",
    }

    // Add common sensitive field patterns
    sensitivePatterns := []string{
        "api_key", "apikey", "apiKey",
        "password", "passwd",
        "token", "secret",
        "access_key", "secret_key",
        "private_key", "privatekey",
        "auth_token", "authToken",
    }
    
    for _, pattern := range sensitivePatterns {
        masker.sensitiveFields[strings.ToLower(pattern)] = true
    }
    
    return masker
}

// AddSensitiveField adds a custom sensitive field pattern
func (sm *SecretMasker) AddSensitiveField(fieldName string) {
    sm.sensitiveFields[strings.ToLower(fieldName)] = true
}

// MaskConfig masks sensitive fields in a config structure
func (sm *SecretMasker) MaskConfig(ctx context.Context, cfg *iface.Config) (map[string]interface{}, error) {
    ctx, span := tracer.Start(ctx, "masker.mask_config")
    defer span.End()
    
    // Convert config to map for masking
    cfgMap, err := sm.configToMap(cfg)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, "failed to convert config")
        return nil, err
    }
    
    // Mask sensitive fields recursively
    masked := sm.maskMap(cfgMap, "")
    
    span.SetAttributes(attribute.Int("masked.fields", sm.countMaskedFields(masked)))
    span.SetStatus(trace.StatusOK, "config masked")
    
    return masked, nil
}

// configToMap converts config struct to map
func (sm *SecretMasker) configToMap(cfg *iface.Config) (map[string]interface{}, error) {
    data, err := json.Marshal(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal config: %w", err)
    }

    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    return result, nil
}

// maskMap recursively masks sensitive fields in a map
func (sm *SecretMasker) maskMap(m map[string]interface{}, prefix string) map[string]interface{} {
    masked := make(map[string]interface{})

    for key, value := range m {
        fullKey := key
        if prefix != "" {
            fullKey = prefix + "." + key
        }
        
        lowerKey := strings.ToLower(key)
        
        // Check if this field should be masked
        if sm.shouldMask(lowerKey) {
            masked[key] = sm.maskValue
            continue
        }
        
        // Recursively process nested maps
        if nestedMap, ok := value.(map[string]interface{}); ok {
            masked[key] = sm.maskMap(nestedMap, fullKey)
            continue
        }
        
        // Process slices
        if slice, ok := value.([]interface{}); ok {
            masked[key] = sm.maskSlice(slice, fullKey)
            continue
        }
        
        // Keep non-sensitive values as-is
        masked[key] = value
    }
    
    return masked
}

// maskSlice masks sensitive fields in a slice
func (sm *SecretMasker) maskSlice(slice []interface{}, prefix string) []interface{} {
    masked := make([]interface{}, len(slice))

    for i, item := range slice {
        if itemMap, ok := item.(map[string]interface{}); ok {
            masked[i] = sm.maskMap(itemMap, prefix)
        } else {
            masked[i] = item
        }
    }
    
    return masked
}

// shouldMask checks if a field should be masked
func (sm *SecretMasker) shouldMask(fieldName string) bool {
    lowerName := strings.ToLower(fieldName)

    // Check exact match
    if sm.sensitiveFields[lowerName] {
        return true
    }
    
    // Check if field name contains sensitive pattern
    for pattern := range sm.sensitiveFields {
        if strings.Contains(lowerName, pattern) {
            return true
        }
    }
    
    return false
}

// countMaskedFields counts how many fields were masked
func (sm *SecretMasker) countMaskedFields(m map[string]interface{}) int {
    count := 0
    for _, value := range m {
        if str, ok := value.(string); ok && str == sm.maskValue {
            count++
        } else if nested, ok := value.(map[string]interface{}); ok {
            count += sm.countMaskedFields(nested)
        } else if slice, ok := value.([]interface{}); ok {
            for _, item := range slice {
                if itemMap, ok := item.(map[string]interface{}); ok {
                    count += sm.countMaskedFields(itemMap)
                }
            }
        }
    }
    return count
}

// SafeLogConfig logs config with masked secrets
func SafeLogConfig(ctx context.Context, cfg *iface.Config) {
    masker := NewSecretMasker()
    masked, err := masker.MaskConfig(ctx, cfg)
    if err != nil {
        log.Printf("Failed to mask config: %v", err)
        return
    }
    
    maskedJSON, _ := json.MarshalIndent(masked, "", "  ")
    log.Printf("Config (secrets masked):\n%s", maskedJSON)
}

// SafeString returns a safe string representation of config
func SafeString(cfg *iface.Config) string {
    masker := NewSecretMasker()
    masked, err := masker.MaskConfig(context.Background(), cfg)
    if err != nil {
        return "<error masking config>"
    }
    
    maskedJSON, _ := json.MarshalIndent(masked, "", "  ")
    return string(maskedJSON)
}

func main() {
    ctx := context.Background()

    // Example: Load config
    // cfg, _ := config.LoadConfig()
    
    // Log config safely
    // SafeLogConfig(ctx, cfg)
    
    // Or get safe string representation
    // safeStr := SafeString(cfg)
    // fmt.Println(safeStr)
    fmt.Println("Secret masker created successfully")
}
```

## Explanation

Let's break down what's happening:

1. **Pattern-based detection** - Notice how we check field names against a set of sensitive patterns. This catches variations like "api_key", "apiKey", and "API_KEY" without needing exact matches. This is important because different config sources might use different naming conventions.

2. **Recursive masking** - We recursively process nested maps and slices. This ensures that sensitive fields deep in the config structure (like `llm_providers[0].api_key`) are also masked.

3. **Non-destructive masking** - The original config is never modified. We create a new masked structure, so the original config remains intact for actual use.

```go
**Key insight:** Always mask at the logging boundary, not in the config structure itself. This keeps the config usable while protecting logs.

## Testing

```
Here's how to test this solution:
```go
func TestSecretMasker_MasksAPIKeys(t *testing.T) {
    masker := NewSecretMasker()
    
    cfg := &iface.Config{
        LLMProviders: []schema.LLMProviderConfig{
            {
                Name:   "test",
                APIKey: "sk-secret-key-12345",
            },
        },
    }
    
    masked, err := masker.MaskConfig(context.Background(), cfg)
    require.NoError(t, err)
    
    // Check that API key is masked
    providers := masked["llm_providers"].([]interface{})
    provider := providers[0].(map[string]interface{})
    
    require.Equal(t, "***REDACTED***", provider["api_key"])
    require.Equal(t, "test", provider["name"]) // Non-sensitive field preserved
}

## Variations

### Custom Mask Values

Use different mask values per field type:
type FieldMasker struct {
    maskValues map[string]string
}

func (fm *FieldMasker) GetMaskValue(fieldName string) string {
    if mask, exists := fm.maskValues[fieldName]; exists {
        return mask
    }
    return "***REDACTED***"
}
```

### Partial Masking

Show partial values (e.g., last 4 characters):
```go
func (sm *SecretMasker) partialMask(value string) string {
    if len(value) <= 4 {
        return sm.maskValue
    }
    return "****" + value[len(value)-4:]
}
```

## Related Recipes

- **[Config Hot-reloading in Production](./config-hot-reloading-production.md)** - Hot-reload configs safely
- **[Safety PII Redaction in Logs](./safety-pii-redaction-logs.md)** - General PII redaction patterns
- **[Config Package Guide](../guides/config-providers.md)** - For a deeper understanding of config management
