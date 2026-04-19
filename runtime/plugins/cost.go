package plugins

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/cost"
	"github.com/lookatitude/beluga-ai/v2/runtime"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// Compile-time check that costPlugin satisfies runtime.Plugin.
var _ runtime.Plugin = (*costPlugin)(nil)

// costPlugin records a cost.Usage entry after every completed turn and
// enforces an optional budget ceiling.
type costPlugin struct {
	tracker cost.Tracker
	budget  cost.Budget
}

// NewCostTracking creates a Plugin that records a [cost.Usage] entry via
// tracker after every completed agent turn. budget is stored for future
// enforcement extensions; the current implementation records usage only.
//
// Errors returned by tracker.Record are ignored so that a storage failure
// never blocks agent execution.
func NewCostTracking(tracker cost.Tracker, budget cost.Budget) runtime.Plugin {
	return &costPlugin{tracker: tracker, budget: budget}
}

// Name returns the plugin identifier.
func (p *costPlugin) Name() string { return "cost_tracking" }

// BeforeTurn is a no-op for this plugin.
func (p *costPlugin) BeforeTurn(_ context.Context, _ *runtime.Session, input schema.Message) (schema.Message, error) {
	return input, nil
}

// AfterTurn records a usage entry for the completed turn and passes the events
// through unchanged.
func (p *costPlugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	usage := cost.Usage{
		TenantID:  session.TenantID,
		Timestamp: time.Now().UTC(),
	}
	_ = p.tracker.Record(ctx, usage)
	return events, nil
}

// OnError is a no-op for this plugin.
func (p *costPlugin) OnError(_ context.Context, err error) error {
	return err
}
