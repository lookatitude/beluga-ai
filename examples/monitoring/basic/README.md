# Monitoring Basic Example

This example demonstrates how to use the Monitoring package for observability, safety, and ethical AI monitoring.

## Prerequisites

- Go 1.21+
- Optional: OpenTelemetry collector (for distributed tracing)

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating a comprehensive monitoring system
2. Using structured logging
3. Creating trace spans for distributed tracing
4. Performing safety checks on content
5. Recording metrics automatically
6. Performing health checks

## Configuration Options

- `WithServiceName`: Service name for identification
- `WithOpenTelemetry`: OpenTelemetry endpoint URL
- `WithSafetyChecks`: Enable content safety validation
- `WithEthicalValidation`: Enable ethical AI monitoring
- `WithLogger`: Custom logger implementation
- `WithTracer`: Custom tracer implementation
- `WithMeter`: Custom metrics meter

## Using OpenTelemetry

To use OpenTelemetry for distributed tracing:

```go
monitor, err := monitoring.NewMonitor(
	monitoring.WithServiceName("my-service"),
	monitoring.WithOpenTelemetry("localhost:4317"),
)
```

## See Also

- [Monitoring Package Documentation](../../../pkg/monitoring/README.md)
- [Observability Integration Example](../../integration/observability/main.go)
