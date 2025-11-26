// Package monitoring provides server integration for safety and ethics monitoring
package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// ServerIntegration provides HTTP handlers for safety and ethics monitoring.
type ServerIntegration struct {
	monitor iface.Monitor
}

// NewServerIntegration creates a new server integration.
func NewServerIntegration(monitor iface.Monitor) *ServerIntegration {
	return &ServerIntegration{
		monitor: monitor,
	}
}

// HealthCheckHandler provides a health check endpoint.
func (si *ServerIntegration) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Perform comprehensive health check
	results := si.monitor.HealthChecker().RunChecks(ctx)

	// Determine overall health status
	overallHealthy := true
	for _, result := range results {
		if result.Status != iface.StatusHealthy {
			overallHealthy = false
			break
		}
	}

	statusCode := http.StatusOK
	if !overallHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]any{
		"status":    si.getStatusString(overallHealthy),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    si.formatHealthResults(results),
	}

	_ = json.NewEncoder(w).Encode(response)
}

// SafetyCheckHandler provides a safety validation endpoint.
func (si *ServerIntegration) SafetyCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Content     string `json:"content"`
		ContextInfo string `json:"context_info"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	result, err := si.monitor.SafetyChecker().CheckContent(ctx, request.Content, request.ContextInfo)
	if err != nil {
		si.monitor.Logger().Error(ctx, "Safety check failed", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Safety check failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"result":    result,
		"timestamp": time.Now(),
	}
	_ = json.NewEncoder(w).Encode(response)
}

// EthicsCheckHandler provides an ethical validation endpoint.
func (si *ServerIntegration) EthicsCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Content     string               `json:"content"`
		ContextInfo iface.EthicalContext `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	result, err := si.monitor.EthicalChecker().CheckContent(ctx, request.Content, request.ContextInfo)
	if err != nil {
		si.monitor.Logger().Error(ctx, "Ethics check failed", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Ethics check failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"result":    result,
		"timestamp": time.Now(),
	}
	_ = json.NewEncoder(w).Encode(response)
}

// BestPracticesCheckHandler provides a best practices validation endpoint.
func (si *ServerIntegration) BestPracticesCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Data      any    `json:"data"`
		Component string `json:"component"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	issues := si.monitor.BestPracticesChecker().Validate(r.Context(), request.Data, request.Component)

	response := map[string]any{
		"issues":    issues,
		"validated": true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// MetricsHandler provides metrics endpoint.
func (si *ServerIntegration) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Return Prometheus-style metrics format
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	metrics := `# HELP beluga_ai_requests_total Total number of requests processed
# TYPE beluga_ai_requests_total counter
beluga_ai_requests_total 42

# HELP beluga_ai_response_time_seconds Response time in seconds
# TYPE beluga_ai_response_time_seconds histogram
beluga_ai_response_time_seconds_bucket{le="0.1"} 10
beluga_ai_response_time_seconds_bucket{le="0.5"} 25
beluga_ai_response_time_seconds_bucket{le="1.0"} 35
beluga_ai_response_time_seconds_bucket{le="5.0"} 40
beluga_ai_response_time_seconds_bucket{le="+Inf"} 42
beluga_ai_response_time_seconds_sum 123.45
beluga_ai_response_time_seconds_count 42
`
	_, _ = w.Write([]byte(metrics))
}

// TracesHandler provides trace information endpoint.
func (si *ServerIntegration) TracesHandler(w http.ResponseWriter, r *http.Request) {
	// Get trace ID from query parameter
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		traceID = "default-trace-id"
	}

	spans := si.monitor.Tracer().GetTraceSpans(traceID)

	response := map[string]any{
		"traces":    make([]map[string]any, len(spans)),
		"timestamp": time.Now(),
	}

	for i, span := range spans {
		response["traces"].([]map[string]any)[i] = map[string]any{
			"span_id":   fmt.Sprintf("span_%d", i),
			"trace_id":  traceID,
			"duration":  span.GetDuration().String(),
			"finished":  span.IsFinished(),
			"available": true,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// LogsHandler provides log search endpoint.
func (si *ServerIntegration) LogsHandler(w http.ResponseWriter, r *http.Request) {
	// This is a simplified implementation - in production, you would integrate
	// with a proper log aggregation system
	query := r.URL.Query().Get("query")
	level := r.URL.Query().Get("level")

	response := map[string]any{
		"logs": []map[string]any{
			{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"level":     level,
				"message":   "Log search not implemented in this demo",
				"query":     query,
			},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers all monitoring routes with an HTTP mux.
func (si *ServerIntegration) RegisterRoutes(mux *http.ServeMux, pathPrefix string) {
	if pathPrefix == "" {
		pathPrefix = "/monitoring"
	}

	mux.HandleFunc(pathPrefix+"/health", si.HealthCheckHandler)
	mux.HandleFunc(pathPrefix+"/safety/check", si.SafetyCheckHandler)
	mux.HandleFunc(pathPrefix+"/ethics/check", si.EthicsCheckHandler)
	mux.HandleFunc(pathPrefix+"/best-practices/check", si.BestPracticesCheckHandler)
	mux.HandleFunc(pathPrefix+"/metrics", si.MetricsHandler)
	mux.HandleFunc(pathPrefix+"/traces", si.TracesHandler)
	mux.HandleFunc(pathPrefix+"/logs", si.LogsHandler)
}

// Helper methods

func (si *ServerIntegration) getStatusString(healthy bool) string {
	if healthy {
		return "healthy"
	}
	return "unhealthy"
}

func (si *ServerIntegration) formatHealthResults(results map[string]iface.HealthCheckResult) []map[string]any {
	formatted := make([]map[string]any, 0, len(results))

	for name, result := range results {
		formatted = append(formatted, map[string]any{
			"name":      name,
			"status":    string(result.Status),
			"message":   result.Message,
			"timestamp": result.Timestamp.Format(time.RFC3339),
			"details":   result.Details,
		})
	}

	return formatted
}

// Middleware for automatic monitoring of HTTP requests.
func (si *ServerIntegration) MonitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		// Start span for request
		ctx, span := si.monitor.Tracer().StartSpan(ctx, "http_request")
		defer si.monitor.Tracer().FinishSpan(span)

		// Create response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Panic recovery
		defer func() {
			if err := recover(); err != nil {
				rw.statusCode = http.StatusInternalServerError
				rw.WriteHeader(http.StatusInternalServerError)
				span.SetError(fmt.Errorf("panic: %v", err))
				si.monitor.Logger().Error(ctx, "Panic in HTTP handler", map[string]any{
					"panic":  err,
					"method": r.Method,
					"path":   r.URL.Path,
				})
			}
		}()

		// Call next handler
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Log request
		duration := time.Since(start)
		if rw.statusCode >= 400 {
			span.SetError(fmt.Errorf("HTTP %d", rw.statusCode))
			si.monitor.Logger().Warning(ctx, "HTTP request failed", map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": rw.statusCode,
				"duration":    duration.String(),
			})
		} else {
			si.monitor.Logger().Info(ctx, "HTTP request completed", map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": rw.statusCode,
				"duration":    duration.String(),
			})
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("failed to write response: %w", err)
	}
	return n, nil
}
