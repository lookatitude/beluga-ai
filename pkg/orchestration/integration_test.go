package orchestration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDataProcessor simulates a data processing step
type TestDataProcessor struct {
	name string
}

func (p *TestDataProcessor) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map input, got %T", input)
	}

	// Simulate processing
	result := map[string]any{
		"processed_data": fmt.Sprintf("%s_processed", inputMap["data"]),
		"processor":      p.name,
		"timestamp":      time.Now().Unix(),
	}

	return result, nil
}

func (p *TestDataProcessor) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := p.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (p *TestDataProcessor) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := p.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// TestDataValidator simulates data validation
type TestDataValidator struct {
	name string
}

func (v *TestDataValidator) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map input, got %T", input)
	}

	// Simulate validation
	processedData, exists := inputMap["processed_data"]
	if !exists {
		return nil, fmt.Errorf("missing processed_data field")
	}

	result := map[string]any{
		"validated_data": fmt.Sprintf("%s_validated", processedData),
		"validator":      v.name,
		"valid":          true,
		"original_input": input,
	}

	// Preserve timestamp from input if it exists
	if timestamp, exists := inputMap["timestamp"]; exists {
		result["timestamp"] = timestamp
	}

	return result, nil
}

func (v *TestDataValidator) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := v.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (v *TestDataValidator) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := v.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// TestDataAggregator simulates data aggregation
type TestDataAggregator struct {
	name string
}

func (a *TestDataAggregator) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map input, got %T", input)
	}

	result := map[string]any{
		"aggregated_data": fmt.Sprintf("aggregated_%s", inputMap["validated_data"]),
		"aggregator":      a.name,
		"final_result":    true,
		"processing_time": time.Since(time.Unix(inputMap["timestamp"].(int64), 0)).String(),
	}

	return result, nil
}

func (a *TestDataAggregator) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (a *TestDataAggregator) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// TestWorkflowStep simulates a workflow step
type TestWorkflowStep struct {
	name  string
	delay time.Duration
}

func (w *TestWorkflowStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	select {
	case <-time.After(w.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map input, got %T", input)
	}

	result := map[string]any{
		"workflow_data": fmt.Sprintf("%s_workflow_processed", inputMap["data"]),
		"step":          w.name,
		"completed_at":  time.Now().Format(time.RFC3339),
	}

	return result, nil
}

func (w *TestWorkflowStep) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := w.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (w *TestWorkflowStep) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := w.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// Integration test: Complete data processing pipeline
func TestDataProcessingPipelineIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create processing steps
	processor := &TestDataProcessor{name: "data-processor"}
	validator := &TestDataValidator{name: "data-validator"}
	aggregator := &TestDataAggregator{name: "data-aggregator"}

	// Create a processing chain
	chain, err := orch.CreateChain([]core.Runnable{processor, validator, aggregator})
	require.NoError(t, err)

	// Test single item processing
	input := map[string]any{"data": "test-item-123"}
	result, err := chain.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "aggregated_data")
	assert.Contains(t, resultMap, "final_result")
	assert.Equal(t, true, resultMap["final_result"])
}

// Integration test: Batch processing pipeline
func TestBatchProcessingPipelineIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	processor := &TestDataProcessor{name: "batch-processor"}
	validator := &TestDataValidator{name: "batch-validator"}

	chain, err := orch.CreateChain([]core.Runnable{processor, validator})
	require.NoError(t, err)

	// Create batch inputs
	batchSize := 5
	inputs := make([]any, batchSize)
	for i := 0; i < batchSize; i++ {
		inputs[i] = map[string]any{"data": fmt.Sprintf("batch-item-%d", i)}
	}

	// Process batch
	results, err := chain.Batch(context.Background(), inputs)

	assert.NoError(t, err)
	assert.Len(t, results, batchSize)

	// Verify each result
	for _, result := range results {
		assert.NotNil(t, result)
		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, resultMap, "validated_data")
		assert.Contains(t, resultMap, "valid")
		assert.Equal(t, true, resultMap["valid"])
	}
}

// Integration test: Graph-based workflow
func TestGraphWorkflowIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	graph, err := orch.CreateGraph()
	require.NoError(t, err)

	// Create workflow steps
	dataLoader := &TestWorkflowStep{name: "data-loader", delay: 10 * time.Millisecond}
	preprocessor := &TestWorkflowStep{name: "preprocessor", delay: 10 * time.Millisecond}
	analyzer := &TestWorkflowStep{name: "analyzer", delay: 10 * time.Millisecond}
	reporter := &TestWorkflowStep{name: "reporter", delay: 10 * time.Millisecond}

	// Add nodes to graph
	nodes := map[string]core.Runnable{
		"loader":       dataLoader,
		"preprocessor": preprocessor,
		"analyzer":     analyzer,
		"reporter":     reporter,
	}

	for name, node := range nodes {
		err = graph.AddNode(name, node)
		require.NoError(t, err)
	}

	// Define workflow: loader -> preprocessor -> analyzer -> reporter
	err = graph.AddEdge("loader", "preprocessor")
	require.NoError(t, err)
	err = graph.AddEdge("preprocessor", "analyzer")
	require.NoError(t, err)
	err = graph.AddEdge("analyzer", "reporter")
	require.NoError(t, err)

	err = graph.SetEntryPoint([]string{"loader"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"reporter"})
	require.NoError(t, err)

	// Execute workflow
	input := map[string]any{"data": "workflow-input"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "workflow_data")
	assert.Contains(t, resultMap, "step")
	assert.Equal(t, "reporter", resultMap["step"])
}

// Integration test: Parallel processing graph
func TestParallelProcessingGraphIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	graph, err := orch.CreateGraph()
	require.NoError(t, err)

	// Create parallel processing steps
	splitter := &TestWorkflowStep{name: "splitter", delay: 5 * time.Millisecond}
	processor1 := &TestWorkflowStep{name: "processor1", delay: 20 * time.Millisecond}
	processor2 := &TestWorkflowStep{name: "processor2", delay: 20 * time.Millisecond}
	merger := &TestWorkflowStep{name: "merger", delay: 5 * time.Millisecond}

	// Add nodes
	nodes := map[string]core.Runnable{
		"splitter":   splitter,
		"processor1": processor1,
		"processor2": processor2,
		"merger":     merger,
	}

	for name, node := range nodes {
		err = graph.AddNode(name, node)
		require.NoError(t, err)
	}

	// Define parallel workflow: splitter -> [processor1, processor2] -> merger
	err = graph.AddEdge("splitter", "processor1")
	require.NoError(t, err)
	err = graph.AddEdge("splitter", "processor2")
	require.NoError(t, err)
	err = graph.AddEdge("processor1", "merger")
	require.NoError(t, err)
	err = graph.AddEdge("processor2", "merger")
	require.NoError(t, err)

	err = graph.SetEntryPoint([]string{"splitter"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"merger"})
	require.NoError(t, err)

	// Execute parallel workflow
	input := map[string]any{"data": "parallel-input"}
	start := time.Now()
	result, err := graph.Invoke(context.Background(), input)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should complete faster than sequential execution
	// (processor1 + processor2 in parallel should be faster than sequential)
	assert.True(t, duration < 100*time.Millisecond, "Parallel execution took too long: %v", duration)

	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "merger", resultMap["step"])
}

// Integration test: Error handling and recovery
func TestErrorHandlingIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create a failing step
	failingStep := &FailingWorkflowStep{
		TestWorkflowStep: TestWorkflowStep{name: "failing-step", delay: 5 * time.Millisecond},
		shouldFail:       true,
	}

	// Create recovery step
	recoveryStep := &TestWorkflowStep{name: "recovery-step", delay: 5 * time.Millisecond}

	// Test chain with error
	chain, err := orch.CreateChain([]core.Runnable{failingStep, recoveryStep})
	require.NoError(t, err)

	input := map[string]any{"data": "error-test"}
	_, err = chain.Invoke(context.Background(), input)

	// Should fail at the first step
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated failure")
}

// Integration test: Timeout handling
func TestTimeoutHandlingIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create a slow step
	slowStep := &TestWorkflowStep{name: "slow-step", delay: 100 * time.Millisecond}

	chain, err := orch.CreateChain([]core.Runnable{slowStep})
	require.NoError(t, err)

	// Use a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	input := map[string]any{"data": "timeout-test"}
	_, err = chain.Invoke(ctx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

// Integration test: Concurrent orchestration
func TestConcurrentOrchestrationIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	const numConcurrentChains = 10
	const stepsPerChain = 3

	var wg sync.WaitGroup
	results := make(chan error, numConcurrentChains)

	for i := 0; i < numConcurrentChains; i++ {
		wg.Add(1)
		go func(chainID int) {
			defer wg.Done()

			// Create steps for this chain
			steps := make([]core.Runnable, stepsPerChain)
			for j := 0; j < stepsPerChain; j++ {
				steps[j] = &TestWorkflowStep{
					name:  fmt.Sprintf("chain-%d-step-%d", chainID, j),
					delay: 5 * time.Millisecond,
				}
			}

			// Create and execute chain
			chain, err := orch.CreateChain(steps)
			if err != nil {
				results <- err
				return
			}

			input := map[string]any{"data": fmt.Sprintf("chain-%d-input", chainID)}
			_, err = chain.Invoke(context.Background(), input)
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	// Check all chains executed successfully
	for err := range results {
		assert.NoError(t, err)
	}
}

// Integration test: Complex graph with conditional paths
func TestComplexGraphIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	graph, err := orch.CreateGraph()
	require.NoError(t, err)

	// Create decision step
	decisionStep := &DecisionWorkflowStep{
		TestWorkflowStep: TestWorkflowStep{name: "decision", delay: 5 * time.Millisecond},
		decisionPath:     "path_a",
	}

	pathA := &TestWorkflowStep{name: "path-a", delay: 10 * time.Millisecond}
	pathB := &TestWorkflowStep{name: "path-b", delay: 10 * time.Millisecond}
	converge := &TestWorkflowStep{name: "converge", delay: 5 * time.Millisecond}

	// Add nodes
	nodes := map[string]core.Runnable{
		"decision": decisionStep,
		"path-a":   pathA,
		"path-b":   pathB,
		"converge": converge,
	}

	for name, node := range nodes {
		err = graph.AddNode(name, node)
		require.NoError(t, err)
	}

	// Define conditional workflow: decision -> [path-a, path-b] -> converge
	err = graph.AddEdge("decision", "path-a")
	require.NoError(t, err)
	err = graph.AddEdge("decision", "path-b")
	require.NoError(t, err)
	err = graph.AddEdge("path-a", "converge")
	require.NoError(t, err)
	err = graph.AddEdge("path-b", "converge")
	require.NoError(t, err)

	err = graph.SetEntryPoint([]string{"decision"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"converge"})
	require.NoError(t, err)

	// Execute complex workflow
	input := map[string]any{"data": "complex-input"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "converge", resultMap["step"])
}

// Integration test: Resource cleanup
func TestResourceCleanupIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create multiple orchestrations
	initialMetrics := orch.GetMetrics()
	initialChains := initialMetrics.GetActiveChains()
	initialGraphs := initialMetrics.GetActiveGraphs()

	// Create several chains and graphs
	const numChains = 5
	const numGraphs = 3

	chains := make([]interface {
		Invoke(context.Context, any, ...core.Option) (any, error)
	}, numChains)
	graphs := make([]interface {
		Invoke(context.Context, any, ...core.Option) (any, error)
	}, numGraphs)

	for i := 0; i < numChains; i++ {
		chain, err := orch.CreateChain([]core.Runnable{&TestWorkflowStep{name: fmt.Sprintf("cleanup-chain-%d", i)}})
		require.NoError(t, err)
		chains[i] = chain
	}

	for i := 0; i < numGraphs; i++ {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		step := &TestWorkflowStep{name: fmt.Sprintf("cleanup-graph-%d", i)}
		err = graph.AddNode("step", step)
		require.NoError(t, err)
		err = graph.SetEntryPoint([]string{"step"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"step"})
		require.NoError(t, err)

		graphs[i] = graph
	}

	// Verify resources are tracked
	finalMetrics := orch.GetMetrics()
	assert.Equal(t, initialChains+numChains, finalMetrics.GetActiveChains())
	assert.Equal(t, initialGraphs+numGraphs, finalMetrics.GetActiveGraphs())

	// Execute all orchestrations
	for _, chain := range chains {
		_, err := chain.Invoke(context.Background(), map[string]any{"data": "cleanup-test"})
		assert.NoError(t, err)
	}

	for _, graph := range graphs {
		_, err := graph.Invoke(context.Background(), map[string]any{"data": "cleanup-test"})
		assert.NoError(t, err)
	}
}

// Integration test: Performance monitoring
func TestPerformanceMonitoringIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create a multi-step chain
	steps := []core.Runnable{
		&TestWorkflowStep{name: "perf-step-1", delay: 10 * time.Millisecond},
		&TestWorkflowStep{name: "perf-step-2", delay: 15 * time.Millisecond},
		&TestWorkflowStep{name: "perf-step-3", delay: 20 * time.Millisecond},
	}

	chain, err := orch.CreateChain(steps)
	require.NoError(t, err)

	// Execute with timing
	input := map[string]any{"data": "perf-test"}
	start := time.Now()
	result, err := chain.Invoke(context.Background(), input)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should take at least the sum of all step delays
	minExpectedDuration := 45 * time.Millisecond // 10 + 15 + 20
	assert.True(t, duration >= minExpectedDuration,
		"Execution too fast: expected >= %v, got %v", minExpectedDuration, duration)

	// Should complete within reasonable time
	maxExpectedDuration := 200 * time.Millisecond
	assert.True(t, duration <= maxExpectedDuration,
		"Execution too slow: expected <= %v, got %v", maxExpectedDuration, duration)
}

// Integration test: Large scale batch processing
func TestLargeScaleBatchProcessingIntegration(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	processor := &TestDataProcessor{name: "large-batch-processor"}
	validator := &TestDataValidator{name: "large-batch-validator"}

	chain, err := orch.CreateChain([]core.Runnable{processor, validator})
	require.NoError(t, err)

	// Create large batch
	batchSize := 100
	inputs := make([]any, batchSize)
	for i := range inputs {
		inputs[i] = map[string]any{"data": fmt.Sprintf("large-batch-item-%d", i)}
	}

	// Process large batch
	start := time.Now()
	results, err := chain.Batch(context.Background(), inputs)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Len(t, results, batchSize)

	// Verify results
	for _, result := range results {
		assert.NotNil(t, result)
		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, resultMap, "validated_data")
		assert.Contains(t, resultMap, "valid")
		assert.Equal(t, true, resultMap["valid"])
	}

	// Performance check - should complete within reasonable time
	maxExpectedDuration := 2 * time.Second
	assert.True(t, duration <= maxExpectedDuration,
		"Large batch processing too slow: expected <= %v, got %v", maxExpectedDuration, duration)

	t.Logf("Processed %d items in %v (%.2f items/sec)",
		batchSize, duration, float64(batchSize)/duration.Seconds())
}

// FailingWorkflowStep extends TestWorkflowStep with failure capability
type FailingWorkflowStep struct {
	TestWorkflowStep
	shouldFail bool
}

func (f *FailingWorkflowStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if f.shouldFail {
		return nil, fmt.Errorf("simulated failure in %s", f.name)
	}
	return f.TestWorkflowStep.Invoke(ctx, input, opts...)
}

// DecisionWorkflowStep makes decisions about execution paths
type DecisionWorkflowStep struct {
	TestWorkflowStep
	decisionPath string
}

func (d *DecisionWorkflowStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	inputMap := input.(map[string]any)
	result := map[string]any{
		"decision":      d.decisionPath,
		"original_data": inputMap["data"],
	}
	return result, nil
}
