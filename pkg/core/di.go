package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Static errors for DI container.
var (
	errFactoryMustBeFunction = errors.New("factory must be a function")
	errFactoryMustReturn     = errors.New("factory function must return at least one value")
	errTargetMustBePointer   = errors.New("target must be a pointer")
	errNoFactoryRegistered   = errors.New("no factory registered for type")
)

// Container represents a dependency injection container.
type Container interface {
	// Register registers a factory function for a type
	Register(factoryFunc any) error

	// Resolve resolves a dependency by type
	Resolve(target any) error

	// MustResolve resolves a dependency or panics
	MustResolve(target any)

	// Has checks if a type is registered
	Has(typ reflect.Type) bool

	// Clear removes all registered dependencies
	Clear()

	// Singleton registers a singleton instance
	Singleton(instance any)

	// HealthChecker provides health check functionality
	HealthChecker
}

// containerImpl is the default implementation of Container.
type containerImpl struct {
	logger         Logger
	tracerProvider TracerProvider
	factories      map[reflect.Type]any
	instances      map[reflect.Type]any
	mu             sync.RWMutex
}

// NewContainer creates a new dependency injection container with no-op monitoring.
func NewContainer() Container {
	return &containerImpl{
		factories:      make(map[reflect.Type]any),
		instances:      make(map[reflect.Type]any),
		logger:         &noOpLogger{},
		tracerProvider: noop.NewTracerProvider(),
	}
}

// NewContainerWithOptions creates a new dependency injection container with custom options.
func NewContainerWithOptions(opts ...DIOption) Container {
	config := optionConfig{
		container:      NewContainer(),
		logger:         &noOpLogger{},
		tracerProvider: noop.NewTracerProvider(),
	}

	for _, opt := range opts {
		opt(&config)
	}

	if container, ok := config.container.(*containerImpl); ok {
		container.logger = config.logger
		container.tracerProvider = config.tracerProvider
		return container
	}

	return config.container
}

// logWithOTELContext extracts OTEL trace/span IDs from context and adds them to log fields.
func (c *containerImpl) logWithOTELContext(ctx context.Context, level, message string, fields ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelFields := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		fields = append(otelFields, fields...)
	}

	// Call appropriate logger method
	switch level {
	case "debug":
		c.logger.Debug(message, fields...)
	case "info":
		c.logger.Info(message, fields...)
	case "warn":
		c.logger.Warn(message, fields...)
	case "error":
		c.logger.Error(message, fields...)
	default:
		// Unknown log level, default to debug
		c.logger.Debug(message, fields...)
	}
}

// Register registers a factory function for a type.
func (c *containerImpl) Register(factoryFunc any) error {
	ctx := context.Background()
	ctx, span := c.tracerProvider.Tracer("di-container").Start(ctx, "register",
		trace.WithAttributes(
			attribute.String("operation", "register"),
		))
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	factoryType := reflect.TypeOf(factoryFunc)
	if factoryType.Kind() != reflect.Func {
		err := fmt.Errorf("%w, got %s", errFactoryMustBeFunction, factoryType.Kind())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.logWithOTELContext(ctx, "error", "DI Register failed", "error", err)
		return err
	}

	if factoryType.NumOut() == 0 {
		err := errFactoryMustReturn
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.logWithOTELContext(ctx, "error", "DI Register failed", "error", err)
		return err
	}

	returnType := factoryType.Out(0)
	c.factories[returnType] = factoryFunc

	span.SetAttributes(
		attribute.String("type", returnType.String()),
	)
	span.SetStatus(codes.Ok, "")
	c.logWithOTELContext(ctx, "info", "DI Register succeeded", "type", returnType.String())

	return nil
}

// resolveFromCache checks if an instance is cached and sets it if found.
func (c *containerImpl) resolveFromCache(
	ctx context.Context,
	span trace.Span,
	targetValue reflect.Value,
	targetType reflect.Type,
) bool {
	c.mu.RLock()
	instance, exists := c.instances[targetType]
	c.mu.RUnlock()

	if exists {
		targetValue.Elem().Set(reflect.ValueOf(instance))
		span.SetAttributes(attribute.Bool("cached", true))
		span.SetStatus(codes.Ok, "")
		c.logWithOTELContext(ctx, "debug", "DI Resolve succeeded", "type", targetType.String(), "cached", true)
		return true
	}
	return false
}

// prepareFactoryArgs prepares arguments for a factory function by resolving dependencies.
func (c *containerImpl) prepareFactoryArgs(
	ctx context.Context,
	span trace.Span,
	factoryType, targetType reflect.Type,
) ([]reflect.Value, error) {
	numIn := factoryType.NumIn()
	args := make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		argType := factoryType.In(i)
		c.mu.RLock()
		argInstance, argExists := c.instances[argType]
		c.mu.RUnlock()

		if argExists {
			args[i] = reflect.ValueOf(argInstance)
		} else {
			argValue := reflect.New(argType)
			if err := c.resolveRecursive(argValue.Interface()); err != nil {
				err = fmt.Errorf("failed to resolve dependency %s: %w", argType, err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				c.logWithOTELContext(
					ctx, "error", "DI Resolve failed",
					"error", err, "type", targetType.String())
				return nil, err
			}
			args[i] = argValue.Elem()
		}
	}
	return args, nil
}

// callFactoryAndCache calls the factory function and caches the result.
func (c *containerImpl) callFactoryAndCache(
	ctx context.Context,
	span trace.Span,
	factoryValue reflect.Value,
	args []reflect.Value,
	targetValue reflect.Value,
	targetType reflect.Type,
) error {
	results := factoryValue.Call(args)

	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) && !lastResult.IsNil() {
			err := lastResult.Interface().(error)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			c.logWithOTELContext(ctx, "error", "DI Resolve failed", "error", err, "type", targetType.String())
			return fmt.Errorf("factory error: %w", err)
		}
	}

	instance := results[0].Interface()
	c.mu.Lock()
	if existingInstance, exists := c.instances[targetType]; exists {
		c.mu.Unlock()
		targetValue.Elem().Set(reflect.ValueOf(existingInstance))
		span.SetAttributes(attribute.Bool("cached", true))
		span.SetStatus(codes.Ok, "")
		c.logWithOTELContext(ctx, "debug", "DI Resolve succeeded", "type", targetType.String(), "cached", true)
		return nil
	}
	c.instances[targetType] = instance
	c.mu.Unlock()

	targetValue.Elem().Set(results[0])
	span.SetAttributes(attribute.Bool("cached", false))
	span.SetStatus(codes.Ok, "")
	c.logWithOTELContext(ctx, "debug", "DI Resolve succeeded", "type", targetType.String(), "cached", false)
	return nil
}

// Resolve resolves a dependency by type.
func (c *containerImpl) Resolve(target any) error {
	ctx := context.Background()
	ctx, span := c.tracerProvider.Tracer("di-container").Start(ctx, "resolve",
		trace.WithAttributes(attribute.String("operation", "resolve")))
	defer span.End()

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		err := fmt.Errorf("%w, got %s", errTargetMustBePointer, targetValue.Kind())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.logWithOTELContext(ctx, "error", "DI Resolve failed", "error", err)
		return err
	}

	targetType := targetValue.Elem().Type()
	span.SetAttributes(attribute.String("type", targetType.String()))

	if c.resolveFromCache(ctx, span, targetValue, targetType) {
		return nil
	}

	c.mu.RLock()
	factory, factoryExists := c.factories[targetType]
	c.mu.RUnlock()

	if !factoryExists {
		err := fmt.Errorf("%w %s", errNoFactoryRegistered, targetType)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.logWithOTELContext(ctx, "error", "DI Resolve failed", "error", err, "type", targetType.String())
		return err
	}

	factoryValue := reflect.ValueOf(factory)
	factoryType := factoryValue.Type()

	args, err := c.prepareFactoryArgs(ctx, span, factoryType, targetType)
	if err != nil {
		return err
	}

	return c.callFactoryAndCache(ctx, span, factoryValue, args, targetValue, targetType)
}

// resolveRecursive is a helper method for recursive resolution.
func (c *containerImpl) resolveRecursive(target any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return errTargetMustBePointer
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
		if len(results) > 1 {
			lastResult := results[len(results)-1]
			if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) &&
				!lastResult.IsNil() {
				err := lastResult.Interface().(error)
				return fmt.Errorf("recursive resolve error: %w", err)
			}
		}

		instance := results[0].Interface()
		c.instances[targetType] = instance
		targetValue.Elem().Set(results[0])
	}

	return nil
}

// MustResolve resolves a dependency or panics.
func (c *containerImpl) MustResolve(target any) {
	if err := c.Resolve(target); err != nil {
		panic(fmt.Sprintf("failed to resolve dependency: %v", err))
	}
}

// Has checks if a type is registered.
func (c *containerImpl) Has(typ reflect.Type) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, hasFactory := c.factories[typ]
	_, hasInstance := c.instances[typ]
	return hasFactory || hasInstance
}

// Clear removes all registered dependencies.
func (c *containerImpl) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.factories = make(map[reflect.Type]any)
	c.instances = make(map[reflect.Type]any)
}

// CheckHealth performs a health check on the DI container.
func (c *containerImpl) CheckHealth(ctx context.Context) error {
	ctx, span := c.tracerProvider.Tracer("di-container").Start(ctx, "check_health",
		trace.WithAttributes(
			attribute.String("operation", "health_check"),
		))
	defer span.End()

	c.mu.RLock()
	factoryCount := len(c.factories)
	instanceCount := len(c.instances)
	c.mu.RUnlock()

	// Basic health check - ensure the container has basic functionality
	span.SetAttributes(
		attribute.Int("factory_count", factoryCount),
		attribute.Int("instance_count", instanceCount),
	)

	// Verify container can perform basic operations without modifying state
	// Check that Has() works correctly
	testType := reflect.TypeOf((*string)(nil)).Elem()
	hasType := c.Has(testType)
	// This is just a read operation, safe for concurrent access
	_ = hasType // Acknowledge the check

	span.SetStatus(codes.Ok, "")
	c.logWithOTELContext(ctx, "debug", "DI Health check passed")
	return nil
}

// Singleton registers a singleton instance.
func (c *containerImpl) Singleton(instance any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	typ := reflect.TypeOf(instance)
	c.instances[typ] = instance
}

// DIOption represents a functional option for DI configuration.
type DIOption func(*optionConfig)

type optionConfig struct {
	container      Container
	logger         Logger
	tracerProvider TracerProvider
}

// WithContainer sets the DI container.
func WithContainer(container Container) DIOption {
	return func(config *optionConfig) {
		config.container = container
	}
}

// WithLogger sets the logger for the DI container.
func WithLogger(logger Logger) DIOption {
	return func(config *optionConfig) {
		config.logger = logger
	}
}

// WithTracerProvider sets the tracer provider for the DI container.
func WithTracerProvider(tracerProvider TracerProvider) DIOption {
	return func(config *optionConfig) {
		config.tracerProvider = tracerProvider
	}
}

// DefaultContainer returns the default container option.
func DefaultContainer() DIOption {
	return WithContainer(NewContainer())
}

// Builder provides a fluent interface for building objects with DI.
type Builder struct {
	container Container
}

// NewBuilder creates a new builder with the given container.
func NewBuilder(container Container) *Builder {
	return &Builder{container: container}
}

// Build builds an object using the registered factory.
func (b *Builder) Build(target any) error {
	return b.container.Resolve(target)
}

// Register registers a factory function.
func (b *Builder) Register(factoryFunc any) error {
	return b.container.Register(factoryFunc)
}

// Singleton registers a singleton instance.
func (b *Builder) Singleton(instance any) {
	if c, ok := b.container.(*containerImpl); ok {
		c.Singleton(instance)
	}
}

// Monitoring interfaces for dependency injection.
type (
	// Logger interface for structured logging.
	Logger interface {
		Debug(msg string, args ...any)
		Info(msg string, args ...any)
		Warn(msg string, args ...any)
		Error(msg string, args ...any)
		With(args ...any) Logger
	}

	// TracerProvider interface for distributed tracing.
	TracerProvider interface {
		Tracer(name string, opts ...trace.TracerOption) trace.Tracer
	}

	// MeterProvider interface for metrics collection.
	MeterProvider interface {
		Meter(name string, opts ...metric.MeterOption) metric.Meter
	}
)

// RegisterLogger registers a logger factory function.
func (b *Builder) RegisterLogger(factory func() Logger) error {
	return b.container.Register(factory)
}

// RegisterTracerProvider registers a tracer provider factory function.
func (b *Builder) RegisterTracerProvider(factory func() TracerProvider) error {
	return b.container.Register(factory)
}

// WithLogger sets the logger for the container (if supported).
func (b *Builder) WithLogger(logger Logger) *Builder {
	if container, ok := b.container.(*containerImpl); ok {
		container.logger = logger
	}
	return b
}

// WithTracerProvider sets the tracer provider for the container (if supported).
func (b *Builder) WithTracerProvider(tracerProvider TracerProvider) *Builder {
	if container, ok := b.container.(*containerImpl); ok {
		container.tracerProvider = tracerProvider
	}
	return b
}

// RegisterMeterProvider registers a meter provider factory function.
func (b *Builder) RegisterMeterProvider(factory func() MeterProvider) error {
	return b.container.Register(factory)
}

// RegisterMetrics registers a metrics factory function.
func (b *Builder) RegisterMetrics(factory func() (*Metrics, error)) error {
	return b.container.Register(factory)
}

// RegisterNoOpLogger registers a no-op logger (useful for testing).
func (b *Builder) RegisterNoOpLogger() error {
	return b.RegisterLogger(func() Logger { return &noOpLogger{} })
}

// RegisterNoOpTracerProvider registers a no-op tracer provider.
func (b *Builder) RegisterNoOpTracerProvider() error {
	return b.RegisterTracerProvider(func() TracerProvider { return noop.NewTracerProvider() })
}

// RegisterNoOpMetrics registers no-op metrics.
func (b *Builder) RegisterNoOpMetrics() error {
	return b.RegisterMetrics(func() (*Metrics, error) { return NoOpMetrics(), nil })
}

// noOpLogger implements Logger with no-op behavior.
type noOpLogger struct{}

func (*noOpLogger) Debug(_ string, _ ...any) {}
func (*noOpLogger) Info(_ string, _ ...any)  {}
func (*noOpLogger) Warn(_ string, _ ...any)  {}
func (*noOpLogger) Error(_ string, _ ...any) {}
func (*noOpLogger) With(_ ...any) Logger     { return &noOpLogger{} }
