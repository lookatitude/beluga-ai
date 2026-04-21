package teambuilder

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

const opPoolRegister = "teambuilder.pool.register"

// AgentPool is a thread-safe registry of candidate agents with their
// capabilities and performance metrics. It serves as the source of
// agents for dynamic team formation.
type AgentPool struct {
	mu      sync.RWMutex
	entries map[string]*PoolEntry
}

// PoolOption configures an AgentPool.
type PoolOption func(*AgentPool)

// NewAgentPool creates a new empty AgentPool.
func NewAgentPool(opts ...PoolOption) *AgentPool {
	p := &AgentPool{
		entries: make(map[string]*PoolEntry),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Register adds an agent to the pool with the given capabilities.
// Returns an error if the agent is nil or already registered.
func (p *AgentPool) Register(a agent.Agent, capabilities ...string) error {
	if a == nil {
		return core.NewError(opPoolRegister, core.ErrInvalidInput,
			"agent must not be nil", nil)
	}

	id := a.ID()
	if id == "" {
		return core.NewError(opPoolRegister, core.ErrInvalidInput,
			"agent must have a non-empty ID", nil)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.entries[id]; exists {
		return core.NewError(opPoolRegister, core.ErrInvalidInput,
			fmt.Sprintf("agent %q is already registered", id), nil)
	}

	caps := make([]string, len(capabilities))
	copy(caps, capabilities)

	p.entries[id] = &PoolEntry{
		Agent:        a,
		Capabilities: caps,
		Metrics:      NewAgentMetrics(),
	}
	return nil
}

// Unregister removes an agent from the pool by ID.
// Returns an error if the agent is not found.
func (p *AgentPool) Unregister(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.entries[id]; !exists {
		return core.NewError("teambuilder.pool.unregister", core.ErrNotFound,
			fmt.Sprintf("agent %q not found in pool", id), nil)
	}
	delete(p.entries, id)
	return nil
}

// List returns a snapshot of all pool entries. The returned slice is a
// copy; modifications do not affect the pool.
func (p *AgentPool) List() []PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entries := make([]PoolEntry, 0, len(p.entries))
	for _, e := range p.entries {
		entries = append(entries, *e)
	}
	return entries
}

// Select uses the given selector to pick agents from the pool for a task.
// Returns an error if the pool is empty or the selector fails.
func (p *AgentPool) Select(ctx context.Context, task string, selector Selector) ([]agent.Agent, error) {
	if selector == nil {
		return nil, core.NewError("teambuilder.pool.select", core.ErrInvalidInput,
			"selector must not be nil", nil)
	}

	candidates := p.List()
	if len(candidates) == 0 {
		return nil, core.NewError("teambuilder.pool.select", core.ErrNotFound,
			"agent pool is empty", nil)
	}

	selected, err := selector.Select(ctx, task, candidates)
	if err != nil {
		return nil, core.NewError("teambuilder.pool.select", core.ErrToolFailed,
			"selector failed", err)
	}

	agents := make([]agent.Agent, len(selected))
	for i, e := range selected {
		agents[i] = e.Agent
	}
	return agents, nil
}

// Get returns the pool entry for the given agent ID, or false if not found.
func (p *AgentPool) Get(id string) (PoolEntry, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.entries[id]
	if !ok {
		return PoolEntry{}, false
	}
	return *e, true
}

// Metrics returns the metrics for the given agent ID. Returns nil if not found.
func (p *AgentPool) Metrics(id string) *AgentMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.entries[id]
	if !ok {
		return nil
	}
	return e.Metrics
}
