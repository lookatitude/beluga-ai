// Package orchestration provides comprehensive tests for orchestration implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package orchestration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockOrchestrator tests the advanced mock orchestrator functionality
func TestAdvancedMockOrchestrator(t *testing.T) {
	tests := []struct {
		name           string
		orchestrator   *AdvancedMockOrchestrator
		input          interface{}
		expectedError  bool
		expectedResult string
	}{
		{
			name: "successful execution",
			orchestrator: NewAdvancedMockOrchestrator("test-orch", "chain",
				WithMockResponses([]interface{}{"success result"})),
			input:          "test input",
			expectedError:  false,
			expectedResult: "success result",
		},
		{
			name: "error execution",
			orchestrator: NewAdvancedMockOrchestrator("error-orch", "chain",
				WithMockError(true, fmt.Errorf("test error"))),
			input:         "test input",
			expectedError: true,
		},
		{
			name: "execution with delay",
			orchestrator: NewAdvancedMockOrchestrator("delay-orch", "chain",
				WithExecutionDelay(10*time.Millisecond),
				WithMockResponses([]interface{}{"delayed result"})),
			input:          "test input",
			expectedError:  false,
			expectedResult: "delayed result",
		},
		{
			name: "multiple responses",
			orchestrator: NewAdvancedMockOrchestrator("multi-orch", "chain",
				WithMockResponses([]interface{}{"result1", "result2", "result3"})),
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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

// TestChainExecution tests chain execution scenarios
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result.(string), tt.chainName)
				assert.Contains(t, result.(string), fmt.Sprintf("%d steps", len(tt.steps)))
			}

			// Test batch execution
			batchResults, err := chain.Batch(ctx, []interface{}{"input1", "input2"})
			assert.NoError(t, err)
			assert.Len(t, batchResults, 2)

			// Test stream execution
			streamCh, err := chain.Stream(ctx, "stream input")
			assert.NoError(t, err)

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

// TestGraphExecution tests graph execution scenarios
func TestGraphExecution(t *testing.T) {
	tests := []struct {
		name          string
		graphName     string
		nodes         []string
		edges         map[string][]string
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result.(string), tt.graphName)
				assert.Contains(t, result.(string), fmt.Sprintf("%d nodes", len(tt.nodes)))
			}

			// Test batch execution
			batchResults, err := graph.Batch(ctx, []interface{}{"input1", "input2"})
			assert.NoError(t, err)
			assert.Len(t, batchResults, 2)

			// Test stream execution
			streamCh, err := graph.Stream(ctx, "stream input")
			assert.NoError(t, err)

			select {
			case streamResult := <-streamCh:
				assert.NotNil(t, streamResult)
			case <-time.After(100 * time.Millisecond):
				t.Error("Stream timeout")
			}

			// Test graph-specific operations
			err = graph.AddNode("new_node", nil)
			assert.NoError(t, err)

			err = graph.AddEdge("node1", "new_node")
			assert.NoError(t, err)

			err = graph.SetEntryPoint([]string{"node1"})
			assert.NoError(t, err)

			err = graph.SetFinishPoint([]string{"new_node"})
			assert.NoError(t, err)
		})
	}
}

// TestWorkflowExecution tests workflow execution scenarios
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, workflowID)
				assert.NotEmpty(t, runID)
				assert.Contains(t, workflowID, tt.workflowName)
			}

			if !tt.expectedError {
				// Test workflow result retrieval
				result, err := workflow.GetResult(ctx, workflowID, runID)
				assert.NoError(t, err)
				assert.Contains(t, result.(string), workflowID)

				// Test workflow signaling
				err = workflow.Signal(ctx, workflowID, runID, "test_signal", "signal_data")
				assert.NoError(t, err)

				// Test workflow querying
				queryResult, err := workflow.Query(ctx, workflowID, runID, "status")
				assert.NoError(t, err)
				assert.NotNil(t, queryResult)

				// Test workflow cancellation
				err = workflow.Cancel(ctx, workflowID, runID)
				assert.NoError(t, err)

				// Test workflow termination
				err = workflow.Terminate(ctx, workflowID, runID, "test termination")
				assert.NoError(t, err)
			}
		})
	}
}

// TestConcurrencyAdvanced tests concurrent execution scenarios
func TestConcurrencyAdvanced(t *testing.T) {
	orchestrator := NewAdvancedMockOrchestrator("concurrent-test", "chain",
		WithMockResponses([]interface{}{"concurrent result"}))

	const numGoroutines = 10
	const numRequests = 5

	t.Run("concurrent_execution", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan interface{}, numGoroutines*numRequests)
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

// TestLoadTesting performs load testing on orchestration components
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	orchestrator := NewAdvancedMockOrchestrator("load-test", "chain",
		WithMockResponses([]interface{}{"load result"}))

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

// TestMetricsRecording tests metrics recording functionality
func TestMetricsRecording(t *testing.T) {
	metrics := NewMockMetricsRecorder()

	tests := []struct {
		name      string
		operation func()
		check     func(t *testing.T, recordings []MetricRecord)
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

// TestIntegrationTestHelper tests the integration test helper functionality
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
	assert.Len(t, recordings, 0)
}

// TestErrorHandling tests error handling scenarios
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
					WithMockError(true, fmt.Errorf("execution failed")))
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

			assert.Error(t, err)
			// Add specific error type checking if available
		})
	}
}

// BenchmarkOrchestratorExecution benchmarks orchestrator execution performance
func BenchmarkOrchestratorExecution(b *testing.B) {
	orchestrator := NewAdvancedMockOrchestrator("bench-orch", "chain",
		WithMockResponses([]interface{}{"benchmark result"}))
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

// BenchmarkChainExecution benchmarks chain execution performance
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

// BenchmarkGraphExecution benchmarks graph execution performance
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

// BenchmarkWorkflowExecution benchmarks workflow execution performance
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
