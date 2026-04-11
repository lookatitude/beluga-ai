package procedural

// Option configures a ProceduralMemory.
type Option func(*options)

type options struct {
	hooks         Hooks
	minConfidence float64
}

func defaults() options {
	return options{
		minConfidence: 0.5,
	}
}

// WithHooks sets the hooks for procedural memory operations.
func WithHooks(h Hooks) Option {
	return func(o *options) {
		o.hooks = h
	}
}

// WithMinConfidence sets the minimum confidence threshold for skill retrieval.
// Skills below this threshold are excluded from search results. Default is 0.5.
func WithMinConfidence(c float64) Option {
	return func(o *options) {
		o.minConfidence = c
	}
}
