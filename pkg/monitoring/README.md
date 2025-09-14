# Monitoring Package

The `monitoring` package provides comprehensive observability, safety, and ethical monitoring capabilities for AI systems within the Beluga AI framework. It integrates structured logging, distributed tracing, metrics collection, health checks, safety validation, and ethical AI monitoring to ensure reliable and responsible AI operations.

## Features

- **Structured Logging**: Context-aware logging with JSON and colored console output
- **Distributed Tracing**: OpenTelemetry-based tracing with span management
- **Metrics Collection**: Comprehensive metrics with histograms, counters, and gauges
- **Health Monitoring**: Automated health checks for system components
- **Safety Validation**: Content safety checks with configurable risk thresholds
- **Ethical AI Monitoring**: Bias detection, privacy protection, and fairness metrics
- **Best Practices Validation**: Code and system best practices enforcement
- **Multiple Backends**: Support for OpenTelemetry, Prometheus, DataDog, and CloudWatch
- **Functional Options**: Flexible configuration using functional options pattern
- **Interface Segregation**: Clean, focused interfaces following SOLID principles

## Installation

```bash
go get github.com/lookatitude/beluga-ai/pkg/monitoring
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

func main() {
    // Create a comprehensive monitoring system
    monitor, err := monitoring.NewMonitor(
        monitoring.WithServiceName("my-ai-service"),
        monitoring.WithOpenTelemetry("localhost:4317"),
        monitoring.WithSafetyChecks(true),
        monitoring.WithEthicalValidation(true),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Start the monitoring system
    ctx := context.Background()
    if err := monitor.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer monitor.Stop(ctx)

    // Use in your AI operations
    result, err := monitor.SafetyChecker().CheckContent(ctx, userInput, "chat")
    if err != nil {
        monitor.Logger().Error(ctx, "Safety check failed", map[string]interface{}{
            "error": err.Error(),
        })
        return err
    }

    if !result.Safe {
        monitor.Logger().Warning(ctx, "Content flagged as unsafe", map[string]interface{}{
            "risk_score": result.RiskScore,
        })
        return errors.New("content flagged as unsafe")
    }
}
```

### Advanced Configuration

```go
// Load configuration from various sources
config, err := monitoring.LoadConfig(
    monitoring.WithServiceName("advanced-ai-service"),
    monitoring.WithOpenTelemetry("otel-collector:4317"),
    monitoring.WithLogging("debug", "json"),
    monitoring.WithSafety(0.8, true), // 80% risk threshold, human review enabled
    monitoring.WithEthics(0.7, true), // 70% fairness threshold, approval required
    monitoring.WithHealth(15*time.Second), // Health check every 15 seconds
)
if err != nil {
    log.Fatal(err)
}

// Create monitor with full configuration
monitor, err := monitoring.NewMonitor(
    monitoring.WithConfig(config),
)
```

## Architecture

The monitoring package follows the Beluga AI framework's design patterns:

```
pkg/monitoring/
├── monitoring.go          # Main interfaces and factory functions
├── config.go             # Configuration structs and validation
├── metrics.go            # Package-specific metrics
├── iface/                # Interface definitions
│   ├── core.go          # Core monitoring interfaces
│   ├── safety.go        # Safety and ethics interfaces
│   └── best_practices.go # Best practices interfaces
├── internal/             # Implementation details
│   ├── logger/          # Logging implementations
│   ├── tracer/          # Tracing implementations
│   ├── metrics/         # Metrics implementations
│   ├── health/          # Health check implementations
│   ├── safety/          # Safety implementations
│   ├── ethics/          # Ethics implementations
│   └── best_practices/  # Best practices implementations
└── providers/           # Backend provider implementations
    ├── opentelemetry/   # OpenTelemetry provider
    ├── prometheus/      # Prometheus provider
    ├── datadog/        # DataDog provider
    └── cloudwatch/     # CloudWatch provider
```

## Core Components

### Logger

Structured logging with context support and multiple output formats.

```go
logger := monitor.Logger()

// Basic logging
logger.Info(ctx, "Operation completed", map[string]interface{}{
    "duration": "150ms",
    "items": 42,
})

// Context logger with persistent fields
ctxLogger := logger.WithFields(map[string]interface{}{
    "user_id": "12345",
    "session_id": "abc-123",
})
ctxLogger.Info(ctx, "User action performed")
```

### Tracer

Distributed tracing with span management and OpenTelemetry integration.

```go
tracer := monitor.Tracer()

// Create a new span
ctx, span := tracer.StartSpan(ctx, "ai_inference",
    monitoring.WithTag("model", "gpt-4"),
    monitoring.WithTag("tokens", 150),
)
defer tracer.FinishSpan(span)

// Add logs to span
span.Log("Starting inference", map[string]interface{}{
    "input_length": len(input),
})

// Set error on span
if err != nil {
    span.SetError(err)
}
```

### Metrics Collector

Comprehensive metrics collection with counters, gauges, and histograms.

```go
metrics := monitor.Metrics()

// Counters
metrics.Counter(ctx, "requests_total", "Total requests", 1, map[string]string{
    "endpoint": "/api/chat",
    "method": "POST",
})

// Gauges
metrics.Gauge(ctx, "active_connections", "Active connections", 42, nil)

// Histograms
metrics.Histogram(ctx, "request_duration", "Request duration", 0.150, map[string]string{
    "endpoint": "/api/chat",
})

// Timers
timer := metrics.StartTimer(ctx, "operation_duration", map[string]string{
    "operation": "text_generation",
})
defer timer.Stop(ctx, "Text generation completed")
```

### Health Checker

Automated health monitoring for system components.

```go
healthChecker := monitor.HealthChecker()

// Register custom health check
healthChecker.RegisterCheck("database", func(ctx context.Context) monitoring.HealthCheckResult {
    if err := db.Ping(ctx); err != nil {
        return monitoring.HealthCheckResult{
            Status:    monitoring.StatusUnhealthy,
            Message:   "Database connection failed",
            CheckName: "database",
            Timestamp: time.Now(),
        }
    }
    return monitoring.HealthCheckResult{
        Status:    monitoring.StatusHealthy,
        Message:   "Database is healthy",
        CheckName: "database",
        Timestamp: time.Now(),
    }
})

// Check overall health
if !healthChecker.IsHealthy(ctx) {
    log.Fatal("System is unhealthy")
}
```

### Safety Checker

Content safety validation with configurable risk thresholds.

```go
safetyChecker := monitor.SafetyChecker()

result, err := safetyChecker.CheckContent(ctx, userInput, "chat")
if err != nil {
    return err
}

if result.RiskScore > 0.8 {
    // Request human review for high-risk content
    decision, err := safetyChecker.RequestHumanReview(ctx, userInput, "chat", result.RiskScore)
    if err != nil {
        return err
    }
    if !decision.Approved {
        return errors.New("content rejected by human reviewer")
    }
}
```

### Ethical Checker

Ethical AI validation including bias detection and fairness metrics.

```go
ethicalChecker := monitor.EthicalChecker()

ethicalCtx := monitoring.EthicalContext{
    UserDemographics: map[string]interface{}{
        "age_group": "25-34",
        "region": "US",
    },
    ContentType: "social_media",
    Domain: "content_moderation",
}

analysis, err := ethicalChecker.CheckContent(ctx, content, ethicalCtx)
if err != nil {
    return err
}

if analysis.OverallRisk == "high" {
    monitor.Logger().Warning(ctx, "High ethical risk detected", map[string]interface{}{
        "bias_issues": len(analysis.BiasIssues),
        "fairness_score": analysis.FairnessScore,
    })
}
```

### Best Practices Checker

Validation of code and system best practices.

```go
bestPracticesChecker := monitor.BestPracticesChecker()

issues := bestPracticesChecker.Validate(ctx, codeSnippet, "ai_model")
for _, issue := range issues {
    monitor.Logger().Warning(ctx, "Best practice violation", map[string]interface{}{
        "validator": issue.Validator,
        "issue": issue.Issue,
        "severity": issue.Severity,
        "suggestion": issue.Suggestion,
    })
}
```

## Configuration

The monitoring package supports comprehensive configuration through functional options and external configuration files.

### Configuration Options

```go
type Config struct {
    ServiceName string                    // Service identification
    OpenTelemetry OpenTelemetryConfig     // OpenTelemetry settings
    Logging       LoggingConfig          // Logging configuration
    Tracing       TracingConfig          // Tracing configuration
    Metrics       MetricsConfig          // Metrics configuration
    Safety        SafetyConfig           // Safety validation config
    Ethics        EthicsConfig           // Ethical AI config
    Health        HealthConfig           // Health monitoring config
    BestPractices BestPracticesConfig     // Best practices config
}
```

### Environment Variables

```bash
# Service
SERVICE_NAME=my-ai-service

# OpenTelemetry
OTEL_ENABLED=true
OTEL_ENDPOINT=localhost:4317
OTEL_SERVICE_NAME=my-ai-service

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT_FILE=/var/log/ai-service.log

# Safety
SAFETY_ENABLED=true
SAFETY_RISK_THRESHOLD=0.7
SAFETY_AUTO_BLOCK=true

# Ethics
ETHICS_ENABLED=true
ETHICS_FAIRNESS_THRESHOLD=0.7
ETHICS_HUMAN_APPROVAL=false
```

### YAML Configuration

```yaml
service_name: "my-ai-service"

opentelemetry:
  enabled: true
  endpoint: "localhost:4317"
  service_name: "my-ai-service"
  environment: "production"

logging:
  enabled: true
  level: "info"
  format: "json"
  output_file: "/var/log/ai-service.log"

safety:
  enabled: true
  risk_threshold: 0.7
  auto_block_high_risk: true
  enable_human_review: false

ethics:
  enabled: true
  fairness_threshold: 0.7
  require_human_approval: false
```

## Backend Providers

### OpenTelemetry Provider

```go
import "github.com/lookatitude/beluga-ai/pkg/monitoring/providers/opentelemetry"

// Create OpenTelemetry provider
provider, err := opentelemetry.NewProvider(opentelemetry.Config{
    Endpoint:        "localhost:4317",
    ServiceName:     "my-service",
    ServiceVersion:  "1.0.0",
    Environment:     "production",
    SampleRate:      1.0,
})
```

### Prometheus Provider

```go
import "github.com/lookatitude/beluga-ai/pkg/monitoring/providers/prometheus"

// Create Prometheus provider
provider, err := prometheus.NewProvider(prometheus.Config{
    Endpoint: "/metrics",
    Namespace: "ai_service",
})
```

## Best Practices

### Error Handling

```go
// Always check for errors
result, err := monitor.SafetyChecker().CheckContent(ctx, content, "chat")
if err != nil {
    // Log error with context
    monitor.Logger().Error(ctx, "Safety check failed",
        map[string]interface{}{
            "error": err.Error(),
            "content_length": len(content),
        })
    return err
}
```

### Context Propagation

```go
// Always pass context through the call chain
func processRequest(ctx context.Context, req *Request) error {
    // Start a new span
    ctx, span := monitor.Tracer().StartSpan(ctx, "process_request")
    defer span.End()

    // Add span to context for logging
    monitor.Logger().Info(ctx, "Processing request",
        map[string]interface{}{
            "request_id": req.ID,
        })

    return processContent(ctx, req.Content)
}

func processContent(ctx context.Context, content string) error {
    // Context is automatically propagated
    result, err := monitor.SafetyChecker().CheckContent(ctx, content, "text")
    return err
}
```

### Resource Management

```go
// Always defer cleanup
monitor, err := monitoring.NewMonitor(...)
if err != nil {
    return err
}
defer monitor.Stop(ctx)

// Start monitoring
if err := monitor.Start(ctx); err != nil {
    return err
}
```

### Configuration Validation

```go
// Always validate configuration
config := monitoring.DefaultConfig()
config.ServiceName = "my-service"

// Validate before using
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}
```

## Extensibility

The monitoring package is designed for easy extension:

### Custom Validators

```go
// Implement custom validator
type CustomValidator struct{}

func (cv *CustomValidator) Name() string { return "custom" }

func (cv *CustomValidator) Validate(ctx context.Context, data interface{}) []monitoring.ValidationIssue {
    // Custom validation logic
    return issues
}

// Add to best practices checker
checker.AddValidator(&CustomValidator{})
```

### Custom Providers

```go
// Implement custom provider
type CustomProvider struct {
    // implementation
}

func (cp *CustomProvider) Tracer() iface.Tracer {
    // Return custom tracer implementation
}

// Use in monitor
monitor := &defaultMonitor{
    tracer: &CustomProvider{},
}
```

### Custom Metrics

```go
// Define custom metrics
customMetrics := monitor.Metrics()

// Register custom metric
customMetrics.Counter(ctx, "custom_metric", "Custom metric description", 1, map[string]string{
    "label": "value",
})
```

## Performance Considerations

- **Sampling**: Configure appropriate sampling rates for high-throughput systems
- **Buffering**: Use span buffering for high-frequency operations
- **Async Operations**: Health checks and metrics collection run asynchronously
- **Resource Limits**: Configure appropriate limits for concurrent operations
- **Cleanup**: Always clean up resources to prevent memory leaks

## Security Considerations

- **PII Detection**: Automatic detection and masking of personal information
- **Access Control**: Role-based access to sensitive monitoring data
- **Audit Logging**: Comprehensive audit trails for all operations
- **Encryption**: Secure transmission of monitoring data
- **Input Validation**: Validation of all input parameters

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   - Reduce span buffer size
   - Implement sampling for high-frequency operations
   - Clean up completed spans regularly

2. **Slow Performance**
   - Check OpenTelemetry exporter configuration
   - Optimize health check intervals
   - Use asynchronous logging

3. **Missing Traces**
   - Verify OpenTelemetry endpoint configuration
   - Check network connectivity
   - Validate span context propagation

4. **Health Check Failures**
   - Review timeout configurations
   - Check component dependencies
   - Validate health check logic

### Debug Mode

```go
// Enable debug logging
monitor, err := monitoring.NewMonitor(
    monitoring.WithLogLevel(monitoring.DEBUG),
    monitoring.WithTracing(true),
)
```

## Contributing

1. Follow the Beluga AI framework design patterns
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Ensure backward compatibility
5. Run the full test suite before submitting

## License

This package is part of the Beluga AI framework and follows the same license terms.

## Related Packages

- [`github.com/lookatitude/beluga-ai/pkg/agents`](https://github.com/lookatitude/beluga-ai/pkg/agents) - AI agent implementations
- [`github.com/lookatitude/beluga-ai/pkg/vectorstores`](https://github.com/lookatitude/beluga-ai/pkg/vectorstores) - Vector database integrations
- [`github.com/lookatitude/beluga-ai/pkg/memory`](https://github.com/lookatitude/beluga-ai/pkg/memory) - Memory management systems
