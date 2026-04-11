package plancache

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
)

// CachedPlanner wraps an agent.Planner to add plan caching. On Plan(), it
// checks the cache for a matching template and returns cached actions on a
// hit. On a miss, it delegates to the inner planner and extracts a new
// template. On Replan(), it always delegates to the inner planner and tracks
// deviation.
type CachedPlanner struct {
	inner   agent.Planner
	store   Store
	matcher Matcher
	opts    options

	// updateMu serializes read-modify-write updates to template counters to
	// prevent lost updates under concurrent Plan/Replan calls on the same
	// template.
	updateMu sync.Mutex
}

var _ agent.Planner = (*CachedPlanner)(nil)

// Wrap creates a CachedPlanner that wraps the given inner planner with
// caching behavior.
func Wrap(inner agent.Planner, store Store, matcher Matcher, opts ...Option) *CachedPlanner {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &CachedPlanner{
		inner:   inner,
		store:   store,
		matcher: matcher,
		opts:    o,
	}
}

// Plan checks the cache for a matching template. If a match with a score
// above the minimum threshold is found, the cached actions are returned. On
// a miss, the inner planner is called and the result is extracted as a new
// template.
func (cp *CachedPlanner) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	agentID := agentIDFromState(state)

	// Try to find a cached match.
	tmpl, score, err := cp.findBestMatch(ctx, agentID, state.Input)
	if err != nil {
		// Cache errors are non-fatal; fall through to inner planner.
		return cp.planAndCache(ctx, state, agentID)
	}

	if tmpl != nil && score >= cp.opts.minScore {
		// Cache hit.
		if cp.opts.hooks.OnCacheHit != nil {
			cp.opts.hooks.OnCacheHit(ctx, tmpl, score)
		}

		// Serialize counter updates to avoid lost updates under concurrency.
		cp.updateMu.Lock()
		if fresh, err := cp.store.Get(ctx, tmpl.ID); err == nil && fresh != nil {
			fresh.SuccessCount++
			_ = cp.store.Save(ctx, fresh) // best-effort update
		} else {
			tmpl.SuccessCount++
			_ = cp.store.Save(ctx, tmpl)
		}
		cp.updateMu.Unlock()

		return templateToActions(tmpl), nil
	}

	// Cache miss.
	if cp.opts.hooks.OnCacheMiss != nil {
		cp.opts.hooks.OnCacheMiss(ctx, state.Input)
	}

	return cp.planAndCache(ctx, state, agentID)
}

// Replan always delegates to the inner planner. It increments the deviation
// count on any matching template and evicts templates that exceed the
// eviction threshold.
func (cp *CachedPlanner) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	agentID := agentIDFromState(state)

	// Track deviation on existing template if one matched.
	tmpl, _, _ := cp.findBestMatch(ctx, agentID, state.Input)
	if tmpl != nil {
		// Serialize read-modify-write updates to avoid lost deviations under
		// concurrent Replan calls on the same template.
		cp.updateMu.Lock()
		fresh, err := cp.store.Get(ctx, tmpl.ID)
		if err != nil || fresh == nil {
			fresh = tmpl
		}
		fresh.DeviationCount++
		_ = cp.store.Save(ctx, fresh)
		evict := fresh.DeviationRatio() > cp.opts.evictionThreshold
		cp.updateMu.Unlock()

		if evict {
			_ = cp.store.Delete(ctx, fresh.ID)
			if cp.opts.hooks.OnTemplateEvicted != nil {
				cp.opts.hooks.OnTemplateEvicted(ctx, fresh)
			}
		}
	}

	return cp.inner.Replan(ctx, state)
}

// findBestMatch searches the store for the best matching template for the
// given input. Returns nil, 0, nil if no templates exist.
func (cp *CachedPlanner) findBestMatch(ctx context.Context, agentID, input string) (*Template, float64, error) {
	templates, err := cp.store.List(ctx, agentID)
	if err != nil {
		return nil, 0, err
	}

	var bestTmpl *Template
	var bestScore float64

	for _, tmpl := range templates {
		score, err := cp.matcher.Score(ctx, input, tmpl)
		if err != nil {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestTmpl = tmpl
		}
	}

	return bestTmpl, bestScore, nil
}

// planAndCache calls the inner planner and caches the result as a template.
func (cp *CachedPlanner) planAndCache(ctx context.Context, state agent.PlannerState, agentID string) ([]agent.Action, error) {
	actions, err := cp.inner.Plan(ctx, state)
	if err != nil {
		return nil, err
	}

	if len(actions) > 0 {
		tmpl := ExtractTemplate(agentID, state.Input, actions, cp.opts.keywordExtractor)

		// Enforce max templates per agent.
		if err := cp.enforceMaxTemplates(ctx, agentID); err == nil {
			if saveErr := cp.store.Save(ctx, tmpl); saveErr == nil {
				if cp.opts.hooks.OnTemplateExtracted != nil {
					cp.opts.hooks.OnTemplateExtracted(ctx, tmpl)
				}
			}
		}
	}

	return actions, nil
}

// enforceMaxTemplates checks the per-agent template count and returns a
// cache error when the configured cap has been reached. Callers use the
// returned error as a signal to skip saving new templates, keeping the
// per-agent limit independent of the store's own global capacity.
func (cp *CachedPlanner) enforceMaxTemplates(ctx context.Context, agentID string) error {
	templates, err := cp.store.List(ctx, agentID)
	if err != nil {
		return err
	}
	if len(templates) >= cp.opts.maxTemplates {
		return newCacheError("plancache.enforceMaxTemplates", "per-agent template limit reached", nil)
	}
	return nil
}

// templateToActions converts a template's actions back to agent actions. Tool
// actions preserve the original captured Arguments payload so executors can
// invoke the plan directly.
func templateToActions(tmpl *Template) []agent.Action {
	actions := make([]agent.Action, len(tmpl.Actions))
	for i, ta := range tmpl.Actions {
		a := agent.Action{
			Type:    ta.Type,
			Message: ta.Description,
		}
		if ta.Type == agent.ActionTool && ta.ToolName != "" {
			a.ToolCall = &schema.ToolCall{
				Name:      ta.ToolName,
				Arguments: ta.Arguments,
			}
		}
		actions[i] = a
	}
	return actions
}

// defaultAgentID is the fallback namespace used when planner state does not
// carry an explicit "agent_id" metadata entry. Callers sharing the same
// fallback namespace will share a template pool — always set
// state.Metadata["agent_id"] explicitly to isolate unrelated planners.
const defaultAgentID = "default"

// agentIDFromState extracts or derives an agent ID from planner state.
//
// NOTE: when state.Metadata lacks "agent_id", every planner — regardless of
// its toolset or purpose — falls into the same defaultAgentID namespace.
// Templates extracted for one agent can therefore match queries from another
// agent in that pool. Set state.Metadata["agent_id"] explicitly to isolate
// unrelated planners.
func agentIDFromState(state agent.PlannerState) string {
	if id, ok := state.Metadata["agent_id"].(string); ok && id != "" {
		return id
	}
	return defaultAgentID
}
