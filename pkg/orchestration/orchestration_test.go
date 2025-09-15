package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/stretchr/testify/assert"
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
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	steps := []core.Runnable{
		&MockRunnable{name: "step1"},
		&MockRunnable{name: "step2"},
	}

	chain, err := orch.CreateChain(steps)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}

	if chain == nil {
		t.Fatal("Chain should not be nil")
	}

	// Test chain execution
	input := map[string]any{"test": "data"}
	result, err := chain.Invoke(context.Background(), input)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if result == nil {
		t.Fatal("Chain result should not be nil")
	}

	t.Logf("Chain execution result: %v", result)
}

func TestGraphCreation(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	graph, err := orch.CreateGraph()
	if err != nil {
		t.Fatalf("Failed to create graph: %v", err)
	}

	if graph == nil {
		t.Fatal("Graph should not be nil")
	}

	// Test graph with nodes
	step1 := &MockRunnable{name: "step1"}
	step2 := &MockRunnable{name: "step2"}

	err = graph.AddNode("processor", step1)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	err = graph.AddNode("validator", step2)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	err = graph.AddEdge("processor", "validator")
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	err = graph.SetEntryPoint([]string{"processor"})
	if err != nil {
		t.Fatalf("Failed to set entry point: %v", err)
	}

	err = graph.SetFinishPoint([]string{"validator"})
	if err != nil {
		t.Fatalf("Failed to set finish point: %v", err)
	}

	// Test graph execution
	input := map[string]any{"data": "test"}
	result, err := graph.Invoke(context.Background(), input)
	if err != nil {
		t.Fatalf("Graph execution failed: %v", err)
	}

	if result == nil {
		t.Fatal("Graph result should not be nil")
	}

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

	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	if config.Chain.DefaultTimeout != 30 {
		t.Errorf("Expected chain timeout 30, got %v", config.Chain.DefaultTimeout)
	}

	if config.Graph.MaxWorkers != 5 {
		t.Errorf("Expected graph max workers 5, got %v", config.Graph.MaxWorkers)
	}

	if config.Workflow.TaskQueue != "test-queue" {
		t.Errorf("Expected workflow task queue 'test-queue', got %v", config.Workflow.TaskQueue)
	}

	if config.Observability.MetricsPrefix != "test.orchestration" {
		t.Errorf("Expected metrics prefix 'test.orchestration', got %v", config.Observability.MetricsPrefix)
	}
}

func TestOrchestratorWithOptions(t *testing.T) {
	orch, err := NewOrchestratorWithOptions(
		WithChainTimeout(60),
		WithGraphMaxWorkers(10),
		WithWorkflowTaskQueue("custom-queue"),
		WithMetricsPrefix("custom.orchestration"),
	)

	if err != nil {
		t.Fatalf("Failed to create orchestrator with options: %v", err)
	}

	if orch == nil {
		t.Fatal("Orchestrator should not be nil")
	}

	// Test that the orchestrator works with the custom configuration
	steps := []core.Runnable{&MockRunnable{name: "test"}}
	chain, err := orch.CreateChain(steps)
	if err != nil {
		t.Fatalf("Failed to create chain with custom orchestrator: %v", err)
	}

	if chain == nil {
		t.Fatal("Chain should not be nil")
	}
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
		name          string
		operation     func() error
		expectRetryable bool
		expectedCode   string
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
