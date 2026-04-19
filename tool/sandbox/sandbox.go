package sandbox

import (
	"context"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Sandbox is the interface for sandboxed code execution. Implementations
// execute arbitrary code in an isolated environment and return the result.
type Sandbox interface {
	// Execute runs code in the sandbox with the given configuration and
	// returns the execution result. The context controls cancellation and
	// deadline; if the context is cancelled the execution is aborted.
	Execute(ctx context.Context, code string, cfg SandboxConfig) (ExecutionResult, error)

	// Close releases all resources held by the sandbox. After Close returns,
	// Execute must not be called.
	Close(ctx context.Context) error
}

// Factory is a constructor function that creates a new Sandbox instance.
type Factory func() (Sandbox, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// RegisterSandbox registers a sandbox factory under the given name.
// This is typically called from an init() function.
func RegisterSandbox(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = f
}

// NewSandbox creates a new Sandbox using the registered factory for name.
// Returns an error if no factory is registered under that name.
func NewSandbox(name string) (Sandbox, error) {
	mu.RLock()
	f, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "sandbox: unknown provider %q (registered: %v)", name, ListSandboxes())
	}
	return f()
}

// ListSandboxes returns a sorted list of all registered sandbox provider names.
func ListSandboxes() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	RegisterSandbox("process", func() (Sandbox, error) {
		return NewProcessSandbox(), nil
	})
}
