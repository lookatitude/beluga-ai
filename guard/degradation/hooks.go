package degradation

import "context"

// Hooks provides optional callbacks for degradation lifecycle events. All
// fields are optional; nil fields are skipped during invocation.
type Hooks struct {
	// OnLevelChanged is called when the autonomy level transitions to a
	// different value. It receives the previous and new levels.
	OnLevelChanged func(ctx context.Context, prev, next AutonomyLevel)

	// OnAnomalyDetected is called when a new security event is recorded by
	// the monitor.
	OnAnomalyDetected func(ctx context.Context, event SecurityEvent)

	// OnRecovery is called when the autonomy level transitions from a more
	// restrictive level back toward Full.
	OnRecovery func(ctx context.Context, prev, next AutonomyLevel)
}

// ComposeHooks merges multiple Hooks into a single Hooks where each
// callback invokes all non-nil callbacks from the input hooks in order.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnLevelChanged:    composeOnLevelChanged(hooks),
		OnAnomalyDetected: composeOnAnomalyDetected(hooks),
		OnRecovery:        composeOnRecovery(hooks),
	}
}

func composeOnLevelChanged(hooks []Hooks) func(context.Context, AutonomyLevel, AutonomyLevel) {
	var fns []func(context.Context, AutonomyLevel, AutonomyLevel)
	for _, h := range hooks {
		if h.OnLevelChanged != nil {
			fns = append(fns, h.OnLevelChanged)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, prev, next AutonomyLevel) {
		for _, fn := range fns {
			fn(ctx, prev, next)
		}
	}
}

func composeOnAnomalyDetected(hooks []Hooks) func(context.Context, SecurityEvent) {
	var fns []func(context.Context, SecurityEvent)
	for _, h := range hooks {
		if h.OnAnomalyDetected != nil {
			fns = append(fns, h.OnAnomalyDetected)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, event SecurityEvent) {
		for _, fn := range fns {
			fn(ctx, event)
		}
	}
}

func composeOnRecovery(hooks []Hooks) func(context.Context, AutonomyLevel, AutonomyLevel) {
	var fns []func(context.Context, AutonomyLevel, AutonomyLevel)
	for _, h := range hooks {
		if h.OnRecovery != nil {
			fns = append(fns, h.OnRecovery)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, prev, next AutonomyLevel) {
		for _, fn := range fns {
			fn(ctx, prev, next)
		}
	}
}
