package plugins

import (
	"context"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/audit"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// compile-time interface check.
var _ runtime.Plugin = (*auditPlugin)(nil)

// auditPlugin logs turn lifecycle events to an audit.Logger.
type auditPlugin struct {
	logger *audit.Logger
}

// NewAuditPlugin returns a runtime.Plugin that records each turn phase
// (start, end, error) as an audit entry via the given store.
func NewAuditPlugin(store audit.Store) runtime.Plugin {
	return &auditPlugin{logger: audit.NewLogger(store)}
}

// Name returns the plugin identifier.
func (a *auditPlugin) Name() string { return "audit" }

// BeforeTurn logs an "agent.turn.start" audit entry.
func (a *auditPlugin) BeforeTurn(ctx context.Context, session *runtime.Session, input schema.Message) (schema.Message, error) {
	meta := map[string]any{
		"turn_count": session.TurnCount,
		"role":       string(input.GetRole()),
	}
	if err := a.logger.TurnStart(ctx, session.AgentID, session.SessionID, meta); err != nil {
		// Audit failures are logged but must not abort the turn.
		_ = err
	}
	return input, nil
}

// AfterTurn logs an "agent.turn.end" audit entry.
func (a *auditPlugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	meta := map[string]any{
		"turn_count":  session.TurnCount,
		"event_count": len(events),
	}
	if err := a.logger.TurnEnd(ctx, session.AgentID, session.SessionID, meta); err != nil {
		_ = err
	}
	return events, nil
}

// OnError logs an "agent.turn.error" audit entry and returns the original error.
func (a *auditPlugin) OnError(ctx context.Context, err error) error {
	// Use a background-safe context in case the original was cancelled.
	// We still want the audit record even if the turn context was cancelled.
	logCtx := ctx
	if logErr := a.logger.TurnError(logCtx, "", "", err); logErr != nil {
		_ = logErr
	}
	return err
}
