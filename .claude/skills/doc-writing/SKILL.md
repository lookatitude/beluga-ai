---
name: doc-writing
description: Documentation writing patterns for Beluga AI v2. Use when creating package documentation, tutorials, API guides, or any teaching-oriented content. Covers documentation structure, example patterns, and enterprise documentation standards.
---

# Documentation Writing Patterns for Beluga AI v2

## Package README Template

Every package should have documentation following this structure:

### 1. Header
One paragraph describing what the package does, who uses it, and why it exists.

### 2. Quick Start
Minimal working example (3-10 lines of Go code). Must compile. Include full imports.

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

model, err := llm.New("openai", llm.ProviderConfig{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "gpt-4o"})
resp, err := model.Generate(context.Background(), []schema.Message{schema.HumanMessage("Hello")})
```

### 3. Core Interface
Show the Go interface definition that users need to know.

### 4. Usage Examples
- **Basic** — simplest usage
- **With middleware** — adding cross-cutting concerns
- **With hooks** — lifecycle interception
- **Custom implementation** — extending the interface

### 5. Configuration
All `WithX()` options with their defaults and descriptions.

### 6. Extension Guide
How to create a custom provider or implementation:
1. Implement the interface
2. Register via `init()`
3. Map errors to `core.Error`
4. Write tests with recorded responses

## Code Example Standards

All code examples must:
1. Include full import paths (`github.com/lookatitude/beluga-ai/...`)
2. Handle errors explicitly (never `_` for error returns)
3. Use `context.Background()` or explain the context source
4. Show both the setup AND the usage
5. Be complete enough to compile (or clearly marked as pseudocode)

```go
// GOOD: complete, compilable example
func Example() {
    ctx := context.Background()
    model, err := llm.New("openai", llm.ProviderConfig{APIKey: "key", Model: "gpt-4o"})
    if err != nil {
        log.Fatal(err)
    }
    resp, err := model.Generate(ctx, []schema.Message{schema.HumanMessage("Hello")})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Text())
}

// BAD: incomplete, won't compile
model, _ := llm.New("openai", cfg)
resp, _ := model.Generate(ctx, msgs)
```

## Tutorial Structure

Tutorials follow progressive complexity:

1. **Basic** (5 min read) — single concept, one code example, immediate result
2. **Intermediate** (10 min read) — combines 2-3 concepts, realistic scenario
3. **Advanced** (15 min read) — production patterns, error handling, testing

Each tutorial section should:
- State what the reader will learn (1 sentence)
- Show the code (annotated)
- Explain what happened (1-2 paragraphs)
- Link to deeper documentation

## Documentation Do's and Don'ts

### DO
- Focus on what the user needs to DO, not internal implementation details
- Use consistent terminology from `CLAUDE.md` (Registry, Middleware, Hooks, Provider)
- Show patterns with concrete implementations
- Include both simple and advanced examples
- Cross-reference related packages with relative links
- Use tables for configuration options and error codes

### DON'T
- Reference implementation phases, timelines, or build order
- List every provider — point to `docs/providers.md` or the filesystem instead
- Duplicate architecture docs — link to them
- Use jargon without explaining it on first use
- Show incomplete examples that won't compile
- Add marketing language ("powerful", "revolutionary", "easy-to-use")

## Enterprise Documentation Tone

- **Professional and precise** — state facts, not opinions
- **Active voice** — "The registry stores factories" not "Factories are stored by the registry"
- **Present tense** — "The function returns" not "The function will return"
- **Imperative mood for instructions** — "Create a new file" not "You should create a new file"
- **No emojis** in technical documentation
- **No filler words** — remove "basically", "simply", "just", "obviously"

## Cross-Reference Conventions

When referencing other parts of the framework:
- **Other packages**: `` `llm/` `` or `` `rag/retriever/` `` with brief purpose reminder
- **Architecture docs**: `See docs/concepts.md Section 3.1 for the design rationale`
- **Skills**: `See the go-framework skill for the full registry pattern`
- **Code locations**: `Defined in llm/llm.go` with the interface name

## Diagram Guidelines

Use mermaid diagrams sparingly and only when they add clarity:
- **Architecture overviews** — layer diagrams showing package dependencies
- **Data flows** — request/response paths through the system
- **State machines** — lifecycle states (e.g., circuit breaker, voice session)

Keep diagrams simple (< 15 nodes). Complex diagrams should be split into focused sub-diagrams.
