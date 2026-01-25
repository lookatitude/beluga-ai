# {Title}

<!--
Template Guidelines:
- Write in a teacher-like, conversational tone
- Use "you" and "we" to create a collaborative feel
- Explain the "why" behind decisions, not just the "what"
- Include practical examples at every step
- Anticipate questions and address them proactively
-->

## Introduction

<!--
Start with a welcoming explanation of what this guide teaches and why it matters.
Use concrete examples to illustrate value.
Example: "In this guide, you'll learn how to..." rather than "This guide covers..."
-->

Welcome to this guide on {feature/topic}. By the end, you'll understand how to...

**What you'll learn:**
- {Learning objective 1}
- {Learning objective 2}
- {Learning objective 3}

**Why this matters:**
{Explain the business value or technical benefit of mastering this feature}

## Prerequisites

<!--
Be specific about requirements.
Explain why each prerequisite matters.
Include version numbers where relevant.
-->

Before we begin, make sure you have:

- **Go 1.24+** installed ([installation guide](https://go.dev/doc/install))
- **Beluga AI Framework** installed (`go get github.com/lookatitude/beluga-ai`)
- **{API key or other requirement}** - you'll need this because {explanation}
- **Understanding of {concept}** - if you're new to this, check out our [concept guide](../concepts/{topic}.md)

## Concepts

<!--
Break down key concepts before diving into implementation.
Use analogies where helpful.
Include diagrams (ASCII or Mermaid) for complex concepts.
-->

Before we start coding, let's understand the key concepts:

### {Concept 1}

{Clear explanation with an analogy if helpful}

### {Concept 2}

{Clear explanation}

```
┌─────────────────┐       ┌─────────────────┐
│   Component A   │──────▶│   Component B   │
└─────────────────┘       └─────────────────┘
         │
         ▼
┌─────────────────┐
│   Component C   │
└─────────────────┘
```

## Step-by-Step Tutorial

<!--
Number each step clearly.
Start each step with a clear action verb.
Include "What you'll see" subsections showing expected output.
Include "Why this works" explanations for complex steps.
-->

Now let's build this step by step.

### Step 1: {Action verb + description}

First, we'll {explanation of what we're doing and why}.

```go
package main

import (
    // We import these packages because...
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/{package}"
)

func main() {
    // We create a context for proper cancellation handling
    ctx := context.Background()
    
    // {Explanation of what this code does}
    // ...
}
```

**What you'll see:**

```
{Expected output}
```

**Why this works:** {Brief explanation of the mechanics}

### Step 2: {Action verb + description}

{Continue pattern for each step...}

## Code Examples

<!--
Show complete, production-ready code.
Include inline comments explaining each section.
Demonstrate proper error handling, OTEL instrumentation, and DI patterns.
-->

Here's a complete, production-ready example combining everything we've learned:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/{package}"
)

// We define a tracer for observability - this helps you debug in production
var tracer = otel.Tracer("beluga.{package}.example")

func main() {
    ctx := context.Background()
    
    // Start a span for tracing - you'll see this in your observability dashboard
    ctx, span := tracer.Start(ctx, "example.main")
    defer span.End()
    
    // Initialize the component with proper configuration
    component, err := {package}.New{Component}(
        {package}.WithOption("value"),
    )
    if err != nil {
        // Always wrap errors with context for better debugging
        span.RecordError(err)
        log.Fatalf("failed to create component: %v", err)
    }
    
    // Use the component
    result, err := component.Execute(ctx, input)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("execution failed: %v", err)
    }
    
    // Add attributes for debugging
    span.SetAttributes(
        attribute.String("result.status", "success"),
    )
    
    fmt.Printf("Result: %v\n", result)
}
```

## Testing

<!--
Explain how to test the feature.
Include example tests using framework patterns.
Show how to interpret results.
-->

Testing is crucial for production code. Here's how to test what we've built:

### Unit Tests

```go
func TestComponent_Execute(t *testing.T) {
    // Table-driven tests for comprehensive coverage
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "successful execution",
            input: "test input",
            want:  "expected output",
        },
        {
            name:    "handles empty input",
            input:   "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            component := // ... setup with mock
            
            got, err := component.Execute(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Execute() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v -run TestComponent_Execute ./...
```

**What to look for:**
- All tests should pass
- Coverage should be above 80%
- No race conditions (run with `-race` flag)

## Best Practices

<!--
Share lessons learned and common pitfalls.
Use a mentor-like tone: "In production, you'll want to..."
-->

After using this feature in production, here are the patterns that work best:

### Do

- **Always handle errors explicitly** - Beluga AI uses errors for control flow, so check every error
- **Use context for cancellation** - This prevents resource leaks and improves responsiveness
- **Add OTEL instrumentation** - You'll thank yourself when debugging production issues
- **Use functional options for configuration** - This makes your code testable and flexible

### Don't

- **Don't ignore context cancellation** - Always check `ctx.Done()` in long-running operations
- **Don't skip error wrapping** - Use `fmt.Errorf("operation failed: %w", err)` for context
- **Don't hardcode configuration** - Use environment variables or config files

### Performance Tips

- {Specific performance recommendation}
- {Another recommendation}

## Troubleshooting

<!--
Anticipate common issues.
Format as Q&A: "Q: I see error X. A: This usually means..."
Be empathetic and solution-focused.
-->

### Q: I see error "{common error message}"

**A:** This usually means {explanation}. Here's how to fix it:

1. Check that {verification step}
2. Ensure {another check}
3. Try {solution}

### Q: {Another common issue}

**A:** {Solution with empathy}

### Q: How do I debug {specific scenario}?

**A:** Enable debug logging by setting `BELUGA_LOG_LEVEL=debug`. You'll see:

```
DEBUG: {example log output}
```

This tells you {interpretation}.

## Related Resources

<!--
Link to related guides, examples, cookbooks, and use cases.
Include brief descriptions of why each is relevant.
-->

Now that you understand {topic}, you might want to explore:

- **[{Related Guide}](../guides/{guide}.md)** - Learn how to {brief description}
- **[{Example}](https://github.com/lookatitude/beluga-ai/blob/main/examples/{example}/README.md)** - See a complete implementation of {description}
- **[{Cookbook Recipe}](../cookbook/{recipe}.md)** - Quick solution for {specific task}
- **[{Use Case}](../use-cases/{use-case}.md)** - Real-world scenario showing {description}
