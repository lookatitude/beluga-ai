// Package server provides metrics definitions for server implementations.
package server

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics contains all metrics for the server package
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal     Int64Counter
	httpRequestDuration   Float64Histogram
	httpActiveConnections metric.Int64UpDownCounter
	httpErrorsTotal       Int64Counter

	// MCP metrics
	mcpToolsTotal           Int64Counter
	mcpToolCallsTotal       Int64Counter
	mcpToolCallDuration     Float64Histogram
	mcpResourcesTotal       Int64Counter
	mcpResourceReadsTotal   Int64Counter
	mcpResourceReadDuration Float64Histogram
	mcpErrorsTotal          Int64Counter

	// Server metrics
	serverUptime       Float64Histogram
	serverHealthChecks Int64Counter
}

// NewMetrics creates a new Metrics instance with the given meter
func NewMetrics(meter Meter) *Metrics {
	httpRequestsTotal, _ := meter.Int64Counter("server_http_requests_total",
		metric.WithDescription("Total number of HTTP requests"))
	httpRequestDuration, _ := meter.Float64Histogram("server_http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"))
	httpActiveConnections, _ := meter.Int64UpDownCounter("server_http_active_connections",
		metric.WithDescription("Number of active HTTP connections"))
	httpErrorsTotal, _ := meter.Int64Counter("server_http_errors_total",
		metric.WithDescription("Total number of HTTP errors"))

	mcpToolsTotal, _ := meter.Int64Counter("server_mcp_tools_total",
		metric.WithDescription("Total number of registered MCP tools"))
	mcpToolCallsTotal, _ := meter.Int64Counter("server_mcp_tool_calls_total",
		metric.WithDescription("Total number of MCP tool calls"))
	mcpToolCallDuration, _ := meter.Float64Histogram("server_mcp_tool_call_duration_seconds",
		metric.WithDescription("Duration of MCP tool calls in seconds"),
		metric.WithUnit("s"))
	mcpResourcesTotal, _ := meter.Int64Counter("server_mcp_resources_total",
		metric.WithDescription("Total number of registered MCP resources"))
	mcpResourceReadsTotal, _ := meter.Int64Counter("server_mcp_resource_reads_total",
		metric.WithDescription("Total number of MCP resource reads"))
	mcpResourceReadDuration, _ := meter.Float64Histogram("server_mcp_resource_read_duration_seconds",
		metric.WithDescription("Duration of MCP resource reads in seconds"),
		metric.WithUnit("s"))
	mcpErrorsTotal, _ := meter.Int64Counter("server_mcp_errors_total",
		metric.WithDescription("Total number of MCP errors"))

	serverUptime, _ := meter.Float64Histogram("server_uptime_seconds",
		metric.WithDescription("Server uptime in seconds"),
		metric.WithUnit("s"))
	serverHealthChecks, _ := meter.Int64Counter("server_health_checks_total",
		metric.WithDescription("Total number of health checks"))

	return &Metrics{
		httpRequestsTotal:       httpRequestsTotal,
		httpRequestDuration:     httpRequestDuration,
		httpActiveConnections:   httpActiveConnections,
		httpErrorsTotal:         httpErrorsTotal,
		mcpToolsTotal:           mcpToolsTotal,
		mcpToolCallsTotal:       mcpToolCallsTotal,
		mcpToolCallDuration:     mcpToolCallDuration,
		mcpResourcesTotal:       mcpResourcesTotal,
		mcpResourceReadsTotal:   mcpResourceReadsTotal,
		mcpResourceReadDuration: mcpResourceReadDuration,
		mcpErrorsTotal:          mcpErrorsTotal,
		serverUptime:            serverUptime,
		serverHealthChecks:      serverHealthChecks,
	}
}

// RecordHTTPRequest records an HTTP request with its duration and status
func (m *Metrics) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	m.httpRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.Int("status_code", statusCode),
		),
	)
	m.httpRequestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.Int("status_code", statusCode),
		),
	)

	if statusCode >= 400 {
		m.httpErrorsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", method),
				attribute.String("path", path),
				attribute.Int("status_code", statusCode),
			),
		)
	}
}

// RecordActiveConnections records the number of active connections
func (m *Metrics) RecordActiveConnections(ctx context.Context, count int64) {
	m.httpActiveConnections.Add(ctx, count)
}

// RecordMCPToolCall records an MCP tool call with its duration and success status
func (m *Metrics) RecordMCPToolCall(ctx context.Context, toolName string, success bool, duration time.Duration) {
	m.mcpToolCallsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.Bool("success", success),
		),
	)
	m.mcpToolCallDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.Bool("success", success),
		),
	)

	if !success {
		m.mcpErrorsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("tool_name", toolName),
				attribute.Bool("success", success),
			),
		)
	}
}

// RecordMCPResourceRead records an MCP resource read with its duration and success status
func (m *Metrics) RecordMCPResourceRead(ctx context.Context, resourceURI string, success bool, duration time.Duration) {
	m.mcpResourceReadsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("resource_uri", resourceURI),
			attribute.Bool("success", success),
		),
	)
	m.mcpResourceReadDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("resource_uri", resourceURI),
			attribute.Bool("success", success),
		),
	)

	if !success {
		m.mcpErrorsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("resource_uri", resourceURI),
				attribute.Bool("success", success),
			),
		)
	}
}

// RecordToolRegistration records the registration of an MCP tool
func (m *Metrics) RecordToolRegistration(ctx context.Context, toolName string) {
	m.mcpToolsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
		),
	)
}

// RecordResourceRegistration records the registration of an MCP resource
func (m *Metrics) RecordResourceRegistration(ctx context.Context, resourceURI string) {
	m.mcpResourcesTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("resource_uri", resourceURI),
		),
	)
}

// RecordHealthCheck records a health check
func (m *Metrics) RecordHealthCheck(ctx context.Context, healthy bool) {
	m.serverHealthChecks.Add(ctx, 1,
		metric.WithAttributes(
			attribute.Bool("healthy", healthy),
		),
	)
}

// RecordServerUptime records server uptime
func (m *Metrics) RecordServerUptime(ctx context.Context, uptime time.Duration) {
	m.serverUptime.Record(ctx, uptime.Seconds())
}
