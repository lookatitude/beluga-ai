// Package server provides registry for server providers.
package server

import (
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// ServerFactory creates a new server instance.
type ServerFactory func(opts ...iface.Option) (iface.Server, error)

// RESTServerFactory creates a new REST server instance.
type RESTServerFactory func(opts ...iface.Option) (RESTServer, error)

// MCPServerFactory creates a new MCP server instance.
type MCPServerFactory func(opts ...iface.Option) (MCPServer, error)

// Registry manages server provider registration and retrieval.
// It supports REST servers, MCP servers, and other server types.
type Registry struct {
	serverFactories     map[string]ServerFactory
	restServerFactories map[string]RESTServerFactory
	mcpServerFactories  map[string]MCPServerFactory
	mu                  sync.RWMutex
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
		serverFactories:     make(map[string]ServerFactory),
		restServerFactories: make(map[string]RESTServerFactory),
		mcpServerFactories:  make(map[string]MCPServerFactory),
	}
}

// RegisterServer registers a generic server provider factory.
func (r *Registry) RegisterServer(name string, factory ServerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.serverFactories[name] = factory
}

// RegisterRESTServer registers a REST server provider factory.
func (r *Registry) RegisterRESTServer(name string, factory RESTServerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.restServerFactories[name] = factory
}

// RegisterMCPServer registers an MCP server provider factory.
func (r *Registry) RegisterMCPServer(name string, factory MCPServerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mcpServerFactories[name] = factory
}

// CreateServer creates a server using the registered provider.
func (r *Registry) CreateServer(name string, opts ...iface.Option) (iface.Server, error) {
	r.mu.RLock()
	factory, exists := r.serverFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.NewConfigValidationError("CreateServer", fmt.Sprintf("server provider '%s' not registered", name))
	}

	return factory(opts...)
}

// CreateRESTServer creates a REST server using the registered provider.
func (r *Registry) CreateRESTServer(name string, opts ...iface.Option) (RESTServer, error) {
	r.mu.RLock()
	factory, exists := r.restServerFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.NewConfigValidationError("CreateRESTServer", fmt.Sprintf("REST server provider '%s' not registered", name))
	}

	return factory(opts...)
}

// CreateMCPServer creates an MCP server using the registered provider.
func (r *Registry) CreateMCPServer(name string, opts ...iface.Option) (MCPServer, error) {
	r.mu.RLock()
	factory, exists := r.mcpServerFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.NewConfigValidationError("CreateMCPServer", fmt.Sprintf("MCP server provider '%s' not registered", name))
	}

	return factory(opts...)
}

// ListServerProviders returns a list of all registered server provider names.
func (r *Registry) ListServerProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.serverFactories))
	for name := range r.serverFactories {
		names = append(names, name)
	}
	return names
}

// ListRESTServerProviders returns a list of all registered REST server provider names.
func (r *Registry) ListRESTServerProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.restServerFactories))
	for name := range r.restServerFactories {
		names = append(names, name)
	}
	return names
}

// ListMCPServerProviders returns a list of all registered MCP server provider names.
func (r *Registry) ListMCPServerProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.mcpServerFactories))
	for name := range r.mcpServerFactories {
		names = append(names, name)
	}
	return names
}

// IsServerRegistered checks if a server provider is registered.
func (r *Registry) IsServerRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.serverFactories[name]
	return exists
}

// IsRESTServerRegistered checks if a REST server provider is registered.
func (r *Registry) IsRESTServerRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.restServerFactories[name]
	return exists
}

// IsMCPServerRegistered checks if an MCP server provider is registered.
func (r *Registry) IsMCPServerRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.mcpServerFactories[name]
	return exists
}

// Clear removes all providers from the registry.
// This is mainly useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.serverFactories = make(map[string]ServerFactory)
	r.restServerFactories = make(map[string]RESTServerFactory)
	r.mcpServerFactories = make(map[string]MCPServerFactory)
}
