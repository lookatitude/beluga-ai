package plugins

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/runtime"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"golang.org/x/time/rate"
)

// Compile-time check that rateLimitPlugin satisfies runtime.Plugin.
var _ runtime.Plugin = (*rateLimitPlugin)(nil)

// rateLimitPlugin enforces a per-minute request ceiling using a token bucket.
type rateLimitPlugin struct {
	limiter *rate.Limiter
}

// NewRateLimit creates a Plugin that rejects turns when the request rate
// exceeds requestsPerMinute. It uses a token bucket that refills at
// requestsPerMinute tokens per minute with a burst equal to requestsPerMinute,
// allowing brief bursts up to the full per-minute allowance.
//
// BeforeTurn returns a [core.ErrRateLimit] error when the bucket is empty.
func NewRateLimit(requestsPerMinute int) runtime.Plugin {
	// Convert requests per minute to a per-second rate.
	r := rate.Limit(float64(requestsPerMinute) / 60.0)
	// Burst is equal to the per-minute allowance so short bursts are permitted.
	burst := requestsPerMinute
	if burst < 1 {
		burst = 1
	}
	return &rateLimitPlugin{limiter: rate.NewLimiter(r, burst)}
}

// Name returns the plugin identifier.
func (p *rateLimitPlugin) Name() string { return "rate_limit" }

// BeforeTurn consumes one token from the bucket. It returns a retryable
// [core.Error] with code [core.ErrRateLimit] if the bucket is exhausted.
func (p *rateLimitPlugin) BeforeTurn(_ context.Context, _ *runtime.Session, input schema.Message) (schema.Message, error) {
	if !p.limiter.Allow() {
		return input, core.NewError(
			"runtime.plugins.ratelimit",
			core.ErrRateLimit,
			"request rate limit exceeded",
			nil,
		)
	}
	return input, nil
}

// AfterTurn is a no-op for this plugin.
func (p *rateLimitPlugin) AfterTurn(_ context.Context, _ *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	return events, nil
}

// OnError is a no-op for this plugin.
func (p *rateLimitPlugin) OnError(_ context.Context, err error) error {
	return err
}
