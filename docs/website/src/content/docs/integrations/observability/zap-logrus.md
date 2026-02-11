---
title: Zap and Logrus Logger Providers
description: Integrate structured logging with Zap or Logrus in Beluga AI applications for production-grade observability and trace correlation.
---

Go's standard `slog` package covers basic logging needs, but production AI applications often require more: Zap's near-zero-allocation performance for high-throughput agent pipelines, Logrus's ecosystem of hooks for log routing, or integration with an existing logging stack that your team already knows. This guide shows how to integrate Uber's Zap and Sirupsen's Logrus as logger providers, including context-aware logging with OpenTelemetry trace correlation so you can follow a single request from HTTP handler through agent execution and LLM call.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- One of: Zap or Logrus

## Installation

Install your preferred logging library:

```bash
# Zap
go get go.uber.org/zap

# Logrus
go get github.com/sirupsen/logrus
```

## Zap Integration

Zap provides high-performance, structured, leveled logging. Create a wrapper that accepts `context.Context` for trace propagation:

```go
package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(level zapcore.Level) (*ZapLogger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create zap logger: %w", err)
	}

	return &ZapLogger{logger: logger}, nil
}

func (l *ZapLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.logger.Info(msg, l.toZapFields(fields)...)
}

func (l *ZapLogger) Error(ctx context.Context, msg string, err error, fields ...interface{}) {
	zapFields := append(l.toZapFields(fields), zap.Error(err))
	l.logger.Error(msg, zapFields...)
}

func (l *ZapLogger) toZapFields(fields []interface{}) []zap.Field {
	var zapFields []zap.Field
	for i := 0; i+1 < len(fields); i += 2 {
		key := fmt.Sprintf("%v", fields[i])
		zapFields = append(zapFields, zap.Any(key, fields[i+1]))
	}
	return zapFields
}

func main() {
	logger, err := NewZapLogger(zapcore.InfoLevel)
	if err != nil {
		panic(err)
	}
	defer logger.logger.Sync()

	ctx := context.Background()
	logger.Info(ctx, "Application started", "version", "1.0.0")
	logger.Error(ctx, "Operation failed", fmt.Errorf("connection timeout"), "operation", "llm_call")
}
```

Running this produces structured JSON output:

```json
{"level":"info","ts":1700000000.000,"msg":"Application started","version":"1.0.0"}
{"level":"error","ts":1700000000.001,"msg":"Operation failed","operation":"llm_call","error":"connection timeout"}
```

## Logrus Integration

Logrus provides a structured logger with a familiar API. Wrap it with the same context-aware interface:

```go
package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

type LogrusLogger struct {
	logger *logrus.Logger
}

func NewLogrusLogger(level logrus.Level) *LogrusLogger {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &LogrusLogger{logger: logger}
}

func (l *LogrusLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.logger.WithFields(l.toLogrusFields(fields)).Info(msg)
}

func (l *LogrusLogger) Error(ctx context.Context, msg string, err error, fields ...interface{}) {
	l.logger.WithFields(l.toLogrusFields(fields)).WithError(err).Error(msg)
}

func (l *LogrusLogger) toLogrusFields(fields []interface{}) logrus.Fields {
	logFields := make(logrus.Fields)
	for i := 0; i+1 < len(fields); i += 2 {
		key := fmt.Sprintf("%v", fields[i])
		logFields[key] = fields[i+1]
	}
	return logFields
}

func main() {
	logger := NewLogrusLogger(logrus.InfoLevel)

	ctx := context.Background()
	logger.Info(ctx, "Application started", "version", "1.0.0")
	logger.Error(ctx, "Operation failed", fmt.Errorf("connection timeout"), "operation", "llm_call")
}
```

## Context-Aware Logging

Extract trace and request IDs from the context and attach them to every log entry. This enables correlation between logs and distributed traces:

```go
func (l *ZapLogger) WithContext(ctx context.Context) *zap.Logger {
	logger := l.logger

	if traceID := getTraceID(ctx); traceID != "" {
		logger = logger.With(zap.String("trace_id", traceID))
	}

	if requestID := getRequestID(ctx); requestID != "" {
		logger = logger.With(zap.String("request_id", requestID))
	}

	return logger
}
```

## Configuration Reference

| Option | Description | Default |
|--------|-------------|---------|
| `Level` | Minimum log level (Debug, Info, Warn, Error) | `Info` |
| `Format` | Output format (JSON, Text) | `JSON` |
| `Output` | Destination (stdout, stderr, file path) | `stdout` |

## Production Example with OTel Trace Correlation

Combine Zap with OpenTelemetry to automatically inject trace and span IDs into every log message:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ProductionLogger struct {
	logger *zap.Logger
	tracer trace.Tracer
}

func NewProductionLogger() (*ProductionLogger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	logger, err := config.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &ProductionLogger{
		logger: logger,
		tracer: otel.Tracer("beluga.core.logger"),
	}, nil
}

func (l *ProductionLogger) LogWithContext(ctx context.Context, level zapcore.Level, msg string, fields ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		fields = append(fields,
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	if ce := l.logger.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}

func main() {
	logger, err := NewProductionLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.logger.Sync()

	ctx := context.Background()
	logger.LogWithContext(ctx, zapcore.InfoLevel, "Application started",
		zap.String("version", "1.0.0"),
		zap.String("environment", "production"),
	)
}
```

## Troubleshooting

### Logger not initialized

If the logger is used before initialization, the program will panic on a nil pointer. Initialize the logger as early as possible in the application lifecycle:

```go
var globalLogger *ZapLogger

func init() {
	var err error
	globalLogger, err = NewZapLogger(zapcore.InfoLevel)
	if err != nil {
		panic(err)
	}
}
```

### Logs not appearing

The log level may be higher than the messages being emitted. For development, set the level to Debug:

```go
logger, err := NewZapLogger(zapcore.DebugLevel)
```

## Production Considerations

- **Use JSON format** for log aggregation systems (Elasticsearch, Loki, Datadog).
- **Set appropriate levels** -- Debug in development, Info or Warn in production.
- **Include context fields** -- trace IDs, request IDs, and tenant IDs enable log correlation across services.
- **Monitor log volume** -- excessive Debug or Info logging can degrade performance and inflate storage costs.
- **Enable sampling** for high-throughput paths to reduce volume without losing visibility.

## Related Resources

- [Context Deep Dive](/integrations/context-deep-dive) -- Advanced context patterns for cancellation and timeouts
- [Infrastructure Integrations](/integrations/infrastructure) -- Deployment and infrastructure overview
