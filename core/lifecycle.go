package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthState represents the health of a component.
type HealthState string

const (
	// HealthHealthy indicates the component is operating normally.
	HealthHealthy HealthState = "healthy"

	// HealthDegraded indicates the component is working but with reduced
	// capability.
	HealthDegraded HealthState = "degraded"

	// HealthUnhealthy indicates the component is not operational.
	HealthUnhealthy HealthState = "unhealthy"
)

// HealthStatus reports the current health of a component.
type HealthStatus struct {
	// Status is the overall health state.
	Status HealthState

	// Message provides additional context about the health state.
	Message string

	// Timestamp is when the health status was last checked.
	Timestamp time.Time
}

// Lifecycle is the interface for components that require explicit
// start/stop management. Components are started in registration order
// and stopped in reverse order.
type Lifecycle interface {
	// Start initialises the component. It should block until the component
	// is ready to serve, or return an error.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the component. It should drain in-flight
	// work before returning.
	Stop(ctx context.Context) error

	// Health returns the current health status of the component.
	Health() HealthStatus
}

// App manages a set of Lifecycle components, starting them in registration
// order and shutting them down in reverse order.
type App struct {
	mu         sync.Mutex
	components []Lifecycle
	running    bool
}

// NewApp creates a new App.
func NewApp() *App {
	return &App{}
}

// Register adds one or more Lifecycle components to the app. Components
// will be started in the order they are registered.
func (a *App) Register(components ...Lifecycle) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.components = append(a.components, components...)
}

// Start starts all registered components in registration order. If any
// component fails to start, previously started components are stopped in
// reverse order and the first error is returned.
func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, c := range a.components {
		if err := c.Start(ctx); err != nil {
			// Roll back: stop previously started components in reverse.
			for j := i - 1; j >= 0; j-- {
				_ = a.components[j].Stop(ctx)
			}
			return fmt.Errorf("starting component %d: %w", i, err)
		}
	}
	a.running = true
	return nil
}

// Shutdown stops all components in reverse registration order. It collects
// all errors and returns a combined error if any component fails to stop.
func (a *App) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	var errs []error
	for i := len(a.components) - 1; i >= 0; i-- {
		if err := a.components[i].Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stopping component %d: %w", i, err))
		}
	}
	a.running = false

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// HealthCheck returns the health status of all registered components.
func (a *App) HealthCheck() []HealthStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	statuses := make([]HealthStatus, len(a.components))
	for i, c := range a.components {
		statuses[i] = c.Health()
	}
	return statuses
}
