// Package orchestration provides comprehensive orchestration capabilities for Beluga AI.
// It supports chains, graphs, and workflows with full observability and configuration management.
package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/providers/chain"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/providers/graph"
)

// Orchestrator implements the main orchestration interface
type Orchestrator struct {
	config  *Config
	metrics *Metrics
	tracer  trace.Tracer
	mu      sync.RWMutex

	// Active orchestration instances
	activeChains    map[string]iface.Chain
	activeGraphs    map[string]iface.Graph
	activeWorkflows map[string]iface.Workflow
}

// NewOrchestrator creates a new orchestrator with the given configuration
func NewOrchestrator(config *Config) (*Orchestrator, error) {
	if config == nil {
		return nil, iface.ErrInvalidConfig("orchestrator_creation", fmt.Errorf("config cannot be nil"))
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	tracer := otel.Tracer("beluga.orchestration")

	var metrics *Metrics
	if config.Observability.EnableMetrics {
		meter := otel.Meter("beluga.orchestration")
		tracer := otel.Tracer("beluga.orchestration")
		var err error
		metrics, err = NewMetrics(meter, tracer)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics: %w", err)
		}
	}

	return &Orchestrator{
		config:          config,
		metrics:         metrics,
		tracer:          tracer,
		activeChains:    make(map[string]iface.Chain),
		activeGraphs:    make(map[string]iface.Graph),
		activeWorkflows: make(map[string]iface.Workflow),
	}, nil
}

// NewOrchestratorWithOptions creates a new orchestrator with functional options
func NewOrchestratorWithOptions(opts ...Option) (*Orchestrator, error) {
	config, err := NewConfig(opts...)
	if err != nil {
		return nil, err
	}

	return NewOrchestrator(config)
}

// NewDefaultOrchestrator creates a new orchestrator with default configuration
func NewDefaultOrchestrator() (*Orchestrator, error) {
	config := DefaultConfig()
	return NewOrchestrator(config)
}

// CreateChain creates a new chain orchestration
func (o *Orchestrator) CreateChain(steps []core.Runnable, opts ...iface.ChainOption) (iface.Chain, error) {
	if !o.config.Enabled.Chains {
		return nil, iface.ErrInvalidState("create_chain", "chains_disabled", "chains_enabled")
	}

	ctx, span := o.tracer.Start(context.Background(), "orchestrator.create_chain",
		trace.WithAttributes(
			attribute.Int("steps_count", len(steps)),
		))
	defer span.End()

	config := &iface.ChainConfig{
		Name:    fmt.Sprintf("chain-%d", time.Now().UnixNano()),
		Steps:   steps,
		Timeout: int(o.config.Chain.DefaultTimeout.Seconds()),
		Retries: o.config.Chain.DefaultRetries,
	}

	// Apply functional options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, iface.ErrInvalidConfig("create_chain", err)
		}
	}

	chain, err := o.createChainImplementation(ctx, config)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Track active chain
	o.mu.Lock()
	chainID := config.Name
	o.activeChains[chainID] = chain
	o.mu.Unlock()

	if o.metrics != nil {
		o.metrics.RecordChainActive(ctx, 1, config.Name)
	}

	return chain, nil
}

// CreateGraph creates a new graph orchestration
func (o *Orchestrator) CreateGraph(opts ...iface.GraphOption) (iface.Graph, error) {
	if !o.config.Enabled.Graphs {
		return nil, iface.ErrInvalidState("create_graph", "graphs_disabled", "graphs_enabled")
	}

	ctx, span := o.tracer.Start(context.Background(), "orchestrator.create_graph")
	defer span.End()

	config := &iface.GraphConfig{
		Name:       fmt.Sprintf("graph-%d", time.Now().UnixNano()),
		Nodes:      make(map[string]core.Runnable),
		Timeout:    int(o.config.Graph.DefaultTimeout.Seconds()),
		MaxWorkers: o.config.Graph.MaxWorkers,
	}

	// Apply functional options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, iface.ErrInvalidConfig("create_graph", err)
		}
	}

	graph, err := o.createGraphImplementation(ctx, config)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Track active graph
	o.mu.Lock()
	graphID := config.Name
	o.activeGraphs[graphID] = graph
	o.mu.Unlock()

	if o.metrics != nil {
		o.metrics.RecordGraphActive(ctx, 1, config.Name)
	}

	return graph, nil
}

// CreateWorkflow creates a new workflow orchestration
func (o *Orchestrator) CreateWorkflow(workflowFn any, opts ...iface.WorkflowOption) (iface.Workflow, error) {
	if !o.config.Enabled.Workflows {
		return nil, iface.ErrInvalidState("create_workflow", "workflows_disabled", "workflows_enabled")
	}

	ctx, span := o.tracer.Start(context.Background(), "orchestrator.create_workflow")
	defer span.End()

	config := &iface.WorkflowConfig{
		Name:      fmt.Sprintf("workflow-%d", time.Now().UnixNano()),
		TaskQueue: o.config.Workflow.TaskQueue,
		Timeout:   int(o.config.Workflow.DefaultTimeout.Seconds()),
		Retries:   o.config.Workflow.DefaultRetries,
	}

	// Apply functional options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, iface.ErrInvalidConfig("create_workflow", err)
		}
	}

	workflow, err := o.createWorkflowImplementation(ctx, workflowFn, config)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Track active workflow
	o.mu.Lock()
	workflowID := config.Name
	o.activeWorkflows[workflowID] = workflow
	o.mu.Unlock()

	if o.metrics != nil {
		o.metrics.RecordWorkflowActive(ctx, 1, config.Name)
	}

	return workflow, nil
}

// GetMetrics returns orchestration metrics
func (o *Orchestrator) GetMetrics() iface.OrchestratorMetrics {
	return &orchestratorMetrics{
		activeChains:    len(o.activeChains),
		activeGraphs:    len(o.activeGraphs),
		activeWorkflows: len(o.activeWorkflows),
	}
}

// Health check implementation
func (o *Orchestrator) Check(ctx context.Context) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Check if we can create a simple chain (basic functionality test)
	if o.config.Enabled.Chains {
		testSteps := []core.Runnable{} // Empty chain should work
		_, err := o.CreateChain(testSteps)
		if err != nil {
			return iface.ErrExecutionFailed("health_check", err)
		}
	}

	return nil
}

// Internal implementations that delegate to providers
func (o *Orchestrator) createChainImplementation(ctx context.Context, config *iface.ChainConfig) (iface.Chain, error) {
	// Create the chain using the provider
	tracer := otel.Tracer("beluga.orchestration.chain")

	// Create the chain using the provider
	simpleChain := chain.NewSimpleChain(*config, config.Memory, tracer)

	return simpleChain, nil
}

func (o *Orchestrator) createGraphImplementation(ctx context.Context, config *iface.GraphConfig) (iface.Graph, error) {
	// Create the graph using the provider
	tracer := otel.Tracer("beluga.orchestration.graph")

	// Create the graph using the provider
	basicGraph := graph.NewBasicGraph(*config, tracer)

	return basicGraph, nil
}

func (o *Orchestrator) createWorkflowImplementation(ctx context.Context, workflowFn any, config *iface.WorkflowConfig) (iface.Workflow, error) {
	// This would need a temporal client - for now return an error
	return nil, iface.ErrInvalidState("create_workflow", "temporal_client_not_configured", "temporal_client_required")
}

// orchestratorMetrics implements the OrchestratorMetrics interface
type orchestratorMetrics struct {
	activeChains    int
	activeGraphs    int
	activeWorkflows int
	totalExecutions int64
	errorCount      int64
}

func (m *orchestratorMetrics) GetActiveChains() int {
	return m.activeChains
}

func (m *orchestratorMetrics) GetActiveGraphs() int {
	return m.activeGraphs
}

func (m *orchestratorMetrics) GetActiveWorkflows() int {
	return m.activeWorkflows
}

func (m *orchestratorMetrics) GetTotalExecutions() int64 {
	return m.totalExecutions
}

func (m *orchestratorMetrics) GetErrorCount() int64 {
	return m.errorCount
}

// Factory functions for backward compatibility and convenience

// NewChain creates a new chain with default orchestrator
func NewChain(steps []core.Runnable, opts ...iface.ChainOption) (iface.Chain, error) {
	orch, err := NewDefaultOrchestrator()
	if err != nil {
		return nil, err
	}
	return orch.CreateChain(steps, opts...)
}

// NewGraph creates a new graph with default orchestrator
func NewGraph(opts ...iface.GraphOption) (iface.Graph, error) {
	orch, err := NewDefaultOrchestrator()
	if err != nil {
		return nil, err
	}
	return orch.CreateGraph(opts...)
}

// NewWorkflow creates a new workflow with default orchestrator
func NewWorkflow(workflowFn any, opts ...iface.WorkflowOption) (iface.Workflow, error) {
	orch, err := NewDefaultOrchestrator()
	if err != nil {
		return nil, err
	}
	return orch.CreateWorkflow(workflowFn, opts...)
}
