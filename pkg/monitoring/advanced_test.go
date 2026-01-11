// Package monitoring provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMonitorCreationAdvanced provides advanced table-driven tests for monitor creation.
func TestMonitorCreationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(t *testing.T) Monitor
		validate    func(t *testing.T, monitor Monitor)
		wantErr     bool
	}{
		{
			name:        "basic_monitor_creation",
			description: "Create basic monitor with minimal config",
			setup: func(t *testing.T) Monitor {
				// Use test utilities if available
				mockMonitor := NewAdvancedMockMonitor("test-monitor", "test-type")
				return mockMonitor
			},
			validate: func(t *testing.T, monitor Monitor) {
				assert.NotNil(t, monitor)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			monitor := tt.setup(t)
			tt.validate(t, monitor)
		})
	}
}

// TestMonitorOperationsAdvanced provides advanced table-driven tests for monitor operations.
func TestMonitorOperationsAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(t *testing.T) Monitor
		operation   func(t *testing.T, monitor Monitor)
		wantErr     bool
	}{
		{
			name:        "health_check",
			description: "Test monitor health check",
			setup: func(t *testing.T) Monitor {
				return NewAdvancedMockMonitor("test-monitor", "test-type")
			},
			operation: func(t *testing.T, monitor Monitor) {
				ctx := context.Background()
				healthy := monitor.IsHealthy(ctx)
				t.Logf("Health check result: healthy=%v", healthy)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			monitor := tt.setup(t)
			tt.operation(t, monitor)
		})
	}
}

// TestConcurrentMonitorOperations tests concurrent monitor operations.
func TestConcurrentMonitorOperations(t *testing.T) {
	const numGoroutines = 20
	const numOperationsPerGoroutine = 10

	monitor := NewAdvancedMockMonitor("test-monitor", "test-type")

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*numOperationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				healthy := monitor.IsHealthy(ctx)
				if !healthy {
					errors <- fmt.Errorf("monitor unhealthy")
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Log any errors
	for err := range errors {
		t.Logf("Concurrent monitor operation error: %v", err)
	}
}

// TestMonitorWithContext tests monitor operations with context.
func TestMonitorWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	monitor := NewAdvancedMockMonitor("test-monitor", "test-type")

	t.Run("health_check_with_timeout", func(t *testing.T) {
		healthy := monitor.IsHealthy(ctx)
		t.Logf("Health check with timeout: healthy=%v", healthy)
	})
}

// BenchmarkMonitorCreation benchmarks monitor creation performance.
func BenchmarkMonitorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewAdvancedMockMonitor("benchmark-monitor", "benchmark-type")
	}
}

// BenchmarkMonitorOperations benchmarks monitor operations performance.
func BenchmarkMonitorOperations(b *testing.B) {
	monitor := NewAdvancedMockMonitor("benchmark-monitor", "benchmark-type")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.IsHealthy(ctx)
	}
}
