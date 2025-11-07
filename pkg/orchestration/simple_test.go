package orchestration

import (
	"testing"
)

// Simple test that doesn't depend on external packages
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestOrchestratorCreation(t *testing.T) {
	// Test config creation
	config := DefaultConfig()
	if config == nil {
		t.Fatal("Default config should not be nil")
	}

	// Test orchestrator creation with default config
	orch, err := NewOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	if orch == nil {
		t.Fatal("Orchestrator should not be nil")
	}

	// Test metrics
	metrics := orch.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	activeChains := metrics.GetActiveChains()
	activeGraphs := metrics.GetActiveGraphs()
	activeWorkflows := metrics.GetActiveWorkflows()

	t.Logf("Orchestrator created successfully:")
	t.Logf("  Active chains: %d", activeChains)
	t.Logf("  Active graphs: %d", activeGraphs)
	t.Logf("  Active workflows: %d", activeWorkflows)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestConfigurationOptions(t *testing.T) {
	// Test configuration with options
	config, err := NewConfig(
		WithChainTimeout(45),
		WithGraphMaxWorkers(8),
		WithMetricsPrefix("test.prefix"),
	)

	if err != nil {
		t.Fatalf("Failed to create config with options: %v", err)
	}

	if config.Chain.DefaultTimeout != 45 {
		t.Errorf("Expected chain timeout 45, got %v", config.Chain.DefaultTimeout)
	}

	if config.Graph.MaxWorkers != 8 {
		t.Errorf("Expected graph max workers 8, got %v", config.Graph.MaxWorkers)
	}

	if config.Observability.MetricsPrefix != "test.prefix" {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Expected metrics prefix 'test.prefix', got %v", config.Observability.MetricsPrefix)
	}
}

func TestWorkflowConfig(t *testing.T) {
	// Test workflow configuration functions
	builder := NewWorkflowBuilder()
	if builder == nil {
		t.Fatal("Workflow builder should not be nil")
	}

	// Build default config
	config := builder.Build()
	if config == nil {
		t.Fatal("Workflow config should not be nil")
	}

	// Test fluent interface
	config = NewWorkflowBuilder().
		WithName("test-workflow").
		WithDescription("Test workflow").
		WithTimeout(120).
		WithRetries(5).
		Build()

	if config.Name != "test-workflow" {
		t.Errorf("Expected name 'test-workflow', got %v", config.Name)
	}

	if config.Description != "Test workflow" {
		t.Errorf("Expected description 'Test workflow', got %v", config.Description)
	}

	if config.Timeout != 120 {
		t.Errorf("Expected timeout 120, got %v", config.Timeout)
	}

	if config.Retries != 5 {
		t.Errorf("Expected retries 5, got %v", config.Retries)
	}
}
