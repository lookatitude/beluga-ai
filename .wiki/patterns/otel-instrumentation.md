# Pattern: OTel GenAI Instrumentation

**Status:** stub — populate with `/wiki-learn`

## Contract

Every exported method on a package boundary opens an OTel span with `gen_ai.*` namespace attributes. Tenant, session, model, and tool info go on the span.

```go
ctx, span := tracer.Start(ctx, "gen_ai.chat.complete",
    trace.WithAttributes(
        attribute.String("gen_ai.system", "openai"),
        attribute.String("gen_ai.request.model", req.Model),
    ))
defer span.End()
```

## Canonical example

(populate via `/wiki-learn`)

## Anti-patterns

- Spans without `gen_ai.*` attributes (breaks downstream dashboards).
- Logging tokens or secrets as span attributes.
- Wrapping errors without `span.RecordError(err)`.

## Related

- `architecture/invariants.md#8-otel-genai-conventions`
- `patterns/error-handling.md`
