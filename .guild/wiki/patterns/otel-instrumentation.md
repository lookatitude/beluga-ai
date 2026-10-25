---
type: standard
title: "OpenTelemetry Instrumentation Pattern"
description: "Migrated from .wiki/patterns/otel-instrumentation.md."
owner: orchestrator
confidence: high
trust_tier: reviewed
importance: high
source_refs: [wiki-patterns-otel-instrumentation]
created_at: 2026-04-11
updated_at: 2026-04-11
expires_at: null
supersedes: null
sensitivity: internal
recall_importance: 3
---
# OpenTelemetry Instrumentation Pattern

> Migrated from `.wiki/patterns/otel-instrumentation.md` on 2026-10-25 (Guild 3.x; structure kept). Verbatim original: `.guild/knowledge/sources/wiki-patterns-otel-instrumentation/original.md` (sha256 `ce17b290b237…`).

GenAI semantic conventions (v1.37+) with gen_ai.* attributes for operation tracing.

## Canonical Example

**File:** `o11y/tracer.go:15-47`

```go
const (
	AttrAgentName = "gen_ai.agent.name"
	AttrOperationName = "gen_ai.operation.name"
	AttrToolName = "gen_ai.tool.name"
	AttrRequestModel = "gen_ai.request.model"
	AttrResponseModel = "gen_ai.response.model"
	AttrInputTokens = "gen_ai.usage.input_tokens"
	AttrOutputTokens = "gen_ai.usage.output_tokens"
	AttrSystem = "gen_ai.system"
	AttrReasoningTokens = "gen_ai.usage.reasoning_tokens"
	AttrReasoningEffort = "gen_ai.request.reasoning_effort"
)
```

## Variations

1. **StartSpan with attributes** — `o11y/tracer.go:118-121`
   ```go
   ctx, span := tracer.Start(ctx, "llm.generate",
       trace.WithAttributes(attrsToOTel(Attrs{
           AttrAgentName: "agent-v2",
           AttrOperationName: "chat",
       })...))
   ```

2. **Status recording** — `o11y/tracer.go:100-107`
   - Maps StatusOK → otelcodes.Ok, StatusError → otelcodes.Error

## Anti-Patterns

- **Custom attribute names**: Using non-standard keys instead of gen_ai.* conventions
- **Missing operation type**: Not recording AttrOperationName; breaks observability
- **Token counts omitted**: Losing usage telemetry for cost tracking
- **Span not ended**: Resource leaks; span never recorded

## Invariants

- All span attributes use gen_ai.* prefix (OTel GenAI v1.37+ compliance)
- StatusCode always maps to otelcodes.Ok or otelcodes.Error before SetStatus
- Token counts (input, output, reasoning) recorded as int attributes
- Span.End() called in defer to guarantee cleanup
