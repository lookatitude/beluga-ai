package cognitive

import (
	"context"
	"time"
)

// ComplexityLevel classifies how complex an input query is, determining
// which cognitive tier should handle it.
type ComplexityLevel int

const (
	// Simple indicates a straightforward query that System 1 can handle
	// with heuristic, pattern-matched responses.
	Simple ComplexityLevel = iota

	// Moderate indicates a query that may need some reasoning but could
	// potentially be handled by System 1 with high confidence.
	Moderate

	// Complex indicates a query requiring multi-step reasoning, analysis,
	// or deliberation that should be routed to System 2.
	Complex
)

// String returns a human-readable name for the complexity level.
func (l ComplexityLevel) String() string {
	switch l {
	case Simple:
		return "simple"
	case Moderate:
		return "moderate"
	case Complex:
		return "complex"
	default:
		return "unknown"
	}
}

// ComplexityScore is the result of scoring input complexity. It carries
// the classified level, a confidence value between 0.0 and 1.0, and an
// optional human-readable reason.
type ComplexityScore struct {
	// Level is the classified complexity.
	Level ComplexityLevel

	// Confidence is a value between 0.0 and 1.0 indicating how certain
	// the scorer is about the classification.
	Confidence float64

	// Reason is an optional human-readable explanation for the classification.
	Reason string
}

// ComplexityScorer classifies the complexity of an input query to determine
// routing between System 1 and System 2 agents.
type ComplexityScorer interface {
	// Score evaluates the input and returns a complexity classification.
	Score(ctx context.Context, input string) (ComplexityScore, error)
}

// Hooks provides optional callback functions invoked at various points
// during dual-process agent execution. All fields are optional; nil hooks
// are skipped.
type Hooks struct {
	// OnRouted is called after routing a request to either S1 or S2.
	OnRouted func(ctx context.Context, input string, level ComplexityLevel, target string)

	// OnEscalated is called when a cascading Invoke escalates from S1 to S2
	// because the S1 output did not meet the confidence threshold.
	OnEscalated func(ctx context.Context, input string, s1Output string, reason string)

	// OnCompleted is called when execution finishes, reporting which tier
	// handled the request and how long it took.
	OnCompleted func(ctx context.Context, tier string, latency time.Duration)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are invoked in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnRouted: func(ctx context.Context, input string, level ComplexityLevel, target string) {
			for _, h := range hooks {
				if h.OnRouted != nil {
					h.OnRouted(ctx, input, level, target)
				}
			}
		},
		OnEscalated: func(ctx context.Context, input string, s1Output string, reason string) {
			for _, h := range hooks {
				if h.OnEscalated != nil {
					h.OnEscalated(ctx, input, s1Output, reason)
				}
			}
		},
		OnCompleted: func(ctx context.Context, tier string, latency time.Duration) {
			for _, h := range hooks {
				if h.OnCompleted != nil {
					h.OnCompleted(ctx, tier, latency)
				}
			}
		},
	}
}
