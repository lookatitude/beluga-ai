package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestNewContainer(t *testing.T) {
	container := NewContainer()

	if container == nil {
		t.Fatal("NewContainer() returned nil")
	}

	// Test basic functionality
	if err := container.Register(func() string { return "test" }); err != nil {
		t.Errorf("Register() error = %v", err)
	}

	var result string
	if err := container.Resolve(&result); err != nil {
		t.Errorf("Resolve() error = %v", err)
	}

	if result != "test" {
		t.Errorf("Resolve() = %q, expected %q", result, "test")
	}
}

func TestNewContainerWithOptions(t *testing.T) {
	logger := &testLogger{}
	tracerProvider := trace.NewNoopTracerProvider()

	container := NewContainerWithOptions(
		WithLogger(logger),
		WithTracerProvider(tracerProvider),
	)

	if container == nil {
		t.Fatal("NewContainerWithOptions() returned nil")
	}

	// Verify the container has the configured components
	if impl, ok := container.(*containerImpl); ok {
		if impl.logger != logger {
			t.Error("Logger not set correctly")
		}
		if impl.tracerProvider != tracerProvider {
			t.Error("TracerProvider not set correctly")
		}
	}
}

func TestContainer_Register(t *testing.T) {
	tests := []struct {
		factoryFunc any
		name        string
		wantErr     bool
	}{
		{
			name:        "valid factory function",
			factoryFunc: func() string { return "test" },
			wantErr:     false,
		},
		{
			name:        "factory with dependency",
			factoryFunc: func(s string) int { return len(s) },
			wantErr:     false,
		},
		{
			name:        "invalid factory - not a function",
			factoryFunc: "not a function",
			wantErr:     true,
		},
		{
			name:        "invalid factory - returns nothing",
			factoryFunc: func() {},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()

			err := container.Register(tt.factoryFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainer_Resolve(t *testing.T) {
	tests := []struct {
		target  any
		setup   func(Container)
		name    string
		wantErr bool
	}{
		{
			name: "resolve simple type",
			setup: func(c Container) {
				_ = c.Register(func() string { return "test" })
			},
			target:  func() *string { var s string; return &s }(),
			wantErr: false,
		},
		{
			name: "resolve with dependency",
			setup: func(c Container) {
				_ = c.Register(func() string { return "test" })
				_ = c.Register(func(s string) int { return len(s) })
			},
			target:  func() *int { var i int; return &i }(),
			wantErr: false,
		},
		{
			name: "resolve unregistered type",
			setup: func(c Container) {
				// No registration
			},
			target:  func() *string { var s string; return &s }(),
			wantErr: true,
		},
		{
			name: "resolve non-pointer",
			setup: func(c Container) {
				_ = c.Register(func() string { return "test" })
			},
			target:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()
			tt.setup(container)

			err := container.Resolve(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainer_Singleton(t *testing.T) {
	container := NewContainer()

	instance := "singleton_test"
	container.Singleton(instance)

	var result string
	if err := container.Resolve(&result); err != nil {
		t.Errorf("Resolve() error = %v", err)
	}

	if result != instance {
		t.Errorf("Resolve() = %q, expected %q", result, instance)
	}
}

func TestContainer_Has(t *testing.T) {
	container := NewContainer()

	// Test with registered factory
	_ = container.Register(func() string { return "test" })
	if !container.Has(stringType()) {
		t.Error("Has() should return true for registered type")
	}

	// Test with singleton
	container.Singleton(42)
	if !container.Has(intType()) {
		t.Error("Has() should return true for singleton type")
	}

	// Test with unregistered type
	if container.Has(boolType()) {
		t.Error("Has() should return false for unregistered type")
	}
}

func TestContainer_Clear(t *testing.T) {
	container := NewContainer()

	_ = container.Register(func() string { return "test" })
	container.Singleton(42)

	if !container.Has(stringType()) || !container.Has(intType()) {
		t.Error("Setup failed: types should be registered")
	}

	container.Clear()

	if container.Has(stringType()) || container.Has(intType()) {
		t.Error("Clear() should remove all registrations")
	}
}

func TestContainer_CheckHealth(t *testing.T) {
	tests := []struct {
		setup   func(Container)
		name    string
		wantErr bool
	}{
		{
			name:    "healthy container",
			setup:   func(c Container) {},
			wantErr: false,
		},
		{
			name: "container with existing registrations",
			setup: func(c Container) {
				_ = c.Register(func() string { return "existing" })
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()
			tt.setup(container)

			err := container.CheckHealth(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHealth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuilder_WithLogger(t *testing.T) {
	logger := &testLogger{}
	builder := NewBuilder(NewContainer()).WithLogger(logger)

	if impl, ok := builder.container.(*containerImpl); ok {
		if impl.logger != logger {
			t.Error("WithLogger() did not set logger correctly")
		}
	}
}

func TestBuilder_WithTracerProvider(t *testing.T) {
	tracerProvider := trace.NewNoopTracerProvider()
	builder := NewBuilder(NewContainer()).WithTracerProvider(tracerProvider)

	if impl, ok := builder.container.(*containerImpl); ok {
		if impl.tracerProvider != tracerProvider {
			t.Error("WithTracerProvider() did not set tracer provider correctly")
		}
	}
}

// Helper functions for type reflection.
func stringType() reflect.Type { return reflect.TypeOf("") }
func intType() reflect.Type    { return reflect.TypeOf(0) }
func boolType() reflect.Type   { return reflect.TypeOf(false) }

// testLogger is a simple logger implementation for testing.
type testLogger struct {
	logs []string
}

func (t *testLogger) Debug(msg string, args ...any) {
	t.logs = append(t.logs, "DEBUG: "+msg)
}

func (t *testLogger) Info(msg string, args ...any) {
	t.logs = append(t.logs, "INFO: "+msg)
}

func (t *testLogger) Warn(msg string, args ...any) {
	t.logs = append(t.logs, "WARN: "+msg)
}

func (t *testLogger) Error(msg string, args ...any) {
	t.logs = append(t.logs, "ERROR: "+msg)
}

func (t *testLogger) With(args ...any) Logger {
	return t
}

// Concurrency tests for DI container

func TestContainer_ConcurrentRegistration(t *testing.T) {
	container := NewContainer()
	numGoroutines := 10
	numRegistrations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Use a type that creates unique types per registration
	type testType struct {
		ID int
	}

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numRegistrations; j++ {
				key := id*numRegistrations + j
				// Create a unique type for each registration by using a closure
				// that captures a unique value, forcing separate registrations
				localKey := key
				// Register as singleton with unique instance
				container.Singleton(&testType{ID: localKey})
			}
		}(i)
	}

	wg.Wait()

	// Verify that registrations were thread-safe (no panic, can resolve)
	// Note: Since we're using Singleton with the same type, only the last one
	// will be stored, but the test verifies thread-safety of registration
	var result *testType
	err := container.Resolve(&result)
	if err != nil {
		t.Errorf("Failed to resolve after concurrent registration: %v", err)
	}
	if result == nil {
		t.Error("Resolved instance is nil")
	}
}

func TestContainer_ConcurrentResolution(t *testing.T) {
	container := NewContainer()

	// Register a single service that returns a constant value
	_ = container.Register(func() int { return 42 })

	numGoroutines := 10
	numResolutions := 100
	results := make([][]int, numGoroutines)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			results[goroutineID] = make([]int, numResolutions)
			for j := 0; j < numResolutions; j++ {
				var result int
				err := container.Resolve(&result)
				if err != nil {
					t.Errorf("Concurrent resolution failed: %v", err)
					return
				}
				results[goroutineID][j] = result
			}
		}(i)
	}

	wg.Wait()

	// All results should be the same (singleton behavior)
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numResolutions; j++ {
			if results[i][j] != 42 {
				t.Errorf("Expected all resolutions to return 42, got %d", results[i][j])
			}
		}
	}
}

func TestContainer_ConcurrentMixedOperations(t *testing.T) {
	container := NewContainer()
	numGoroutines := 20
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Use a type that won't conflict with health check
	type testService struct {
		ID int
	}

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 4 {
				case 0: // Register
					_ = container.Register(func() testService {
						return testService{ID: goroutineID*operationsPerGoroutine + j}
					})
				case 1: // Resolve
					var result testService
					_ = container.Resolve(&result)
				case 2: // Has
					_ = container.Has(reflect.TypeOf(testService{}))
				case 3: // Singleton
					container.Singleton(testService{ID: j})
				}
			}
		}(i)
	}

	wg.Wait()

	// Container should still be in a valid state
	err := container.CheckHealth(context.Background())
	if err != nil {
		t.Errorf("Container health check failed after concurrent operations: %v", err)
	}
}

func TestContainer_ConcurrentHealthChecks(t *testing.T) {
	container := NewContainer()
	numGoroutines := 5
	healthChecksPerGoroutine := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	var mu sync.Mutex
	var errors []error

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < healthChecksPerGoroutine; j++ {
				err := container.CheckHealth(context.Background())
				if err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
				}
				time.Sleep(time.Millisecond) // Small delay to increase chance of race conditions
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		t.Errorf("Concurrent health checks failed: %v", errors)
	}
}

func TestContainer_ConcurrentBuilderOperations(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	numGoroutines := 10
	operationsPerGoroutine := 25

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 3 {
				case 0: // Register
					_ = builder.Register(func() int { return id*operationsPerGoroutine + j })
				case 1: // Singleton
					builder.Singleton(fmt.Sprintf("singleton_%d_%d", id, j))
				case 2: // Build
					var result int
					_ = builder.Build(&result)
				}
			}
		}(i)
	}

	wg.Wait()

	// Builder should still be functional
	var result int
	err := builder.Build(&result)
	if err != nil {
		t.Errorf("Builder failed after concurrent operations: %v", err)
	}
}

// Builder integration tests

func TestBuilder_FluentInterface(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Test fluent interface chaining
	result := builder.
		WithLogger(&testLogger{}).
		WithTracerProvider(trace.NewNoopTracerProvider())

	if result != builder {
		t.Error("Fluent interface methods should return the builder instance")
	}
}

func TestBuilder_RegisterMultipleServices(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register multiple services
	err := builder.Register(func() string { return "service1" })
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	err = builder.Register(func() int { return 42 })
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	err = builder.Register(func() bool { return true })
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Verify all services can be resolved
	var s string
	err = builder.Build(&s)
	if err != nil {
		t.Errorf("Build(string) error = %v", err)
	}
	if s != "service1" {
		t.Errorf("Expected 'service1', got %q", s)
	}

	var i int
	err = builder.Build(&i)
	if err != nil {
		t.Errorf("Build(int) error = %v", err)
	}
	if i != 42 {
		t.Errorf("Expected 42, got %d", i)
	}

	var b bool
	err = builder.Build(&b)
	if err != nil {
		t.Errorf("Build(bool) error = %v", err)
	}
	if !b {
		t.Error("Expected true, got false")
	}
}

func TestBuilder_ServiceDependencies(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register service with dependency
	err := builder.Register(func() string { return "dependency" })
	if err != nil {
		t.Errorf("Register(dependency) error = %v", err)
	}

	err = builder.Register(func(s string) ServiceWithDep { return &serviceWithDepImpl{dep: s} })
	if err != nil {
		t.Errorf("Register(service) error = %v", err)
	}

	// Resolve service with dependency
	var service ServiceWithDep
	err = builder.Build(&service)
	if err != nil {
		t.Errorf("Build(service) error = %v", err)
	}

	if service.GetDep() != "dependency" {
		t.Errorf("Expected dependency 'dependency', got %q", service.GetDep())
	}
}

func TestBuilder_SingletonServices(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register singleton
	builder.Singleton("singleton_value")

	// Resolve multiple times
	var s1 string
	err := builder.Build(&s1)
	if err != nil {
		t.Errorf("Build(s1) error = %v", err)
	}

	var s2 string
	err = builder.Build(&s2)
	if err != nil {
		t.Errorf("Build(s2) error = %v", err)
	}

	// Both should be the same instance
	if s1 != s2 {
		t.Errorf("Singleton services should return same instance: %q != %q", s1, s2)
	}
}

func TestBuilder_RegisterMonitoringComponents(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register monitoring components
	err := builder.RegisterLogger(func() Logger { return &testLogger{} })
	if err != nil {
		t.Errorf("RegisterLogger() error = %v", err)
	}

	err = builder.RegisterTracerProvider(func() TracerProvider { return trace.NewNoopTracerProvider() })
	if err != nil {
		t.Errorf("RegisterTracerProvider() error = %v", err)
	}

	err = builder.RegisterMetrics(func() (*Metrics, error) { return NoOpMetrics(), nil })
	if err != nil {
		t.Errorf("RegisterMetrics() error = %v", err)
	}

	// Resolve monitoring components
	var logger Logger
	err = builder.Build(&logger)
	if err != nil {
		t.Errorf("Build(logger) error = %v", err)
	}
	if logger == nil {
		t.Error("Logger should not be nil")
	}

	var tracerProvider TracerProvider
	err = builder.Build(&tracerProvider)
	if err != nil {
		t.Errorf("Build(tracerProvider) error = %v", err)
	}
	if tracerProvider == nil {
		t.Error("TracerProvider should not be nil")
	}

	var metrics *Metrics
	err = builder.Build(&metrics)
	if err != nil {
		t.Errorf("Build(metrics) error = %v", err)
	}
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}
}

func TestBuilder_NoOpMonitoringComponents(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register no-op monitoring components
	err := builder.RegisterNoOpLogger()
	if err != nil {
		t.Errorf("RegisterNoOpLogger() error = %v", err)
	}

	err = builder.RegisterNoOpTracerProvider()
	if err != nil {
		t.Errorf("RegisterNoOpTracerProvider() error = %v", err)
	}

	err = builder.RegisterNoOpMetrics()
	if err != nil {
		t.Errorf("RegisterNoOpMetrics() error = %v", err)
	}

	// Verify no-op components work
	var logger Logger
	err = builder.Build(&logger)
	if err != nil {
		t.Errorf("Build(logger) error = %v", err)
	}

	var tracerProvider TracerProvider
	err = builder.Build(&tracerProvider)
	if err != nil {
		t.Errorf("Build(tracerProvider) error = %v", err)
	}

	var metrics *Metrics
	err = builder.Build(&metrics)
	if err != nil {
		t.Errorf("Build(metrics) error = %v", err)
	}

	// Test that no-op components don't panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
}

func TestBuilder_ErrorHandling(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Test registering invalid factory
	err := builder.Register("not a function")
	if err == nil {
		t.Error("Expected error when registering non-function")
	}

	// Test registering function with error
	err = builder.Register(func() (string, error) { return "", errors.New("factory error") })
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Test resolving service that fails
	var result string
	err = builder.Build(&result)
	if err == nil {
		t.Error("Expected error when resolving failing service")
	}
}

func TestBuilder_MixedRegistrationTypes(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register mix of factories and singletons
	err := builder.Register(func() string { return "factory" })
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	builder.Singleton(42)

	// Both should be resolvable
	var s string
	err = builder.Build(&s)
	if err != nil {
		t.Errorf("Build(string) error = %v", err)
	}
	if s != "factory" {
		t.Errorf("Expected 'factory', got %q", s)
	}

	var i int
	err = builder.Build(&i)
	if err != nil {
		t.Errorf("Build(int) error = %v", err)
	}
	if i != 42 {
		t.Errorf("Expected 42, got %d", i)
	}
}

func TestBuilder_ComplexDependencyChain(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Create a complex dependency chain: A -> B -> C
	err := builder.Register(func() string { return "C" })
	if err != nil {
		t.Errorf("Register(C) error = %v", err)
	}

	err = builder.Register(func(c string) ServiceB { return &serviceBImpl{dep: c} })
	if err != nil {
		t.Errorf("Register(B) error = %v", err)
	}

	err = builder.Register(func(b ServiceB) ServiceA { return &serviceAImpl{dep: b} })
	if err != nil {
		t.Errorf("Register(A) error = %v", err)
	}

	// Resolve the top-level service
	var service ServiceA
	err = builder.Build(&service)
	if err != nil {
		t.Errorf("Build(A) error = %v", err)
	}

	// Verify dependency chain
	if service.GetB().GetDep() != "C" {
		t.Errorf("Expected dependency chain A->B->C, got A->B->%q", service.GetB().GetDep())
	}
}

// Test interfaces and implementations for dependency testing.
type ServiceWithDep interface {
	GetDep() string
}

type serviceWithDepImpl struct {
	dep string
}

func (s *serviceWithDepImpl) GetDep() string {
	return s.dep
}

type ServiceA interface {
	GetB() ServiceB
}

type ServiceB interface {
	GetDep() string
}

type serviceAImpl struct {
	dep ServiceB
}

func (s *serviceAImpl) GetB() ServiceB {
	return s.dep
}

type serviceBImpl struct {
	dep string
}

func (s *serviceBImpl) GetDep() string {
	return s.dep
}
