package metrics

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/eval"
)

// DefaultMaxLatencyMs is the default maximum latency in milliseconds used for
// normalization. Latencies above this value result in a score of 0.0.
const DefaultMaxLatencyMs = 10000.0

// Latency evaluates response latency by reading Metadata["latency_ms"] from
// the sample. It returns a normalized score in [0.0, 1.0] where 1.0 means
// instantaneous and 0.0 means at or above the maximum threshold.
type Latency struct {
	maxLatencyMs float64
}

// LatencyOption configures a Latency metric.
type LatencyOption func(*Latency)

// WithMaxLatencyMs sets the maximum latency threshold for normalization.
func WithMaxLatencyMs(ms float64) LatencyOption {
	return func(l *Latency) {
		if ms > 0 {
			l.maxLatencyMs = ms
		}
	}
}

// NewLatency creates a new Latency metric with the given options.
func NewLatency(opts ...LatencyOption) *Latency {
	l := &Latency{
		maxLatencyMs: DefaultMaxLatencyMs,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Name returns "latency".
func (l *Latency) Name() string { return "latency" }

// Score reads latency_ms from sample metadata and returns a normalized score.
// Returns 1.0 for 0ms latency and 0.0 for latency >= maxLatencyMs.
func (l *Latency) Score(_ context.Context, sample eval.EvalSample) (float64, error) {
	raw, ok := sample.Metadata["latency_ms"]
	if !ok {
		return 0, fmt.Errorf("latency: missing metadata key %q", "latency_ms")
	}

	ms, err := toFloat64(raw)
	if err != nil {
		return 0, fmt.Errorf("latency: %w", err)
	}

	if ms <= 0 {
		return 1.0, nil
	}
	score := 1.0 - ms/l.maxLatencyMs
	if score < 0 {
		score = 0
	}
	return score, nil
}

// toFloat64 converts numeric interface values to float64.
func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T for value %v", v, v)
	}
}
