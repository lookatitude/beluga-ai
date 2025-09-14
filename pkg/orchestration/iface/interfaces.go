// Package iface defines the core interfaces for orchestration components.
// These interfaces follow the Interface Segregation Principle and provide
// focused contracts for different orchestration patterns.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory"
)

// Chain represents a sequence of components executed one after another.
// The output of one step is typically the input to the next.
type Chain interface {
	core.Runnable // Chains are Runnable

	// GetInputKeys returns the expected input keys for the chain.
	GetInputKeys() []string

	// GetOutputKeys returns the keys produced by the chain.
	GetOutputKeys() []string

	// GetMemory returns the memory associated with the chain, if any.
	GetMemory() memory.Memory
}

// Graph represents a more complex orchestration where components can be executed
// based on dependencies or conditions, forming a directed acyclic graph (DAG).
type Graph interface {
	core.Runnable // Graphs are Runnable

	// AddNode adds a Runnable component as a node in the graph.
	AddNode(name string, runnable core.Runnable) error

	// AddEdge defines a dependency between two nodes.
	AddEdge(sourceNode string, targetNode string) error

	// SetEntryPoint defines the starting node(s) of the graph.
	SetEntryPoint(nodeNames []string) error

	// SetFinishPoint defines the final node(s) whose output is the graph's output.
	SetFinishPoint(nodeNames []string) error
}

// Workflow represents a long-running, potentially distributed orchestration.
// This interface can be implemented using systems like Temporal or other workflow engines.
type Workflow interface {
	// Execute starts the workflow execution.
	Execute(ctx context.Context, input any) (workflowID string, runID string, err error)

	// GetResult retrieves the final result of a completed workflow instance.
	GetResult(ctx context.Context, workflowID string, runID string) (any, error)

	// Signal sends a signal to a running workflow instance.
	Signal(ctx context.Context, workflowID string, runID string, signalName string, data any) error

	// Query queries the state of a running workflow instance.
	Query(ctx context.Context, workflowID string, runID string, queryType string, args ...any) (any, error)

	// Cancel requests cancellation of a running workflow instance.
	Cancel(ctx context.Context, workflowID string, runID string) error

	// Terminate forcefully stops a running workflow instance.
	Terminate(ctx context.Context, workflowID string, runID string, reason string, details ...any) error
}

// Activity represents a unit of work within a workflow, often corresponding to a Beluga Runnable.
// This interface helps bridge Beluga components with workflow systems.
type Activity interface {
	// Execute performs the activity's logic.
	Execute(ctx context.Context, input any) (any, error)
}

// Orchestrator defines the main orchestration interface that combines
// different orchestration patterns.
type Orchestrator interface {
	// CreateChain creates a new chain orchestration
	CreateChain(steps []core.Runnable, opts ...ChainOption) (Chain, error)

	// CreateGraph creates a new graph orchestration
	CreateGraph(opts ...GraphOption) (Graph, error)

	// CreateWorkflow creates a new workflow orchestration
	CreateWorkflow(workflowFn any, opts ...WorkflowOption) (Workflow, error)

	// GetMetrics returns orchestration metrics
	GetMetrics() OrchestratorMetrics
}

// OrchestratorMetrics provides observability into orchestration performance.
type OrchestratorMetrics interface {
	// GetActiveChains returns the number of currently active chains
	GetActiveChains() int

	// GetActiveGraphs returns the number of currently active graphs
	GetActiveGraphs() int

	// GetActiveWorkflows returns the number of currently active workflows
	GetActiveWorkflows() int

	// GetTotalExecutions returns total number of executions across all orchestration types
	GetTotalExecutions() int64

	// GetErrorCount returns total number of errors across all orchestration types
	GetErrorCount() int64
}

// HealthChecker provides health check capabilities for orchestration components.
type HealthChecker interface {
	// Check performs a health check and returns an error if unhealthy
	Check(ctx context.Context) error
}

// ChainOption represents a functional option for configuring chains
type ChainOption func(*ChainConfig) error

// GraphOption represents a functional option for configuring graphs
type GraphOption func(*GraphConfig) error

// WorkflowOption represents a functional option for configuring workflows
type WorkflowOption func(*WorkflowConfig) error

// ChainConfig holds configuration for chain orchestration
type ChainConfig struct {
	Name        string
	Description string
	Memory      memory.Memory
	Steps       []core.Runnable
	InputKeys   []string
	OutputKeys  []string
	Timeout     int // seconds
	Retries     int
}

// GraphConfig holds configuration for graph orchestration
type GraphConfig struct {
	Name                    string
	Description             string
	Nodes                   map[string]core.Runnable
	Edges                   []GraphEdge
	EntryPoints             []string
	ExitPoints              []string
	Timeout                 int // seconds
	MaxWorkers              int
	EnableParallelExecution bool
}

// GraphEdge represents an edge in a graph orchestration
type GraphEdge struct {
	Source    string
	Target    string
	Condition string // optional condition for conditional edges
}

// WorkflowConfig holds configuration for workflow orchestration
type WorkflowConfig struct {
	Name        string
	Description string
	TaskQueue   string
	Timeout     int // seconds
	Retries     int
	Container   any // DI container
	Metadata    map[string]any
}
