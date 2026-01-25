// Package orchestration provides comprehensive tests for orchestration implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
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

// TestAdvancedMockOrchestrator tests the advanced mock orchestrator functionality.
func TestAdvancedMockOrchestrator(t *testing.T) {
	tests := []struct {
		input          any
		orchestrator   *AdvancedMockOrchestrator
		name           string
		expectedResult string
		expectedError  bool
	}{
		{
			name: "successful execution",
			orchestrator: NewAdvancedMockOrchestrator("test-orch", "chain",
				WithMockResponses([]any{"success result"})),
			input:          "test input",
			expectedError:  false,
			expectedResult: "success result",
		},
		{
			name: "error execution",
			orchestrator: NewAdvancedMockOrchestrator("error-orch", "chain",
				WithMockError(true, errors.New("test error"))),
			input:         "test input",
			expectedError: true,
		},
		{
			name: "execution with delay",
			orchestrator: NewAdvancedMockOrchestrator("delay-orch", "chain",
				WithExecutionDelay(10*time.Millisecond),
				WithMockResponses([]any{"delayed result"})),
			input:          "test input",
			expectedError:  false,
			expectedResult: "delayed result",
		},
		{
			name: "multiple responses",
			orchestrator: NewAdvancedMockOrchestrator("multi-orch", "chain",
				WithMockResponses([]any{"result1", "result2", "result3"})),
			input:          "test input",
			expectedError:  false,
			expectedResult: "result1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			start := time.Now()
			result, err := tt.orchestrator.Execute(ctx, tt.input)
			duration := time.Since(start)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			// Verify call count
			assert.Equal(t, 1, tt.orchestrator.GetCallCount())

			// Verify execution delay was respected (with some tolerance)
			if tt.orchestrator.executionDelay > 0 {
				assert.GreaterOrEqual(t, duration, tt.orchestrator.executionDelay)
			}
		})
	}
}

// TestChainExecution tests chain execution scenarios.
func TestChainExecution(t *testing.T) {
	tests := []struct {
		name          string
		chainName     string
		steps         []string
		expectedError bool
	}{
		{
			name:          "simple chain",
			chainName:     "simple",
			steps:         []string{"step1", "step2"},
			expectedError: false,
		},
		{
			name:          "complex chain",
			chainName:     "complex",
			steps:         []string{"step1", "step2", "step3", "step4"},
			expectedError: false,
		},
		{
			name:          "empty chain",
			chainName:     "empty",
			steps:         []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := CreateTestChain(tt.chainName, tt.steps)
			ctx := context.Background()

			result, err := chain.Invoke(ctx, "test input")

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, result.(string), tt.chainName)
				assert.Contains(t, result.(string), fmt.Sprintf("%d steps", len(tt.steps)))
			}

			// Test batch execution
			batchResults, err := chain.Batch(ctx, []any{"input1", "input2"})
			require.NoError(t, err)
			assert.Len(t, batchResults, 2)

			// Test stream execution
			streamCh, err := chain.Stream(ctx, "stream input")
			require.NoError(t, err)

			select {
			case streamResult := <-streamCh:
				assert.NotNil(t, streamResult)
			case <-time.After(100 * time.Millisecond):
				t.Error("Stream timeout")
			}

			// Test interface methods
			assert.Equal(t, []string{"input"}, chain.GetInputKeys())
			assert.Equal(t, []string{"output"}, chain.GetOutputKeys())
			assert.Nil(t, chain.GetMemory())
		})
	}
}

// TestGraphExecution tests graph execution scenarios.
func TestGraphExecution(t *testing.T) {
	tests := []struct {
		edges         map[string][]string
		name          string
		graphName     string
		nodes         []string
		expectedError bool
	}{
		{
			name:      "simple graph",
			graphName: "simple",
			nodes:     []string{"node1", "node2"},
			edges:     map[string][]string{"node1": {"node2"}},
		},
		{
			name:      "complex graph",
			graphName: "complex",
			nodes:     []string{"start", "middle1", "middle2", "end"},
			edges: map[string][]string{
				"start":   {"middle1", "middle2"},
				"middle1": {"end"},
				"middle2": {"end"},
			},
		},
		{
			name:      "single node graph",
			graphName: "single",
			nodes:     []string{"only"},
			edges:     map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := CreateTestGraph(tt.graphName, tt.nodes, tt.edges)
			ctx := context.Background()

			result, err := graph.Invoke(ctx, "test input")

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, result.(string), tt.graphName)
				assert.Contains(t, result.(string), fmt.Sprintf("%d nodes", len(tt.nodes)))
			}

			// Test batch execution
			batchResults, err := graph.Batch(ctx, []any{"input1", "input2"})
			require.NoError(t, err)
			assert.Len(t, batchResults, 2)

			// Test stream execution
			streamCh, err := graph.Stream(ctx, "stream input")
			require.NoError(t, err)

			select {
			case streamResult := <-streamCh:
				assert.NotNil(t, streamResult)
			case <-time.After(100 * time.Millisecond):
				t.Error("Stream timeout")
			}

			// Test graph-specific operations
			err = graph.AddNode("new_node", nil)
			require.NoError(t, err)

			err = graph.AddEdge("node1", "new_node")
			require.NoError(t, err)

			err = graph.SetEntryPoint([]string{"node1"})
			require.NoError(t, err)

			err = graph.SetFinishPoint([]string{"new_node"})
			require.NoError(t, err)
		})
	}
}

// TestWorkflowExecution tests workflow execution scenarios.
func TestWorkflowExecution(t *testing.T) {
	tests := []struct {
		name          string
		workflowName  string
		tasks         []string
		expectedError bool
	}{
		{
			name:         "simple workflow",
			workflowName: "simple",
			tasks:        []string{"task1", "task2"},
		},
		{
			name:         "complex workflow",
			workflowName: "complex",
			tasks:        []string{"init", "process", "validate", "finalize"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := CreateTestWorkflow(tt.workflowName, tt.tasks)
			ctx := context.Background()

			// Test workflow execution
			workflowID, runID, err := workflow.Execute(ctx, "test input")

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, workflowID)
				assert.NotEmpty(t, runID)
				assert.Contains(t, workflowID, tt.workflowName)
			}

			if !tt.expectedError {
				// Test workflow result retrieval
				result, err := workflow.GetResult(ctx, workflowID, runID)
				require.NoError(t, err)
				assert.Contains(t, result.(string), workflowID)

				// Test workflow signaling
				err = workflow.Signal(ctx, workflowID, runID, "test_signal", "signal_data")
				require.NoError(t, err)

				// Test workflow querying
				queryResult, err := workflow.Query(ctx, workflowID, runID, "status")
				require.NoError(t, err)
				assert.NotNil(t, queryResult)

				// Test workflow cancellation
				err = workflow.Cancel(ctx, workflowID, runID)
				require.NoError(t, err)

				// Test workflow termination
				err = workflow.Terminate(ctx, workflowID, runID, "test termination")
				require.NoError(t, err)
			}
		})
	}
}

// TestConcurrencyAdvanced tests concurrent execution scenarios.
func TestConcurrencyAdvanced(t *testing.T) {
	orchestrator := NewAdvancedMockOrchestrator("concurrent-test", "chain",
		WithMockResponses([]any{"concurrent result"}))

	const numGoroutines = 10
	const numRequests = 5

	t.Run("concurrent_execution", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan any, numGoroutines*numRequests)
		errors := make(chan error, numGoroutines*numRequests)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < numRequests; j++ {
					ctx := context.Background()
					result, err := orchestrator.Execute(ctx, fmt.Sprintf("input-%d-%d", goroutineID, j))

					if err != nil {
						errors <- err
					} else {
						results <- result
					}
				}
			}(i)
		}

		wg.Wait()
		close(results)
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent execution error: %v", err)
		}

		// Verify results
		resultCount := 0
		for result := range results {
			assert.Equal(t, "concurrent result", result)
			resultCount++
		}

		assert.Equal(t, numGoroutines*numRequests, resultCount)
		assert.Equal(t, numGoroutines*numRequests, orchestrator.GetCallCount())
	})
}

// TestLoadTesting performs load testing on orchestration components.
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	orchestrator := NewAdvancedMockOrchestrator("load-test", "chain",
		WithMockResponses([]any{"load result"}))

	const numRequests = 100
	const concurrency = 10

	t.Run("load_test", func(t *testing.T) {
		RunLoadTest(t, orchestrator, numRequests, concurrency)

		// Verify health after load test
		health := orchestrator.CheckHealth()
		AssertHealthCheck(t, health, "healthy")
		assert.Equal(t, numRequests, health["call_count"])
	})
}

// TestMetricsRecording tests metrics recording functionality.
func TestMetricsRecording(t *testing.T) {
	metrics := NewMockMetricsRecorder()

	tests := []struct {
		operation func()
		check     func(t *testing.T, recordings []MetricRecord)
		name      string
	}{
		{
			name: "chain execution metrics",
			operation: func() {
				ctx := context.Background()
				metrics.RecordChainExecution(ctx, 100*time.Millisecond, true, "test-chain")
			},
			check: func(t *testing.T, recordings []MetricRecord) {
				require.Len(t, recordings, 1)
				record := recordings[0]
				assert.Equal(t, "chain_execution", record.Operation)
				assert.Equal(t, "duration", record.Type)
				assert.Equal(t, 100*time.Millisecond, record.Value)
				assert.Equal(t, "test-chain", record.Labels["chain_name"])
				assert.Equal(t, "true", record.Labels["success"])
			},
		},
		{
			name: "graph execution metrics",
			operation: func() {
				ctx := context.Background()
				metrics.RecordGraphExecution(ctx, 200*time.Millisecond, true, "test-graph", 5)
			},
			check: func(t *testing.T, recordings []MetricRecord) {
				require.Len(t, recordings, 1)
				record := recordings[0]
				assert.Equal(t, "graph_execution", record.Operation)
				assert.Equal(t, "test-graph", record.Labels["graph_name"])
				assert.Equal(t, "5", record.Labels["node_count"])
			},
		},
		{
			name: "workflow execution metrics",
			operation: func() {
				ctx := context.Background()
				metrics.RecordWorkflowExecution(ctx, 300*time.Millisecond, false, "test-workflow")
			},
			check: func(t *testing.T, recordings []MetricRecord) {
				require.Len(t, recordings, 1)
				record := recordings[0]
				assert.Equal(t, "workflow_execution", record.Operation)
				assert.Equal(t, "test-workflow", record.Labels["workflow_name"])
				assert.Equal(t, "false", record.Labels["success"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.Clear()
			tt.operation()
			recordings := metrics.GetRecordings()
			tt.check(t, recordings)
		})
	}
}

// TestIntegrationTestHelper tests the integration test helper functionality.
func TestIntegrationTestHelper(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add orchestrators
	chainOrch := NewAdvancedMockOrchestrator("chain-orch", "chain")
	graphOrch := NewAdvancedMockOrchestrator("graph-orch", "graph")
	workflowOrch := NewAdvancedMockOrchestrator("workflow-orch", "workflow")

	helper.AddOrchestrator("chain", chainOrch)
	helper.AddOrchestrator("graph", graphOrch)
	helper.AddOrchestrator("workflow", workflowOrch)

	// Test orchestrator retrieval
	assert.Equal(t, chainOrch, helper.GetOrchestrator("chain"))
	assert.Equal(t, graphOrch, helper.GetOrchestrator("graph"))
	assert.Equal(t, workflowOrch, helper.GetOrchestrator("workflow"))

	// Test metrics recording
	ctx := context.Background()
	helper.GetMetrics().RecordChainExecution(ctx, 100*time.Millisecond, true, "integration-chain")

	recordings := helper.GetMetrics().GetRecordings()
	assert.Len(t, recordings, 1)

	// Test reset functionality
	helper.Reset()
	recordings = helper.GetMetrics().GetRecordings()
	assert.Empty(t, recordings)
}

// TestErrorHandling tests error handling scenarios.
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *AdvancedMockOrchestrator
		operation func(orch *AdvancedMockOrchestrator) error
		errorCode string
	}{
		{
			name: "execution error",
			setup: func() *AdvancedMockOrchestrator {
				return NewAdvancedMockOrchestrator("error-orch", "chain",
					WithMockError(true, errors.New("execution failed")))
			},
			operation: func(orch *AdvancedMockOrchestrator) error {
				ctx := context.Background()
				_, err := orch.Execute(ctx, "test")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch := tt.setup()
			err := tt.operation(orch)

			require.Error(t, err)
			// Add specific error type checking if available
		})
	}
}

// BenchmarkOrchestratorExecution benchmarks orchestrator execution performance.
func BenchmarkOrchestratorExecution(b *testing.B) {
	orchestrator := NewAdvancedMockOrchestrator("bench-orch", "chain",
		WithMockResponses([]any{"benchmark result"}))
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := orchestrator.Execute(ctx, "benchmark input")
			if err != nil {
				b.Errorf("Benchmark execution error: %v", err)
			}
		}
	})
}

// BenchmarkChainExecution benchmarks chain execution performance.
func BenchmarkChainExecution(b *testing.B) {
	chain := CreateTestChain("benchmark-chain", []string{"step1", "step2", "step3"})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := chain.Invoke(ctx, fmt.Sprintf("input-%d", i))
		if err != nil {
			b.Errorf("Chain execution error: %v", err)
		}
	}
}

// BenchmarkGraphExecution benchmarks graph execution performance.
func BenchmarkGraphExecution(b *testing.B) {
	edges := map[string][]string{
		"node1": {"node2", "node3"},
		"node2": {"node4"},
		"node3": {"node4"},
	}
	graph := CreateTestGraph("benchmark-graph", []string{"node1", "node2", "node3", "node4"}, edges)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.Invoke(ctx, fmt.Sprintf("input-%d", i))
		if err != nil {
			b.Errorf("Graph execution error: %v", err)
		}
	}
}

// BenchmarkWorkflowExecution benchmarks workflow execution performance.
func BenchmarkWorkflowExecution(b *testing.B) {
	workflow := CreateTestWorkflow("benchmark-workflow", []string{"task1", "task2", "task3"})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflowID, runID, err := workflow.Execute(ctx, fmt.Sprintf("input-%d", i))
		if err != nil {
			b.Errorf("Workflow execution error: %v", err)
		}

		// Also benchmark result retrieval
		_, err = workflow.GetResult(ctx, workflowID, runID)
		if err != nil {
			b.Errorf("Workflow result retrieval error: %v", err)
		}
	}
}

// TestOrchestratorCreationAdvanced provides advanced table-driven tests for orchestrator creation.
func TestOrchestratorCreationAdvanced(t *testing.T) {
	tests := []struct {
		setup       func(t *testing.T) (*Orchestrator, error)
		validate    func(t *testing.T, orch *Orchestrator, err error)
		name        string
		description string
		wantErr     bool
	}{
		{
			name:        "basic_orchestrator_creation",
			description: "Create orchestrator with default config",
			setup: func(t *testing.T) (*Orchestrator, error) {
				config := DefaultConfig()
				return NewOrchestrator(config)
			},
			validate: func(t *testing.T, orch *Orchestrator, err error) {
				require.NoError(t, err)
				assert.NotNil(t, orch)
				metrics := orch.GetMetrics()
				assert.NotNil(t, metrics)
			},
		},
		{
			name:        "orchestrator_with_nil_config",
			description: "Create orchestrator with nil config should fail",
			setup: func(t *testing.T) (*Orchestrator, error) {
				return NewOrchestrator(nil)
			},
			validate: func(t *testing.T, orch *Orchestrator, err error) {
				require.Error(t, err)
				assert.Nil(t, orch)
			},
			wantErr: true,
		},
		{
			name:        "orchestrator_with_options",
			description: "Create orchestrator with functional options",
			setup: func(t *testing.T) (*Orchestrator, error) {
				return NewOrchestratorWithOptions(
					WithChainTimeout(30*time.Second),
					WithGraphMaxWorkers(10),
				)
			},
			validate: func(t *testing.T, orch *Orchestrator, err error) {
				require.NoError(t, err)
				assert.NotNil(t, orch)
			},
		},
		{
			name:        "default_orchestrator_creation",
			description: "Create orchestrator with NewDefaultOrchestrator",
			setup: func(t *testing.T) (*Orchestrator, error) {
				return NewDefaultOrchestrator()
			},
			validate: func(t *testing.T, orch *Orchestrator, err error) {
				require.NoError(t, err)
				assert.NotNil(t, orch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			orch, err := tt.setup(t)
			tt.validate(t, orch, err)
		})
	}
}

// TestOrchestratorChainCreationAdvanced tests chain creation with various scenarios.
func TestOrchestratorChainCreationAdvanced(t *testing.T) {
	tests := []struct {
		setup         func(t *testing.T) (*Orchestrator, []core.Runnable)
		validate      func(t *testing.T, chain iface.Chain, err error)
		name          string
		description   string
		wantErr       bool
		chainsEnabled bool
	}{
		{
			name:        "create_chain_with_steps",
			description: "Create chain with multiple steps",
			setup: func(t *testing.T) (*Orchestrator, []core.Runnable) {
				orch, err := NewDefaultOrchestrator()
				require.NoError(t, err)
				steps := []core.Runnable{
					&MockRunnable{name: "step1"},
					&MockRunnable{name: "step2"},
				}
				return orch, steps
			},
			validate: func(t *testing.T, chain iface.Chain, err error) {
				require.NoError(t, err)
				assert.NotNil(t, chain)
			},
			chainsEnabled: true,
		},
		{
			name:        "create_chain_disabled",
			description: "Create chain when chains are disabled should fail",
			setup: func(t *testing.T) (*Orchestrator, []core.Runnable) {
				config := DefaultConfig()
				config.Enabled.Chains = false
				orch, err := NewOrchestrator(config)
				require.NoError(t, err)
				steps := []core.Runnable{&MockRunnable{name: "step1"}}
				return orch, steps
			},
			validate: func(t *testing.T, chain iface.Chain, err error) {
				require.Error(t, err)
				assert.Nil(t, chain)
			},
			wantErr:       true,
			chainsEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			orch, steps := tt.setup(t)
			chain, err := orch.CreateChain(steps)
			tt.validate(t, chain, err)
		})
	}
}

// TestOrchestratorGraphCreationAdvanced tests graph creation with various scenarios.
func TestOrchestratorGraphCreationAdvanced(t *testing.T) {
	tests := []struct {
		setup         func(t *testing.T) *Orchestrator
		validate      func(t *testing.T, graph iface.Graph, err error)
		name          string
		description   string
		wantErr       bool
		graphsEnabled bool
	}{
		{
			name:        "create_graph_basic",
			description: "Create graph with nodes and edges",
			setup: func(t *testing.T) *Orchestrator {
				orch, err := NewDefaultOrchestrator()
				require.NoError(t, err)
				return orch
			},
			validate: func(t *testing.T, graph iface.Graph, err error) {
				require.NoError(t, err)
				assert.NotNil(t, graph)
			},
			graphsEnabled: true,
		},
		{
			name:        "create_graph_disabled",
			description: "Create graph when graphs are disabled should fail",
			setup: func(t *testing.T) *Orchestrator {
				config := DefaultConfig()
				config.Enabled.Graphs = false
				orch, err := NewOrchestrator(config)
				require.NoError(t, err)
				return orch
			},
			validate: func(t *testing.T, graph iface.Graph, err error) {
				require.Error(t, err)
				assert.Nil(t, graph)
			},
			wantErr:       true,
			graphsEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			orch := tt.setup(t)
			graph, err := orch.CreateGraph()
			tt.validate(t, graph, err)
		})
	}
}

// TestOrchestratorWorkflowCreationAdvanced tests workflow creation with various scenarios.
func TestOrchestratorWorkflowCreationAdvanced(t *testing.T) {
	tests := []struct {
		workflowFn       any
		setup            func(t *testing.T) *Orchestrator
		validate         func(t *testing.T, workflow iface.Workflow, err error)
		name             string
		description      string
		wantErr          bool
		workflowsEnabled bool
	}{
		{
			name:        "create_workflow_basic",
			description: "Create workflow with basic function",
			setup: func(t *testing.T) *Orchestrator {
				orch, err := NewDefaultOrchestrator()
				require.NoError(t, err)
				return orch
			},
			workflowFn: func(ctx context.Context, input string) (string, error) {
				return "result", nil
			},
			validate: func(t *testing.T, workflow iface.Workflow, err error) {
				// Workflow creation will fail without Temporal client, which is expected
				require.Error(t, err)
				assert.Nil(t, workflow)
			},
			wantErr: true,
		},
		{
			name:        "create_workflow_disabled",
			description: "Create workflow when workflows are disabled should fail",
			setup: func(t *testing.T) *Orchestrator {
				config := DefaultConfig()
				config.Enabled.Workflows = false
				orch, err := NewOrchestrator(config)
				require.NoError(t, err)
				return orch
			},
			workflowFn: func(ctx context.Context, input string) (string, error) {
				return "result", nil
			},
			validate: func(t *testing.T, workflow iface.Workflow, err error) {
				require.Error(t, err)
				assert.Nil(t, workflow)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			orch := tt.setup(t)
			workflow, err := orch.CreateWorkflow(tt.workflowFn)
			tt.validate(t, workflow, err)
		})
	}
}

// TestOrchestratorHealthCheckAdvanced tests health check functionality.
func TestOrchestratorHealthCheckAdvanced(t *testing.T) {
	tests := []struct {
		setup       func(t *testing.T) *Orchestrator
		validate    func(t *testing.T, err error)
		name        string
		description string
		wantErr     bool
	}{
		{
			name:        "health_check_success",
			description: "Health check should succeed for healthy orchestrator",
			setup: func(t *testing.T) *Orchestrator {
				orch, err := NewDefaultOrchestrator()
				require.NoError(t, err)
				return orch
			},
			validate: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			orch := tt.setup(t)
			ctx := context.Background()
			err := orch.Check(ctx)
			tt.validate(t, err)
		})
	}
}

// TestNewChain tests the convenience NewChain function.
func TestNewChain(t *testing.T) {
	steps := []core.Runnable{
		&MockRunnable{name: "step1"},
		&MockRunnable{name: "step2"},
	}

	chain, err := NewChain(steps)
	require.NoError(t, err)
	assert.NotNil(t, chain)

	ctx := context.Background()
	result, err := chain.Invoke(ctx, "test input")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestNewGraph tests the convenience NewGraph function.
func TestNewGraph(t *testing.T) {
	graph, err := NewGraph()
	require.NoError(t, err)
	assert.NotNil(t, graph)

	ctx := context.Background()
	result, err := graph.Invoke(ctx, "test input")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestNewWorkflow tests the convenience NewWorkflow function.
func TestNewWorkflow(t *testing.T) {
	workflowFn := func(ctx context.Context, input string) (string, error) {
		return "result", nil
	}

	workflow, err := NewWorkflow(workflowFn)
	// Workflow creation will fail without Temporal client
	require.Error(t, err)
	assert.Nil(t, workflow)
}

// TestConfigValidationAdvanced provides advanced table-driven tests for config validation.
func TestConfigValidationAdvanced(t *testing.T) {
	tests := []struct {
		config      *Config
		validate    func(t *testing.T, err error)
		name        string
		description string
		wantErr     bool
	}{
		{
			name:        "valid_config",
			description: "Validate valid configuration",
			config:      DefaultConfig(),
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:        "invalid_chain_max_concurrent",
			description: "Handle invalid chain max concurrent chains",
			config: &Config{
				Chain: ChainConfig{
					MaxConcurrentChains: 0, // Invalid: must be >= 1
				},
				Graph: GraphConfig{
					MaxWorkers: 5,
				},
				Workflow: WorkflowConfig{
					MaxConcurrentWorkflows: 50,
				},
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "invalid_graph_max_workers",
			description: "Handle invalid graph max workers",
			config: &Config{
				Chain: ChainConfig{
					MaxConcurrentChains: 10,
				},
				Graph: GraphConfig{
					MaxWorkers: 0, // Invalid: must be >= 1
				},
				Workflow: WorkflowConfig{
					MaxConcurrentWorkflows: 50,
				},
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "invalid_workflow_max_concurrent",
			description: "Handle invalid workflow max concurrent workflows",
			config: &Config{
				Chain: ChainConfig{
					MaxConcurrentChains: 10,
				},
				Graph: GraphConfig{
					MaxWorkers: 5,
				},
				Workflow: WorkflowConfig{
					MaxConcurrentWorkflows: 0, // Invalid: must be >= 1
				},
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "config_with_custom_timeouts",
			description: "Validate config with custom timeouts",
			config: &Config{
				Chain: ChainConfig{
					DefaultTimeout:      30 * time.Second,
					DefaultRetries:      3,
					MaxConcurrentChains: 10,
				},
				Graph: GraphConfig{
					DefaultTimeout: 60 * time.Second,
					MaxWorkers:     5,
					QueueSize:      100, // Required: must be >= 1
				},
				Workflow: WorkflowConfig{
					DefaultTimeout:         90 * time.Minute,
					MaxConcurrentWorkflows: 50,
					TaskQueue:              "test-queue",
				},
			},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			err := tt.config.Validate()
			tt.validate(t, err)
		})
	}
}

// TestConfigOptionsAdvanced provides advanced table-driven tests for config options.
func TestConfigOptionsAdvanced(t *testing.T) {
	tests := []struct {
		validate    func(t *testing.T, config *Config, err error)
		name        string
		description string
		options     []Option
		wantErr     bool
	}{
		{
			name:        "config_with_chain_timeout",
			description: "Create config with chain timeout option",
			options: []Option{
				WithChainTimeout(30 * time.Second),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, 30*time.Second, config.Chain.DefaultTimeout)
			},
		},
		{
			name:        "config_with_graph_max_workers",
			description: "Create config with graph max workers option",
			options: []Option{
				WithGraphMaxWorkers(10),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, 10, config.Graph.MaxWorkers)
			},
		},
		{
			name:        "config_with_workflow_task_queue",
			description: "Create config with workflow task queue option",
			options: []Option{
				WithWorkflowTaskQueue("custom-queue"),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, "custom-queue", config.Workflow.TaskQueue)
			},
		},
		{
			name:        "config_with_metrics_prefix",
			description: "Create config with metrics prefix option",
			options: []Option{
				WithMetricsPrefix("custom.prefix"),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.NoError(t, err)
				assert.Equal(t, "custom.prefix", config.Observability.MetricsPrefix)
			},
		},
		{
			name:        "config_with_features",
			description: "Create config with features option",
			options: []Option{
				WithFeatures(EnabledFeatures{
					Chains:    true,
					Graphs:    true,
					Workflows: false,
				}),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.NoError(t, err)
				assert.True(t, config.Enabled.Chains)
				assert.True(t, config.Enabled.Graphs)
				assert.False(t, config.Enabled.Workflows)
			},
		},
		{
			name:        "invalid_chain_timeout_zero",
			description: "Create config with zero chain timeout should fail",
			options: []Option{
				WithChainTimeout(0),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "invalid_graph_max_workers_zero",
			description: "Create config with zero graph max workers should fail",
			options: []Option{
				WithGraphMaxWorkers(0),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "invalid_workflow_task_queue_empty",
			description: "Create config with empty workflow task queue should fail",
			options: []Option{
				WithWorkflowTaskQueue(""),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "invalid_metrics_prefix_empty",
			description: "Create config with empty metrics prefix should fail",
			options: []Option{
				WithMetricsPrefix(""),
			},
			validate: func(t *testing.T, config *Config, err error) {
				require.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			config, err := NewConfig(tt.options...)
			tt.validate(t, config, err)
		})
	}
}

// TestErrorHandlingAdvanced provides advanced table-driven tests for error handling.
func TestErrorHandlingAdvanced(t *testing.T) {
	tests := []struct {
		setup       func() *OrchestrationError
		validate    func(t *testing.T, err *OrchestrationError)
		name        string
		description string
	}{
		{
			name:        "basic_error_creation",
			description: "Create basic orchestration error",
			setup: func() *OrchestrationError {
				return NewOrchestrationError("test_op", ErrCodeInvalidConfig, errors.New("test error"))
			},
			validate: func(t *testing.T, err *OrchestrationError) {
				assert.NotNil(t, err)
				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeInvalidConfig, err.Code)
				assert.Error(t, err)
			},
		},
		{
			name:        "error_with_message",
			description: "Create orchestration error with custom message",
			setup: func() *OrchestrationError {
				return NewOrchestrationErrorWithMessage("test_op", ErrCodeExecutionFailed, "test message", errors.New("underlying error"))
			},
			validate: func(t *testing.T, err *OrchestrationError) {
				assert.NotNil(t, err)
				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeExecutionFailed, err.Code)
				assert.Equal(t, "test message", err.Message)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "test message")
			},
		},
		{
			name:        "error_unwrap",
			description: "Unwrap orchestration error to get underlying error",
			setup: func() *OrchestrationError {
				underlying := errors.New("underlying error")
				return NewOrchestrationError("test_op", ErrCodeTimeout, underlying)
			},
			validate: func(t *testing.T, err *OrchestrationError) {
				underlying := err.Unwrap()
				assert.Error(t, underlying)
				assert.Equal(t, "underlying error", underlying.Error())
			},
		},
		{
			name:        "is_orchestration_error",
			description: "Check if error is an orchestration error",
			setup: func() *OrchestrationError {
				return NewOrchestrationError("test_op", ErrCodeInvalidInput, errors.New("test error"))
			},
			validate: func(t *testing.T, err *OrchestrationError) {
				assert.True(t, IsOrchestrationError(err))
				regularErr := errors.New("regular error")
				assert.False(t, IsOrchestrationError(regularErr))
			},
		},
		{
			name:        "as_orchestration_error",
			description: "Convert error to orchestration error",
			setup: func() *OrchestrationError {
				return NewOrchestrationError("test_op", ErrCodeDependencyError, errors.New("test error"))
			},
			validate: func(t *testing.T, err *OrchestrationError) {
				orchErr, ok := AsOrchestrationError(err)
				assert.True(t, ok)
				assert.NotNil(t, orchErr)
				assert.Equal(t, "test_op", orchErr.Op)
				assert.Equal(t, ErrCodeDependencyError, orchErr.Code)

				regularErr := errors.New("regular error")
				_, ok = AsOrchestrationError(regularErr)
				assert.False(t, ok)
			},
		},
		{
			name:        "error_with_all_codes",
			description: "Test error creation with all error codes",
			setup: func() *OrchestrationError {
				// Test with different error codes
				return NewOrchestrationError("test_op", ErrCodeSchedulingFailed, errors.New("scheduling failed"))
			},
			validate: func(t *testing.T, err *OrchestrationError) {
				assert.Equal(t, ErrCodeSchedulingFailed, err.Code)
				assert.Contains(t, err.Error(), "scheduling_failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			err := tt.setup()
			tt.validate(t, err)
		})
	}
}
