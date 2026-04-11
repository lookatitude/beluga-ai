// Package temporal provides bi-temporal knowledge graph memory for agents.
// It implements the Graphiti-inspired 2-condition conflict resolution algorithm
// and supports querying the knowledge graph as it existed at any point in time.
package temporal

// Option configures a TemporalMemory.
type Option func(*options)

type options struct {
	resolver ConflictResolver
	hooks    Hooks
}

func defaultOptions() options {
	return options{
		resolver: NewTemporalResolver(),
	}
}

// WithConflictResolver sets a custom conflict resolver for the temporal memory.
// If not set, the default TemporalResolver (Graphiti 2-condition algorithm) is used.
func WithConflictResolver(r ConflictResolver) Option {
	return func(o *options) {
		if r != nil {
			o.resolver = r
		}
	}
}

// WithHooks sets the hooks for the temporal memory.
func WithHooks(h Hooks) Option {
	return func(o *options) {
		o.hooks = h
	}
}
