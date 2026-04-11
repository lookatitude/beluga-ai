package teambuilder

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Selector defines the strategy for selecting agents from a pool of candidates
// based on a task description. Implementations range from simple keyword
// matching to LLM-based semantic selection.
type Selector interface {
	// Select returns a ranked subset of candidates suitable for the given task.
	// The returned entries are ordered by relevance (best match first).
	// An empty result indicates no suitable agents were found.
	Select(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, error)
}

// ScoredPoolEntry pairs a PoolEntry with the concrete relevance score the
// selector computed for it. Scores are normalized to the range [0.0, 1.0]
// where 1.0 is the strongest match.
type ScoredPoolEntry struct {
	// Entry is the selected pool entry.
	Entry PoolEntry
	// Score is the selector's computed relevance score in [0.0, 1.0].
	Score float64
}

// ScoredSelector is an optional capability interface that a Selector may
// implement to expose the concrete relevance scores it computed for each
// selected entry. Consumers (e.g. TeamBuilder hooks) should prefer
// SelectScored when the selector supports it, and fall back to Select for
// plain selectors that do not carry score information.
//
// Results must be ordered by Score descending, matching the ordering
// contract of Selector.Select.
type ScoredSelector interface {
	Selector
	// SelectScored returns a ranked subset of candidates with their
	// concrete relevance scores in [0.0, 1.0], ordered by Score descending.
	SelectScored(ctx context.Context, task string, candidates []PoolEntry) ([]ScoredPoolEntry, error)
}

// SelectorFactory creates a Selector from a configuration map.
type SelectorFactory func(cfg map[string]any) (Selector, error)

var (
	selectorMu       sync.RWMutex
	selectorRegistry = make(map[string]SelectorFactory)
)

// RegisterSelector registers a named selector factory. This is typically
// called from init() in packages providing selector implementations.
func RegisterSelector(name string, f SelectorFactory) {
	selectorMu.Lock()
	defer selectorMu.Unlock()
	selectorRegistry[name] = f
}

// NewSelector creates a selector by name from the registry.
func NewSelector(name string, cfg map[string]any) (Selector, error) {
	selectorMu.RLock()
	f, ok := selectorRegistry[name]
	selectorMu.RUnlock()
	if !ok {
		return nil, core.NewError("teambuilder.new_selector", core.ErrNotFound,
			fmt.Sprintf("unknown selector %q (registered: %v)", name, ListSelectors()), nil)
	}
	return f(cfg)
}

// ListSelectors returns sorted names of all registered selectors.
func ListSelectors() []string {
	selectorMu.RLock()
	defer selectorMu.RUnlock()
	names := make([]string, 0, len(selectorRegistry))
	for name := range selectorRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
