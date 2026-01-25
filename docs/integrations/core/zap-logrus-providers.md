# Zap/Logrus Logger Providers

Welcome, colleague! In this integration guide, we're going to integrate structured logging with Zap or Logrus in Beluga AI applications. Structured logging is essential for production debugging and observability.

## What you will build

You will create logger providers that integrate Zap or Logrus with Beluga AI's core package, enabling structured logging throughout your application with consistent formatting and log levels.

## Learning Objectives

- ✅ Configure Zap logger for Beluga AI
- ✅ Configure Logrus logger for Beluga AI
- ✅ Use structured logging in Beluga AI components
- ✅ Understand logging best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Choose one: Zap or Logrus

## Step 1: Install Dependencies

Install your chosen logger:
# For Zap
```bash
go get go.uber.org/zap
```

# For Logrus
bash
```bash
go get github.com/sirupsen/logrus
```

## Step 2: Zap Integration

Create a Zap logger provider:
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
    l.logger.Info(msg, l.fieldsToZap(fields)...)
}

func (l *ZapLogger) Error(ctx context.Context, msg string, err error, fields ...interface{}) {
    fields = append(fields, zap.Error(err))
    l.logger.Error(msg, l.fieldsToZap(fields)...)
}

func (l *ZapLogger) fieldsToZap(fields []interface{}) []zap.Field {
    var zapFields []zap.Field
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprintf("%v", fields[i])
            zapFields = append(zapFields, zap.Any(key, fields[i+1]))
        }
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
    logger.Error(ctx, "Operation failed", fmt.Errorf("test error"), "operation", "test")
}
```

### Verification

Run the example:
bash
```bash
go run main.go
```

You should see structured JSON logs.

## Step 3: Logrus Integration

Create a Logrus logger provider:
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
    entry := l.logger.WithFields(l.fieldsToLogrus(fields))
    entry.Info(msg)
}

func (l *LogrusLogger) Error(ctx context.Context, msg string, err error, fields ...interface{}) {
    entry := l.logger.WithFields(l.fieldsToLogrus(fields))
    entry.WithError(err).Error(msg)
}

func (l *LogrusLogger) fieldsToLogrus(fields []interface{}) logrus.Fields {
    logFields := make(logrus.Fields)
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprintf("%v", fields[i])
            logFields[key] = fields[i+1]
        }
    }
    return logFields
}

func main() {
    logger := NewLogrusLogger(logrus.InfoLevel)

    ctx := context.Background()
    logger.Info(ctx, "Application started", "version", "1.0.0")
    logger.Error(ctx, "Operation failed", fmt.Errorf("test error"), "operation", "test")
}
```

## Step 4: Integration with Beluga AI

Integrate logger with Beluga AI components:
```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/core"
    "go.uber.org/zap"
)

type LoggedRunnable struct {
    runnable core.Runnable
    logger   *ZapLogger
}

func NewLoggedRunnable(runnable core.Runnable, logger *ZapLogger) *LoggedRunnable {
    return &LoggedRunnable{
        runnable: runnable,
        logger:   logger,
    }
}

func (l *LoggedRunnable) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
    l.logger.Info(ctx, "Invoking runnable", "input", input)
    
    result, err := l.runnable.Invoke(ctx, input)
    if err != nil {
        l.logger.Error(ctx, "Runnable failed", err, "input", input)
        return nil, err
    }
    
    l.logger.Info(ctx, "Runnable completed", "result", result)
    return result, nil
}

func main() {
    logger, _ := NewZapLogger(zapcore.InfoLevel)
    defer logger.logger.Sync()

    runnable := core.NewRunnable(func(ctx context.Context, input interface{}) (interface{}, error) {
        return "result", nil
    })

    loggedRunnable := NewLoggedRunnable(runnable, logger)
    ctx := context.Background()
    result, _ := loggedRunnable.Invoke(ctx, "test")
    fmt.Println(result)
}
```

## Step 5: Context-Aware Logging

Add context values to logs:
```go
func (l *ZapLogger) WithContext(ctx context.Context) *zap.Logger {
    logger := l.logger
    
    // Add trace ID if available
    if traceID := getTraceID(ctx); traceID != "" {
        logger = logger.With(zap.String("trace_id", traceID))
    }
    
    // Add request ID if available
    if requestID := getRequestID(ctx); requestID != "" {
        logger = logger.With(zap.String("request_id", requestID))
    }

    
    return logger
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Level` | Log level (Debug, Info, Warn, Error) | `Info` | No |
| `Format` | Log format (JSON, Text) | `JSON` | No |
| `Output` | Log output (stdout, file) | `stdout` | No |

## Common Issues

### "Logger not initialized"

**Problem**: Logger used before initialization.

**Solution**: Initialize logger at application startup:var globalLogger *ZapLogger

```go
func init() {
    var err error
    globalLogger, err = NewZapLogger(zapcore.InfoLevel)
    if err != nil {
        panic(err)
    }
}
```

### "Logs not appearing"

**Problem**: Log level too high.

**Solution**: Set appropriate log level:logger, _ := NewZapLogger(zapcore.DebugLevel)
```

## Production Considerations

When using structured logging in production:

- **Use JSON format**: Easier to parse and search
- **Set appropriate levels**: Debug in dev, Info in prod
- **Include context**: Add trace IDs, request IDs
- **Monitor log volume**: Avoid excessive logging
- **Use sampling**: For high-volume logs

## Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/core"
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
    
    // Add caller information
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

## Next Steps

Congratulations! You've integrated structured logging with Beluga AI. Next, learn how to:

- **[Context Deep Dive](./context-deep-dive.md)** - Advanced context patterns
- **[Core Package Documentation](../../api-docs/packages/core.md)** - Deep dive into core package
- **[Observability Guide](../../guides/observability-tracing.md)** - Complete observability setup

---

**Ready for more?** Check out the Integrations Index for more integration guides!
