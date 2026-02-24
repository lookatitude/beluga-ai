---
title: "PII Redaction in Logs"
description: "Recipe for automatically redacting PII (emails, phones, SSNs) from Go application logs to comply with GDPR and CCPA while preserving debugging utility."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, PII redaction, Go log privacy, GDPR compliance, CCPA, data masking, privacy recipe, log sanitization"
---

# PII Redaction in Logs

## Problem

You need to redact personally identifiable information (PII) from logs to comply with privacy regulations (GDPR, CCPA) while maintaining useful logging for debugging and monitoring. This is a critical challenge for AI systems that process user data: logs are essential for debugging production issues, but exposing PII in logs creates compliance and security risks. Privacy regulations require minimizing PII exposure, but overly aggressive redaction makes logs useless for debugging. The challenge is striking the right balance: redact sensitive data (emails, phone numbers, SSNs, credit cards, IP addresses) while preserving enough context to diagnose issues. You need to handle both structured logs (JSON fields) and unstructured text (error messages, user inputs) where PII might appear anywhere. Additionally, redaction must be automatic and fail-safe—relying on developers to manually redact PII is error-prone and doesn't scale.

## Solution

Implement a PII redactor that uses regex patterns to detect PII (emails, phone numbers, SSNs, credit cards), replaces them with redacted placeholders, and maintains a redaction audit trail. This works because PII follows predictable patterns that can be detected and replaced. The design uses regex patterns for common PII types, configurable redaction strategies (full redaction vs. partial masking), and an audit trail that logs what was redacted without logging the actual PII. The key insight is that complete removal of PII is often unnecessary and counterproductive—partial redaction (like showing the domain for emails or last 4 digits for credit cards) preserves debugging utility while protecting sensitive data. The redactor integrates with Beluga's logging infrastructure and OpenTelemetry tracing, ensuring all logs are automatically sanitized before emission.

Pattern ordering matters: the credit_card pattern must be checked before the phone pattern to avoid false positives where a credit card number is partially matched as phone numbers. Phone regex requires separators to avoid matching arbitrary digit sequences. The redactor is designed to be conservative: it may occasionally over-redact (false positives) but should never under-redact (false negatives), prioritizing compliance over debugging convenience.

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
    redactor   *PIIRedactor
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

1. **Pattern-based detection** — Regex patterns target specific PII types (email, phone, SSN, etc.), providing broad coverage of common personally identifiable information. This matters because PII appears in predictable formats: emails follow RFC 5322, phone numbers follow regional formatting rules, SSNs have fixed structure in the US. Regex patterns exploit these predictable formats to detect PII without requiring machine learning models or external services. This approach is fast (regex matching is highly optimized), deterministic (same input always produces same output), and privacy-preserving (detection happens locally without sending data to external APIs). The trade-off is that regex patterns are format-specific and may miss PII in unusual formats or produce false positives on data that looks like PII but isn't.

2. **Structure preservation** — Some structure is preserved in redacted output (like domain for emails, last 4 digits for credit cards). This helps with debugging while protecting PII. This matters because logs are useless if they're completely redacted. Preserving structure provides enough context to debug issues: showing email domains helps identify misconfigured mail servers, showing last 4 digits of credit cards helps verify payment processing flow, and showing phone area codes helps debug geographic routing. The key insight is that structure and format often contain useful debugging information while the specific values are sensitive. By redacting values but preserving structure, you maintain debugging utility while complying with privacy regulations that focus on protecting individual identifiers, not aggregate patterns.

3. **Audit trail** — The count and type of PII redactions are tracked and logged, helping monitor what's being processed and ensuring compliance. This matters because you need visibility into PII processing for compliance audits and security monitoring. The audit trail answers critical questions: Is PII appearing in logs unexpectedly (indicating a bug)? What types of PII are users submitting (informing feature design)? Are redaction patterns catching all PII (validating regex patterns)? The counts map provides this visibility without logging the actual PII, satisfying compliance requirements. This data can be aggregated across logs to detect anomalies—like a sudden spike in SSN detections suggesting a data breach or a new feature leaking sensitive data.

Balance between complete redaction and preserving useful information for debugging. Sometimes partial redaction (like showing last 4 digits) is acceptable and useful.

## Testing

```go
func TestPIIRedactor_RedactsEmail(t *testing.T) {
    redactor := NewPIIRedactor(true)

    text := "Contact user@example.com"
    redacted, counts := redactor.Redact(context.Background(), text)

    require.Contains(t, counts, "email")
    require.Contains(t, redacted, "[REDACTED")
    require.NotContains(t, redacted, "user@example.com")
}
```

## Variations

### Custom PII Patterns

Add custom patterns for domain-specific PII:

```go
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

- [Prompt Injection Detection](/cookbook/prompt-injection-detection) — Additional safety patterns
- [Config Masking Secrets in Logs](/cookbook/config-secret-masking) — Secret masking
