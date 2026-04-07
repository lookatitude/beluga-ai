package plugins

import (
	"context"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/cost"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// compile-time interface check.
var _ runtime.Plugin = (*costTracking)(nil)

// costOptions holds configuration for the costTracking plugin.
type costOptions struct {
	tracker cost.Tracker
}

// CostOption is a functional option for configuring the costTracking plugin.
type CostOption func(*costOptions)

// WithTracker sets a custom cost.Tracker on the plugin. If not provided, the
// plugin uses the tracker supplied to NewCostTracking.
func WithTracker(t cost.Tracker) CostOption {
	return func(o *costOptions) {
		if t != nil {
			o.tracker = t
		}
	}
}

// costTracking records token usage extracted from agent events after each turn.
type costTracking struct {
	tracker cost.Tracker
}

// NewCostTracking returns a runtime.Plugin that records token usage from agent
// events after each turn. The budget is enforced by the tracker; use
// cost.NewInMemoryTracker with a non-zero cost.Budget to limit consumption.
// Additional options (e.g. WithTracker) override the default tracker.
func NewCostTracking(budget cost.Budget, opts ...CostOption) runtime.Plugin {
	o := &costOptions{
		tracker: cost.NewInMemoryTracker(budget),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &costTracking{tracker: o.tracker}
}

// Name returns the plugin identifier.
func (c *costTracking) Name() string { return "cost_tracking" }

// BeforeTurn is a no-op; it passes the message through unchanged.
func (c *costTracking) BeforeTurn(_ context.Context, _ *runtime.Session, input schema.Message) (schema.Message, error) {
	return input, nil
}

// AfterTurn extracts token usage from each event's Metadata and records it
// via the tracker. Usage is stored under the "usage" key as a cost.Usage or
// schema.Usage value. If both are absent the event is skipped.
func (c *costTracking) AfterTurn(ctx context.Context, _ *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	for _, ev := range events {
		if ev.Metadata == nil {
			continue
		}
		u, ok := extractUsage(ev.Metadata)
		if !ok {
			continue
		}
		if err := c.tracker.Record(ctx, u); err != nil {
			// Budget exhausted or tracker error — propagate to abort the turn.
			return events, err
		}
	}
	return events, nil
}

// OnError passes the error through unchanged.
func (c *costTracking) OnError(_ context.Context, err error) error { return err }

// extractUsage attempts to read token usage from metadata. It supports two
// value shapes: cost.Usage (native) and schema.Usage (from LLM providers).
func extractUsage(meta map[string]any) (cost.Usage, bool) {
	v, ok := meta["usage"]
	if !ok {
		return cost.Usage{}, false
	}
	switch u := v.(type) {
	case cost.Usage:
		return u, true
	case schema.Usage:
		return cost.Usage{
			InputTokens:  u.InputTokens,
			OutputTokens: u.OutputTokens,
			TotalTokens:  u.TotalTokens,
			CachedTokens: u.CachedTokens,
		}, true
	case *schema.Usage:
		if u == nil {
			return cost.Usage{}, false
		}
		return cost.Usage{
			InputTokens:  u.InputTokens,
			OutputTokens: u.OutputTokens,
			TotalTokens:  u.TotalTokens,
			CachedTokens: u.CachedTokens,
		}, true
	}
	return cost.Usage{}, false
}
