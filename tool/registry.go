package tool

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe, name-based collection of tools.
// Tools are registered as instances and looked up by name.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new empty tool Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Add registers a tool in the registry. Returns an error if a tool with the
// same name is already registered.
func (r *Registry) Add(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = t
	return nil
}

// Get returns the tool with the given name, or an error if not found.
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t, nil
}

// List returns a sorted list of all registered tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Remove unregisters the tool with the given name. Returns an error if the
// tool is not found.
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tools[name]; !ok {
		return fmt.Errorf("tool %q not found", name)
	}
	delete(r.tools, name)
	return nil
}

// All returns all registered tools as a slice, sorted by name.
func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		tools = append(tools, r.tools[name])
	}
	return tools
}

// Definitions returns schema.ToolDefinition for all registered tools,
// sorted by name.
func (r *Registry) Definitions() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	defs := make([]map[string]any, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
		defs = append(defs, map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"inputSchema": t.InputSchema(),
		})
	}
	return defs
}
