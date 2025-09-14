package core

import (
	"fmt"
	"reflect"
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Container represents a dependency injection container
type Container interface {
	// Register registers a factory function for a type
	Register(factoryFunc interface{}) error

	// Resolve resolves a dependency by type
	Resolve(target interface{}) error

	// MustResolve resolves a dependency or panics
	MustResolve(target interface{})

	// Has checks if a type is registered
	Has(typ reflect.Type) bool

	// Clear removes all registered dependencies
	Clear()
}

// containerImpl is the default implementation of Container
type containerImpl struct {
	mu        sync.RWMutex
	factories map[reflect.Type]interface{}
	instances map[reflect.Type]interface{}
}

// NewContainer creates a new dependency injection container
func NewContainer() Container {
	return &containerImpl{
		factories: make(map[reflect.Type]interface{}),
		instances: make(map[reflect.Type]interface{}),
	}
}

// Register registers a factory function for a type
func (c *containerImpl) Register(factoryFunc interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	factoryType := reflect.TypeOf(factoryFunc)
	if factoryType.Kind() != reflect.Func {
		return fmt.Errorf("factory must be a function, got %s", factoryType.Kind())
	}

	if factoryType.NumOut() == 0 {
		return fmt.Errorf("factory function must return at least one value")
	}

	returnType := factoryType.Out(0)
	c.factories[returnType] = factoryFunc

	return nil
}

// Resolve resolves a dependency by type
func (c *containerImpl) Resolve(target interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer, got %s", targetValue.Kind())
	}

	targetType := targetValue.Elem().Type()

	// Check if we have a cached instance
	if instance, exists := c.instances[targetType]; exists {
		targetValue.Elem().Set(reflect.ValueOf(instance))
		return nil
	}

	// Check if we have a factory
	factory, exists := c.factories[targetType]
	if !exists {
		return fmt.Errorf("no factory registered for type %s", targetType)
	}

	factoryValue := reflect.ValueOf(factory)
	factoryType := factoryValue.Type()

	// Prepare arguments for the factory function
	numIn := factoryType.NumIn()
	args := make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		argType := factoryType.In(i)

		// Try to resolve the argument from the container
		if argInstance, exists := c.instances[argType]; exists {
			args[i] = reflect.ValueOf(argInstance)
		} else {
			// Try to recursively resolve the dependency
			argValue := reflect.New(argType)
			if err := c.resolveRecursive(argValue.Interface()); err != nil {
				return fmt.Errorf("failed to resolve dependency %s: %w", argType, err)
			}
			args[i] = argValue.Elem()
		}
	}

	// Call the factory function
	results := factoryValue.Call(args)

	// Check for errors in the results
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) && !lastResult.IsNil() {
			return lastResult.Interface().(error)
		}
	}

	instance := results[0].Interface()
	c.instances[targetType] = instance
	targetValue.Elem().Set(results[0])

	return nil
}

// resolveRecursive is a helper method for recursive resolution
func (c *containerImpl) resolveRecursive(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetType := targetValue.Elem().Type()

	if instance, exists := c.instances[targetType]; exists {
		targetValue.Elem().Set(reflect.ValueOf(instance))
		return nil
	}

	if factory, exists := c.factories[targetType]; exists {
		factoryValue := reflect.ValueOf(factory)
		factoryType := factoryValue.Type()

		args := make([]reflect.Value, factoryType.NumIn())
		for i := 0; i < factoryType.NumIn(); i++ {
			argType := factoryType.In(i)
			argValue := reflect.New(argType)
			if err := c.resolveRecursive(argValue.Interface()); err != nil {
				return err
			}
			args[i] = argValue.Elem()
		}

		results := factoryValue.Call(args)
		if len(results) > 1 && results[len(results)-1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) && !results[len(results)-1].IsNil() {
			return results[len(results)-1].Interface().(error)
		}

		instance := results[0].Interface()
		c.instances[targetType] = instance
		targetValue.Elem().Set(results[0])
	}

	return nil
}

// MustResolve resolves a dependency or panics
func (c *containerImpl) MustResolve(target interface{}) {
	if err := c.Resolve(target); err != nil {
		panic(fmt.Sprintf("failed to resolve dependency: %v", err))
	}
}

// Has checks if a type is registered
func (c *containerImpl) Has(typ reflect.Type) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, hasFactory := c.factories[typ]
	_, hasInstance := c.instances[typ]
	return hasFactory || hasInstance
}

// Clear removes all registered dependencies
func (c *containerImpl) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.factories = make(map[reflect.Type]interface{})
	c.instances = make(map[reflect.Type]interface{})
}

// Singleton registers a singleton instance
func (c *containerImpl) Singleton(instance interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	typ := reflect.TypeOf(instance)
	c.instances[typ] = instance
}

// DIOption represents a functional option for DI configuration
type DIOption func(*optionConfig)

type optionConfig struct {
	container Container
}

// WithContainer sets the DI container
func WithContainer(container Container) DIOption {
	return func(config *optionConfig) {
		config.container = container
	}
}

// DefaultContainer returns the default container option
func DefaultContainer() DIOption {
	return WithContainer(NewContainer())
}

// Builder provides a fluent interface for building objects with DI
type Builder struct {
	container Container
}

// NewBuilder creates a new builder with the given container
func NewBuilder(container Container) *Builder {
	return &Builder{container: container}
}

// Build builds an object using the registered factory
func (b *Builder) Build(target interface{}) error {
	return b.container.Resolve(target)
}

// Register registers a factory function
func (b *Builder) Register(factoryFunc interface{}) error {
	return b.container.Register(factoryFunc)
}

// Singleton registers a singleton instance
func (b *Builder) Singleton(instance interface{}) {
	if c, ok := b.container.(*containerImpl); ok {
		c.Singleton(instance)
	}
}

// Monitoring interfaces for dependency injection
type (
	// Logger interface for structured logging
	Logger interface {
		Debug(msg string, args ...interface{})
		Info(msg string, args ...interface{})
		Warn(msg string, args ...interface{})
		Error(msg string, args ...interface{})
		With(args ...interface{}) Logger
	}

	// TracerProvider interface for distributed tracing
	TracerProvider interface {
		Tracer(name string, opts ...trace.TracerOption) trace.Tracer
	}

	// MeterProvider interface for metrics collection
	MeterProvider interface {
		Meter(name string, opts ...metric.MeterOption) metric.Meter
	}
)

// RegisterLogger registers a logger factory function
func (b *Builder) RegisterLogger(factory func() Logger) error {
	return b.container.Register(factory)
}

// RegisterTracerProvider registers a tracer provider factory function
func (b *Builder) RegisterTracerProvider(factory func() TracerProvider) error {
	return b.container.Register(factory)
}

// RegisterMeterProvider registers a meter provider factory function
func (b *Builder) RegisterMeterProvider(factory func() MeterProvider) error {
	return b.container.Register(factory)
}

// RegisterMetrics registers a metrics factory function
func (b *Builder) RegisterMetrics(factory func() (*Metrics, error)) error {
	return b.container.Register(factory)
}

// RegisterNoOpLogger registers a no-op logger (useful for testing)
func (b *Builder) RegisterNoOpLogger() error {
	return b.RegisterLogger(func() Logger { return &noOpLogger{} })
}

// RegisterNoOpTracerProvider registers a no-op tracer provider
func (b *Builder) RegisterNoOpTracerProvider() error {
	return b.RegisterTracerProvider(func() TracerProvider { return trace.NewNoopTracerProvider() })
}

// RegisterNoOpMetrics registers no-op metrics
func (b *Builder) RegisterNoOpMetrics() error {
	return b.RegisterMetrics(func() (*Metrics, error) { return NoOpMetrics(), nil })
}

// noOpLogger implements Logger with no-op behavior
type noOpLogger struct{}

func (n *noOpLogger) Debug(msg string, args ...interface{}) {}
func (n *noOpLogger) Info(msg string, args ...interface{})  {}
func (n *noOpLogger) Warn(msg string, args ...interface{})  {}
func (n *noOpLogger) Error(msg string, args ...interface{}) {}
func (n *noOpLogger) With(args ...interface{}) Logger       { return n }
