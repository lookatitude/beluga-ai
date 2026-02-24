---
title: "Guard Providers — AI Safety"
description: "5 guard providers for AI safety: Azure Safety, Guardrails AI, Lakera, LLM Guard, NeMo. Three-stage content validation pipeline in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AI safety, guard providers, content moderation, prompt injection, Lakera, NeMo Guardrails, Go, Beluga AI"
---

Beluga AI v2 provides a three-stage safety pipeline for validating content at every point in the agent lifecycle. The `guard.Guard` interface defines a unified contract for content validation, and providers integrate with external safety APIs for enterprise-grade moderation.

## Interface

```go
type Guard interface {
    Name() string
    Validate(ctx context.Context, input GuardInput) (GuardResult, error)
}
```

### Input and Result Types

```go
type GuardInput struct {
    Content  string         // Text to validate
    Role     string         // Pipeline stage: "input", "output", or "tool"
    Metadata map[string]any // Guard-specific key-value pairs
}

type GuardResult struct {
    Allowed   bool   // True when content passes validation
    Reason    string // Explanation when blocked or modified
    Modified  string // Optional sanitized content
    GuardName string // Which guard produced this result
}
```

## Three-Stage Pipeline

The guard pipeline validates content at three stages, each running an ordered sequence of guards:

```go
import "github.com/lookatitude/beluga-ai/guard"

pipeline := guard.NewPipeline(
    guard.Input(inputGuard1, inputGuard2),   // Validate user messages
    guard.Output(outputGuard1),               // Validate model responses
    guard.Tool(toolGuard1),                   // Validate tool call arguments
)
```

### Stage Execution

Guards within each stage execute sequentially. The first guard that blocks content stops the pipeline for that stage. If a guard modifies content (e.g., PII redaction), subsequent guards in the same stage see the modified version.

```go
// Validate user input
result, err := pipeline.ValidateInput(ctx, userMessage)
if err != nil {
    log.Fatal(err)
}
if !result.Allowed {
    fmt.Printf("Blocked by %s: %s\n", result.GuardName, result.Reason)
    return
}

// Validate model output
result, err = pipeline.ValidateOutput(ctx, modelResponse)

// Validate tool call
result, err = pipeline.ValidateTool(ctx, "search_web", toolInput)
```

## Built-in Guards

The `guard` package includes four built-in guards that require no external dependencies:

| Guard | Registry Name | Description |
|---|---|---|
| Prompt Injection Detector | `prompt_injection_detector` | Regex-based detection of injection patterns |
| PII Redactor | `pii_redactor` | Detects and replaces PII (email, phone, SSN, credit card, IP) |
| Content Filter | `content_filter` | Keyword-based content moderation with threshold |
| Spotlighting | `spotlighting` | Wraps untrusted content in delimiter markers |

### Using Built-in Guards

```go
pipeline := guard.NewPipeline(
    guard.Input(
        guard.NewPromptInjectionDetector(),
        guard.NewSpotlighting(""),
    ),
    guard.Output(
        guard.NewPIIRedactor(guard.DefaultPIIPatterns...),
        guard.NewContentFilter(guard.WithKeywords("harmful", "dangerous")),
    ),
)
```

## External Guard Providers

| Provider | Guard Name | Description |
|---|---|---|
| [Azure Content Safety](/docs/providers/guard/azuresafety) | `azure_content_safety` | Microsoft Azure AI Content Safety |
| [Guardrails AI](/docs/providers/guard/guardrailsai) | `guardrails_ai` | Guardrails AI validators (PII, toxicity, hallucination) |
| [Lakera Guard](/docs/providers/guard/lakera) | `lakera_guard` | Lakera prompt injection and content detection |
| [LLM Guard](/docs/providers/guard/llmguard) | `llm_guard` | LLM Guard prompt and output scanning |
| [NeMo Guardrails](/docs/providers/guard/nemo) | `nemo_guardrails` | NVIDIA NeMo Guardrails with Colang configs |

## Registry Usage

Guards can be created through the registry or constructed directly. Built-in guards register via `init()`:

```go
// Via registry (built-in guards)
g, err := guard.New("prompt_injection_detector", nil)
if err != nil {
    log.Fatal(err)
}

// Direct construction (external providers)
g, err := azuresafety.New(
    azuresafety.WithEndpoint(os.Getenv("AZURE_SAFETY_ENDPOINT")),
    azuresafety.WithAPIKey(os.Getenv("AZURE_SAFETY_KEY")),
)
```

### Provider Discovery

List all registered guards at runtime:

```go
names := guard.List()
// Returns sorted list: ["content_filter", "pii_redactor", "prompt_injection_detector", "spotlighting"]
```

## Composing a Full Safety Pipeline

A production safety pipeline typically combines built-in and external guards:

```go
import (
    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/guard/providers/lakera"
    "github.com/lookatitude/beluga-ai/guard/providers/azuresafety"
)

lakeraGuard, err := lakera.New(
    lakera.WithAPIKey(os.Getenv("LAKERA_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

azureGuard, err := azuresafety.New(
    azuresafety.WithEndpoint(os.Getenv("AZURE_SAFETY_ENDPOINT")),
    azuresafety.WithAPIKey(os.Getenv("AZURE_SAFETY_KEY")),
)
if err != nil {
    log.Fatal(err)
}

pipeline := guard.NewPipeline(
    guard.Input(
        lakeraGuard,                            // Prompt injection detection
        guard.NewPromptInjectionDetector(),      // Regex-based fallback
    ),
    guard.Output(
        azureGuard,                              // Content moderation
        guard.NewPIIRedactor(guard.DefaultPIIPatterns...), // PII redaction
    ),
    guard.Tool(
        guard.NewContentFilter(guard.WithKeywords("rm -rf", "DROP TABLE")),
    ),
)
```
