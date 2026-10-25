package memory

import "context"

// Hooks provides optional callback functions for memory guard events. All
// fields are optional — nil hooks are skipped. Hooks can be composed via
// ComposeHooks.
type Hooks struct {
	// OnPoisoningDetected is called when one or more detectors flag content
	// as potentially poisoned. The results slice contains all detector
	// findings that exceeded the threshold.
	OnPoisoningDetected func(ctx context.Context, content string, results []AnomalyResult)

	// OnSignatureInvalid is called when a memory entry's HMAC signature
	// fails verification. The reason describes why (missing or mismatch).
	OnSignatureInvalid func(ctx context.Context, reason string)

	// OnCircuitTripped is called when a circuit breaker transitions from
	// closed/half-open to open for a writer-reader pair, indicating the
	// writer has been isolated.
	OnCircuitTripped func(ctx context.Context, writer, reader string)
}

// ComposeHooks merges multiple Hooks into a single Hooks value. Callbacks are
// called in the order the hooks were provided. All hooks in the chain are
// always called (no short-circuit).
func ComposeHooks(hooks ...Hooks) Hooks {
	if len(hooks) == 0 {
		return Hooks{}
	}
	if len(hooks) == 1 {
		return hooks[0]
	}

	return Hooks{
		OnPoisoningDetected: composeOnPoisoningDetected(hooks),
		OnSignatureInvalid:  composeOnSignatureInvalid(hooks),
		OnCircuitTripped:    composeOnCircuitTripped(hooks),
	}
}

// composeOnPoisoningDetected returns an OnPoisoningDetected callback that
// invokes every non-nil OnPoisoningDetected in hooks, in order.
func composeOnPoisoningDetected(hooks []Hooks) func(ctx context.Context, content string, results []AnomalyResult) {
	return func(ctx context.Context, content string, results []AnomalyResult) {
		for _, h := range hooks {
			if h.OnPoisoningDetected != nil {
				h.OnPoisoningDetected(ctx, content, results)
			}
		}
	}
}

// composeOnSignatureInvalid returns an OnSignatureInvalid callback that
// invokes every non-nil OnSignatureInvalid in hooks, in order.
func composeOnSignatureInvalid(hooks []Hooks) func(ctx context.Context, reason string) {
	return func(ctx context.Context, reason string) {
		for _, h := range hooks {
			if h.OnSignatureInvalid != nil {
				h.OnSignatureInvalid(ctx, reason)
			}
		}
	}
}

// composeOnCircuitTripped returns an OnCircuitTripped callback that invokes
// every non-nil OnCircuitTripped in hooks, in order.
func composeOnCircuitTripped(hooks []Hooks) func(ctx context.Context, writer, reader string) {
	return func(ctx context.Context, writer, reader string) {
		for _, h := range hooks {
			if h.OnCircuitTripped != nil {
				h.OnCircuitTripped(ctx, writer, reader)
			}
		}
	}
}
