package contract

import (
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/schema"
)

// Factory creates a Contract from a name and options.
type Factory func(name string, opts ...Option) *schema.Contract

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a contract template factory to the global registry.
//
// Register MUST be called only from init() functions. Calling Register
// after program startup is unsupported and may result in race conditions
// or unexpected behavior; the mutex is present solely to guard against
// concurrent init() ordering issues across packages.
func Register(templateName string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[templateName] = f
}

// NewFromTemplate creates a new Contract from a registered template.
// The name parameter becomes the contract's Name; the templateName selects
// the factory. Returns an error if the template is not registered.
func NewFromTemplate(templateName, name string, opts ...Option) (*schema.Contract, error) {
	mu.RLock()
	f, ok := registry[templateName]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("contract: unknown template %q (registered: %v)", templateName, List())
	}
	return f(name, opts...), nil
}

// List returns the sorted names of all registered contract templates.
func List() []string {
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
	// text-in-text-out: agent that takes a string and produces a string.
	Register("text-in-text-out", func(name string, opts ...Option) *schema.Contract {
		c := &schema.Contract{
			Name:         name,
			Description:  "Accepts text input and produces text output.",
			InputSchema:  map[string]any{"type": "string"},
			OutputSchema: map[string]any{"type": "string"},
		}
		for _, opt := range opts {
			opt(c)
		}
		return c
	})

	// json-object: agent that takes a JSON object and produces a JSON object.
	Register("json-object", func(name string, opts ...Option) *schema.Contract {
		c := &schema.Contract{
			Name:         name,
			Description:  "Accepts a JSON object and produces a JSON object.",
			InputSchema:  map[string]any{"type": "object"},
			OutputSchema: map[string]any{"type": "object"},
		}
		for _, opt := range opts {
			opt(c)
		}
		return c
	})
}
