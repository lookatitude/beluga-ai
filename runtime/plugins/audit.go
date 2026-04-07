package plugins

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/audit"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compile-time check that auditPlugin satisfies runtime.Plugin.
var _ runtime.Plugin = (*auditPlugin)(nil)

// auditPlugin records structured audit entries at the start and end of every
// agent turn, as well as on error.
type auditPlugin struct {
	store audit.Store
}

// NewAuditPlugin creates a Plugin that logs audit entries to store for every
// turn lifecycle event:
//   - BeforeTurn logs "agent.turn.start"
//   - AfterTurn logs "agent.turn.end"
//   - OnError logs "agent.turn.error"
//
// Errors emitted by the store are currently ignored so that a logging failure
// never blocks agent execution.
func NewAuditPlugin(store audit.Store) runtime.Plugin {
	return &auditPlugin{store: store}
}

// Name returns the plugin identifier.
func (p *auditPlugin) Name() string { return "audit" }

// BeforeTurn logs an "agent.turn.start" entry and passes the input message
// through unchanged.
func (p *auditPlugin) BeforeTurn(ctx context.Context, session *runtime.Session, input schema.Message) (schema.Message, error) {
	_ = p.store.Log(ctx, audit.Entry{
		Timestamp: time.Now().UTC(),
		TenantID:  session.TenantID,
		AgentID:   session.AgentID,
		SessionID: session.ID,
		Action:    "agent.turn.start",
	})
	return input, nil
}

// AfterTurn logs an "agent.turn.end" entry and passes the events through
// unchanged.
func (p *auditPlugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	_ = p.store.Log(ctx, audit.Entry{
		Timestamp: time.Now().UTC(),
		TenantID:  session.TenantID,
		AgentID:   session.AgentID,
		SessionID: session.ID,
		Action:    "agent.turn.end",
	})
	return events, nil
}

// OnError logs an "agent.turn.error" entry and returns the original error
// unmodified.
func (p *auditPlugin) OnError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	entry := audit.Entry{
		Timestamp: time.Now().UTC(),
		Action:    "agent.turn.error",
		Error:     err.Error(),
	}
	_ = p.store.Log(ctx, entry)
	return err
}
