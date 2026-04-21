package degradation

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// RuntimeDegrader enforces autonomy level restrictions on agent execution.
// It queries a SecurityMonitor for severity, evaluates a PolicyEvaluator,
// and applies the resulting autonomy level as constraints on the wrapped
// agent. RuntimeDegrader is safe for concurrent use.
type RuntimeDegrader struct {
	monitor   *SecurityMonitor
	policy    PolicyEvaluator
	hooks     Hooks
	allowlist map[string]bool
	logger    *slog.Logger

	mu    sync.RWMutex
	level AutonomyLevel
}

// degraderOptions holds configuration for RuntimeDegrader.
type degraderOptions struct {
	hooks     Hooks
	allowlist []string
	logger    *slog.Logger
}

// DegraderOption configures a RuntimeDegrader.
type DegraderOption func(*degraderOptions)

// WithToolAllowlist sets the tools permitted in Restricted mode.
func WithToolAllowlist(tools ...string) DegraderOption {
	return func(o *degraderOptions) {
		o.allowlist = tools
	}
}

// WithHooks sets lifecycle hooks on the RuntimeDegrader.
func WithHooks(h Hooks) DegraderOption {
	return func(o *degraderOptions) {
		o.hooks = h
	}
}

// WithLogger sets a structured logger for the RuntimeDegrader.
func WithLogger(l *slog.Logger) DegraderOption {
	return func(o *degraderOptions) {
		o.logger = l
	}
}

// NewRuntimeDegrader creates a RuntimeDegrader that uses the given monitor
// and policy to determine and enforce autonomy levels.
func NewRuntimeDegrader(monitor *SecurityMonitor, policy PolicyEvaluator, opts ...DegraderOption) *RuntimeDegrader {
	o := degraderOptions{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	allowlist := make(map[string]bool, len(o.allowlist))
	for _, name := range o.allowlist {
		allowlist[name] = true
	}

	return &RuntimeDegrader{
		monitor:   monitor,
		policy:    policy,
		hooks:     o.hooks,
		allowlist: allowlist,
		logger:    o.logger,
		level:     Full,
	}
}

// CurrentLevel returns the last evaluated autonomy level.
func (d *RuntimeDegrader) CurrentLevel() AutonomyLevel {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.level
}

// RecordEvent forwards a security event to the underlying monitor and fires
// the OnAnomalyDetected hook so SIEM and alerting integrations can observe
// the event in real time. Callers that previously talked directly to the
// monitor should prefer this wrapper so hooks are consistently delivered.
func (d *RuntimeDegrader) RecordEvent(ctx context.Context, event SecurityEvent) {
	d.monitor.RecordEvent(ctx, event)
	if d.hooks.OnAnomalyDetected != nil {
		d.hooks.OnAnomalyDetected(ctx, event)
	}
}

// Evaluate queries the monitor and policy, updates the current level, and
// fires hooks if the level changed. It returns the new level.
func (d *RuntimeDegrader) Evaluate(ctx context.Context) AutonomyLevel {
	severity := d.monitor.CurrentSeverity()
	newLevel := d.policy.Evaluate(severity)

	d.mu.Lock()
	prev := d.level
	d.level = newLevel
	d.mu.Unlock()

	if prev != newLevel {
		d.logger.InfoContext(ctx, "autonomy level changed",
			"prev", prev.String(),
			"next", newLevel.String(),
			"severity", severity,
		)
		if d.hooks.OnLevelChanged != nil {
			d.hooks.OnLevelChanged(ctx, prev, newLevel)
		}
		// Recovery: transitioning to a less restrictive level.
		if newLevel < prev && d.hooks.OnRecovery != nil {
			d.hooks.OnRecovery(ctx, prev, newLevel)
		}
	}

	return newLevel
}

// Middleware returns an agent.Middleware that enforces the current autonomy
// level on every Invoke and Stream call.
func (d *RuntimeDegrader) Middleware() agent.Middleware {
	return func(a agent.Agent) agent.Agent {
		return &degradedAgent{
			inner:    a,
			degrader: d,
		}
	}
}

// degradedAgent wraps an Agent and applies autonomy level restrictions.
type degradedAgent struct {
	inner    agent.Agent
	degrader *RuntimeDegrader
}

// Compile-time check that degradedAgent implements agent.Agent.
var _ agent.Agent = (*degradedAgent)(nil)

// ID delegates to the inner agent.
func (a *degradedAgent) ID() string { return a.inner.ID() }

// Persona delegates to the inner agent.
func (a *degradedAgent) Persona() agent.Persona { return a.inner.Persona() }

// Tools returns the tools available under the current autonomy level.
//
// agent.Agent.Tools() has no context parameter, so Evaluate is called with
// context.Background() here. This means any OnLevelChanged / OnRecovery hooks
// that fire inside Tools() receive an empty context: no tracing span, no
// tenant ID, and no cancellation. Callers that need hooks with a populated
// context should invoke RuntimeDegrader.Evaluate(ctx) directly with their
// request context before calling Tools().
func (a *degradedAgent) Tools() []tool.Tool {
	level := a.degrader.Evaluate(context.Background())
	caps := LevelCapabilities(level)

	if !caps.CanExecuteTools {
		return nil
	}

	innerTools := a.inner.Tools()
	if !caps.ToolsAllowlisted {
		return innerTools
	}

	// Filter to allowlisted tools only.
	var filtered []tool.Tool
	for _, t := range innerTools {
		if a.degrader.allowlist[t.Name()] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// Children delegates to the inner agent.
func (a *degradedAgent) Children() []agent.Agent { return a.inner.Children() }

// Invoke executes the agent under the current autonomy level constraints.
func (a *degradedAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	level := a.degrader.Evaluate(ctx)
	caps := LevelCapabilities(level)

	if !caps.CanRespond {
		a.degrader.logger.WarnContext(ctx, "agent invoke blocked by sequestered level",
			"agent", a.inner.ID(),
			"level", level.String(),
		)
		return "", core.NewError(
			"degradation.invoke",
			core.ErrGuardBlocked,
			fmt.Sprintf("agent %q is sequestered; invoke is not permitted", a.inner.ID()),
			nil,
		)
	}

	return a.inner.Invoke(ctx, input, opts...)
}

// Stream executes the agent in streaming mode under autonomy level
// constraints.
func (a *degradedAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	level := a.degrader.Evaluate(ctx)
	caps := LevelCapabilities(level)

	if !caps.CanRespond {
		return func(yield func(agent.Event, error) bool) {
			a.degrader.logger.WarnContext(ctx, "agent stream blocked by sequestered level",
				"agent", a.inner.ID(),
				"level", level.String(),
			)
			yield(agent.Event{}, core.NewError(
				"degradation.stream",
				core.ErrGuardBlocked,
				fmt.Sprintf("agent %q is sequestered; streaming is not permitted", a.inner.ID()),
				nil,
			))
		}
	}

	return a.inner.Stream(ctx, input, opts...)
}
