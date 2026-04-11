// Package plancache provides agentic plan caching for the Beluga AI framework.
//
// Plan caching reduces redundant LLM calls by caching successful action plans
// as reusable templates. When a similar input arrives, the cached plan template
// is returned instead of invoking the planner, significantly reducing latency
// and cost.
//
// # Architecture
//
// The package consists of:
//   - [Template] — a cached plan with keyword metadata and success/deviation tracking
//   - [Matcher] — scores how well an input matches a cached template (1-method interface)
//   - [Store] — persists and retrieves templates (4-method interface)
//   - [CachedPlanner] — wraps any [agent.Planner] to add caching behavior
//
// # Usage
//
// Wrap an existing planner with caching:
//
//	store := plancache.NewInMemoryStore(100) // capacity of 100 templates
//	matcher, _ := plancache.NewMatcher("keyword")
//
//	cached := plancache.Wrap(myPlanner, store, matcher,
//	    plancache.WithMinScore(0.6),
//	    plancache.WithMaxTemplates(50),
//	    plancache.WithEvictionThreshold(0.5),
//	)
//
// The cached planner implements [agent.Planner] and can be used anywhere a
// planner is expected.
//
// # Cache Behavior
//
// On Plan(): the cached planner extracts keywords from the input, scores all
// stored templates, and returns the best match if it exceeds the minimum score.
// On a miss, the inner planner is called and the result is extracted as a new
// template.
//
// On Replan(): the inner planner is always called (replanning implies the
// cached plan was insufficient). The template's deviation count is incremented,
// and templates with excessive deviation ratios are evicted.
//
// # Registry
//
// Matcher implementations are registered via the matcher registry:
//
//	plancache.RegisterMatcher("custom", func() (plancache.Matcher, error) {
//	    return &CustomMatcher{}, nil
//	})
//
//	matchers := plancache.ListMatchers() // ["custom", "keyword"]
package plancache
