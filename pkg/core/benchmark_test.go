package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
)

// Static errors for benchmark tests.
var errBenchmarkTestError = errors.New("test error")

// Benchmark tests for performance measurement

func BenchmarkContainer_Resolve(b *testing.B) {
	container := NewContainer()
	if err := container.Register(func() string { return "benchmark_value" }); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result string
		err := container.Resolve(&result)
		if err != nil {
			b.Fatal(err)
		}
		if result != "benchmark_value" {
			b.Fatalf("Expected 'benchmark_value', got %q", result)
		}
	}
}

func BenchmarkContainer_Resolve_WithDependencyChain(b *testing.B) {
	container := NewContainer()

	// Create a dependency chain: A -> B -> C -> D
	if err := container.Register(func() string { return "D" }); err != nil {
		b.Fatal(err)
	}
	if err := container.Register(func(d string) BenchmarkServiceD { return &benchmarkServiceDImpl{dep: d} }); err != nil {
		b.Fatal(err)
	}
	if err := container.Register(
		func(d BenchmarkServiceD) BenchmarkServiceC {
			return &benchmarkServiceCImpl{dep: d}
		}); err != nil {
		b.Fatal(err)
	}
	if err := container.Register(
		func(c BenchmarkServiceC) BenchmarkServiceB {
			return &benchmarkServiceBImpl{dep: c}
		}); err != nil {
		b.Fatal(err)
	}
	if err := container.Register(
		func(b BenchmarkServiceB) BenchmarkServiceA {
			return &benchmarkServiceAImpl{dep: b}
		}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result BenchmarkServiceA
		err := container.Resolve(&result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContainer_ConcurrentResolve(b *testing.B) {
	container := NewContainer()
	if err := container.Register(func() string { return "concurrent_value" }); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result string
			err := container.Resolve(&result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkContainer_Register(b *testing.B) {
	container := NewContainer()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := container.Register(func() int { return i }); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunnable_Invoke(b *testing.B) {
	mock := NewMockRunnable().WithInvokeResult("benchmark_result")

	b.ResetTimer()
	b.ReportAllocs()

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		result, err := mock.Invoke(ctx, "input")
		if err != nil {
			b.Fatal(err)
		}
		if result != "benchmark_result" {
			b.Fatalf("Expected 'benchmark_result', got %v", result)
		}
	}
}

func BenchmarkRunnable_Batch(b *testing.B) {
	mock := NewMockRunnable()

	inputs := []any{"input1", "input2", "input3", "input4", "input5"}

	b.ResetTimer()
	b.ReportAllocs()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < b.N; i++ {
		results, err := mock.Batch(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
		if len(results) != len(inputs) {
			b.Fatalf("Expected %d results, got %d", len(inputs), len(results))
		}
	}
}

func BenchmarkTracedRunnable_Invoke(b *testing.B) {
	mock := NewMockRunnable().WithInvokeResult("traced_result")
	tracer := noop.NewTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "benchmark", "test")

	b.ResetTimer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := traced.Invoke(ctx, "input")
		if err != nil {
			b.Fatal(err)
		}
		if result != "traced_result" {
			b.Fatalf("Expected 'traced_result', got %v", result)
		}
	}
}

func BenchmarkTracedRunnable_Stream(b *testing.B) {
	mock := NewMockRunnable().WithStreamChunks("chunk1", "chunk2", "chunk3")
	tracer := noop.NewTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "benchmark", "test")

	b.ResetTimer()
	b.ReportAllocs()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < b.N; i++ {
		streamChan, err := traced.Stream(ctx, "input")
		if err != nil {
			b.Fatal(err)
		}

		// Consume all chunks
		chunks := 0
		for range streamChan {
			chunks++
		}
		if chunks != 3 {
			b.Fatalf("Expected 3 chunks, got %d", chunks)
		}
	}
}

func BenchmarkContainer_CheckHealth(b *testing.B) {
	container := NewContainer()
	if err := container.Register(func() string { return "health_check" }); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		err := container.CheckHealth(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkErrorHandling(b *testing.B) {
	testErr := errBenchmarkTestError

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := NewInternalError("benchmark error", testErr)
		if err == nil {
			b.Fatal("Expected error")
		}
		if err.Type != ErrorTypeInternal {
			b.Fatal("Expected Internal error type")
		}
	}
}

func BenchmarkErrorTypeChecking(b *testing.B) {
	err := NewValidationError("test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := IsErrorType(err, ErrorTypeValidation)
		if !result {
			b.Fatal("Expected true for Validation error type")
		}
	}
}

func BenchmarkMetrics_Recording(b *testing.B) {
	metrics := NoOpMetrics() // Use no-op metrics for benchmarking

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics.RecordRunnableInvoke(ctx, "benchmark", time.Microsecond*100, nil)
	}
}

func BenchmarkBuilder_Build(b *testing.B) {
	container := NewContainer()
	builder := NewBuilder(container)

	if err := builder.Register(func() string { return "builder_test" }); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result string
		err := builder.Build(&result)
		if err != nil {
			b.Fatal(err)
		}
		if result != "builder_test" {
			b.Fatalf("Expected 'builder_test', got %q", result)
		}
	}
}

func BenchmarkBuilder_ComplexDependencyChain(b *testing.B) {
	container := NewContainer()
	builder := NewBuilder(container)

	// Create a dependency chain
	if err := builder.Register(func() string { return "dep1" }); err != nil {
		b.Fatal(err)
	}
	if err := builder.Register(func(s string) BenchmarkServiceB {
		return &benchmarkServiceBImpl{dep: &benchmarkServiceCImpl{dep: &benchmarkServiceDImpl{dep: s}}}
	}); err != nil {
		b.Fatal(err)
	}
	if err := builder.Register(
		func(b BenchmarkServiceB) BenchmarkServiceA {
			return &benchmarkServiceAImpl{dep: b}
		}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result BenchmarkServiceA
		err := builder.Build(&result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Memory benchmark tests

func BenchmarkContainer_Resolve_Memory(b *testing.B) {
	container := NewContainer()
	if err := container.Register(func() *LargeStruct {
		return &LargeStruct{
			data: make([]byte, 1024), // 1KB of data
		}
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result *LargeStruct
		err := container.Resolve(&result)
		if err != nil {
			b.Fatal(err)
		}
		if len(result.data) != 1024 {
			b.Fatal("Unexpected data size")
		}
	}
}

func BenchmarkContainer_Singleton_Memory(b *testing.B) {
	container := NewContainer()
	largeInstance := &LargeStruct{
		data: make([]byte, 1024*1024), // 1MB of data
	}
	container.Singleton(largeInstance)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result *LargeStruct
		err := container.Resolve(&result)
		if err != nil {
			b.Fatal(err)
		}
		if result != largeInstance {
			b.Fatal("Expected same singleton instance")
		}
	}
}

// Supporting types for benchmarks

type LargeStruct struct {
	data []byte
}

type benchmarkServiceAImpl struct {
	dep BenchmarkServiceB
}

func (s *benchmarkServiceAImpl) GetB() BenchmarkServiceB {
	return s.dep
}

type benchmarkServiceBImpl struct {
	dep BenchmarkServiceC
}

func (s *benchmarkServiceBImpl) GetDep() string {
	return s.dep.GetDep()
}

type benchmarkServiceCImpl struct {
	dep BenchmarkServiceD
}

func (s *benchmarkServiceCImpl) GetDep() string {
	return s.dep.GetDep()
}

type benchmarkServiceDImpl struct {
	dep string
}

func (s *benchmarkServiceDImpl) GetDep() string {
	return s.dep
}

// Interface definitions for benchmark services.
type BenchmarkServiceA interface {
	GetB() BenchmarkServiceB
}

type BenchmarkServiceB interface {
	GetDep() string
}

type BenchmarkServiceC interface {
	GetDep() string
}

type BenchmarkServiceD interface {
	GetDep() string
}
