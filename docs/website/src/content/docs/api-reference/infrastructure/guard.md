---
title: "Guard API — Safety Pipeline, PII, Injection"
description: "Guard package API reference for Beluga AI. Three-stage safety pipeline with prompt injection, PII redaction, content filtering, and providers."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "guard API, safety pipeline, prompt injection, PII redaction, content filter, Lakera, NeMo, Beluga AI, Go, reference"
---

## guard

```go
import "github.com/lookatitude/beluga-ai/guard"
```

Package guard provides a three-stage safety pipeline for the Beluga AI
framework. It validates content at three points: input (user messages),
output (model responses), and tool (tool call arguments). Each stage runs
a configurable set of Guard implementations that can block, modify, or
allow content to pass through.

## Guard Interface

The core Guard interface requires two methods:

- Name returns a unique identifier for the guard.
- Validate checks content and returns a GuardResult indicating whether
  the content is allowed, along with an optional modified version.

## Built-in Guards

The package ships with four built-in guard implementations:

- PromptInjectionDetector detects common prompt injection patterns using
  configurable regular expressions.
- PIIRedactor detects and redacts personally identifiable information
  (email, phone, SSN, credit card, IP address) using regex-based patterns.
- ContentFilter performs keyword-based content moderation with a
  configurable match threshold.
- Spotlighting wraps untrusted content in delimiters to isolate it
  from trusted instructions, reducing prompt injection effectiveness.

## Pipeline

Guards are composed into a Pipeline using the Input, Output, and Tool
stage options. The Pipeline runs guards sequentially within each stage;
the first guard that blocks stops the pipeline for that stage. Modified
content from one guard is passed to subsequent guards.

## Registry

The package follows the standard Beluga registry pattern with Register,
New, and List functions. Built-in guards register themselves via init.
External guard providers (Azure Content Safety, Lakera, NeMo, etc.) are
available under guard/providers/.

## Usage

Create a pipeline with input, output, and tool guards:

```go
p := guard.NewPipeline(
    guard.Input(guard.NewPromptInjectionDetector()),
    guard.Output(guard.NewPIIRedactor(guard.DefaultPIIPatterns...)),
    guard.Tool(guard.NewContentFilter(guard.WithKeywords("drop", "delete"))),
)
result, err := p.ValidateInput(ctx, userMessage)
if err != nil {
    log.Fatal(err)
}
if !result.Allowed {
    fmt.Println("blocked:", result.Reason)
}
```

Use the registry to create guards by name:

```go
g, err := guard.New("prompt_injection_detector", nil)
if err != nil {
    log.Fatal(err)
}
result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
```

---

## azuresafety

```go
import "github.com/lookatitude/beluga-ai/guard/providers/azuresafety"
```

Package azuresafety provides an Azure Content Safety guard implementation for
the Beluga AI safety pipeline. It implements the guard.Guard interface and
sends content validation requests to the Azure Content Safety API.

Azure Content Safety provides text moderation across categories including
Hate, SelfHarm, Sexual, and Violence with configurable severity thresholds.

## Configuration

The guard is configured using functional options:

- WithEndpoint sets the Azure Content Safety endpoint URL (required).
- WithAPIKey sets the API key for authentication (required).
- WithThreshold sets the severity threshold (0-6); content at or above
  this severity in any category is blocked. Defaults to 2.
- WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.

## Usage

```go
g, err := azuresafety.New(
    azuresafety.WithEndpoint("https://myinstance.cognitiveservices.azure.com"),
    azuresafety.WithAPIKey("key-..."),
    azuresafety.WithThreshold(4),
)
if err != nil {
    log.Fatal(err)
}
result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
```

---

## guardrailsai

```go
import "github.com/lookatitude/beluga-ai/guard/providers/guardrailsai"
```

Package guardrailsai provides a Guardrails AI guard implementation for the
Beluga AI safety pipeline. It implements the guard.Guard interface and sends
content validation requests to a Guardrails AI API endpoint.

Guardrails AI provides validators for PII detection, toxicity, hallucination,
prompt injection, and custom rules defined via RAIL specifications.

## Configuration

The guard is configured using functional options:

- WithBaseURL sets the Guardrails AI API base URL. Defaults to
  "http://localhost:8000".
- WithAPIKey sets the API key for authentication (optional).
- WithGuardName sets the guard name to invoke on the server. Defaults to
  "default".
- WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.

## Usage

```go
g, err := guardrailsai.New(
    guardrailsai.WithBaseURL("http://localhost:8000"),
    guardrailsai.WithGuardName("my-guard"),
)
if err != nil {
    log.Fatal(err)
}
result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "output"})
```

---

## lakera

```go
import "github.com/lookatitude/beluga-ai/guard/providers/lakera"
```

Package lakera provides a Lakera Guard API guard implementation for the
Beluga AI safety pipeline. It implements the guard.Guard interface and sends
content validation requests to the Lakera Guard API endpoint.

Lakera Guard detects prompt injections, jailbreaks, PII, and harmful content.

## Configuration

The guard is configured using functional options:

- WithAPIKey sets the Lakera Guard API key (required).
- WithBaseURL sets the API base URL. Defaults to "https://api.lakera.ai".
- WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.

## Usage

```go
g, err := lakera.New(
    lakera.WithAPIKey("lk-..."),
)
if err != nil {
    log.Fatal(err)
}
result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
```

---

## llmguard

```go
import "github.com/lookatitude/beluga-ai/guard/providers/llmguard"
```

Package llmguard provides an LLM Guard API guard implementation for the
Beluga AI safety pipeline. It implements the guard.Guard interface and sends
content validation requests to an LLM Guard API endpoint.

LLM Guard provides prompt injection detection, toxicity filtering, and
sensitive data detection via its REST API. It uses the /analyze/prompt
endpoint for input content and /analyze/output for output content.

## Configuration

The guard is configured using functional options:

- WithBaseURL sets the LLM Guard API base URL. Defaults to
  "http://localhost:8000".
- WithAPIKey sets the API key for authentication (optional).
- WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.

## Usage

```go
g, err := llmguard.New(
    llmguard.WithBaseURL("http://localhost:8000"),
)
if err != nil {
    log.Fatal(err)
}
result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
```

---

## nemo

```go
import "github.com/lookatitude/beluga-ai/guard/providers/nemo"
```

Package nemo provides an NVIDIA NeMo Guardrails guard implementation for the
Beluga AI safety pipeline. It implements the guard.Guard interface and sends
content validation requests to a NeMo Guardrails API endpoint.

NeMo Guardrails can be configured to check for topic safety, jailbreak
detection, fact-checking, and more via Colang configurations.

## Configuration

The guard is configured using functional options:

- WithBaseURL sets the NeMo Guardrails API base URL. Defaults to
  "http://localhost:8080".
- WithAPIKey sets the API key for authentication (optional).
- WithConfigID sets the guardrails configuration ID. Defaults to "default".
- WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.

## Usage

```go
g, err := nemo.New(
    nemo.WithBaseURL("http://localhost:8080"),
    nemo.WithConfigID("my-config"),
)
if err != nil {
    log.Fatal(err)
}
result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
```
