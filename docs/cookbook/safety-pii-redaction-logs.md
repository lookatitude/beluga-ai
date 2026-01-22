---
title: "PII Redaction in Logs"
package: "safety"
category: "security"
complexity: "intermediate"
---

# PII Redaction in Logs

## Problem

You need to redact personally identifiable information (PII) from logs to comply with privacy regulations (GDPR, CCPA) while maintaining useful logging for debugging and monitoring.

## Solution

Implement a PII redactor that uses regex patterns and ML models to detect PII (emails, phone numbers, SSNs, credit cards), replaces them with redacted placeholders, and maintains a redaction audit trail. This works because PII follows predictable patterns that can be detected and replaced.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "regexp"
    "strings"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.safety.pii_redaction")

// PIIRedactor redacts PII from text
type PIIRedactor struct {
    patterns map[string]*regexp.Regexp
    enabled  bool
}

// PII patterns
var piiPatterns = map[string]string{
    "email":       `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
    "phone":       `\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`,
    "ssn":         `\b\d{3}-?\d{2}-?\d{4}\b`,
    "credit_card": `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`,
    "ip_address":  `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,
}

// NewPIIRedactor creates a new PII redactor
func NewPIIRedactor(enabled bool) *PIIRedactor {
    patterns := make(map[string]*regexp.Regexp)
    for piiType, pattern := range piiPatterns {
        patterns[piiType] = regexp.MustCompile(pattern)
    }

    return &PIIRedactor{
        patterns: patterns,
        enabled:  enabled,
    }
}

// Redact redacts PII from text
func (pr *PIIRedactor) Redact(ctx context.Context, text string) (string, map[string]int) {
    ctx, span := tracer.Start(ctx, "pii_redactor.redact")
    defer span.End()
    
    if !pr.enabled {
        span.SetAttributes(attribute.Bool("redaction_enabled", false))
        return text, nil
    }
    
    redacted := text
    counts := make(map[string]int)
    
    for piiType, pattern := range pr.patterns {
        matches := pattern.FindAllString(text, -1)
        count := len(matches)
        
        if count > 0 {
            counts[piiType] = count
            
            // Replace matches with redacted placeholder
            redacted = pattern.ReplaceAllStringFunc(redacted, func(match string) string {
                return pr.redactMatch(match, piiType)
            })
        }
    }
    
    span.SetAttributes(
        attribute.Int("total_redactions", pr.sumCounts(counts)),
        attribute.String("pii_types", pr.formatPIITypes(counts)),
    )
    
    if len(counts) > 0 {
        span.SetStatus(trace.StatusOK, fmt.Sprintf("redacted %d PII items", pr.sumCounts(counts)))
    } else {
        span.SetStatus(trace.StatusOK, "no PII detected")
    }
    
    return redacted, counts
}

// redactMatch creates a redacted placeholder for a match
func (pr *PIIRedactor) redactMatch(match string, piiType string) string {
    // Preserve structure for debugging (e.g., "***@***.com")
    switch piiType {
    case "email":
        parts := strings.Split(match, "@")
        if len(parts) == 2 {
            return fmt.Sprintf("[REDACTED_EMAIL]@%s", parts[1][:min(3, len(parts[1]))]+"...")
        }
    case "phone":
        return "[REDACTED_PHONE]"
    case "ssn":
        return "[REDACTED_SSN]"
    case "credit_card":
        // Show last 4 digits
        if len(match) >= 4 {
            return fmt.Sprintf("[REDACTED_CC]...%s", match[len(match)-4:])
        }
        return "[REDACTED_CC]"
    }
    
    return fmt.Sprintf("[REDACTED_%s]", strings.ToUpper(piiType))
}

// RedactStructured redacts PII from structured data
func (pr *PIIRedactor) RedactStructured(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
    ctx, span := tracer.Start(ctx, "pii_redactor.redact_structured")
    defer span.End()
    
    redacted := make(map[string]interface{})
    totalRedactions := 0
    
    for key, value := range data {
        switch v := value.(type) {
        case string:
            redactedValue, counts := pr.Redact(ctx, v)
            redacted[key] = redactedValue
            totalRedactions += pr.sumCounts(counts)
        case map[string]interface{}:
            subRedacted, err := pr.RedactStructured(ctx, v)
            if err != nil {
                return nil, err
            }
            redacted[key] = subRedacted
        default:
            redacted[key] = value
        }
    }
    
    span.SetAttributes(attribute.Int("total_redactions", totalRedactions))
    span.SetStatus(trace.StatusOK, "structured data redacted")
    
    return redacted, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func (pr *PIIRedactor) sumCounts(counts map[string]int) int {
    total := 0
    for _, count := range counts {
        total += count
    }
    return total
}

func (pr *PIIRedactor) formatPIITypes(counts map[string]int) string {
    types := []string{}
    for piiType, count := range counts {
        types = append(types, fmt.Sprintf("%s:%d", piiType, count))
    }
    return strings.Join(types, ",")
}

// SafeLogger wraps logging with PII redaction
type SafeLogger struct {
    redactor *PIIRedactor
    baseLogger *log.Logger
}

// NewSafeLogger creates a new safe logger
func NewSafeLogger(redactor *PIIRedactor) *SafeLogger {
    return &SafeLogger{
        redactor:   redactor,
        baseLogger: log.Default(),
    }
}

// Log logs with PII redaction
func (sl *SafeLogger) Log(ctx context.Context, message string) {
    redacted, counts := sl.redactor.Redact(ctx, message)

    if len(counts) > 0 {
        sl.baseLogger.Printf("[PII_REDACTED: %v] %s", counts, redacted)
    } else {
        sl.baseLogger.Print(redacted)
    }
}

func main() {
    ctx := context.Background()

    // Create redactor
    redactor := NewPIIRedactor(true)
    
    // Test redaction
    text := "Contact john.doe@example.com or call 555-123-4567"
    redacted, counts := redactor.Redact(ctx, text)
    fmt.Printf("Original: %s\n", text)
    fmt.Printf("Redacted: %s\n", redacted)
    fmt.Printf("Counts: %v\n", counts)
}
```

## Explanation

Let's break down what's happening:

1. **Pattern-based detection** - Notice how we use regex patterns for common PII types. Each pattern targets a specific type of PII (email, phone, SSN, etc.).

2. **Structure preservation** - We preserve some structure in redacted output (like domain for emails, last 4 digits for credit cards). This helps with debugging while protecting PII.

3. **Audit trail** - We count and log what types of PII were redacted. This helps monitor what's being logged and ensures compliance.

```go
**Key insight:** Balance between complete redaction and preserving useful information for debugging. Sometimes partial redaction (like showing last 4 digits) is acceptable and useful.

## Testing

```
Here's how to test this solution:
```go
func TestPIIRedactor_RedactsEmail(t *testing.T) {
    redactor := NewPIIRedactor(true)
    
    text := "Contact user@example.com"
    redacted, counts := redactor.Redact(context.Background(), text)
    
    require.Contains(t, counts, "email")
    require.Contains(t, redacted, "[REDACTED")
    require.NotContains(t, redacted, "user@example.com")
}

## Variations

### Custom PII Patterns

Add custom patterns for domain-specific PII:
func (pr *PIIRedactor) AddPattern(piiType string, pattern string) {
    pr.patterns[piiType] = regexp.MustCompile(pattern)
}
```

### ML-based Detection

Combine with ML models for better detection:
```go
type MLPIIRedactor struct {
    regexRedactor *PIIRedactor
    mlModel       *PIIModel
}
```
## Related Recipes

- **[Safety Mitigating Prompt Injection with Regex](./safety-mitigating-prompt-injection-regex.md)** - Additional safety patterns
- **[Config Masking Secrets in Logs](./config-masking-secrets-logs.md)** - Secret masking
- **[Safety Package Guide](../package_design_patterns.md)** - For a deeper understanding of safety
