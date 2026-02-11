---
title: "Masking Secrets in Logs"
description: "Automatically mask sensitive configuration data like API keys and passwords before they appear in logs."
---

## Problem

You need to log configuration values for debugging but must prevent sensitive data (API keys, passwords, tokens) from appearing in logs, which could be exposed in log aggregation systems, error reports, or debugging output. This is a pervasive problem in production systems: configuration often contains both sensitive credentials (API keys, database passwords, OAuth tokens) and non-sensitive operational settings (timeouts, feature flags, resource limits). Logging full configurations is invaluable for debugging—knowing which config values were active when an issue occurred is essential for reproducing and diagnosing bugs. However, logs flow through multiple systems (log collectors, aggregation pipelines, search indices, error tracking tools) and are accessed by many people (developers, operators, support staff). Exposing secrets in logs creates security risks: compromised credentials, compliance violations, and potential data breaches. The challenge is logging enough configuration detail to debug issues while automatically protecting sensitive fields without requiring developers to manually redact every log statement.

## Solution

Implement a config logger that automatically masks sensitive fields using field name patterns and custom masking rules. This works because Beluga AI's config structures use consistent field naming, allowing you to identify and mask sensitive fields before logging. The design uses pattern-based field detection: fields with names like "api_key", "password", "token", or "secret" are automatically masked. The masker recursively traverses config structures (nested maps, slices, structs), applying masking rules at every level. This approach is fail-safe: it operates on a copy of the config, never modifying the original, and it errs on the side of over-masking rather than under-masking. The key insight is that sensitive field names follow predictable conventions, making pattern-based detection reliable. This integrates with Beluga's config package and structured logging, ensuring all config logs are automatically sanitized.

The masker converts configs to maps via JSON serialization, enabling recursive traversal without reflection. Pattern matching is case-insensitive and supports both exact matches ("api_key") and substring matches (catching "user_api_key", "llm_api_key"). The masker tracks how many fields were masked, providing observability into redaction operations without exposing the actual secrets.

## Code Example

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "strings"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/config"
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
func (sm *SecretMasker) MaskConfig(ctx context.Context, cfg *config.Config) (map[string]interface{}, error) {
    ctx, span := tracer.Start(ctx, "masker.mask_config")
    defer span.End()

    cfgMap, err := sm.configToMap(cfg)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, "failed to convert config")
        return nil, err
    }

    masked := sm.maskMap(cfgMap, "")

    span.SetAttributes(attribute.Int("masked.fields", sm.countMaskedFields(masked)))
    span.SetStatus(trace.StatusOK, "config masked")

    return masked, nil
}

// configToMap converts config struct to map
func (sm *SecretMasker) configToMap(cfg *config.Config) (map[string]interface{}, error) {
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

        if sm.shouldMask(lowerKey) {
            masked[key] = sm.maskValue
            continue
        }

        if nestedMap, ok := value.(map[string]interface{}); ok {
            masked[key] = sm.maskMap(nestedMap, fullKey)
            continue
        }

        if slice, ok := value.([]interface{}); ok {
            masked[key] = sm.maskSlice(slice, fullKey)
            continue
        }

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

    if sm.sensitiveFields[lowerName] {
        return true
    }

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
func SafeLogConfig(ctx context.Context, cfg *config.Config) {
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
func SafeString(cfg *config.Config) string {
    masker := NewSecretMasker()
    masked, err := masker.MaskConfig(context.Background(), cfg)
    if err != nil {
        return "<error masking config>"
    }

    maskedJSON, _ := json.MarshalIndent(masked, "", "  ")
    return string(maskedJSON)
}

func main() {
    // Load config and log safely
    // cfg, _ := config.LoadFromFile("./config.yaml")
    // SafeLogConfig(context.Background(), cfg)
    // safeStr := SafeString(cfg)
    // fmt.Println(safeStr)
    fmt.Println("Secret masker created successfully")
}
```

## Explanation

1. **Pattern-based detection** — Field names are checked against a set of sensitive patterns. This catches variations like "api_key", "apiKey", and "API_KEY" without needing exact matches. This matters because configuration formats vary: YAML configs might use snake_case ("api_key"), JSON might use camelCase ("apiKey"), and environment variables use UPPER_SNAKE_CASE ("API_KEY"). Case-insensitive substring matching handles all these variations automatically. This approach is also forward-compatible: adding new sensitive fields (like "jwt_secret" or "oauth_token") only requires adding patterns to the registry, not modifying masking logic. The trade-off is potential false positives—a field named "secret_count" would be masked—but this is acceptable because over-masking is safer than under-masking in security contexts.

2. **Recursive masking** — Nested maps and slices are processed recursively. This ensures that sensitive fields deep in the config structure (like `llm_providers[0].api_key`) are also masked. This matters because modern configs are deeply nested: a Beluga config might have LLM provider configs nested under providers, each with their own credentials; RAG configs with vector store credentials; and auth configs with OAuth client secrets. Without recursive traversal, nested secrets would leak through. The recursive design handles arbitrary nesting depth, making it robust to config structure changes. The masker processes both maps (nested objects) and slices (arrays of configs), ensuring comprehensive coverage of all possible config shapes.

3. **Non-destructive masking** — The original config is never modified. A new masked structure is created, so the original config remains intact for actual use. This matters because masking must not interfere with config consumption—the actual system needs real API keys to function, not redacted placeholders. By creating a copy (via JSON round-trip), the masker produces a sanitized version for logging while leaving the original unchanged. This design also makes masking composable: you can log both masked and unmasked versions (to different destinations) without interference. The JSON round-trip approach works because Beluga configs are JSON-serializable, and it automatically handles private fields (which won't appear in the JSON output).

Always mask at the logging boundary, not in the config structure itself. This keeps the config usable while protecting logs.

## Testing

```go
func TestSecretMasker_MasksAPIKeys(t *testing.T) {
    masker := NewSecretMasker()

    cfg := &config.Config{
        LLMProviders: []schema.LLMProviderConfig{
            {
                Name:   "test",
                APIKey: "sk-secret-key-12345",
            },
        },
    }

    masked, err := masker.MaskConfig(context.Background(), cfg)
    require.NoError(t, err)

    providers := masked["llm_providers"].([]interface{})
    provider := providers[0].(map[string]interface{})

    require.Equal(t, "***REDACTED***", provider["api_key"])
    require.Equal(t, "test", provider["name"])
}
```

## Variations

### Custom Mask Values

Use different mask values per field type:

```go
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

- **[Config Hot-reloading](./config-hot-reload)** — Hot-reload configs safely
- **[PII Redaction](./pii-redaction)** — General PII redaction patterns
