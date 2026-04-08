---
title: "Masking Secrets in Logs"
description: "Recipe for automatically masking sensitive values in Go config structures before logging — fail-safe pattern-based secret protection for any config type."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, secret masking, Go log security, credential masking, config safety, production logging recipe"
---

## Problem

You need to log configuration values for debugging but must prevent sensitive data (passwords, tokens) from appearing in logs, which could be exposed in log aggregation systems, error reports, or debugging output.

## Solution

Implement a `SecretMasker` that converts any config struct to a map via JSON round-trip and recursively replaces values whose field names match sensitive patterns. The original struct is never modified.

## Why This Matters

Logs flow through multiple systems (collectors, aggregation pipelines, search indices) and are accessed by many people. The masker operates on a copy of the config, erring on the side of over-masking rather than under-masking. This approach works for any config type — including Beluga AI's `config.ProviderConfig` — because JSON serialization handles struct traversal automatically.

**Note:** Never store raw secrets in config files or structs unless absolutely necessary. Prefer `config.LoadFromEnv` to source sensitive values from the environment. If secrets do appear in config structs (e.g., after loading from a secrets manager), mask them before logging.

## Code Example

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
)

// SecretMasker masks sensitive fields in any JSON-serializable value.
type SecretMasker struct {
	sensitivePatterns []string
	maskValue         string
}

// NewSecretMasker creates a masker with default sensitive field patterns.
func NewSecretMasker() *SecretMasker {
	return &SecretMasker{
		sensitivePatterns: []string{
			"password", "passwd",
			"token", "secret",
			"access_key", "secret_key",
			"private_key", "privatekey",
			"auth_token", "authtoken",
		},
		maskValue: "***REDACTED***",
	}
}

// AddPattern registers an additional field name pattern to mask.
func (sm *SecretMasker) AddPattern(pattern string) {
	sm.sensitivePatterns = append(sm.sensitivePatterns, strings.ToLower(pattern))
}

// MaskAny masks sensitive fields in any JSON-serializable value.
// It returns a map suitable for structured logging.
func (sm *SecretMasker) MaskAny(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("secret masker: marshal: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("secret masker: unmarshal: %w", err)
	}

	return sm.maskMap(raw), nil
}

func (sm *SecretMasker) maskMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if sm.shouldMask(k) {
			out[k] = sm.maskValue
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			out[k] = sm.maskMap(val)
		case []any:
			out[k] = sm.maskSlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

func (sm *SecretMasker) maskSlice(s []any) []any {
	out := make([]any, len(s))
	for i, item := range s {
		if m, ok := item.(map[string]any); ok {
			out[i] = sm.maskMap(m)
		} else {
			out[i] = item
		}
	}
	return out
}

func (sm *SecretMasker) shouldMask(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, pattern := range sm.sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// SafeLog logs a config value with sensitive fields masked.
// Pass any JSON-serializable config struct.
func SafeLog(ctx context.Context, masker *SecretMasker, label string, cfg any) {
	masked, err := masker.MaskAny(cfg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to mask config for logging", "label", label, "error", err)
		return
	}
	// Log the masked map as structured fields — never the original cfg.
	slog.InfoContext(ctx, label, "config", masked)
}

func main() {
	ctx := context.Background()
	masker := NewSecretMasker()

	// Example: mask a ProviderConfig before logging.
	// In production, APIKey would come from os.Getenv, not a literal.
	providerCfg := config.ProviderConfig{
		Model: "gpt-4o",
	}

	SafeLog(ctx, masker, "provider config loaded", providerCfg)

	// Direct use:
	type DBConfig struct {
		Host     string `json:"host"`
		Password string `json:"password"`
		Port     int    `json:"port"`
	}
	db := DBConfig{Host: "localhost", Password: "hunter2", Port: 5432}
	masked, err := masker.MaskAny(db)
	if err != nil {
		slog.Error("mask failed", "error", err)
		return
	}
	out, _ := json.MarshalIndent(masked, "", "  ")
	fmt.Println(string(out))
	// Output:
	// {
	//   "host": "localhost",
	//   "password": "***REDACTED***",
	//   "port": 5432
	// }
}
```

## Explanation

1. **JSON round-trip traversal** — Serializing to JSON and back into `map[string]any` enables recursive traversal of any struct depth without reflection. This handles nested structs, slices of structs, and embedded types uniformly.

2. **Pattern-based detection** — Field names are lowercased and checked for substring matches against the pattern list. This catches variations: `password`, `db_password`, `Password`, `DATABASE_PASSWORD`. Adding new patterns requires no changes to masking logic.

3. **Non-destructive** — The original config value is never modified. A separate masked copy is produced for logging. This ensures the running system continues to use the real (unmasked) values while logs contain only redacted output.

4. **Over-masking preference** — A field named `token_count` would be masked because it contains "token". This is intentional: over-masking is safer than under-masking. If false positives are a problem in your domain, use more specific patterns.

## Testing

```go
func TestSecretMasker_MasksPassword(t *testing.T) {
	masker := NewSecretMasker()

	type Config struct {
		Host     string `json:"host"`
		Password string `json:"password"`
	}
	cfg := Config{Host: "localhost", Password: "hunter2"}

	masked, err := masker.MaskAny(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if masked["password"] != "***REDACTED***" {
		t.Errorf("expected password to be redacted, got %v", masked["password"])
	}
	if masked["host"] != "localhost" {
		t.Errorf("expected host to be preserved, got %v", masked["host"])
	}
}
```

## Variations

### Partial Masking

Show the last four characters of a token for debugging:

```go
func (sm *SecretMasker) partialMask(value string) string {
	if len(value) <= 4 {
		return sm.maskValue
	}
	return "****" + value[len(value)-4:]
}
```

## Related Recipes

- **[Config Hot-reloading](./config-hot-reload)** — Reload configs without restart
- **[PII Redaction](./pii-redaction)** — General PII redaction patterns for user data
