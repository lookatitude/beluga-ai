package speculative

import (
	"context"
	"time"
)

// Result holds the outcome of a speculative execution attempt.
type Result struct {
	// Prediction is the text produced by the predictor.
	Prediction string

	// GroundTruth is the text produced by the ground-truth agent.
	GroundTruth string

	// Validated indicates whether the prediction matched the ground truth.
	Validated bool

	// Confidence is the predictor's confidence score for the prediction.
	Confidence float64

	// Speedup is the time saved by using the prediction (negative if wasted).
	Speedup time.Duration

	// WastedTokens is the number of tokens spent on a failed prediction.
	WastedTokens int

	// Output is the final output text (prediction if validated, ground truth otherwise).
	Output string
}

// Hooks provides optional callback functions invoked at various points
// during speculative execution. All fields are optional; nil hooks are skipped.
type Hooks struct {
	// OnPrediction is called when the predictor produces a result.
	OnPrediction func(ctx context.Context, prediction string, confidence float64)

	// OnValidation is called when validation succeeds (prediction matches ground truth).
	OnValidation func(ctx context.Context, result Result)

	// OnMisprediction is called when validation fails (prediction does not match).
	OnMisprediction func(ctx context.Context, result Result)

	// OnCancel is called when the speculative execution is cancelled.
	OnCancel func(ctx context.Context, err error)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnPrediction:    composeOnPrediction(hooks),
		OnValidation:    composeResultHook(hooks, func(h Hooks) func(context.Context, Result) { return h.OnValidation }),
		OnMisprediction: composeResultHook(hooks, func(h Hooks) func(context.Context, Result) { return h.OnMisprediction }),
		OnCancel:        composeOnCancel(hooks),
	}
}

// composeOnPrediction returns an OnPrediction callback that invokes every
// non-nil OnPrediction in hooks, in order.
func composeOnPrediction(hooks []Hooks) func(ctx context.Context, prediction string, confidence float64) {
	return func(ctx context.Context, prediction string, confidence float64) {
		for _, h := range hooks {
			if h.OnPrediction != nil {
				h.OnPrediction(ctx, prediction, confidence)
			}
		}
	}
}

// composeResultHook returns a callback that invokes the Result-typed hook
// selected by pick from every element of hooks, in order, skipping nil ones.
func composeResultHook(hooks []Hooks, pick func(Hooks) func(context.Context, Result)) func(ctx context.Context, result Result) {
	return func(ctx context.Context, result Result) {
		for _, h := range hooks {
			if fn := pick(h); fn != nil {
				fn(ctx, result)
			}
		}
	}
}

// composeOnCancel returns an OnCancel callback that invokes every non-nil
// OnCancel in hooks, in order.
func composeOnCancel(hooks []Hooks) func(ctx context.Context, err error) {
	return func(ctx context.Context, err error) {
		for _, h := range hooks {
			if h.OnCancel != nil {
				h.OnCancel(ctx, err)
			}
		}
	}
}
