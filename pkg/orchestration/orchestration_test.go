package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRunnable is a simple implementation of core.Runnable for testing
type MockRunnable struct {
	name  string
	input any
}

func (m *MockRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	m.input = input
	return map[string]any{"output": m.name + "_result", "input": input}, nil
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, _ := m.Invoke(ctx, input, opts...)
		results[i] = result
	}
	return results, nil
}

func (m *MockRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, _ := m.Invoke(ctx, input, opts...)
		ch <- result
	}()
	return ch, nil
}

func TestNewOrchestrator(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	if orch == nil {
		t.Fatal("Orchestrator should not be nil")
	}
}

func TestChainCreation(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	steps := []core.Runnable{
		&MockRunnable{name: "step1"},
		&MockRunnable{name: "step2"},
	}

	chain, err := orch.CreateChain(steps)
	require.NoError(t, err)
	assert.NotNil(t, chain)

	// Test chain execution
	input := map[string]any{"test": "data"}
	result, err := chain.Invoke(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("Chain execution result: %v", result)
}

func TestGraphCreation(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	graph, err := orch.CreateGraph()
	require.NoError(t, err)
	assert.NotNil(t, graph)

	// Test graph with nodes
	step1 := &MockRunnable{name: "step1"}
	step2 := &MockRunnable{name: "step2"}

	err = graph.AddNode("processor", step1)
	require.NoError(t, err)

	err = graph.AddNode("validator", step2)
	require.NoError(t, err)

	err = graph.AddEdge("processor", "validator")
	require.NoError(t, err)

	err = graph.SetEntryPoint([]string{"processor"})
	require.NoError(t, err)

	err = graph.SetFinishPoint([]string{"validator"})
	require.NoError(t, err)

	// Test graph execution
	input := map[string]any{"data": "test"}
	result, err := graph.Invoke(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("Graph execution result: %v", result)
}

func TestConfiguration(t *testing.T) {
	config, err := NewConfig(
		WithChainTimeout(30),
		WithGraphMaxWorkers(5),
		WithWorkflowTaskQueue("test-queue"),
		WithMetricsPrefix("test.orchestration"),
		WithFeatures(EnabledFeatures{
			Chains:    true,
			Graphs:    true,
			Workflows: false,
		}),
	)

	require.NoError(t, err)

	assert.Equal(t, 30*time.Second, config.Chain.DefaultTimeout)
	assert.Equal(t, 5, config.Graph.MaxWorkers)
	assert.Equal(t, "test-queue", config.Workflow.TaskQueue)
	assert.Equal(t, "test.orchestration", config.Observability.MetricsPrefix)
}

func TestOrchestratorWithOptions(t *testing.T) {
	orch, err := NewOrchestratorWithOptions(
		WithChainTimeout(60),
		WithGraphMaxWorkers(10),
		WithWorkflowTaskQueue("custom-queue"),
		WithMetricsPrefix("custom.orchestration"),
	)

	require.NoError(t, err)
	assert.NotNil(t, orch)

	// Test that the orchestrator works with the custom configuration
	steps := []core.Runnable{&MockRunnable{name: "test"}}
	chain, err := orch.CreateChain(steps)
	require.NoError(t, err)
	assert.NotNil(t, chain)
}

// Table-driven tests for configuration validation
func TestConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		expectError bool
		errorCode   string
	}{
		{
			name: "valid configuration",
			config: &Config{
				Chain: ChainConfig{
					DefaultTimeout:      5 * time.Minute,
					DefaultRetries:      3,
					MaxConcurrentChains: 10,
				},
				Graph: GraphConfig{
					DefaultTimeout:          10 * time.Minute,
					DefaultRetries:          3,
					MaxWorkers:              5,
					EnableParallelExecution: true,
					QueueSize:               100,
				},
				Workflow: WorkflowConfig{
					DefaultTimeout:         30 * time.Minute,
					DefaultRetries:         5,
					TaskQueue:              "beluga-workflows",
					MaxConcurrentWorkflows: 50,
				},
			},
			expectError: false,
		},
		{
			name: "invalid chain max concurrent",
			config: &Config{
				Chain: ChainConfig{
					DefaultTimeout:      5 * time.Minute,
					DefaultRetries:      3,
					MaxConcurrentChains: 0, // Invalid
				},
				Graph: GraphConfig{
					DefaultTimeout:          10 * time.Minute,
					DefaultRetries:          3,
					MaxWorkers:              5,
					EnableParallelExecution: true,
					QueueSize:               100,
				},
				Workflow: WorkflowConfig{
					DefaultTimeout:         30 * time.Minute,
					DefaultRetries:         5,
					TaskQueue:              "beluga-workflows",
					MaxConcurrentWorkflows: 50,
				},
			},
			expectError: true,
			errorCode:   iface.ErrCodeInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectError {
				assert.Error(t, err)
				if err != nil {
					var orchErr *iface.OrchestratorError
					assert.ErrorAs(t, err, &orchErr)
					if orchErr != nil {
						assert.Equal(t, tc.errorCode, orchErr.Code)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test error handling and retry scenarios
func TestErrorHandlingScenarios(t *testing.T) {
	testCases := []struct {
		name            string
		operation       func() error
		expectRetryable bool
		expectedCode    string
	}{
		{
			name: "timeout error",
			operation: func() error {
				return iface.ErrTimeout("test.op", context.DeadlineExceeded)
			},
			expectRetryable: true,
			expectedCode:    iface.ErrCodeTimeout,
		},
		{
			name: "invalid config error",
			operation: func() error {
				return iface.ErrInvalidConfig("test.op", errors.New("invalid config"))
			},
			expectRetryable: false,
			expectedCode:    iface.ErrCodeInvalidConfig,
		},
		{
			name: "dependency failed error",
			operation: func() error {
				return iface.ErrDependencyFailed("test.op", "dependency1", errors.New("dependency failed"))
			},
			expectRetryable: true,
			expectedCode:    iface.ErrCodeDependencyFailed,
		},
		{
			name: "resource exhausted error",
			operation: func() error {
				return iface.ErrResourceExhausted("test.op", "memory")
			},
			expectRetryable: true,
			expectedCode:    iface.ErrCodeResourceExhausted,
		},
		{
			name: "invalid state error",
			operation: func() error {
				return iface.ErrInvalidState("test.op", "current", "expected")
			},
			expectRetryable: false,
			expectedCode:    iface.ErrCodeInvalidState,
		},
		{
			name: "not found error",
			operation: func() error {
				return iface.ErrNotFound("test.op", "resource")
			},
			expectRetryable: false,
			expectedCode:    iface.ErrCodeNotFound,
		},
		{
			name: "circuit breaker open error",
			operation: func() error {
				return iface.ErrCircuitBreakerOpen("test.op")
			},
			expectRetryable: true,
			expectedCode:    iface.ErrCodeCircuitBreakerOpen,
		},
		{
			name: "rate limit exceeded error",
			operation: func() error {
				return iface.ErrRateLimitExceeded("test.op", 100)
			},
			expectRetryable: true,
			expectedCode:    iface.ErrCodeRateLimitExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.operation()

			// Check if it's an OrchestratorError
			var orchErr *iface.OrchestratorError
			assert.ErrorAs(t, err, &orchErr)

			if orchErr != nil {
				assert.Equal(t, tc.expectedCode, orchErr.Code)
				assert.Equal(t, tc.expectRetryable, iface.IsRetryable(err))
			}
		})
	}
}

// Test comprehensive chain orchestration scenarios
func TestChainOrchestrationScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("empty chain", func(t *testing.T) {
		chain, err := orch.CreateChain([]core.Runnable{})
		require.NoError(t, err)

		result, err := chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("single step chain", func(t *testing.T) {
		step := &MockRunnable{name: "single"}
		chain, err := orch.CreateChain([]core.Runnable{step})
		require.NoError(t, err)

		result, err := chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("multi step chain with data flow", func(t *testing.T) {
		step1 := &MockRunnable{name: "processor"}
		step2 := &MockRunnable{name: "validator"}
		step3 := &MockRunnable{name: "formatter"}

		chain, err := orch.CreateChain([]core.Runnable{step1, step2, step3})
		require.NoError(t, err)

		input := map[string]any{"data": "initial"}
		result, err := chain.Invoke(context.Background(), input)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("chain with context cancellation", func(t *testing.T) {
		step := &MockRunnable{name: "slow"}
		chain, err := orch.CreateChain([]core.Runnable{step})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err = chain.Invoke(ctx, map[string]any{"input": "test"})
		// Context cancellation may result in error or nil depending on timing
		// The important thing is that it doesn't panic
		if err != nil {
			// If there's an error, it should be context-related
			assert.Contains(t, err.Error(), "context")
		}
	})

	t.Run("chain batch execution", func(t *testing.T) {
		step1 := &MockRunnable{name: "batch_processor"}
		step2 := &MockRunnable{name: "batch_validator"}

		chain, err := orch.CreateChain([]core.Runnable{step1, step2})
		require.NoError(t, err)

		inputs := []any{
			map[string]any{"batch": "item1"},
			map[string]any{"batch": "item2"},
			map[string]any{"batch": "item3"},
		}

		results, err := chain.Batch(context.Background(), inputs)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
		for _, result := range results {
			assert.NotNil(t, result)
		}
	})
}

// Test comprehensive graph orchestration scenarios
func TestGraphOrchestrationScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("linear graph", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		step1 := &MockRunnable{name: "start"}
		step2 := &MockRunnable{name: "middle"}
		step3 := &MockRunnable{name: "end"}

		err = graph.AddNode("start", step1)
		require.NoError(t, err)
		err = graph.AddNode("middle", step2)
		require.NoError(t, err)
		err = graph.AddNode("end", step3)
		require.NoError(t, err)

		err = graph.AddEdge("start", "middle")
		require.NoError(t, err)
		err = graph.AddEdge("middle", "end")
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"start"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"end"})
		require.NoError(t, err)

		result, err := graph.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("branched graph", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		start := &MockRunnable{name: "start"}
		branch1 := &MockRunnable{name: "branch1"}
		branch2 := &MockRunnable{name: "branch2"}
		merge := &MockRunnable{name: "merge"}

		// Add nodes
		for name, node := range map[string]core.Runnable{
			"start":   start,
			"branch1": branch1,
			"branch2": branch2,
			"merge":   merge,
		} {
			err = graph.AddNode(name, node)
			require.NoError(t, err)
		}

		// Add edges
		err = graph.AddEdge("start", "branch1")
		require.NoError(t, err)
		err = graph.AddEdge("start", "branch2")
		require.NoError(t, err)
		err = graph.AddEdge("branch1", "merge")
		require.NoError(t, err)
		err = graph.AddEdge("branch2", "merge")
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"start"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"merge"})
		require.NoError(t, err)

		result, err := graph.Invoke(context.Background(), map[string]any{"input": "branched"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("graph with multiple entry points", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		entry1 := &MockRunnable{name: "entry1"}
		entry2 := &MockRunnable{name: "entry2"}
		converge := &MockRunnable{name: "converge"}

		for name, node := range map[string]core.Runnable{
			"entry1":   entry1,
			"entry2":   entry2,
			"converge": converge,
		} {
			err = graph.AddNode(name, node)
			require.NoError(t, err)
		}

		err = graph.AddEdge("entry1", "converge")
		require.NoError(t, err)
		err = graph.AddEdge("entry2", "converge")
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"entry1", "entry2"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"converge"})
		require.NoError(t, err)

		result, err := graph.Invoke(context.Background(), map[string]any{"input": "multi-entry"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("graph with multiple exit points", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		start := &MockRunnable{name: "start"}
		path1 := &MockRunnable{name: "path1"}
		path2 := &MockRunnable{name: "path2"}

		for name, node := range map[string]core.Runnable{
			"start": start,
			"path1": path1,
			"path2": path2,
		} {
			err = graph.AddNode(name, node)
			require.NoError(t, err)
		}

		err = graph.AddEdge("start", "path1")
		require.NoError(t, err)
		err = graph.AddEdge("start", "path2")
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"start"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"path1", "path2"})
		require.NoError(t, err)

		result, err := graph.Invoke(context.Background(), map[string]any{"input": "multi-exit"})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Result should contain both exit node outputs
		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, resultMap, "path1")
		assert.Contains(t, resultMap, "path2")
	})
}

// Test orchestrator feature toggles
func TestOrchestratorFeatureToggles(t *testing.T) {
	t.Run("chains disabled", func(t *testing.T) {
		orch, err := NewOrchestratorWithOptions(
			WithFeatures(EnabledFeatures{
				Chains:    false,
				Graphs:    true,
				Workflows: true,
			}),
		)
		require.NoError(t, err)

		_, err = orch.CreateChain([]core.Runnable{&MockRunnable{name: "test"}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "chains_disabled")
	})

	t.Run("graphs disabled", func(t *testing.T) {
		orch, err := NewOrchestratorWithOptions(
			WithFeatures(EnabledFeatures{
				Chains:    true,
				Graphs:    false,
				Workflows: true,
			}),
		)
		require.NoError(t, err)

		_, err = orch.CreateGraph()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "graphs_disabled")
	})

	t.Run("workflows disabled", func(t *testing.T) {
		orch, err := NewOrchestratorWithOptions(
			WithFeatures(EnabledFeatures{
				Chains:    true,
				Graphs:    true,
				Workflows: false,
			}),
		)
		require.NoError(t, err)

		_, err = orch.CreateWorkflow(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflows_disabled")
	})
}

// Test orchestrator metrics
func TestOrchestratorMetrics(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Initially no active orchestrations
	metrics := orch.GetMetrics()
	assert.Equal(t, 0, metrics.GetActiveChains())
	assert.Equal(t, 0, metrics.GetActiveGraphs())
	assert.Equal(t, 0, metrics.GetActiveWorkflows())

	// Create some chains
	chain1, err := orch.CreateChain([]core.Runnable{&MockRunnable{name: "test1"}})
	require.NoError(t, err)
	assert.NotNil(t, chain1)

	chain2, err := orch.CreateChain([]core.Runnable{&MockRunnable{name: "test2"}})
	require.NoError(t, err)
	assert.NotNil(t, chain2)

	// Create a graph
	graph, err := orch.CreateGraph()
	require.NoError(t, err)
	assert.NotNil(t, graph)

	// Check metrics updated
	metrics = orch.GetMetrics()
	assert.Equal(t, 2, metrics.GetActiveChains())
	assert.Equal(t, 1, metrics.GetActiveGraphs())
}

// Test orchestrator health checks
func TestOrchestratorHealthChecks(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Health check should pass with default orchestrator
	err = orch.Check(context.Background())
	assert.NoError(t, err)
}

// Test concurrent orchestration
func TestConcurrentOrchestration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	const numGoroutines = 10
	const chainsPerGoroutine = 5

	var wg sync.WaitGroup
	results := make(chan error, numGoroutines*chainsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < chainsPerGoroutine; j++ {
				chain, err := orch.CreateChain([]core.Runnable{
					&MockRunnable{name: fmt.Sprintf("step-%d-%d", id, j)},
				})
				if err != nil {
					results <- err
					continue
				}

				_, err = chain.Invoke(context.Background(), map[string]any{"input": "concurrent"})
				results <- err
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Check all operations succeeded
	for err := range results {
		assert.NoError(t, err)
	}
}

// Test orchestrator configuration edge cases
func TestOrchestratorConfigurationEdgeCases(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		_, err := NewOrchestrator(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("invalid timeout values", func(t *testing.T) {
		_, err := NewConfig(WithChainTimeout(-1))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("invalid worker count", func(t *testing.T) {
		_, err := NewConfig(WithGraphMaxWorkers(0))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max workers must be >= 1")
	})

	t.Run("empty task queue", func(t *testing.T) {
		_, err := NewConfig(WithWorkflowTaskQueue(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task queue cannot be empty")
	})

	t.Run("empty metrics prefix", func(t *testing.T) {
		_, err := NewConfig(WithMetricsPrefix(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metrics prefix cannot be empty")
	})
}

// Test orchestrator with various input types
func TestOrchestratorInputTypes(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	step := &MockRunnable{name: "input_tester"}
	chain, err := orch.CreateChain([]core.Runnable{step})
	require.NoError(t, err)

	testCases := []struct {
		name  string
		input any
	}{
		{"map input", map[string]any{"key": "value"}},
		{"string input", "string input"},
		{"int input", 42},
		{"slice input", []string{"item1", "item2"}},
		{"struct input", struct{ Field string }{"value"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := chain.Invoke(context.Background(), tc.input)
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// Test orchestrator performance characteristics
func TestOrchestratorPerformance(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create a moderately complex chain
	steps := make([]core.Runnable, 10)
	for i := range steps {
		steps[i] = &MockRunnable{name: fmt.Sprintf("perf-step-%d", i)}
	}

	chain, err := orch.CreateChain(steps)
	require.NoError(t, err)

	input := map[string]any{"performance": "test"}

	// Measure execution time
	start := time.Now()
	result, err := chain.Invoke(context.Background(), input)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should complete within reasonable time (adjust based on system)
	assert.True(t, duration < 1*time.Second, "Chain execution took too long: %v", duration)
}

// Test orchestrator cleanup and resource management
func TestOrchestratorCleanup(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create multiple orchestrations
	chains := make([]iface.Chain, 5)
	graphs := make([]iface.Graph, 3)

	for i := range chains {
		chains[i], err = orch.CreateChain([]core.Runnable{&MockRunnable{name: fmt.Sprintf("chain-%d", i)}})
		require.NoError(t, err)
	}

	for i := range graphs {
		graphs[i], err = orch.CreateGraph()
		require.NoError(t, err)
	}

	// Verify they're tracked
	metrics := orch.GetMetrics()
	assert.Equal(t, len(chains), metrics.GetActiveChains())
	assert.Equal(t, len(graphs), metrics.GetActiveGraphs())

	// Note: In a real implementation, we'd test cleanup methods
	// but the current orchestrator doesn't expose explicit cleanup
}
