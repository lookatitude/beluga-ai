// Package core provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates constitutional testing practices including table-driven tests,
// concurrency testing, performance benchmarks, and integration test patterns.
// T007: Create advanced_test.go with table-driven tests, concurrency tests, and comprehensive benchmarks
package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T008: Benchmark tests for DI resolution (target <1ms) and Runnable operations (target <100μs)

// BenchmarkContainerOperations tests the performance of DI container operations
func BenchmarkContainerOperations(b *testing.B) {
	tests := []struct {
		name      string
		operation func(container Container) func() error
	}{
		{
			name: "Register",
			operation: func(container Container) func() error {
				return func() error {
					return container.Register(func() string { return "test" })
				}
			},
		},
		{
			name: "Resolve",
			operation: func(container Container) func() error {
				// Pre-register for resolve test
				container.Register(func() string { return "test" })
				return func() error {
					var result string
					return container.Resolve(&result)
				}
			},
		},
		{
			name: "Has",
			operation: func(container Container) func() error {
				stringType := reflect.TypeOf("")
				return func() error {
					container.Has(stringType)
					return nil
				}
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			container := NewContainer()
			op := tt.operation(container)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if err := op(); err != nil {
					b.Fatalf("Operation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkRunnableOperations tests the performance of Runnable operations
func BenchmarkRunnableOperations(b *testing.B) {
	mockRunnable := NewAdvancedMockRunnable("benchmark",
		WithMockResponses([]any{"benchmark result"}))

	ctx := context.Background()
	testInput := "benchmark input"

	tests := []struct {
		name      string
		operation func() error
	}{
		{
			name: "Invoke",
			operation: func() error {
				_, err := mockRunnable.Invoke(ctx, testInput)
				return err
			},
		},
		{
			name: "Batch",
			operation: func() error {
				inputs := []any{testInput}
				_, err := mockRunnable.Batch(ctx, inputs)
				return err
			},
		},
		{
			name: "Stream",
			operation: func() error {
				ch, err := mockRunnable.Stream(ctx, testInput)
				if err != nil {
					return err
				}
				// Read from channel
				<-ch
				return nil
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if err := tt.operation(); err != nil {
					b.Fatalf("Operation failed: %v", err)
				}
			}
		})
	}
}

// T013: Performance benchmark tests for DI container operations
func BenchmarkDIContainerPerformance(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping performance benchmarks in short mode")
	}

	container := NewContainer()

	// Pre-register some factories for resolve benchmarks
	container.Register(func() string { return "test_string" })
	container.Register(func() int { return 42 })
	container.Register(func() iface.Runnable {
		return NewAdvancedMockRunnable("test")
	})

	b.Run("Registration", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			localContainer := NewContainer()
			err := localContainer.Register(func() string {
				return fmt.Sprintf("test_%d", i)
			})
			if err != nil {
				b.Fatal("Registration failed:", err)
			}
		}
	})

	b.Run("Resolution", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var result string
			err := container.Resolve(&result)
			if err != nil {
				b.Fatal("Resolution failed:", err)
			}
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := container.CheckHealth(ctx)
			if err != nil {
				b.Fatal("Health check failed:", err)
			}
		}
	})
}

// T014: Concurrency tests for thread-safe Container and Runnable operations
func TestConcurrentContainerOperations(t *testing.T) {
	tests := []struct {
		name        string
		workers     int
		operations  int
		operation   func(container Container, workerID, opID int) error
		description string
	}{
		{
			name:        "ConcurrentRegistration",
			workers:     10,
			operations:  100,
			description: "Multiple goroutines registering different factories",
			operation: func(container Container, workerID, opID int) error {
				return container.Register(func() string {
					return fmt.Sprintf("worker_%d_op_%d", workerID, opID)
				})
			},
		},
		{
			name:        "ConcurrentResolution",
			workers:     8,
			operations:  50,
			description: "Multiple goroutines resolving the same type",
			operation: func(container Container, workerID, opID int) error {
				var result string
				return container.Resolve(&result)
			},
		},
		{
			name:        "ConcurrentHealthCheck",
			workers:     5,
			operations:  20,
			description: "Multiple goroutines performing health checks",
			operation: func(container Container, workerID, opID int) error {
				ctx := context.Background()
				return container.CheckHealth(ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()

			// Pre-register for resolution tests
			if tt.name == "ConcurrentResolution" || tt.name == "ConcurrentHealthCheck" {
				err := container.Register(func() string { return "concurrent_test" })
				require.NoError(t, err)
			}

			runner := NewConcurrentTestRunner(tt.workers, tt.operations, 10*time.Second)

			runner.Run(t, func(workerID, operationID int) error {
				return tt.operation(container, workerID, operationID)
			})
		})
	}
}

// T015: Load testing for core package scalability
func TestCorePackageLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	t.Run("ContainerScalability", func(t *testing.T) {
		// Test with increasing number of registrations
		scalabilityLevels := []int{100, 500, 1000, 2000}

		for _, numRegistrations := range scalabilityLevels {
			t.Run(fmt.Sprintf("Registrations_%d", numRegistrations), func(t *testing.T) {
				localContainer := NewContainer()

				// Register many factories
				start := time.Now()
				for i := 0; i < numRegistrations; i++ {
					err := localContainer.Register(func(id int) func() string {
						return func() string { return fmt.Sprintf("service_%d", id) }
					}(i))
					require.NoError(t, err)
				}
				registrationTime := time.Since(start)

				// Test resolution performance with many registrations
				var result string
				start = time.Now()
				err := localContainer.Resolve(&result)
				resolutionTime := time.Since(start)

				require.NoError(t, err)

				t.Logf("Registrations: %d, Registration time: %v, Resolution time: %v",
					numRegistrations, registrationTime, resolutionTime)

				// Verify performance targets
				avgRegistrationTime := registrationTime / time.Duration(numRegistrations)
				assert.Less(t, avgRegistrationTime, time.Millisecond,
					"Average registration time should be under 1ms")
				assert.Less(t, resolutionTime, time.Millisecond,
					"Resolution time should be under 1ms")
			})
		}
	})

	t.Run("RunnableLoadTest", func(t *testing.T) {
		mockRunnable := NewAdvancedMockRunnable("load_test")

		// Test with increasing load
		loadLevels := []int{100, 500, 1000}

		for _, operations := range loadLevels {
			t.Run(fmt.Sprintf("Operations_%d", operations), func(t *testing.T) {
				ctx := context.Background()

				start := time.Now()
				for i := 0; i < operations; i++ {
					_, err := mockRunnable.Invoke(ctx, fmt.Sprintf("load_test_%d", i))
					require.NoError(t, err)
				}
				elapsed := time.Since(start)

				avgTime := elapsed / time.Duration(operations)
				opsPerSec := float64(operations) / elapsed.Seconds()

				t.Logf("Operations: %d, Total time: %v, Avg time: %v, Ops/sec: %.2f",
					operations, elapsed, avgTime, opsPerSec)

				// Verify performance targets
				assert.Less(t, avgTime, 100*time.Microsecond,
					"Average operation time should be under 100μs")
				assert.Greater(t, opsPerSec, 10000.0,
					"Should achieve >10,000 ops/sec")
			})
		}
	})
}

// Advanced table-driven test patterns following constitutional requirements

// TestAdvancedContainerScenarios provides comprehensive table-driven tests for Container functionality
func TestAdvancedContainerScenarios(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func() Container
		operations  func(t *testing.T, container Container)
		expectError bool
		cleanup     func(Container)
	}{
		{
			name:        "complex_dependency_graph",
			description: "Tests resolution of complex interdependent services",
			setup:       func() Container { return NewContainer() },
			operations: func(t *testing.T, container Container) {
				// Register interdependent services
				err := container.Register(func() string { return "config_service" })
				require.NoError(t, err)

				err = container.Register(func(config string) int { return 42 })
				require.NoError(t, err)

				err = container.Register(func(config string, number int) iface.Runnable {
					return NewAdvancedMockRunnable(fmt.Sprintf("service_%s_%d", config, number))
				})
				require.NoError(t, err)

				// Resolve complex dependency
				var result iface.Runnable
				err = container.Resolve(&result)
				require.NoError(t, err)
				assert.NotNil(t, result)
			},
		},
		{
			name:        "singleton_vs_factory",
			description: "Tests behavior difference between singletons and factories",
			setup:       func() Container { return NewContainer() },
			operations: func(t *testing.T, container Container) {
				// Register factory
				counter := 0
				err := container.Register(func() int {
					counter++
					return counter
				})
				require.NoError(t, err)

				// Resolve twice (should create new instances)
				var result1, result2 int
				err = container.Resolve(&result1)
				require.NoError(t, err)
				err = container.Resolve(&result2)
				require.NoError(t, err)

				assert.Equal(t, 1, result1)
				assert.Equal(t, 2, result2)

				// Register singleton
				container.Singleton("singleton_value")

				// Resolve singleton twice (should return same instance)
				var single1, single2 string
				err = container.Resolve(&single1)
				require.NoError(t, err)
				err = container.Resolve(&single2)
				require.NoError(t, err)

				assert.Equal(t, "singleton_value", single1)
				assert.Equal(t, "singleton_value", single2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := tt.setup()
			tt.operations(t, container)

			if tt.cleanup != nil {
				tt.cleanup(container)
			}
		})
	}
}

// TestAdvancedRunnableScenarios provides comprehensive tests for Runnable implementations
func TestAdvancedRunnableScenarios(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setupMock   func() *AdvancedMockRunnable
		operations  func(t *testing.T, runnable iface.Runnable)
		expectError bool
	}{
		{
			name:        "standard_invoke_flow",
			description: "Tests standard invoke operation with various inputs",
			setupMock: func() *AdvancedMockRunnable {
				return NewAdvancedMockRunnable("standard",
					WithMockResponses([]any{"response1", "response2", "response3"}))
			},
			operations: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()

				// Test multiple invokes
				for i := 0; i < 3; i++ {
					result, err := runnable.Invoke(ctx, fmt.Sprintf("input_%d", i+1))
					require.NoError(t, err)
					expected := fmt.Sprintf("response%d", i+1)
					assert.Equal(t, expected, result)
				}
			},
		},
		{
			name:        "batch_processing",
			description: "Tests batch processing with multiple inputs",
			setupMock: func() *AdvancedMockRunnable {
				return NewAdvancedMockRunnable("batch",
					WithMockResponses([]any{"batch_result"}))
			},
			operations: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()
				inputs := []any{"input1", "input2", "input3"}

				results, err := runnable.Batch(ctx, inputs)
				require.NoError(t, err)
				assert.Len(t, results, 3)

				// All results should be the batch result
				for _, result := range results {
					assert.Equal(t, "batch_result", result)
				}
			},
		},
		{
			name:        "streaming_operations",
			description: "Tests streaming functionality",
			setupMock: func() *AdvancedMockRunnable {
				return NewAdvancedMockRunnable("streaming",
					WithMockResponses([]any{"stream_chunk"}))
			},
			operations: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()

				ch, err := runnable.Stream(ctx, "stream_input")
				require.NoError(t, err)
				assert.NotNil(t, ch)

				// Read from stream
				select {
				case result := <-ch:
					assert.Equal(t, "stream_chunk", result)
				case <-time.After(time.Second):
					t.Error("Stream should produce result within reasonable time")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunnable := tt.setupMock()
			tt.operations(t, mockRunnable)
		})
	}
}

// TestConcurrentRunnableOperations tests Runnable operations under concurrent load
func TestConcurrentRunnableOperations(t *testing.T) {
	mockRunnable := NewAdvancedMockRunnable("concurrent",
		WithMockResponses([]any{"concurrent_result"}))

	runner := NewConcurrentTestRunner(8, 50, 5*time.Second)

	t.Run("ConcurrentInvoke", func(t *testing.T) {
		runner.Run(t, func(workerID, operationID int) error {
			ctx := context.Background()
			input := fmt.Sprintf("worker_%d_op_%d", workerID, operationID)
			_, err := mockRunnable.Invoke(ctx, input)
			return err
		})
	})

	t.Run("ConcurrentBatch", func(t *testing.T) {
		runner.Run(t, func(workerID, operationID int) error {
			ctx := context.Background()
			inputs := []any{fmt.Sprintf("batch_input_%d_%d", workerID, operationID)}
			_, err := mockRunnable.Batch(ctx, inputs)
			return err
		})
	})

	t.Run("ConcurrentStream", func(t *testing.T) {
		runner.Run(t, func(workerID, operationID int) error {
			ctx := context.Background()
			input := fmt.Sprintf("stream_input_%d_%d", workerID, operationID)
			ch, err := mockRunnable.Stream(ctx, input)
			if err != nil {
				return err
			}

			// Read from channel with timeout
			select {
			case <-ch:
				return nil
			case <-time.After(100 * time.Millisecond):
				return fmt.Errorf("stream timeout")
			}
		})
	})
}

// TestAdvancedErrorHandling tests comprehensive error scenarios
func TestAdvancedErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		description string
		operation   func() error
		expectCode  string
		expectType  ErrorType
		expectRetry bool
	}{
		{
			name:        "validation_error",
			description: "Tests validation error handling",
			operation: func() error {
				return NewFrameworkErrorWithCode(ErrorTypeValidation, "test_validation", "Validation failed", fmt.Errorf("Validation failed"))
			},
			expectCode:  "test_validation",
			expectType:  ErrorTypeValidation,
			expectRetry: false,
		},
		{
			name:        "network_error",
			description: "Tests network error handling",
			operation: func() error {
				return NewFrameworkErrorWithCode(ErrorTypeNetwork, "test_network", "Network connection failed", fmt.Errorf("Network connection failed"))
			},
			expectCode:  "test_network",
			expectType:  ErrorTypeNetwork,
			expectRetry: true,
		},
		{
			name:        "internal_error",
			description: "Tests internal error handling",
			operation: func() error {
				return NewFrameworkErrorWithCode(ErrorTypeInternal, "test_internal", "Internal system error", fmt.Errorf("Internal system error"))
			},
			expectCode:  "test_internal",
			expectType:  ErrorTypeInternal,
			expectRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			require.Error(t, err)

			var frameworkErr *FrameworkError
			ok := AsFrameworkError(err, &frameworkErr)
			require.True(t, ok)
			require.NotNil(t, frameworkErr)

			assert.Equal(t, tt.expectCode, frameworkErr.Code)
			assert.Equal(t, tt.expectType, frameworkErr.Type)
		})
	}
}

// Performance regression tests to validate targets are met
func TestPerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	// Test DI container performance target (<1ms)
	t.Run("DIContainerPerformance", func(t *testing.T) {
		runner := NewPerformanceTestRunner("DI Resolution", time.Millisecond, 1000)

		container := NewContainer()
		container.Register(func() string { return "performance_test" })

		runner.Run(t, func() error {
			var result string
			return container.Resolve(&result)
		})
	})

	// Test Runnable operation performance target (<100μs)
	t.Run("RunnablePerformance", func(t *testing.T) {
		runner := NewPerformanceTestRunner("Runnable Invoke", 100*time.Microsecond, 10000)

		mockRunnable := NewAdvancedMockRunnable("performance")
		ctx := context.Background()

		runner.Run(t, func() error {
			_, err := mockRunnable.Invoke(ctx, "performance_input")
			return err
		})
	})

	// Test overall throughput target (10,000+ ops/sec)
	t.Run("ThroughputTarget", func(t *testing.T) {
		mockRunnable := NewAdvancedMockRunnable("throughput")
		ctx := context.Background()

		operations := 10000
		start := time.Now()

		for i := 0; i < operations; i++ {
			_, err := mockRunnable.Invoke(ctx, "throughput_test")
			require.NoError(t, err)
		}

		elapsed := time.Since(start)
		opsPerSec := float64(operations) / elapsed.Seconds()

		t.Logf("Achieved throughput: %.2f ops/sec", opsPerSec)
		assert.Greater(t, opsPerSec, 10000.0, "Should achieve >10,000 ops/sec")
	})
}

// TestOptionsAndConfiguration tests the Option interface and configuration patterns
func TestOptionsAndConfiguration(t *testing.T) {
	t.Run("OptionApplication", func(t *testing.T) {
		config := make(map[string]any)

		opt1 := iface.OptionFunc(func(cfg *map[string]any) {
			(*cfg)["temperature"] = 0.7
		})

		opt2 := iface.OptionFunc(func(cfg *map[string]any) {
			(*cfg)["max_tokens"] = 1000
		})

		opt1.Apply(&config)
		opt2.Apply(&config)

		assert.Equal(t, 0.7, config["temperature"])
		assert.Equal(t, 1000, config["max_tokens"])
	})

	t.Run("WithOptionFactory", func(t *testing.T) {
		config := make(map[string]any)

		opt := WithOption("test_key", "test_value")
		opt.Apply(&config)

		assert.Equal(t, "test_value", config["test_key"])
	})
}

// TestHealthChecking tests health monitoring functionality
func TestHealthChecking(t *testing.T) {
	t.Run("ContainerHealthCheck", func(t *testing.T) {
		container := NewContainer()
		ctx := context.Background()

		// Healthy container
		err := container.CheckHealth(ctx)
		assert.NoError(t, err, "Healthy container should pass health check")

		// Container with registrations
		container.Register(func() string { return "health_test" })
		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container with registrations should be healthy")
	})

	t.Run("RunnableHealthCheck", func(t *testing.T) {
		// Test with health-aware mocks
		healthyMock := NewAdvancedMockRunnable("healthy")

		// Basic functionality test as health indicator
		ctx := context.Background()
		_, err := healthyMock.Invoke(ctx, "health_test")
		assert.NoError(t, err, "Healthy runnable should work without error")
	})
}

// TestMockUtilities tests the testing utilities themselves
func TestMockUtilities(t *testing.T) {
	t.Run("AdvancedMockRunnable", func(t *testing.T) {
		mock := NewAdvancedMockRunnable("test_mock",
			WithMockResponses([]any{"response1", "response2"}),
			WithMockDelay(time.Millisecond))

		ctx := context.Background()

		// Test configured responses
		result1, err := mock.Invoke(ctx, "test1")
		assert.NoError(t, err)
		assert.Equal(t, "response1", result1)

		result2, err := mock.Invoke(ctx, "test2")
		assert.NoError(t, err)
		assert.Equal(t, "response2", result2)

		// Test call count tracking
		assert.Equal(t, 2, mock.GetCallCount())
	})

	t.Run("AdvancedMockContainer", func(t *testing.T) {
		mock := NewAdvancedMockContainer()

		// Test registration
		err := mock.Register(func() string { return "test" })
		assert.NoError(t, err)

		// Test resolution
		var result string
		err = mock.Resolve(&result)
		assert.NoError(t, err)

		// Test call count tracking
		assert.Greater(t, mock.GetCallCount(), 0)
	})

	t.Run("ConcurrentTestRunner", func(t *testing.T) {
		runner := NewConcurrentTestRunner(4, 20, 5*time.Second)

		counter := 0
		var mu sync.Mutex

		runner.Run(t, func(workerID, operationID int) error {
			mu.Lock()
			counter++
			mu.Unlock()
			return nil
		})

		assert.Equal(t, 20, counter)
	})
}
