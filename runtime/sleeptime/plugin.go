package sleeptime

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// SleeptimePlugin integrates the sleep-time compute scheduler with the
// runtime plugin system. It wakes the scheduler on BeforeTurn (user activity)
// and updates the session state on AfterTurn.
type SleeptimePlugin struct {
	scheduler *Scheduler
}

// Compile-time check that SleeptimePlugin implements runtime.Plugin.
var _ runtime.Plugin = (*SleeptimePlugin)(nil)

// NewSleeptimePlugin creates a plugin that bridges the runtime plugin
// system with the sleep-time compute scheduler.
func NewSleeptimePlugin(scheduler *Scheduler) *SleeptimePlugin {
	return &SleeptimePlugin{scheduler: scheduler}
}

// Name returns the unique identifier for this plugin.
func (p *SleeptimePlugin) Name() string {
	return "sleeptime"
}

// BeforeTurn signals user activity to the scheduler (wake) and passes the
// input message through unmodified.
func (p *SleeptimePlugin) BeforeTurn(ctx context.Context, session *runtime.Session, input schema.Message) (schema.Message, error) {
	p.scheduler.Wake(ctx)
	return input, nil
}

// AfterTurn updates the scheduler's session state from the current session
// and passes events through unmodified.
func (p *SleeptimePlugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	state := SessionState{
		SessionID:    session.ID,
		AgentID:      session.AgentID,
		TurnCount:    len(session.Turns),
		LastActivity: time.Now(),
	}
	if session.State != nil {
		state.Metadata = make(map[string]any, len(session.State))
		for k, v := range session.State {
			state.Metadata[k] = v
		}
	}
	p.scheduler.SetSessionState(state)
	return events, nil
}

// OnError passes errors through unmodified. The sleep-time plugin does not
// modify error handling.
func (p *SleeptimePlugin) OnError(_ context.Context, err error) error {
	return err
}
