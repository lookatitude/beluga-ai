# metrics.go Shape

**Struct:** `Metrics` holds `metric.Int64Counter`, `metric.Float64Histogram`, `metric.Int64UpDownCounter`, and `trace.Tracer` as needed.

**Constructor:** `NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error)`. Always take `meter` and `tracer` from the caller; do not create `otel.Meter`/`otel.Tracer` inside the package.

**NoOpMetrics():** Optional. Provide when the package is often tested or used without a real meter/tracer.

**Global (optional):** `InitMetrics(meter, tracer)` and `GetMetrics()` are allowed when the package uses a single global `Metrics` instance.
