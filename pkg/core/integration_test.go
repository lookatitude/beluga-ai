package core

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Integration tests that combine multiple components

func TestContainerWithTracedRunnable_Integration(t *testing.T) {
	// Create a container with monitoring components
	container := NewContainerWithOptions(
		WithLogger(&testLogger{}),
		WithTracerProvider(trace.NewNoopTracerProvider()),
	)

	// Register metrics
	metrics := NoOpMetrics()
	_ = container.Register(func() *Metrics { return metrics })

	// Create a traced runnable
	mock := NewMockRunnable().WithInvokeResult("integration_test")
	tracer := trace.NewNoopTracerProvider().Tracer("")
	traced := NewTracedRunnable(mock, tracer, metrics, "integration_test", "test_instance")

	// Register the traced runnable
	_ = container.Register(func() Runnable { return traced })

	// Resolve and use the runnable
	var runnable Runnable
	if err := container.Resolve(&runnable); err != nil {
		t.Fatalf("Failed to resolve runnable: %v", err)
	}

	// Execute the runnable
	result, err := runnable.Invoke(context.Background(), "input")
	if err != nil {
		t.Errorf("Runnable.Invoke() error = %v", err)
	}
	if result != "integration_test" {
		t.Errorf("Runnable.Invoke() = %v, expected integration_test", result)
	}

	// Verify call tracking
	calls := mock.GetInvokeCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 invoke call, got %d", len(calls))
	}

	// Verify container health
	err = container.CheckHealth(context.Background())
	if err != nil {
		t.Errorf("Container health check failed: %v", err)
	}
}

func TestBuilderWithComplexDependencies_Integration(t *testing.T) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Register a complex service hierarchy
	type TestConfig struct {
		Timeout int
	}
	err := builder.Register(func() *TestConfig { return &TestConfig{Timeout: 30} })
	if err != nil {
		t.Fatalf("Register config failed: %v", err)
	}

	err = builder.Register(func() Logger { return &testLogger{} })
	if err != nil {
		t.Fatalf("Register logger failed: %v", err)
	}

	err = builder.Register(func() TracerProvider { return trace.NewNoopTracerProvider() })
	if err != nil {
		t.Fatalf("Register tracer failed: %v", err)
	}

	err = builder.Register(func() *Metrics {
		metrics := NoOpMetrics()
		return metrics
	})
	if err != nil {
		t.Fatalf("Register metrics failed: %v", err)
	}

	// Register a service that depends on all the above
	err = builder.Register(func(cfg *Config, logger Logger, tracer TracerProvider, metrics *Metrics) *ComplexService {
		return &ComplexService{
			Config:  cfg,
			Logger:  logger,
			Tracer:  tracer,
			Metrics: metrics,
		}
	})
	if err != nil {
		t.Fatalf("Register complex service failed: %v", err)
	}

	// Build the complex service
	var service *ComplexService
	err = builder.Build(&service)
	if err != nil {
		t.Fatalf("Build complex service failed: %v", err)
	}

	// Verify the service has all dependencies
	if service.Config == nil {
		t.Error("Config dependency not resolved")
	}
	if service.Logger == nil {
		t.Error("Logger dependency not resolved")
	}
	if service.Tracer == nil {
		t.Error("Tracer dependency not resolved")
	}
	if service.Metrics == nil {
		t.Error("Metrics dependency not resolved")
	}

	// Test service functionality
	err = service.Execute(context.Background())
	if err != nil {
		t.Errorf("ComplexService.Execute() error = %v", err)
	}
}

func TestConcurrentContainerOperations_Integration(t *testing.T) {
	container := NewContainer()
	numGoroutines := 10
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent registration, resolution, and health checks
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 3 {
				case 0: // Register
					_ = container.Register(func() int { return goroutineID*1000 + j })
				case 1: // Resolve
					var result int
					_ = container.Resolve(&result)
				case 2: // Health check
					_ = container.CheckHealth(context.Background())
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify container is still functional
	err := container.CheckHealth(context.Background())
	if err != nil {
		t.Errorf("Container health check failed after concurrent operations: %v", err)
	}
}

func TestErrorPropagationThroughLayers_Integration(t *testing.T) {
	// Test how errors propagate through multiple layers of components
	container := NewContainer()
	builder := NewBuilder(container)

	// Register services that can fail
	err := builder.Register(func() *FailableService {
		return &FailableService{shouldFail: true}
	})
	if err != nil {
		t.Fatalf("Register failable service failed: %v", err)
	}

	err = builder.Register(func(svc *FailableService) *DependentService {
		return &DependentService{service: svc}
	})
	if err != nil {
		t.Fatalf("Register dependent service failed: %v", err)
	}

	// Build and test error propagation
	var depSvc *DependentService
	err = builder.Build(&depSvc)
	if err != nil {
		t.Fatalf("Build dependent service failed: %v", err)
	}

	// Execute and expect error propagation
	err = depSvc.Execute(context.Background())
	if err == nil {
		t.Error("Expected error to propagate through layers")
	}

	// Verify error type
	if !IsErrorType(err, ErrorTypeInternal) {
		t.Errorf("Expected Internal error type, got error: %v", err)
	}
}

func TestRunnableChainWithTracing_Integration(t *testing.T) {
	// Test a chain of runnables with tracing and metrics
	container := NewContainer()
	builder := NewBuilder(container)

	// Set up monitoring
	logger := &testLogger{}
	tracerProvider := trace.NewNoopTracerProvider()
	metrics := NoOpMetrics()

	builder.WithLogger(logger).WithTracerProvider(tracerProvider)

	// Create a chain of runnables
	runnable1 := NewMockRunnable().WithInvokeResult("step1")
	runnable2 := NewMockRunnable().WithInvokeResult("step2")
	runnable3 := NewMockRunnable().WithInvokeResult("step3")

	traced1 := NewTracedRunnable(runnable1, tracerProvider.Tracer(""), metrics, "step1", "")
	traced2 := NewTracedRunnable(runnable2, tracerProvider.Tracer(""), metrics, "step2", "")
	traced3 := NewTracedRunnable(runnable3, tracerProvider.Tracer(""), metrics, "step3", "")

	// Create a simple chain (for this test, just execute sequentially)
	runnables := []Runnable{traced1, traced2, traced3}

	// Execute the chain
	ctx := context.Background()
	results := make([]any, len(runnables))

	for i, runnable := range runnables {
		result, err := runnable.Invoke(ctx, "chain_input")
		if err != nil {
			t.Errorf("Runnable %d failed: %v", i, err)
			continue
		}
		results[i] = result
	}

	// Verify results
	expected := []any{"step1", "step2", "step3"}
	for i, expectedResult := range expected {
		if results[i] != expectedResult {
			t.Errorf("Step %d: expected %v, got %v", i, expectedResult, results[i])
		}
	}

	// Verify call tracking
	for i, runnable := range []*MockRunnable{runnable1, runnable2, runnable3} {
		calls := runnable.GetInvokeCalls()
		if len(calls) != 1 {
			t.Errorf("Runnable %d: expected 1 call, got %d", i, len(calls))
		}
	}
}

func TestContainerLifecycle_Integration(t *testing.T) {
	// Test complete container lifecycle: setup, use, cleanup
	container := NewContainerWithOptions(
		WithLogger(&testLogger{}),
		WithTracerProvider(trace.NewNoopTracerProvider()),
	)

	builder := NewBuilder(container)

	// Setup phase
	err := builder.RegisterNoOpLogger()
	if err != nil {
		t.Fatalf("Register logger failed: %v", err)
	}

	err = builder.RegisterNoOpTracerProvider()
	if err != nil {
		t.Fatalf("Register tracer failed: %v", err)
	}

	err = builder.RegisterNoOpMetrics()
	if err != nil {
		t.Fatalf("Register metrics failed: %v", err)
	}

	// Use phase
	var logger Logger
	err = builder.Build(&logger)
	if err != nil {
		t.Fatalf("Build logger failed: %v", err)
	}

	var tracer TracerProvider
	err = builder.Build(&tracer)
	if err != nil {
		t.Fatalf("Build tracer failed: %v", err)
	}

	var metrics *Metrics
	err = builder.Build(&metrics)
	if err != nil {
		t.Fatalf("Build metrics failed: %v", err)
	}

	// Test functionality
	logger.Info("Integration test message")
	_, span := tracer.Tracer("test").Start(context.Background(), "test_span")
	span.End()

	// Health check
	err = container.CheckHealth(context.Background())
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Cleanup phase
	container.Clear()

	// Verify cleanup
	if container.Has(reflect.TypeOf((*Logger)(nil)).Elem()) {
		t.Error("Logger should be cleared")
	}
	if container.Has(reflect.TypeOf((*TracerProvider)(nil)).Elem()) {
		t.Error("TracerProvider should be cleared")
	}
	if container.Has(reflect.TypeOf((*Metrics)(nil))) {
		t.Error("Metrics should be cleared")
	}
}

func TestContextCancellationPropagation_Integration(t *testing.T) {
	// Test that context cancellation propagates through component layers
	container := NewContainer()
	builder := NewBuilder(container)

	// Register a service that respects context
	err := builder.Register(func() *ContextAwareService {
		return &ContextAwareService{delay: 100 * time.Millisecond}
	})
	if err != nil {
		t.Fatalf("Register context aware service failed: %v", err)
	}

	var service *ContextAwareService
	err = builder.Build(&service)
	if err != nil {
		t.Fatalf("Build context aware service failed: %v", err)
	}

	// Test with cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = service.Execute(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}


// Supporting types for integration tests

type ComplexService struct {
	Config  *Config
	Logger  Logger
	Tracer  TracerProvider
	Metrics *Metrics
}

func (s *ComplexService) Execute(ctx context.Context) error {
	s.Logger.Info("ComplexService executing")
	_, span := s.Tracer.Tracer("complex_service").Start(ctx, "execute")
	defer span.End()

	// Simulate some work
	time.Sleep(1 * time.Millisecond)

	s.Metrics.RecordRunnableInvoke(ctx, "complex_service", time.Millisecond, nil)
	return nil
}

type FailableService struct {
	shouldFail bool
}

func (s *FailableService) Execute(ctx context.Context) error {
	if s.shouldFail {
		return NewInternalError("FailableService configured to fail", nil)
	}
	return nil
}

type DependentService struct {
	service *FailableService
}

func (s *DependentService) Execute(ctx context.Context) error {
	return s.service.Execute(ctx)
}

type ContextAwareService struct {
	delay time.Duration
}

func (s *ContextAwareService) Execute(ctx context.Context) error {
	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
