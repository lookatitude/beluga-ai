---
title: "Custom Validation Middleware"
description: "Recipe for adding custom validation rules to Go schema operations with composable, stateless middleware — enforce business logic, content filters, and limits."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, validation middleware, Go schema validation, custom rules, content filtering, business logic, composable validation"
---

## Problem

You need to add custom validation rules to schema operations (messages, documents, agent I/O) that go beyond structural concerns — such as business-specific constraints, content filtering, or domain-specific checks.

Standard schema validation handles type checking and required fields, but cannot enforce application-specific rules such as token length limits for a chosen model, prohibited content for compliance requirements, or quality thresholds before RAG indexing.

## Solution

Create a `CustomValidator` that holds a registry of named `ValidationRule` functions. Rules accept a `schema.Message` and return an error if validation fails. The orchestrator applies all rules in sequence, providing span-level observability for each pass or fail.

## Why This Matters

Validation requirements evolve over time. By separating rules from orchestration logic, you can add new rules without modifying the validator itself. Each rule is self-contained and testable in isolation. Composing rules at construction time means different validator instances can have different rule sets for different contexts (user-facing vs. internal agents).

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.schema.validation")

// ValidationRule validates a single schema.Message.
// Return an error if validation fails; nil if it passes.
type ValidationRule func(ctx context.Context, msg schema.Message) error

// CustomValidator applies a registry of named validation rules to messages.
type CustomValidator struct {
	rules map[string]ValidationRule
}

// NewCustomValidator creates an empty validator. Register rules with RegisterRule.
func NewCustomValidator() *CustomValidator {
	return &CustomValidator{rules: make(map[string]ValidationRule)}
}

// RegisterRule adds a named rule to the validator.
func (v *CustomValidator) RegisterRule(name string, rule ValidationRule) {
	v.rules[name] = rule
}

// ValidateMessage applies all registered rules to msg.
// Returns on the first failure. All rule failures are recorded as OTel events.
func (v *CustomValidator) ValidateMessage(ctx context.Context, msg schema.Message) error {
	ctx, span := tracer.Start(ctx, "validator.validate_message")
	defer span.End()

	span.SetAttributes(
		attribute.String("message.role", string(msg.GetRole())),
	)

	for name, rule := range v.rules {
		if err := rule(ctx, msg); err != nil {
			span.RecordError(err)
			span.SetStatus(trace.StatusError, fmt.Sprintf("rule %s failed", name))
			return fmt.Errorf("validation rule %q failed: %w", name, err)
		}
	}

	span.SetStatus(trace.StatusOK, "validation passed")
	return nil
}

// --- Built-in rules ---

// ContentLengthRule rejects messages whose text content exceeds maxBytes bytes.
func ContentLengthRule(maxBytes int) ValidationRule {
	return func(ctx context.Context, msg schema.Message) error {
		// schema.Message.GetContent() returns []schema.ContentPart.
		// Use the Text() helper on concrete types, or accumulate text parts manually.
		total := 0
		for _, part := range msg.GetContent() {
			// TextPart is the only content part with a string representation.
			if tp, ok := part.(schema.TextPart); ok {
				total += len(tp.Text)
			}
		}
		if total > maxBytes {
			return fmt.Errorf("content length %d exceeds maximum %d bytes", total, maxBytes)
		}
		return nil
	}
}

// ProhibitedWordsRule rejects messages that contain any of the listed words.
func ProhibitedWordsRule(prohibited []string) ValidationRule {
	lower := make([]string, len(prohibited))
	for i, w := range prohibited {
		lower[i] = strings.ToLower(w)
	}
	return func(ctx context.Context, msg schema.Message) error {
		for _, part := range msg.GetContent() {
			if tp, ok := part.(schema.TextPart); ok {
				text := strings.ToLower(tp.Text)
				for _, word := range lower {
					if strings.Contains(text, word) {
						return fmt.Errorf("content contains prohibited word: %q", word)
					}
				}
			}
		}
		return nil
	}
}

// RoleAllowlistRule rejects messages whose role is not in the allowlist.
func RoleAllowlistRule(allowed ...schema.Role) ValidationRule {
	set := make(map[schema.Role]bool, len(allowed))
	for _, r := range allowed {
		set[r] = true
	}
	return func(ctx context.Context, msg schema.Message) error {
		if !set[msg.GetRole()] {
			return fmt.Errorf("role %q is not allowed", msg.GetRole())
		}
		return nil
	}
}

func main() {
	ctx := context.Background()

	validator := NewCustomValidator()
	validator.RegisterRule("content_length", ContentLengthRule(5000))
	validator.RegisterRule("prohibited_words", ProhibitedWordsRule([]string{"spam", "phishing"}))
	validator.RegisterRule("role_allowlist", RoleAllowlistRule(schema.RoleHuman, schema.RoleAI))

	msg := schema.NewHumanMessage("Hello, this is a test message")

	if err := validator.ValidateMessage(ctx, msg); err != nil {
		slog.Error("validation failed", "error", err)
		return
	}
	fmt.Println("Message validated successfully")
}
```

## Explanation

1. **`ValidationRule` function type** — A function type rather than an interface keeps rules lightweight. Factory functions like `ContentLengthRule(maxBytes)` produce configured closures without requiring struct definitions for each rule.

2. **`GetContent() []ContentPart`** — `schema.Message.GetContent()` returns a slice of `ContentPart` values, not a string. Text content is extracted by type-asserting each part to `schema.TextPart`. This handles multimodal messages (images, audio) without discarding non-text parts.

3. **`GetRole() schema.Role`** — Returns `schema.Role` (a typed string alias). Compare against `schema.RoleHuman`, `schema.RoleAI`, `schema.RoleSystem`, `schema.RoleTool` — not raw strings — for type safety.

4. **OTel tracing** — Each `ValidateMessage` call creates a span. Rule failures are recorded as span errors, making it possible to monitor validation health and identify problematic rules in production without reproducing failures locally.

## Testing

```go
import (
	"context"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestCustomValidator_PassesValidMessage(t *testing.T) {
	validator := NewCustomValidator()
	validator.RegisterRule("content_length", ContentLengthRule(100))

	if err := validator.ValidateMessage(context.Background(), schema.NewHumanMessage("Short message")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCustomValidator_RejectsLongMessage(t *testing.T) {
	validator := NewCustomValidator()
	validator.RegisterRule("content_length", ContentLengthRule(100))

	long := schema.NewHumanMessage(strings.Repeat("a", 200))
	if err := validator.ValidateMessage(context.Background(), long); err == nil {
		t.Error("expected validation to fail for long message")
	}
}

func TestCustomValidator_RejectsProhibitedWord(t *testing.T) {
	validator := NewCustomValidator()
	validator.RegisterRule("prohibited_words", ProhibitedWordsRule([]string{"spam"}))

	msg := schema.NewHumanMessage("This is spam")
	if err := validator.ValidateMessage(context.Background(), msg); err == nil {
		t.Error("expected prohibited word rejection")
	}
}
```

## Variations

### Async Validation

For expensive rules (external PII detection, ML classifiers), run rules concurrently:

```go
func (v *CustomValidator) ValidateMessageParallel(ctx context.Context, msg schema.Message) error {
	errCh := make(chan error, len(v.rules))
	for name, rule := range v.rules {
		go func(n string, r ValidationRule) {
			errCh <- r(ctx, msg)
		}(name, rule)
	}
	for range v.rules {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}
```

## Related Recipes

- **[Recursive Schema Handling](./recursive-schemas)** — Handle nested/recursive schema structures
- **[LLM Error Handling](/docs/recipes/llm/llm-error-handling)** — Error handling patterns that complement validation
