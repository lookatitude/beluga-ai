package plugins

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// compile-time interface check.
var _ runtime.Plugin = (*rateLimitPlugin)(nil)

// rateLimitOptions holds configuration for the rate limit plugin.
type rateLimitOptions struct {
	requestsPerMinute int
	burstSize         int
}

// RateLimitOption is a functional option for configuring the rate limit plugin.
type RateLimitOption func(*rateLimitOptions)

// WithRequestsPerMinute sets the maximum number of requests allowed per minute.
// Default is 60.
func WithRequestsPerMinute(n int) RateLimitOption {
	return func(o *rateLimitOptions) {
		if n > 0 {
			o.requestsPerMinute = n
		}
	}
}

// WithBurstSize sets the maximum number of requests that can be processed
// simultaneously before the rate limit applies. Default equals requestsPerMinute.
func WithBurstSize(n int) RateLimitOption {
	return func(o *rateLimitOptions) {
		if n > 0 {
			o.burstSize = n
		}
	}
}

// rateLimitPlugin enforces a sliding-window token-bucket rate limit on turns.
type rateLimitPlugin struct {
	mu        sync.Mutex
	tokens    float64   // current token count
	maxTokens float64   // burst capacity
	refillPer float64   // tokens added per nanosecond
	lastRefil time.Time // time of last token refill
}

// NewRateLimit returns a runtime.Plugin that limits the number of turns
// processed per minute using a token-bucket algorithm. BeforeTurn returns a
// core.ErrRateLimit error when the bucket is empty.
func NewRateLimit(opts ...RateLimitOption) runtime.Plugin {
	o := &rateLimitOptions{
		requestsPerMinute: 60,
		burstSize:         0, // will default to requestsPerMinute below
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.burstSize <= 0 {
		o.burstSize = o.requestsPerMinute
	}

	// refill rate: tokens per nanosecond
	refillPerNS := float64(o.requestsPerMinute) / float64(time.Minute)

	return &rateLimitPlugin{
		tokens:    float64(o.burstSize),
		maxTokens: float64(o.burstSize),
		refillPer: refillPerNS,
		lastRefil: time.Now(),
	}
}

// Name returns the plugin identifier.
func (r *rateLimitPlugin) Name() string { return "rate_limit" }

// BeforeTurn checks whether a token is available in the bucket. If so, it
// consumes one token and allows the turn. Otherwise it returns a retryable
// core.Error with code ErrRateLimit.
func (r *rateLimitPlugin) BeforeTurn(_ context.Context, _ *runtime.Session, input schema.Message) (schema.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefil)
	r.lastRefil = now

	// Refill tokens based on elapsed time.
	r.tokens += elapsed.Seconds() * float64(time.Second) * r.refillPer
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}

	if r.tokens < 1.0 {
		return input, core.NewError(
			"ratelimit.before_turn",
			core.ErrRateLimit,
			"rate limit exceeded: too many requests per minute",
			nil,
		)
	}

	r.tokens--
	return input, nil
}

// AfterTurn is a no-op; it passes events through unchanged.
func (r *rateLimitPlugin) AfterTurn(_ context.Context, _ *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	return events, nil
}

// OnError passes the error through unchanged.
func (r *rateLimitPlugin) OnError(_ context.Context, err error) error { return err }
