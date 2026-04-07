package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// checkFunc is the signature for a named readiness check.
type checkFunc func(ctx context.Context) error

// checkEntry pairs a name with its associated check function.
type checkEntry struct {
	name  string
	check checkFunc
}

// checkResult is the outcome of a single named check.
// The Error field is intentionally omitted from the JSON response to prevent
// information disclosure; error details should be logged internally instead.
type checkResult struct {
	Status string `json:"status"`
}

// healthResponse is the JSON body returned by the health endpoints.
type healthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]checkResult `json:"checks,omitempty"`
}

// HealthEndpoint exposes HTTP handlers for liveness and readiness checks.
//
// Health endpoints are intentionally unauthenticated to support orchestrator
// probes (Kubernetes liveness/readiness). Restrict network access via
// infrastructure controls (NetworkPolicy, firewall rules) rather than
// application-layer authentication.
//
// Use [NewHealthEndpoint] to create an instance, [HealthEndpoint.AddCheck] to
// register named readiness checks, and [HealthEndpoint.Healthz] /
// [HealthEndpoint.Readyz] to obtain the respective http.HandlerFunc values.
type HealthEndpoint struct {
	mu     sync.RWMutex
	checks []checkEntry
}

// NewHealthEndpoint creates and returns a new [HealthEndpoint] with no checks
// registered.
func NewHealthEndpoint() *HealthEndpoint {
	return &HealthEndpoint{}
}

// AddCheck registers a named readiness check. The check function receives a
// context that carries the deadline of the incoming HTTP request. Checks are
// run in registration order by [HealthEndpoint.Readyz]. Calling AddCheck after
// the server has started is safe.
func (h *HealthEndpoint) AddCheck(name string, check func(ctx context.Context) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, checkEntry{name: name, check: check})
}

// Healthz returns an [http.HandlerFunc] that always responds 200 OK with a
// JSON body {"status":"ok"}. It serves as a liveness probe — confirming the
// process is running and able to handle requests.
// Only GET and HEAD requests are accepted; all other methods receive 405.
func (h *HealthEndpoint) Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
	}
}

// Readyz returns an [http.HandlerFunc] that runs all registered checks and
// responds:
//   - 200 OK when every check passes, with body {"status":"ok","checks":{...}}
//   - 503 Service Unavailable when one or more checks fail, with the failing
//     check names listed in the body (error details are intentionally suppressed
//     to prevent information disclosure to unauthenticated callers).
//
// Only GET and HEAD requests are accepted; all other methods receive 405.
// Each check is given a 5-second deadline derived from the request context.
func (h *HealthEndpoint) Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		h.mu.RLock()
		entries := make([]checkEntry, len(h.checks))
		copy(entries, h.checks)
		h.mu.RUnlock()

		results := make(map[string]checkResult, len(entries))
		allOK := true

		for _, e := range entries {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			err := e.check(ctx)
			cancel()

			if err != nil {
				// Error details are intentionally suppressed here to prevent
				// information disclosure to unauthenticated orchestrator probes.
				// Log the full error internally via your observability stack.
				results[e.name] = checkResult{Status: "unhealthy"}
				allOK = false
			} else {
				results[e.name] = checkResult{Status: "ok"}
			}
		}

		status := http.StatusOK
		statusStr := "ok"
		if !allOK {
			status = http.StatusServiceUnavailable
			statusStr = "fail"
		}

		resp := healthResponse{
			Status: statusStr,
		}
		if len(results) > 0 {
			resp.Checks = results
		}
		writeJSON(w, status, resp)
	}
}

// writeJSON serialises v as JSON and writes it to w with the given status code.
// Errors during serialisation are handled internally; if marshalling fails the
// handler writes a plain-text 500 response.
func writeJSON(w http.ResponseWriter, code int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(data)
}
