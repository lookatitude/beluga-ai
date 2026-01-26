# Spans

**Start:** `ctx, span := tracer.Start(ctx, "package.thing.operation")`

**End:** Always `defer span.End()`. If the tracer can be no-op, guard with `if span != nil` or equivalent so `End` is safe.

**On error:** Use both:

- `span.RecordError(err)`
- `span.SetStatus(codes.Error, err.Error())`
