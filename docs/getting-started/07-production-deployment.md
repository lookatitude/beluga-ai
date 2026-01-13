# Part 7: Production Deployment

In this tutorial, you'll learn how to deploy Beluga AI applications to production. We'll cover configuration management, observability, health checks, and deployment strategies.

## Learning Objectives

- ✅ Configure applications for production
- ✅ Set up observability (OTEL)
- ✅ Implement health checks
- ✅ Deploy to different environments
- ✅ Monitor and debug production systems

## Prerequisites

- Completed all previous tutorials
- Basic understanding of deployment concepts
- Access to deployment infrastructure

## Step 1: Configuration Management

### Environment-Based Configuration

```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/config"
    "os"
)

func loadConfig() (*config.Config, error) {
    env := os.Getenv("ENVIRONMENT")
    if env == "" {
        env = "development"
    }

    configFile := fmt.Sprintf("config.%s.yaml", env)
    cfgProvider, err := config.NewViperProvider(configFile)
    if err != nil {
        return nil, err
    }

    return cfgProvider, nil
}
```

### Configuration File Structure

```yaml
# config.production.yaml
app:
  name: "beluga-ai-app"
  version: "1.0.0"
  environment: "production"

llm_providers:
  - name: "openai-production"
    provider: "openai"
    model_name: "gpt-4"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 5
    retry_delay: "2s"

observability:
  tracing:
    enabled: true
    endpoint: "${JAEGER_ENDPOINT}"
  metrics:
    enabled: true
    endpoint: "${PROMETHEUS_ENDPOINT}"
  logging:
    level: "info"
    format: "json"

server:
  port: 8080
  timeout: "30s"
  health_check_path: "/health"
```

## Step 2: Observability Setup

### OpenTelemetry Configuration

```go
package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
)

func setupTracing(ctx context.Context) (*trace.TracerProvider, error) {
    // Create Jaeger exporter
    exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint(os.Getenv("JAEGER_ENDPOINT")),
    ))
    if err != nil {
        return nil, err
    }

    // Create tracer provider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exp),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("beluga-ai-app"),
        )),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### Metrics Collection

```go
import "github.com/lookatitude/beluga-ai/pkg/monitoring"

func setupMetrics() (*monitoring.MetricsCollector, error) {
    metrics := monitoring.NewMetricsCollector()
    
    // Export to Prometheus
    // (implementation depends on your setup)
    
    return metrics, nil
}
```

### Structured Logging

```go
import "github.com/lookatitude/beluga-ai/pkg/monitoring"

func setupLogging() (*monitoring.StructuredLogger, error) {
    logger := monitoring.NewStructuredLogger(
        "beluga-ai-app",
        monitoring.WithJSONOutput(),
        monitoring.WithLogLevel("info"),
        monitoring.WithFileOutput("logs/app.log"),
    )
    
    return logger, nil
}
```

## Step 3: Health Checks

### Implement Health Check Endpoint

```go
package main

import (
    "encoding/json"
    "net/http"
    "time"
)

type HealthStatus struct {
    Status    string            `json:"status"`
    Timestamp time.Time         `json:"timestamp"`
    Checks    map[string]string `json:"checks"`
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    status := HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks:    make(map[string]string),
    }

    // Check LLM provider
    if checkLLMProvider() {
        status.Checks["llm"] = "ok"
    } else {
        status.Checks["llm"] = "error"
        status.Status = "unhealthy"
    }

    // Check vector store
    if checkVectorStore() {
        status.Checks["vectorstore"] = "ok"
    } else {
        status.Checks["vectorstore"] = "error"
        status.Status = "unhealthy"
    }

    w.Header().Set("Content-Type", "application/json")
    if status.Status != "healthy" {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    json.NewEncoder(w).Encode(status)
}
```

### Graceful Shutdown

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // Setup application
    app := setupApplication()

    // Start server
    go app.Start()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := app.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exiting")
}
```

## Step 4: Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/app .
COPY --from=builder /app/config.production.yaml .

EXPOSE 8080
CMD ["./app"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  beluga-app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ENVIRONMENT=production
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    depends_on:
      - postgres
      - jaeger

  postgres:
    image: pgvector/pgvector:pg15
    environment:
      POSTGRES_USER: beluga
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: beluga
    volumes:
      - postgres-data:/var/lib/postgresql/data

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
```

## Step 5: Kubernetes Deployment

### Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: beluga-ai-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: beluga-ai
  template:
    metadata:
      labels:
        app: beluga-ai
    spec:
      containers:
      - name: app
        image: beluga-ai:latest
        ports:
        - containerPort: 8080
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: beluga-secrets
              key: openai-api-key
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Step 6: Monitoring and Alerting

### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "beluga_requests_total",
            Help: "Total number of requests",
        },
        []string{"endpoint", "status"},
    )
)

func init() {
    prometheus.MustRegister(requestsTotal)
}
```

### Logging Best Practices

```go
logger.Info(ctx, "Request processed", map[string]interface{}{
    "user_id": userID,
    "request_id": requestID,
    "duration_ms": duration,
    "status": "success",
})
```

## Step 7: Security Considerations

### API Key Management

```go
// Use environment variables or secret management
apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
}

// Or use a secret manager
apiKey, err := secretManager.GetSecret(ctx, "openai-api-key")
```

### Input Validation

```go
func validateInput(input string) error {
    if len(input) > 10000 {
        return fmt.Errorf("input too long")
    }
    // Add more validation...
    return nil
}
```

## Step 8: Complete Production Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

func main() {
    ctx := context.Background()

    // Setup observability
    logger, _ := setupLogging()
    metrics, _ := setupMetrics()
    tracer, _ := setupTracing(ctx)

    defer tracer.Shutdown(ctx)

    // Load configuration
    cfg, _ := loadConfig()

    // Setup application
    app := NewApplication(cfg, logger, metrics)

    // Setup HTTP server
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthCheckHandler)
    mux.HandleFunc("/api/v1/chat", app.ChatHandler)

    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    // Start server
    go func() {
        log.Println("Server starting on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exited")
}
```

## Exercises

1. **Deploy to cloud**: Deploy your application to AWS, GCP, or Azure
2. **Set up monitoring**: Configure Prometheus and Grafana
3. **Implement alerts**: Set up alerting for errors and performance
4. **Load testing**: Test your application under load
5. **CI/CD pipeline**: Set up automated deployment

## Next Steps

Congratulations! You've completed the Getting Started Tutorial series. Next, explore:

- **[Best Practices](../best-practices.md)** - Production best practices
- **[Troubleshooting](../troubleshooting.md)** - Common issues and solutions
- **[Use Cases](../use-cases/)** - Real-world examples
- **[API Reference](../../website/docs/api/)** - Detailed API documentation

---

**You've completed the Getting Started Tutorial series!** 🎉

You now have the knowledge to build production-ready AI applications with Beluga AI. Continue learning by exploring the advanced concepts and use cases.

