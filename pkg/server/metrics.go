// Package server provides metrics definitions for server implementations.
package server

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics contains all metrics for the server package.
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal     metric.Int64Counter
	httpRequestDuration   metric.Float64Histogram
	httpActiveConnections metric.Int64UpDownCounter
	httpErrorsTotal       metric.Int64Counter

	// MCP metrics
	mcpToolsTotal           metric.Int64Counter
	mcpToolCallsTotal       metric.Int64Counter
	mcpToolCallDuration     metric.Float64Histogram
	mcpResourcesTotal       metric.Int64Counter
	mcpResourceReadsTotal   metric.Int64Counter
	mcpResourceReadDuration metric.Float64Histogram
	mcpErrorsTotal          metric.Int64Counter

	// Server metrics
	serverUptime       metric.Float64Histogram
	serverHealthChecks metric.Int64Counter
}

// NewMetrics creates a new Metrics instance with the given meter.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	// Initialize HTTP metrics
	m.httpRequestsTotal, err = meter.Int64Counter(
		"server_http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.httpRequestDuration, err = meter.Float64Histogram(
		"server_http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.httpActiveConnections, err = meter.Int64UpDownCounter(
		"server_http_active_connections",
		metric.WithDescription("Number of active HTTP connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.httpErrorsTotal, err = meter.Int64Counter(
		"server_http_errors_total",
		metric.WithDescription("Total number of HTTP errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize MCP metrics
	m.mcpToolsTotal, err = meter.Int64Counter(
		"server_mcp_tools_total",
		metric.WithDescription("Total number of registered MCP tools"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.mcpToolCallsTotal, err = meter.Int64Counter(
		"server_mcp_tool_calls_total",
		metric.WithDescription("Total number of MCP tool calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.mcpToolCallDuration, err = meter.Float64Histogram(
		"server_mcp_tool_call_duration_seconds",
		metric.WithDescription("Duration of MCP tool calls in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.mcpResourcesTotal, err = meter.Int64Counter(
		"server_mcp_resources_total",
		metric.WithDescription("Total number of registered MCP resources"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.mcpResourceReadsTotal, err = meter.Int64Counter(
		"server_mcp_resource_reads_total",
		metric.WithDescription("Total number of MCP resource reads"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.mcpResourceReadDuration, err = meter.Float64Histogram(
		"server_mcp_resource_read_duration_seconds",
		metric.WithDescription("Duration of MCP resource reads in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.mcpErrorsTotal, err = meter.Int64Counter(
		"server_mcp_errors_total",
		metric.WithDescription("Total number of MCP errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize server metrics
	m.serverUptime, err = meter.Float64Histogram(
		"server_uptime_seconds",
		metric.WithDescription("Server uptime in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.serverHealthChecks, err = meter.Int64Counter(
		"server_health_checks_total",
		metric.WithDescription("Total number of health checks"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// RecordHTTPRequest records an HTTP request with its duration and status.
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

// RecordActiveConnections records the number of active connections.
func (m *Metrics) RecordActiveConnections(ctx context.Context, count int64) {
	m.httpActiveConnections.Add(ctx, count)
}

// RecordMCPToolCall records an MCP tool call with its duration and success status.
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

// RecordMCPResourceRead records an MCP resource read with its duration and success status.
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

// RecordToolRegistration records the registration of an MCP tool.
func (m *Metrics) RecordToolRegistration(ctx context.Context, toolName string) {
	m.mcpToolsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
		),
	)
}

// RecordResourceRegistration records the registration of an MCP resource.
func (m *Metrics) RecordResourceRegistration(ctx context.Context, resourceURI string) {
	m.mcpResourcesTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("resource_uri", resourceURI),
		),
	)
}

// RecordHealthCheck records a health check.
func (m *Metrics) RecordHealthCheck(ctx context.Context, healthy bool) {
	m.serverHealthChecks.Add(ctx, 1,
		metric.WithAttributes(
			attribute.Bool("healthy", healthy),
		),
	)
}

// RecordServerUptime records server uptime.
func (m *Metrics) RecordServerUptime(ctx context.Context, uptime time.Duration) {
	m.serverUptime.Record(ctx, uptime.Seconds())
}
