package orchestration

import (
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
)

// SimpleChainOption represents a functional option for configuring SimpleChain
type SimpleChainOption func(*SimpleChainConfig)

// SimpleChainConfig holds configuration for SimpleChain
type SimpleChainConfig struct {
	Steps  []core.Runnable
	Memory memory.Memory
}

// WithSteps sets the chain steps
func WithSteps(steps ...core.Runnable) SimpleChainOption {
	return func(config *SimpleChainConfig) {
		config.Steps = steps
	}
}

// WithMemory sets the memory component
func WithMemory(mem memory.Memory) SimpleChainOption {
	return func(config *SimpleChainConfig) {
		config.Memory = mem
	}
}

// DefaultSimpleChainConfig returns default configuration
func DefaultSimpleChainConfig() *SimpleChainConfig {
	return &SimpleChainConfig{
		Steps:  make([]core.Runnable, 0),
		Memory: nil,
	}
}

// NewSimpleChainWithOptions creates a new SimpleChain with functional options
func NewSimpleChainWithOptions(options ...SimpleChainOption) *SimpleChain {
	config := DefaultSimpleChainConfig()

	for _, option := range options {
		option(config)
	}

	return NewSimpleChain(config.Steps, config.Memory)
}

// ChainBuilder provides a fluent interface for building chains
type ChainBuilder struct {
	config *SimpleChainConfig
}

// NewChainBuilder creates a new chain builder
func NewChainBuilder() *ChainBuilder {
	return &ChainBuilder{
		config: DefaultSimpleChainConfig(),
	}
}

// AddStep adds a step to the chain
func (b *ChainBuilder) AddStep(step core.Runnable) *ChainBuilder {
	b.config.Steps = append(b.config.Steps, step)
	return b
}

// WithMemory sets the memory for the chain
func (b *ChainBuilder) WithMemory(memory memory.Memory) *ChainBuilder {
	b.config.Memory = memory
	return b
}

// Build creates the chain
func (b *ChainBuilder) Build() *SimpleChain {
	return NewSimpleChain(b.config.Steps, b.config.Memory)
}

// OrchestrationContainer provides DI for orchestration components
type OrchestrationContainer struct {
	container core.Container
}

// NewOrchestrationContainer creates a new orchestration DI container
func NewOrchestrationContainer() *OrchestrationContainer {
	container := core.NewContainer()

	// Register common orchestration components
	container.Register(func(steps []core.Runnable, memory memory.Memory) (*SimpleChain, error) {
		return NewSimpleChain(steps, memory), nil
	})

	return &OrchestrationContainer{
		container: container,
	}
}

// BuildChain builds a chain using DI
func (oc *OrchestrationContainer) BuildChain(steps []core.Runnable, memory memory.Memory) (*SimpleChain, error) {
	var chain *SimpleChain
	err := oc.container.Resolve(&chain)
	return chain, err
}

// RegisterComponent registers a custom component factory
func (oc *OrchestrationContainer) RegisterComponent(factoryFunc interface{}) error {
	return oc.container.Register(factoryFunc)
}

// GetContainer returns the underlying DI container
func (oc *OrchestrationContainer) GetContainer() core.Container {
	return oc.container
}

// WorkflowOption represents a functional option for workflow configuration
type WorkflowOption func(*iface.WorkflowConfig)

// WithWorkflowName sets the workflow name
func WithWorkflowName(name string) WorkflowOption {
	return func(config *iface.WorkflowConfig) {
		config.Name = name
	}
}

// WithWorkflowDescription sets the workflow description
func WithWorkflowDescription(description string) WorkflowOption {
	return func(config *iface.WorkflowConfig) {
		config.Description = description
	}
}

// WithWorkflowTimeout sets the workflow timeout
func WithWorkflowTimeout(timeout int) WorkflowOption {
	return func(config *iface.WorkflowConfig) {
		config.Timeout = timeout
	}
}

// WithWorkflowRetries sets the workflow retry count
func WithWorkflowRetries(retries int) WorkflowOption {
	return func(config *iface.WorkflowConfig) {
		config.Retries = retries
	}
}

// WithWorkflowContainer sets the DI container
func WithWorkflowContainer(container core.Container) WorkflowOption {
	return func(config *iface.WorkflowConfig) {
		config.Container = container
	}
}

// DefaultWorkflowConfig returns default workflow configuration
func DefaultWorkflowConfig() *iface.WorkflowConfig {
	return &iface.WorkflowConfig{
		Name:      "default-workflow",
		Timeout:   300, // 5 minutes
		Retries:   3,
		Container: core.NewContainer(),
	}
}

// WorkflowBuilder provides a fluent interface for building workflows
type WorkflowBuilder struct {
	config *iface.WorkflowConfig
}

// NewWorkflowBuilder creates a new workflow builder
func NewWorkflowBuilder() *WorkflowBuilder {
	return &WorkflowBuilder{
		config: DefaultWorkflowConfig(),
	}
}

// WithName sets the workflow name
func (b *WorkflowBuilder) WithName(name string) *WorkflowBuilder {
	b.config.Name = name
	return b
}

// WithDescription sets the workflow description
func (b *WorkflowBuilder) WithDescription(description string) *WorkflowBuilder {
	b.config.Description = description
	return b
}

// WithTimeout sets the workflow timeout
func (b *WorkflowBuilder) WithTimeout(timeout int) *WorkflowBuilder {
	b.config.Timeout = timeout
	return b
}

// WithRetries sets the retry count
func (b *WorkflowBuilder) WithRetries(retries int) *WorkflowBuilder {
	b.config.Retries = retries
	return b
}

// WithContainer sets the DI container
func (b *WorkflowBuilder) WithContainer(container core.Container) *WorkflowBuilder {
	b.config.Container = container
	return b
}

// Build creates a workflow configuration (placeholder for future workflow implementation)
func (b *WorkflowBuilder) Build() *iface.WorkflowConfig {
	return b.config
}
