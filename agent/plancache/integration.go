package plancache

import (
	"github.com/lookatitude/beluga-ai/agent"
)

// WrapPlanner is a convenience helper that wraps an agent.Planner with plan
// caching using the default keyword matcher and an in-memory store. This is
// the simplest way to add plan caching to an existing agent.
//
// Example:
//
//	cached, err := plancache.WrapPlanner(myPlanner,
//	    plancache.WithMinScore(0.7),
//	    plancache.WithMaxTemplates(50),
//	)
func WrapPlanner(inner agent.Planner, opts ...Option) (*CachedPlanner, error) {
	matcher, err := NewMatcher("keyword")
	if err != nil {
		return nil, newCacheError("plancache.WrapPlanner", "failed to create default matcher", err)
	}

	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	store := NewInMemoryStore(o.maxTemplates)
	return Wrap(inner, store, matcher, opts...), nil
}
