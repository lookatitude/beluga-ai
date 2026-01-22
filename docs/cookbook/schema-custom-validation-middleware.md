---
title: "Custom Validation Middleware"
package: "schema"
category: "validation"
complexity: "intermediate"
---

# Custom Validation Middleware

## Problem

You need to add custom validation rules to schema operations (messages, documents, agent I/O) that go beyond the built-in validation, such as business-specific constraints, content filtering, or domain-specific checks.

## Solution

Create a validation middleware that wraps schema operations and applies custom validation rules. This works because Beluga AI's schema package uses the validator pattern and provides hooks for custom validation through `SchemaValidationConfig` with `CustomValidationRules`.

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
    
    "github.com/lookatitude/beluga-ai/pkg/schema"
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

// Example: Content length validation rule
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

// Example: Prohibited words filter
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

Let's break down what's happening:

1. **CustomValidator structure** - Notice how we separate validation rules from the validation logic. This allows you to register multiple rules and apply them in sequence. Each rule is independent and can be tested separately.

2. **ValidationRule function type** - We use a function type for rules, which makes it easy to create reusable validation functions. This follows the strategy pattern, allowing different validation strategies to be plugged in.

3. **OTEL tracing integration** - The validator creates spans for each validation operation, recording which rules pass or fail. This is important because validation failures in production need to be traceable for debugging.

```go
**Key insight:** Keep validation rules stateless and composable. Each rule should validate one concern, making it easy to combine rules and test them independently.

## Testing

```
Here's how to test this solution:
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

## Variations

### Integration with SchemaValidationConfig

You can integrate custom validators with Beluga AI's `SchemaValidationConfig`:
config, _ := schema.NewSchemaValidationConfig(
    schema.WithCustomValidationRules(map[string]any{
        "content_length": 5000,
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
        errCh \<- v.ValidateMessage(ctx, msg)
    }()
text
    return errCh
}
```

## Related Recipes

- **[Schema Recursive Schema Handling](./schema-recursive-schema-handling.md)** - Handle nested/recursive schema structures
- **[LLM Error Handling](./llm-error-handling.md)** - Error handling patterns that work with validation
- **[Schema Package Guide](../package_design_patterns.md)** - For a deeper understanding of schema validation
