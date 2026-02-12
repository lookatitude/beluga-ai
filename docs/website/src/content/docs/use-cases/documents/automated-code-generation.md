---
title: AI Code Generation Pipeline
description: "Generate production-ready code from natural language with LLM-powered pattern enforcement, validation, and test generation."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AI code generation, automated code generation, LLM code, code from natural language, pattern enforcement, Beluga AI, Go"
---

Development teams spend 15-20% of their time writing boilerplate — repository implementations, factory methods, API handlers, data transfer objects. This code follows known patterns but still requires manual effort, and inconsistencies creep in as different developers interpret patterns differently. The result is a codebase where the same pattern is implemented three different ways across three teams, making maintenance harder and onboarding slower.

An automated code generation pipeline addresses this by encoding patterns as prompt templates and using LLMs to generate implementations that conform to those patterns. Unlike simple code snippets or IDE templates, LLM-based generation adapts the pattern to specific requirements — a repository for users looks different from one for orders, even though both follow the same pattern.

## Solution Architecture

Beluga AI provides LLM integration with structured output, the `prompt/` package for template management, and `iter.Seq2` streaming for real-time code generation feedback. The system selects the appropriate pattern, populates the prompt template with request-specific context, generates code using the LLM, validates the output with language-specific tooling, and generates accompanying tests.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Code      │───▶│   Pattern    │───▶│     LLM      │
│   Request    │    │   Selector   │    │  Generator   │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Formatted   │◀───│     Code     │◀───│  Generated   │
│    Code +    │    │  Validator   │    │     Code     │
│    Tests     │    │  + Formatter │    └──────────────┘
└──────────────┘    └──────────────┘
```

## Implementation

### Code Generator Setup

The generator uses prompt templates to enforce patterns and coding standards:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/prompt"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

type CodeGenerator struct {
    model    llm.ChatModel
    template *prompt.Template
    patterns map[string]CodePattern
}

type CodePattern struct {
    Name     string
    Template string
    Rules    []string
    Examples []string
}

func NewCodeGenerator(ctx context.Context) (*CodeGenerator, error) {
    model, err := llm.New("openai", llm.ProviderConfig{
        Model: "gpt-4o",
    })
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    template, err := prompt.NewTemplate(`Generate {{.language}} code for: {{.description}}

Pattern: {{.pattern}}

Rules:
{{range .rules}}- {{.}}
{{end}}

Generate production-ready code with:
- Proper error handling
- Comments and documentation
- Following {{.pattern}} pattern
- Type safety and validation
`)
    if err != nil {
        return nil, fmt.Errorf("create template: %w", err)
    }

    return &CodeGenerator{
        model:    model,
        template: template,
        patterns: loadPatterns(),
    }, nil
}

func loadPatterns() map[string]CodePattern {
    return map[string]CodePattern{
        "factory": {
            Name: "factory",
            Rules: []string{
                "Use functional options pattern",
                "Return interface, not concrete type",
                "Validate configuration",
                "Handle errors explicitly",
            },
        },
        "repository": {
            Name: "repository",
            Rules: []string{
                "Use context.Context as first parameter",
                "Return typed errors",
                "Implement CRUD operations",
                "Use transactions where appropriate",
            },
        },
    }
}
```

### Code Generation

Generate code from natural language descriptions:

```go
type CodeRequest struct {
    Description string
    Language    string
    Pattern     string
    Context     map[string]string
}

func (g *CodeGenerator) GenerateCode(ctx context.Context, req CodeRequest) (string, error) {
    pattern, exists := g.patterns[req.Pattern]
    if !exists {
        return "", fmt.Errorf("unknown pattern: %s", req.Pattern)
    }

    // Format prompt from template
    promptText, err := g.template.Format(map[string]interface{}{
        "description": req.Description,
        "language":    req.Language,
        "pattern":     req.Pattern,
        "rules":       pattern.Rules,
    })
    if err != nil {
        return "", fmt.Errorf("format prompt: %w", err)
    }

    // Generate code using LLM
    messages := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "You are an expert code generator. Generate production-ready, well-documented code."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: promptText},
        }},
    }

    resp, err := g.model.Generate(ctx, messages)
    if err != nil {
        return "", fmt.Errorf("generate code: %w", err)
    }

    code := resp.Parts[0].(schema.TextPart).Text

    // Validate generated code
    if err := g.validateCode(ctx, code, req.Language); err != nil {
        return "", fmt.Errorf("validation failed: %w", err)
    }

    return code, nil
}

func (g *CodeGenerator) validateCode(ctx context.Context, code, language string) error {
    // Run language-specific linting and validation
    // Implementation depends on language
    return nil
}
```

### Test Generation

Automatically generate tests for generated code:

```go
type GeneratedCode struct {
    Code     string
    Tests    string
    Language string
}

func (g *CodeGenerator) GenerateCodeWithTests(ctx context.Context, req CodeRequest) (*GeneratedCode, error) {
    code, err := g.GenerateCode(ctx, req)
    if err != nil {
        return nil, err
    }

    // Generate tests
    tests, err := g.generateTests(ctx, code, req)
    if err != nil {
        return nil, fmt.Errorf("generate tests: %w", err)
    }

    // Format code
    formattedCode, err := g.formatCode(ctx, code, req.Language)
    if err != nil {
        return nil, fmt.Errorf("format code: %w", err)
    }

    return &GeneratedCode{
        Code:     formattedCode,
        Tests:    tests,
        Language: req.Language,
    }, nil
}

func (g *CodeGenerator) generateTests(ctx context.Context, code string, req CodeRequest) (string, error) {
    testPrompt := fmt.Sprintf(`Generate comprehensive unit tests for this %s code:

%s

Include:
- Happy path tests
- Error case tests
- Edge case tests
- Table-driven tests where appropriate
`, req.Language, code)

    messages := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "You are an expert at writing comprehensive, maintainable tests."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: testPrompt},
        }},
    }

    resp, err := g.model.Generate(ctx, messages)
    if err != nil {
        return "", fmt.Errorf("generate tests: %w", err)
    }

    return resp.Parts[0].(schema.TextPart).Text, nil
}

func (g *CodeGenerator) formatCode(ctx context.Context, code, language string) (string, error) {
    // Run language-specific formatter
    // Implementation depends on language (gofmt, prettier, etc.)
    return code, nil
}
```

## Production Considerations

### Streaming Generation

For interactive use cases (IDE integrations, developer portals), streaming provides immediate feedback as code is generated. The generator uses Beluga AI's `iter.Seq2[string, error]` pattern, yielding chunks as the LLM produces them rather than waiting for the full response:

```go
import "iter"

func (g *CodeGenerator) StreamCode(ctx context.Context, req CodeRequest) iter.Seq2[string, error] {
    return func(yield func(string, error) bool) {
        pattern, exists := g.patterns[req.Pattern]
        if !exists {
            yield("", fmt.Errorf("unknown pattern: %s", req.Pattern))
            return
        }

        promptText, err := g.template.Format(map[string]interface{}{
            "description": req.Description,
            "language":    req.Language,
            "pattern":     req.Pattern,
            "rules":       pattern.Rules,
        })
        if err != nil {
            yield("", fmt.Errorf("format prompt: %w", err))
            return
        }

        messages := []schema.Message{
            &schema.SystemMessage{Parts: []schema.ContentPart{
                schema.TextPart{Text: "You are an expert code generator."},
            }},
            &schema.HumanMessage{Parts: []schema.ContentPart{
                schema.TextPart{Text: promptText},
            }},
        }

        for chunk, err := range g.model.Stream(ctx, messages) {
            if err != nil {
                yield("", err)
                return
            }
            if !yield(chunk.Text(), nil) {
                return
            }
        }
    }
}
```

### Observability

Track generation quality and performance:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (g *CodeGenerator) GenerateCodeWithTracing(ctx context.Context, req CodeRequest) (string, error) {
    ctx, span := o11y.StartSpan(ctx, "code_generation.generate")
    defer span.End()

    span.SetAttributes(
        attribute.String("language", req.Language),
        attribute.String("pattern", req.Pattern),
    )

    code, err := g.GenerateCode(ctx, req)
    if err != nil {
        span.RecordError(err)
        return "", err
    }

    span.SetAttributes(attribute.Int("code_length", len(code)))
    return code, nil
}
```

### Quality Assurance

- **Linting**: Run language-specific linters on all generated code before returning
- **Validation**: Verify code compiles and passes basic syntax checks
- **Test coverage**: Ensure generated tests achieve minimum coverage thresholds
- **Pattern compliance**: Verify generated code follows selected pattern rules

## Results

After implementing automated code generation, the team achieved:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Boilerplate Time | 15-20% | 3% | 82% reduction |
| Pattern Consistency | 60% | 96% | 60% improvement |
| Generation Accuracy | N/A | 92% | High accuracy |
| Developer Productivity | Baseline | +18% | 18% gain |
| Code Quality Score | 7.0/10 | 9.2/10 | 31% improvement |

## Related Resources

- [LLM Integration Guide](/guides/llm-integration/) for model selection and configuration
- [Prompt Engineering](/guides/prompt-engineering/) for effective prompt design
- [Structured Output](/guides/structured-output/) for type-safe code generation
