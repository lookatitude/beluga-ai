package config

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestNewMetrics(t *testing.T) {
	// Test with noop meter (doesn't require actual OTEL setup)
	meter := noop.Meter{}
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Fatal("NewMetrics() returned nil")
	}

	// Verify that all metric instruments are initialized
	if metrics.configLoadsTotal == nil {
		t.Error("configLoadsTotal should be initialized")
	}
	if metrics.configLoadDuration == nil {
		t.Error("configLoadDuration should be initialized")
	}
	if metrics.configErrorsTotal == nil {
		t.Error("configErrorsTotal should be initialized")
	}
	if metrics.validationDuration == nil {
		t.Error("validationDuration should be initialized")
	}
	if metrics.validationErrorsTotal == nil {
		t.Error("validationErrorsTotal should be initialized")
	}
}

func TestNoOpMetrics(t *testing.T) {
	metrics := NoOpMetrics()

	if metrics == nil {
		t.Fatal("NoOpMetrics() returned nil")
	}

	// Verify that all metric instruments are nil (no-op behavior)
	if metrics.configLoadsTotal != nil {
		t.Error("configLoadsTotal should be nil in NoOpMetrics")
	}
	if metrics.configLoadDuration != nil {
		t.Error("configLoadDuration should be nil in NoOpMetrics")
	}
	if metrics.configErrorsTotal != nil {
		t.Error("configErrorsTotal should be nil in NoOpMetrics")
	}
	if metrics.validationDuration != nil {
		t.Error("validationDuration should be nil in NoOpMetrics")
	}
	if metrics.validationErrorsTotal != nil {
		t.Error("validationErrorsTotal should be nil in NoOpMetrics")
	}
}

func TestMetrics_RecordConfigLoad(t *testing.T) {
	// Use noop meter for testing
	meter := noop.Meter{}
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	duration := 100 * time.Millisecond

	tests := []struct {
		name    string
		source  string
		success bool
	}{
		{"successful load", "file", true},
		{"failed load", "env", false},
		{"empty source", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic even with nil metrics
			metrics.RecordConfigLoad(ctx, duration, tt.success, tt.source)

			// Test with nil metrics (should not panic)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var nilMetrics *Metrics
			nilMetrics.RecordConfigLoad(ctx, duration, tt.success, tt.source)
		})
	}
}

func TestMetrics_RecordValidation(t *testing.T) {
	// Use noop meter for testing
	meter := noop.Meter{}
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	duration := 50 * time.Millisecond

	tests := []struct {
		name    string
		success bool
	}{
		{"successful validation", true},
		{"failed validation", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic even with nil metrics
			metrics.RecordValidation(ctx, duration, tt.success)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Test with nil metrics (should not panic)
			var nilMetrics *Metrics
			nilMetrics.RecordValidation(ctx, duration, tt.success)
		})
	}
}

func TestGetGlobalMetrics(t *testing.T) {
	// Save original global metrics
	originalMetrics := globalMetrics
	defer func() {
		globalMetrics = originalMetrics
	}()

	// Reset global metrics for clean test
	globalMetrics = nil

	// First call should create metrics
	metrics1 := GetGlobalMetrics()
	if metrics1 == nil {
		t.Fatal("GetGlobalMetrics() returned nil")
	}

	// Second call should return the same instance
	metrics2 := GetGlobalMetrics()
	if metrics1 != metrics2 {
		t.Error("GetGlobalMetrics() should return the same instance")
	}
}

func TestSetGlobalMetrics(t *testing.T) {
	// Save original global metrics
	originalMetrics := globalMetrics
	defer func() {
		globalMetrics = originalMetrics
	}()

	// Create a test metrics instance
	testMetrics := NoOpMetrics()
	SetGlobalMetrics(testMetrics)

	// Verify that GetGlobalMetrics returns our test instance
	retrievedMetrics := GetGlobalMetrics()
	if retrievedMetrics != testMetrics {
		t.Error("SetGlobalMetrics() did not set the global metrics instance correctly")
	}

	// Test setting nil
	SetGlobalMetrics(nil)
	if GetGlobalMetrics() != nil {
		t.Error("SetGlobalMetrics(nil) should set global metrics to nil")
	}
}

func TestGlobalMetrics_NoOpFallback(t *testing.T) {
	// Save original state
	originalMetrics := globalMetrics
	originalExplicitlySet := globalMetricsExplicitlySet
	defer func() {
		globalMetrics = originalMetrics
		globalMetricsExplicitlySet = originalExplicitlySet
	}()

	// Reset global metrics and flag to test lazy initialization
	globalMetrics = nil
	globalMetricsExplicitlySet = false

	// Mock a scenario where NewMetrics might fail
	// Since we're using noop meter in tests, this should work fine
	// but in real scenarios with broken OTEL setup, it would fall back to NoOpMetrics

	metrics := GetGlobalMetrics()
	if metrics == nil {
		t.Error("GetGlobalMetrics() should not return nil even if NewMetrics fails")
	}
}

// Simple test using noop meter which is sufficient for our testing needs.
func TestMetrics_WithNoOpMeter(t *testing.T) {
	// Use noop meter which implements the full interface
	meter := noop.Meter{}

	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		t.Fatalf("NewMetrics() with noop meter error = %v", err)
	}

	// Test recording with noop metrics
	ctx := context.Background()
	duration := 200 * time.Millisecond

	// These should not panic
	metrics.RecordConfigLoad(ctx, duration, true, "test")
	metrics.RecordValidation(ctx, duration, false)
}

func TestMetrics_RecordConfigLoad_Attributes(t *testing.T) {
	// Test that the correct attributes are passed (this is more of a documentation test)
	// In a real implementation, you'd use a spy/mock to verify the attributes

	meter := noop.Meter{}
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	duration := 150 * time.Millisecond

	// Test various combinations of success/source
	testCases := []struct {
		source  string
		success bool
	}{
		{"file", true},
		{"env", false},
		{"default", true},
		{"remote", false},
	}

	for _, tc := range testCases {
		// Just ensure it doesn't panic
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		metrics.RecordConfigLoad(ctx, duration, tc.success, tc.source)
	}
}

func BenchmarkMetrics_RecordConfigLoad(b *testing.B) {
	meter := noop.Meter{}
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		b.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordConfigLoad(ctx, duration, true, "benchmark")
	}
}

func BenchmarkMetrics_RecordValidation(b *testing.B) {
	meter := noop.Meter{}
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		b.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 50 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordValidation(ctx, duration, true)
	}
}

func TestMetrics_Concurrency(t *testing.T) {
	// Add test timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	meter := noop.Meter{}
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	duration := 10 * time.Millisecond

	// Test concurrent access to metrics recording
	var wg sync.WaitGroup
	const numGoroutines = 10
	const iterationsPerGoroutine = 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				// Check context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}
				metrics.RecordConfigLoad(ctx, duration, id%2 == 0, "concurrency_test")
				metrics.RecordValidation(ctx, duration, id%2 == 1)
			}
		}(i)
	}

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed successfully
	case <-ctx.Done():
		t.Fatal("Test timed out waiting for goroutines to complete")
	}
}
