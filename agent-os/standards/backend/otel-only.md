# OTEL Only, No Non-OTEL Metrics

Use **OpenTelemetry** (`go.opentelemetry.io/otel`) for all metrics and tracing. Do not add Prometheus client, StatsD, or other instrumentation that exports outside OTEL.

Custom metric names and instruments (e.g. `meter.Int64Counter("myapp.requests.total", ...)`) are part of the OTEL model and are allowed. The rule is: no second telemetry stack.

**Why:** OpenTelemetry is the standard and is supported by many collectors and backends (Prometheus, Jaeger, etc.). A single OTEL pipeline keeps integration and operations simple.
