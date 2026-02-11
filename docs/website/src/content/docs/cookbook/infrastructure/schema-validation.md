---
title: "Custom Validation Middleware"
description: "Add custom validation rules to schema operations with composable, stateless validation functions."
---

# Custom Validation Middleware

## Problem

You need to add custom validation rules to schema operations (messages, documents, agent I/O) that go beyond the built-in validation, such as business-specific constraints, content filtering, or domain-specific checks. Standard schema validation handles structural concerns—required fields, type checking, format validation—but cannot enforce application-specific rules. For example, you might need to validate that user messages don't exceed token limits for your chosen model, that tool call parameters match business logic constraints (like date ranges or enum values), that agent responses don't contain prohibited content for your compliance requirements, or that RAG documents meet quality thresholds before indexing. These domain-specific validations cannot be hardcoded into the framework because they vary by application. You need a way to plug in custom validation logic without modifying Beluga's core schema package.

## Solution

Create a validation middleware that wraps schema operations and applies custom validation rules. This works because Beluga AI's schema package uses the validator pattern and provides hooks for custom validation through `SchemaValidationConfig` with `CustomValidationRules`. The design follows the strategy pattern: validation logic is encapsulated in small, independent functions that each validate one concern. These functions are composed together through a registry, allowing you to mix and match validation rules per operation. The validator is stateless—it doesn't maintain mutable state between validations, making it thread-safe and testable. This approach integrates with Beluga's middleware pattern and OpenTelemetry tracing, providing observability into which validation rules pass or fail.

The key design choice is separating validation rules from validation orchestration. Rules are simple functions that return an error if validation fails. The orchestrator (CustomValidator) manages the rule registry and applies rules in sequence. This separation allows rules to be tested independently, reused across validators, and composed dynamically based on context. The strategy pattern makes validation extensible: adding new rules doesn't require modifying existing code, just registering new functions.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "strings"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.schema.validation")

// CustomValidator defines validation rules for schema objects
type CustomValidator struct {
    rules map[string]ValidationRule
}

// ValidationRule defines a single validation rule
type ValidationRule func(ctx context.Context, obj interface{}) error

// NewCustomValidator creates a new validator with custom rules
func NewCustomValidator() *CustomValidator {
    return &CustomValidator{
        rules: make(map[string]ValidationRule),
    }
}

// RegisterRule registers a validation rule by name
func (v *CustomValidator) RegisterRule(name string, rule ValidationRule) {
    v.rules[name] = rule
}

// ValidateMessage applies custom validation to a message
func (v *CustomValidator) ValidateMessage(ctx context.Context, msg schema.Message) error {
    ctx, span := tracer.Start(ctx, "validator.validate_message")
    defer span.End()

    span.SetAttributes(
        attribute.String("message.type", msg.GetType()),
        attribute.String("message.role", string(msg.GetRole())),
    )

    // Apply all registered rules
    for name, rule := range v.rules {
        if err := rule(ctx, msg); err != nil {
            span.RecordError(err)
            span.SetStatus(trace.StatusError, fmt.Sprintf("rule %s failed", name))
            return fmt.Errorf("validation rule %s failed: %w", name, err)
        }
    }

    span.SetStatus(trace.StatusOK, "validation passed")
    return nil
}

// ContentLengthRule validates content length
func ContentLengthRule(maxLength int) ValidationRule {
    return func(ctx context.Context, obj interface{}) error {
        msg, ok := obj.(schema.Message)
        if !ok {
            return nil // Skip if not a message
        }

        content := msg.GetContent()
        if len(content) > maxLength {
            return fmt.Errorf("content length %d exceeds maximum %d", len(content), maxLength)
        }
        return nil
    }
}

// ProhibitedWordsRule filters content for prohibited words
func ProhibitedWordsRule(prohibited []string) ValidationRule {
    return func(ctx context.Context, obj interface{}) error {
        msg, ok := obj.(schema.Message)
        if !ok {
            return nil
        }

        content := strings.ToLower(msg.GetContent())
        for _, word := range prohibited {
            if strings.Contains(content, strings.ToLower(word)) {
                return fmt.Errorf("content contains prohibited word: %s", word)
            }
        }
        return nil
    }
}

func main() {
    ctx := context.Background()

    // Create validator with custom rules
    validator := NewCustomValidator()
    validator.RegisterRule("content_length", ContentLengthRule(5000))
    validator.RegisterRule("prohibited_words", ProhibitedWordsRule([]string{"spam", "phishing"}))

    // Create a message
    msg := schema.NewHumanMessage("Hello, this is a test message")

    // Validate
    if err := validator.ValidateMessage(ctx, msg); err != nil {
        log.Fatalf("Validation failed: %v", err)
    }
    fmt.Println("Message validated successfully")
}
```

## Explanation

1. **CustomValidator structure** — Validation rules are separated from the validation logic, allowing multiple rules to be registered and applied in sequence. Each rule is independent and can be tested separately. This matters because validation requirements evolve over time: you might start with just length checks, then add content filtering, then add business logic validation. By separating rules from the orchestration logic, you can add new rules without modifying the validator itself. Each rule is self-contained—it either passes or returns an error—making them easy to test in isolation. The registry pattern (map of rules) allows dynamic composition: different validators can have different rule sets, and rules can be enabled or disabled based on configuration.

2. **ValidationRule function type** — A function type is used for rules, making it easy to create reusable validation functions. This follows the strategy pattern, allowing different validation strategies to be plugged in. This matters because validation logic is highly variable: some rules are simple (length checks), others are complex (calling external services for PII detection), and some are domain-specific (validating against business rules). By using a function type, you can create rules in multiple ways: inline lambdas for simple checks, factory functions that return configured validators (like ContentLengthRule(maxLength)), or methods on structs for stateful validators. The function signature includes context.Context, enabling rules to perform async operations, check deadlines, or access request-scoped values.

3. **OTel tracing integration** — The validator creates spans for each validation operation, recording which rules pass or fail. This is important because validation failures in production need to be traceable for debugging. This matters because when validation fails in production, you need to understand why. Was it a single rule that failed, or multiple? Which input triggered the failure? How often does this rule fail? OpenTelemetry tracing provides this observability: each validation creates a span, rule failures are recorded as errors, and attributes capture which rule failed and why. This makes it possible to monitor validation health, identify problematic rules, and debug validation issues without reproducing them locally.

Keep validation rules stateless and composable. Each rule should validate one concern, making it easy to combine rules and test them independently.

## Testing

```go
func TestCustomValidator(t *testing.T) {
    ctx := context.Background()
    validator := NewCustomValidator()
    validator.RegisterRule("content_length", ContentLengthRule(100))

    // Test valid message
    msg := schema.NewHumanMessage("Short message")
    if err := validator.ValidateMessage(ctx, msg); err != nil {
        t.Errorf("Expected validation to pass, got: %v", err)
    }

    // Test invalid message
    longMsg := schema.NewHumanMessage(strings.Repeat("a", 200))
    if err := validator.ValidateMessage(ctx, longMsg); err == nil {
        t.Error("Expected validation to fail for long message")
    }
}
```

## Variations

### Integration with SchemaValidationConfig

Integrate custom validators with Beluga AI's `SchemaValidationConfig`:

```go
config, _ := schema.NewSchemaValidationConfig(
    schema.WithCustomValidationRules(map[string]any{
        "content_length":   5000,
        "prohibited_words": []string{"spam"},
    }),
)
```

### Async Validation

For expensive validation rules, use goroutines:

```go
func (v *CustomValidator) ValidateMessageAsync(ctx context.Context, msg schema.Message) <-chan error {
    errCh := make(chan error, 1)
    go func() {
        errCh <- v.ValidateMessage(ctx, msg)
    }()
    return errCh
}
```

## Related Recipes

- [Schema Recursive Schema Handling](/cookbook/schema-recursive-schema-handling) — Handle nested/recursive schema structures
- [LLM Error Handling](/cookbook/llm-error-handling) — Error handling patterns that work with validation
