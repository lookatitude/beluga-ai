# Prometheus & Grafana Setup

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we will implement a connection between Beluga AI and Prometheus. We'll wire up a **Prometheus Exporter** that exposes your agent's heartbeat to the world and visualize it in Grafana.

## Learning Objectives
By the end of this tutorial, you will:
1.  Understand how `pkg/monitoring` metrics work.
2.  Implement a Prometheus HTTP exporter.
3.  Configure Grafana to visualize your agent's token usage and latency.

## Introduction
Welcome, colleague! Observability is the only way to know if your AI agents are thriving or hallucinating in production. While logs tell you *what* happened, metrics tell you *how often* and *how fast*.

Beluga AI's `pkg/monitoring` is built on top of **OpenTelemetry (OTel)**, the industry standard for observability. This means it is vendor-neutral by design. However, the most popular stack for collecting metrics is **Prometheus** (storage) and **Grafana** (visualization).

## Why This Matters

*   **Real-time Visibility**: See spikes in latency or error rates instantly.
*   **Cost Control**: Track token usage across different models and tenants.
*   **Safety**: Monitor the rate of "unsafe" content flags to detect adversarial attacks.

## Prerequisites

*   A working Go environment.
*   Docker (to run Prometheus/Grafana locally).
*   Understanding of HTTP handlers.

## Concepts

### The Metrics Pipeline
1.  **Instrumentation**: Your code calls `monitor.Metrics().Counter(...)`.
2.  **SDK**: Beluga AI aggregates these numbers in memory.
3.  **Exporter**: An HTTP server exposes these numbers at `/metrics`.
4.  **Scraper**: Prometheus polls `/metrics` every 15 seconds.

## Step-by-Step Implementation

### Step 1: The Monitoring Setup

First, let's look at how we normally initialize monitoring.
```go
package main

import (
    "context"
    "log"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

func main() {
    monitor, err := monitoring.NewMonitor(
        monitoring.WithServiceName("my-agent"),
        monitoring.WithMetrics(true),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

This sets up the internal meters, but it doesn't *expose* them yet.

### Step 2: Wiring the Prometheus Exporter

To expose metrics, we need to use the OpenTelemetry Prometheus exporter. Since `pkg/monitoring` is designed to be extensible, we can inject the exporter initialization.

*Note: You will need to add `go.opentelemetry.io/otel/exporters/prometheus` to your `go.mod`.*
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    "github.com/lookatitude/beluga-ai/pkg/monitoring"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func serveMetrics() {
    // 1. Create the Prometheus Exporter
    exporter, err := prometheus.New()
    if err != nil {
        log.Fatalf("failed to create prometheus exporter: %v", err)
    }

    // 2. Register the exporter with the OTel generic provider
    provider := metric.NewMeterProvider(
        metric.WithReader(exporter),
    )
    
    // 3. Set this provider as the global meter provider
    // Note: pkg/monitoring uses the global OTEL provider by default if configured
    // This is where you would integrate it with your monitor config.
    
    // 4. Start the HTTP server
    // Prometheus scrapes from /metrics
    http.Handle("/metrics", exporter)
    
    fmt.Println("Serving metrics on :2222/metrics")
    go func() {
        if err := http.ListenAndServe(":2222", nil); err != nil {
            log.Fatal(err)
        }
    }()
}
```

### Step 3: Recording Custom Metrics

Now that the server is running, let's record some data using `pkg/monitoring`.
```go
func runAgentLoop(monitor monitoring.Monitor) {
    ctx := context.Background()
    metrics := monitor.Metrics()



    // Define a custom metric
    // In a real app, do this once at startup
    metrics.RegisterCustomCounter("agent_thoughts_total", "Number of reasoning steps taken")

    // Record some data
    metrics.RecordCustomCounter(ctx, "agent_thoughts_total", 1, map[string]string{
        "model": "gpt-4",
        "type":  "reflection",
    })
    
    // Record standard framework metrics
    metrics.RecordAPIRequest(ctx, "POST", "/v1/chat")
}
```

### Step 4: Configuring Prometheus

Create a `prometheus.yml` file to tell Prometheus where your agent is running.
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'beluga-agent'
    static_configs:
      - targets: ['host.docker.internal:2222']
```

Run Prometheus with Docker:docker run -p 9090:9090 -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus
```

### Step 5: Visualizing in Grafana

1.  Run Grafana: `docker run -p 3000:3000 grafana/grafana`
2.  Login (admin/admin).
3.  Add Data Source -> Prometheus -> URL: `http://host.docker.internal:9090`.
4.  Create a Dashboard.

**Useful Queries:**

*   **Request Rate**: `rate(monitoring_api_requests_total[1m])`
*   **Error Rate**: `rate(monitoring_errors_total[1m])`
*   **Latency**: `histogram_quantile(0.95, sum(rate(monitoring_response_time_seconds_bucket[5m])) by (le))`

## Pro-Tips

*   **Cardinality Explosion**: Be careful with labels! Do not use "User ID" or "Message Content" as a metric label. If you have 1 million users, you will create 1 million metric series and crash your Prometheus server. Use high-level buckets like "user_tier" (free/paid) instead.
*   **Push vs Pull**: Prometheus "pulls" (scrapes) metrics. If your agent is a short-lived Lambda function, Prometheus might miss it. In that case, use the **OpenTelemetry Collector** as a gateway to "push" metrics to, which then holds them for Prometheus.

## Troubleshooting

### "Context Missing"
If your metrics aren't showing up, ensure you are passing `context.Background()` (or a derived context) into `Record...` methods. OpenTelemetry relies on context to propagate trace IDs.

### "No metrics endpoint"
Double check that `http.ListenAndServe(":2222", nil)` is actually running and not blocked behind a firewall. You should be able to `curl localhost:2222/metrics` and see text output.

## Conclusion

You have successfully connected your Beluga AI agent to the Prometheus ecosystem. By using the `pkg/monitoring` interfaces, your metrics code remains clean and decoupled from the backend implementation, allowing you to switch to DataDog or CloudWatch in the future just by changing the exporter configuration in Step 2.
