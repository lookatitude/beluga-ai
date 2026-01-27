// Package monitoring provides provider registration system
package monitoring

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// Provider represents a monitoring backend provider.
type Provider interface {
	Name() string
	Initialize(config map[string]any) error
	Shutdown() error
}

// LoggerProvider provides logging functionality.
type LoggerProvider interface {
	Provider
	CreateLogger(name string, config map[string]any) (iface.Logger, error)
}

// TracerProvider provides tracing functionality.
type TracerProvider interface {
	Provider
	CreateTracer(serviceName string, config map[string]any) (iface.Tracer, error)
}

// MetricsProvider provides metrics functionality.
type MetricsProvider interface {
	Provider
	CreateMetricsCollector(config map[string]any) (iface.MetricsCollector, error)
}

// ProviderRegistry manages monitoring providers.
type ProviderRegistry struct {
	loggers map[string]LoggerProvider
	tracers map[string]TracerProvider
	metrics map[string]MetricsProvider
	current map[string]string
	mu      sync.RWMutex
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		loggers: make(map[string]LoggerProvider),
		tracers: make(map[string]TracerProvider),
		metrics: make(map[string]MetricsProvider),
		current: make(map[string]string),
	}
}

// RegisterLoggerProvider registers a logger provider.
func (pr *ProviderRegistry) RegisterLoggerProvider(provider LoggerProvider) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	name := provider.Name()
	if _, exists := pr.loggers[name]; exists {
		return fmt.Errorf("logger provider %s already registered", name)
	}

	pr.loggers[name] = provider
	return nil
}

// RegisterTracerProvider registers a tracer provider.
func (pr *ProviderRegistry) RegisterTracerProvider(provider TracerProvider) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	name := provider.Name()
	if _, exists := pr.tracers[name]; exists {
		return fmt.Errorf("tracer provider %s already registered", name)
	}

	pr.tracers[name] = provider
	return nil
}

// RegisterMetricsProvider registers a metrics provider.
func (pr *ProviderRegistry) RegisterMetricsProvider(provider MetricsProvider) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	name := provider.Name()
	if _, exists := pr.metrics[name]; exists {
		return fmt.Errorf("metrics provider %s already registered", name)
	}

	pr.metrics[name] = provider
	return nil
}

// SetCurrentLogger sets the current logger provider.
func (pr *ProviderRegistry) SetCurrentLogger(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.loggers[name]; !exists {
		return fmt.Errorf("logger provider %s not found", name)
	}

	pr.current["logger"] = name
	return nil
}

// SetCurrentTracer sets the current tracer provider.
func (pr *ProviderRegistry) SetCurrentTracer(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.tracers[name]; !exists {
		return fmt.Errorf("tracer provider %s not found", name)
	}

	pr.current["tracer"] = name
	return nil
}

// SetCurrentMetrics sets the current metrics provider.
func (pr *ProviderRegistry) SetCurrentMetrics(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.metrics[name]; !exists {
		return fmt.Errorf("metrics provider %s not found", name)
	}

	pr.current["metrics"] = name
	return nil
}

// CreateLogger creates a logger using the current provider.
func (pr *ProviderRegistry) CreateLogger(name string, config map[string]any) (iface.Logger, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	providerName, exists := pr.current["logger"]
	if !exists {
		return nil, errors.New("no current logger provider set")
	}

	provider, exists := pr.loggers[providerName]
	if !exists {
		return nil, fmt.Errorf("logger provider %s not found", providerName)
	}

	return provider.CreateLogger(name, config)
}

// CreateTracer creates a tracer using the current provider.
func (pr *ProviderRegistry) CreateTracer(serviceName string, config map[string]any) (iface.Tracer, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	providerName, exists := pr.current["tracer"]
	if !exists {
		return nil, errors.New("no current tracer provider set")
	}

	provider, exists := pr.tracers[providerName]
	if !exists {
		return nil, fmt.Errorf("tracer provider %s not found", providerName)
	}

	return provider.CreateTracer(serviceName, config)
}

// CreateMetricsCollector creates a metrics collector using the current provider.
func (pr *ProviderRegistry) CreateMetricsCollector(config map[string]any) (iface.MetricsCollector, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	providerName, exists := pr.current["metrics"]
	if !exists {
		return nil, errors.New("no current metrics provider set")
	}

	provider, exists := pr.metrics[providerName]
	if !exists {
		return nil, fmt.Errorf("metrics provider %s not found", providerName)
	}

	return provider.CreateMetricsCollector(config)
}

// GetAvailableProviders returns all available providers.
func (pr *ProviderRegistry) GetAvailableProviders() map[string][]string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	providers := make(map[string][]string)
	for name := range pr.loggers {
		providers["loggers"] = append(providers["loggers"], name)
	}
	for name := range pr.tracers {
		providers["tracers"] = append(providers["tracers"], name)
	}
	for name := range pr.metrics {
		providers["metrics"] = append(providers["metrics"], name)
	}

	return providers
}

// Shutdown shuts down all registered providers.
func (pr *ProviderRegistry) Shutdown() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	var errors []error

	// Shutdown all logger providers
	for _, provider := range pr.loggers {
		if err := provider.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown logger provider %s: %w", provider.Name(), err))
		}
	}

	// Shutdown all tracer providers
	for _, provider := range pr.tracers {
		if err := provider.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown tracer provider %s: %w", provider.Name(), err))
		}
	}

	// Shutdown all metrics providers
	for _, provider := range pr.metrics {
		if err := provider.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown metrics provider %s: %w", provider.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("provider shutdown errors: %v", errors)
	}

	return nil
}

// Global registry instance.
var globalRegistry = NewProviderRegistry()

// GetGlobalRegistry returns the global provider registry.
func GetGlobalRegistry() *ProviderRegistry {
	return globalRegistry
}

// Example usage:
//
//	// Register providers
//	registry := monitoring.GetGlobalRegistry()
//	registry.RegisterLoggerProvider(&MyLoggerProvider{})
//	registry.RegisterTracerProvider(&MyTracerProvider{})
//	registry.RegisterMetricsProvider(&MyMetricsProvider{})
//
//	// Set current providers
//	registry.SetCurrentLogger("mylogger")
//	registry.SetCurrentTracer("mytracer")
//	registry.SetCurrentMetrics("mymetrics")
//
//	// Use in monitor creation
//	logger, _ := registry.CreateLogger("my-service", config)
//	tracer, _ := registry.CreateTracer("my-service", config)
//	metrics, _ := registry.CreateMetricsCollector(config)
