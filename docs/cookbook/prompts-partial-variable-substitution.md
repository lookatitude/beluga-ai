---
title: "Partial Variable Substitution"
package: "prompts"
category: "templating"
complexity: "intermediate"
---

# Partial Variable Substitution

## Problem

You need to substitute variables in a prompt template incrementally as data becomes available, rather than requiring all variables at once. This is useful for streaming scenarios or when building prompts dynamically.

## Solution

Implement a partial substitution system that allows replacing variables one at a time, tracks which variables have been substituted, and can complete the substitution when all variables are available. This works because prompt templates use placeholders that can be replaced independently.

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
    
    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

var tracer = otel.Tracer("beluga.prompts.partial_substitution")

// PartialPromptTemplate supports incremental variable substitution
type PartialPromptTemplate struct {
    template      string
    substituted   map[string]string
    remainingVars []string
    pattern       *regexp.Regexp
}

// NewPartialPromptTemplate creates a new partial template
func NewPartialPromptTemplate(template string) *PartialPromptTemplate {
    // Extract variable names
    varPattern := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
    matches := varPattern.FindAllStringSubmatch(template, -1)

    remainingVars := []string{}
    seen := make(map[string]bool)
    for _, match := range matches {
        varName := match[1]
        if !seen[varName] {
            remainingVars = append(remainingVars, varName)
            seen[varName] = true
        }
    }
    
    return &PartialPromptTemplate{
        template:      template,
        substituted:   make(map[string]string),
        remainingVars: remainingVars,
        pattern:       varPattern,
    }
}

// Substitute replaces a variable with a value
func (ppt *PartialPromptTemplate) Substitute(ctx context.Context, varName string, value string) error {
    ctx, span := tracer.Start(ctx, "partial_template.substitute")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("variable", varName),
        attribute.Int("value_length", len(value)),
    )
    
    // Check if variable exists
    if !ppt.hasVariable(varName) {
        err := fmt.Errorf("variable %s not found in template", varName)
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return err
    }
    
    // Store substitution
    ppt.substituted[varName] = value
    
    // Remove from remaining
    for i, v := range ppt.remainingVars {
        if v == varName {
            ppt.remainingVars = append(ppt.remainingVars[:i], ppt.remainingVars[i+1:]...)
            break
        }
    }
    
    span.SetStatus(trace.StatusOK, "variable substituted")
    return nil
}

// GetCurrentPrompt returns the prompt with current substitutions
func (ppt *PartialPromptTemplate) GetCurrentPrompt(ctx context.Context) string {
    ctx, span := tracer.Start(ctx, "partial_template.get_current")
    defer span.End()
    
    result := ppt.template
    
    // Replace all substituted variables
    for varName, value := range ppt.substituted {
        placeholder := fmt.Sprintf("{{.%s}}", varName)
        result = strings.ReplaceAll(result, placeholder, value)
    }
    
    span.SetAttributes(
        attribute.Int("substituted_count", len(ppt.substituted)),
        attribute.Int("remaining_count", len(ppt.remainingVars)),
    )
    
    return result
}

// IsComplete checks if all variables have been substituted
func (ppt *PartialPromptTemplate) IsComplete() bool {
    return len(ppt.remainingVars) == 0
}

// GetRemainingVariables returns variables that still need substitution
func (ppt *PartialPromptTemplate) GetRemainingVariables() []string {
    return ppt.remainingVars
}

// Complete substitutes all remaining variables with defaults or errors
func (ppt *PartialPromptTemplate) Complete(ctx context.Context, defaults map[string]string) (string, error) {
    ctx, span := tracer.Start(ctx, "partial_template.complete")
    defer span.End()
    
    // Substitute remaining variables
    for _, varName := range ppt.remainingVars {
        value, exists := defaults[varName]
        if !exists {
            err := fmt.Errorf("no value or default for variable %s", varName)
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return "", err
        }
        
        if err := ppt.Substitute(ctx, varName, value); err != nil {
            return "", err
        }
    }
    
    result := ppt.GetCurrentPrompt(ctx)
    span.SetStatus(trace.StatusOK, "template completed")
    
    return result, nil
}

// hasVariable checks if a variable exists in the template
func (ppt *PartialPromptTemplate) hasVariable(varName string) bool {
    placeholder := fmt.Sprintf("{{.%s}}", varName)
    return strings.Contains(ppt.template, placeholder)
}

func main() {
    ctx := context.Background()

    // Create template
    template := "Hello {{.name}}, your order {{.orderId}} is ready. Status: {{.status}}"
    ppt := NewPartialPromptTemplate(template)
    
    // Substitute incrementally
    ppt.Substitute(ctx, "name", "Alice")
    fmt.Println(ppt.GetCurrentPrompt(ctx)) // "Hello Alice, your order {{.orderId}} is ready. Status: {{.status}}"
    
    ppt.Substitute(ctx, "orderId", "12345")
    fmt.Println(ppt.GetCurrentPrompt(ctx)) // "Hello Alice, your order 12345 is ready. Status: {{.status}}"
    
    // Complete with defaults
    defaults := map[string]string{"status": "pending"}
    final, _ := ppt.Complete(ctx, defaults)
    fmt.Println(final) // "Hello Alice, your order 12345 is ready. Status: pending"
}
```

## Explanation

Let's break down what's happening:

1. **Incremental substitution** - Notice how we can substitute variables one at a time. Each substitution updates the template state, allowing you to build prompts incrementally as data arrives.

2. **State tracking** - We track which variables have been substituted and which remain. This allows you to check completion status and see what's still needed.

3. **Completion with defaults** - When all variables aren't available, you can complete the template with default values. This is useful for fallback scenarios.

```go
**Key insight:** Partial substitution enables streaming and dynamic prompt building. You can start showing a prompt to users even before all data is available.

## Testing

```
Here's how to test this solution:
```go
func TestPartialPromptTemplate_SubstitutesIncrementally(t *testing.T) {
    template := "Hello {{.name}}, status: {{.status}}"
    ppt := NewPartialPromptTemplate(template)
    
    ppt.Substitute(context.Background(), "name", "Alice")
    
    result := ppt.GetCurrentPrompt(context.Background())
    require.Contains(t, result, "Alice")
    require.Contains(t, result, "{{.status}}")
    require.False(t, ppt.IsComplete())
}

## Variations

### Validation on Substitution

Validate values when substituting:
func (ppt *PartialPromptTemplate) SubstituteWithValidation(ctx context.Context, varName string, value string, validator func(string) error) error {
    // Validate before substituting
}
```

### Template Merging

Merge multiple partial templates:
```go
func (ppt *PartialPromptTemplate) Merge(other *PartialPromptTemplate) *PartialPromptTemplate {
    // Combine templates
}
```

## Related Recipes

- **[Prompts Dynamic Message Chain Templates](./prompts-dynamic-message-chain-templates.md)** - Build message chains dynamically
- **[LLMs Streaming Tool Logic Handler](./llms-streaming-tool-logic-handler.md)** - Streaming patterns
- **[Prompts Package Guide](../package_design_patterns.md)** - For a deeper understanding of prompts
