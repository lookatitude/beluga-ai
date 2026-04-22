package metrics

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/eval"
)

// ExactMatch scores whether [eval.EvalSample.Output] equals
// [eval.EvalSample.ExpectedOutput]. It is the only correctness metric that
// runs with zero credentials and zero API calls — the canonical default for
// the scaffolded-project smoke eval (DX-1 S4, specialist-ai-ml-expert §Q2).
//
// By default the comparison is case-insensitive with leading/trailing
// whitespace trimmed; pass [WithCaseSensitive] for strict equality.
type ExactMatch struct {
	caseSensitive bool
}

// ExactMatchOption configures an [ExactMatch] metric.
type ExactMatchOption func(*ExactMatch)

// WithCaseSensitive flips the comparison to strict byte equality, preserving
// case and whitespace. Useful when evaluating code snippets or JSON output
// where case matters.
func WithCaseSensitive() ExactMatchOption {
	return func(e *ExactMatch) { e.caseSensitive = true }
}

// NewExactMatch creates a new ExactMatch metric with the given options.
func NewExactMatch(opts ...ExactMatchOption) *ExactMatch {
	e := &ExactMatch{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Name returns "exact_match".
func (e *ExactMatch) Name() string { return "exact_match" }

// Score returns 1.0 when Output matches ExpectedOutput and 0.0 otherwise.
// An empty ExpectedOutput is treated as "no ground truth available" and
// yields 0.0 — callers wanting a different policy for unset ground truth
// should filter those rows before scoring.
func (e *ExactMatch) Score(_ context.Context, sample eval.EvalSample) (float64, error) {
	if sample.ExpectedOutput == "" {
		return 0.0, nil
	}
	got, want := sample.Output, sample.ExpectedOutput
	if !e.caseSensitive {
		got = strings.ToLower(strings.TrimSpace(got))
		want = strings.ToLower(strings.TrimSpace(want))
	}
	if got == want {
		return 1.0, nil
	}
	return 0.0, nil
}
