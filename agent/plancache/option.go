package plancache

// options holds configuration for CachedPlanner.
type options struct {
	minScore          float64
	maxTemplates      int
	evictionThreshold float64
	hooks             Hooks
	keywordExtractor  func(string) []string
}

// defaultOptions returns sensible defaults.
func defaultOptions() options {
	return options{
		minScore:          0.6,
		maxTemplates:      100,
		evictionThreshold: 0.5,
		keywordExtractor:  ExtractKeywords,
	}
}

// Option configures a CachedPlanner.
type Option func(*options)

// WithMinScore sets the minimum similarity score for a cache hit. Scores below
// this threshold are treated as cache misses. Must be in [0.0, 1.0].
// Default: 0.6.
func WithMinScore(score float64) Option {
	return func(o *options) {
		if score >= 0.0 && score <= 1.0 {
			o.minScore = score
		}
	}
}

// WithMaxTemplates sets the maximum number of templates to consider per agent.
// Templates beyond this limit are subject to eviction. Default: 100.
func WithMaxTemplates(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxTemplates = n
		}
	}
}

// WithEvictionThreshold sets the deviation ratio above which a template is
// evicted. Must be in [0.0, 1.0]. Default: 0.5.
func WithEvictionThreshold(threshold float64) Option {
	return func(o *options) {
		if threshold >= 0.0 && threshold <= 1.0 {
			o.evictionThreshold = threshold
		}
	}
}

// WithHooks sets the lifecycle hooks for the cached planner.
func WithHooks(hooks Hooks) Option {
	return func(o *options) {
		o.hooks = hooks
	}
}

// WithKeywordExtractor sets a custom keyword extraction function. The function
// takes an input string and returns a slice of keywords. Default:
// ExtractKeywords.
func WithKeywordExtractor(fn func(string) []string) Option {
	return func(o *options) {
		if fn != nil {
			o.keywordExtractor = fn
		}
	}
}
