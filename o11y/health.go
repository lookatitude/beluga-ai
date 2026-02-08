package o11y

import (
	"context"
	"sync"
	"time"
)

// HealthStatus represents the operational state of a component.
type HealthStatus string

const (
	// Healthy indicates the component is fully operational.
	Healthy HealthStatus = "healthy"

	// Degraded indicates the component is operational but impaired.
	Degraded HealthStatus = "degraded"

	// Unhealthy indicates the component has failed.
	Unhealthy HealthStatus = "unhealthy"
)

// HealthResult contains the outcome of a single health check.
type HealthResult struct {
	// Status is the component's operational state.
	Status HealthStatus

	// Message provides human-readable detail about the health check outcome.
	Message string

	// Component identifies which component was checked.
	Component string

	// Timestamp is when the health check was performed.
	Timestamp time.Time
}

// HealthChecker is implemented by any component that can report its health.
type HealthChecker interface {
	// HealthCheck performs a health probe and returns the result.
	HealthCheck(ctx context.Context) HealthResult
}

// HealthRegistry aggregates named health checkers and runs them concurrently.
type HealthRegistry struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
}

// NewHealthRegistry creates an empty HealthRegistry.
func NewHealthRegistry() *HealthRegistry {
	return &HealthRegistry{
		checkers: make(map[string]HealthChecker),
	}
}

// Register adds a named health checker. If a checker with the same name
// already exists, it is replaced.
func (r *HealthRegistry) Register(name string, checker HealthChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[name] = checker
}

// CheckAll runs every registered health checker concurrently and returns
// all results. Each result's Component field is set to the registered name.
func (r *HealthRegistry) CheckAll(ctx context.Context) []HealthResult {
	r.mu.RLock()
	checkers := make(map[string]HealthChecker, len(r.checkers))
	for k, v := range r.checkers {
		checkers[k] = v
	}
	r.mu.RUnlock()

	if len(checkers) == 0 {
		return nil
	}

	type namedResult struct {
		name   string
		result HealthResult
	}

	ch := make(chan namedResult, len(checkers))
	for name, checker := range checkers {
		go func(n string, c HealthChecker) {
			result := c.HealthCheck(ctx)
			result.Component = n
			if result.Timestamp.IsZero() {
				result.Timestamp = time.Now()
			}
			ch <- namedResult{name: n, result: result}
		}(name, checker)
	}

	results := make([]HealthResult, 0, len(checkers))
	for range len(checkers) {
		nr := <-ch
		results = append(results, nr.result)
	}
	return results
}

// HealthCheckerFunc adapts a plain function to the HealthChecker interface.
type HealthCheckerFunc func(ctx context.Context) HealthResult

// HealthCheck calls the underlying function.
func (f HealthCheckerFunc) HealthCheck(ctx context.Context) HealthResult {
	return f(ctx)
}
