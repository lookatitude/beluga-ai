// Package orchestration provides registry for orchestration providers.
package orchestration

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
)

// ChainFactory creates a new Chain instance.
type ChainFactory func(ctx context.Context, steps []core.Runnable, opts ...iface.ChainOption) (iface.Chain, error)

// GraphFactory creates a new Graph instance.
type GraphFactory func(ctx context.Context, opts ...iface.GraphOption) (iface.Graph, error)

// WorkflowFactory creates a new Workflow instance.
type WorkflowFactory func(ctx context.Context, opts ...iface.WorkflowOption) (iface.Workflow, error)

// Registry manages orchestration provider registration and retrieval.
// It supports chains, graphs, and workflows following the standard Beluga AI pattern.
type Registry struct {
	chainFactories    map[string]ChainFactory
	graphFactories    map[string]GraphFactory
	workflowFactories map[string]WorkflowFactory
	mu                sync.RWMutex
}

// Global registry instance.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry returns the global registry instance.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = NewRegistry()
	})
	return globalRegistry
}

// NewRegistry creates a new Registry instance.
// Use GetRegistry() for most cases; this is mainly for testing.
func NewRegistry() *Registry {
	return &Registry{
		chainFactories:    make(map[string]ChainFactory),
		graphFactories:    make(map[string]GraphFactory),
		workflowFactories: make(map[string]WorkflowFactory),
	}
}

// RegisterChain registers a chain provider factory.
func (r *Registry) RegisterChain(name string, factory ChainFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chainFactories[name] = factory
}

// RegisterGraph registers a graph provider factory.
func (r *Registry) RegisterGraph(name string, factory GraphFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.graphFactories[name] = factory
}

// RegisterWorkflow registers a workflow provider factory.
func (r *Registry) RegisterWorkflow(name string, factory WorkflowFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflowFactories[name] = factory
}

// CreateChain creates a chain using the registered provider.
func (r *Registry) CreateChain(ctx context.Context, name string, steps []core.Runnable, opts ...iface.ChainOption) (iface.Chain, error) {
	r.mu.RLock()
	factory, exists := r.chainFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.ErrInvalidConfig("CreateChain", fmt.Errorf("chain provider '%s' not registered", name))
	}

	return factory(ctx, steps, opts...)
}

// CreateGraph creates a graph using the registered provider.
func (r *Registry) CreateGraph(ctx context.Context, name string, opts ...iface.GraphOption) (iface.Graph, error) {
	r.mu.RLock()
	factory, exists := r.graphFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.ErrInvalidConfig("CreateGraph", fmt.Errorf("graph provider '%s' not registered", name))
	}

	return factory(ctx, opts...)
}

// CreateWorkflow creates a workflow using the registered provider.
func (r *Registry) CreateWorkflow(ctx context.Context, name string, opts ...iface.WorkflowOption) (iface.Workflow, error) {
	r.mu.RLock()
	factory, exists := r.workflowFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.ErrInvalidConfig("CreateWorkflow", fmt.Errorf("workflow provider '%s' not registered", name))
	}

	return factory(ctx, opts...)
}

// ListChainProviders returns a list of all registered chain provider names.
func (r *Registry) ListChainProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.chainFactories))
	for name := range r.chainFactories {
		names = append(names, name)
	}
	return names
}

// ListGraphProviders returns a list of all registered graph provider names.
func (r *Registry) ListGraphProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.graphFactories))
	for name := range r.graphFactories {
		names = append(names, name)
	}
	return names
}

// ListWorkflowProviders returns a list of all registered workflow provider names.
func (r *Registry) ListWorkflowProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.workflowFactories))
	for name := range r.workflowFactories {
		names = append(names, name)
	}
	return names
}

// IsChainRegistered checks if a chain provider is registered.
func (r *Registry) IsChainRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.chainFactories[name]
	return exists
}

// IsGraphRegistered checks if a graph provider is registered.
func (r *Registry) IsGraphRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.graphFactories[name]
	return exists
}

// IsWorkflowRegistered checks if a workflow provider is registered.
func (r *Registry) IsWorkflowRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.workflowFactories[name]
	return exists
}

// Clear removes all providers from the registry.
// This is mainly useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chainFactories = make(map[string]ChainFactory)
	r.graphFactories = make(map[string]GraphFactory)
	r.workflowFactories = make(map[string]WorkflowFactory)
}
