package computeruse

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// checkURLOp identifies the guard.CheckURL operation in error codes.
const checkURLOp = "computeruse.guard.check_url"

// guardOptions holds configuration for SafetyGuard.
type guardOptions struct {
	allowedHosts        map[string]bool
	maxActionsPerMinute int
	blockPatterns       []string
}

// GuardOption configures a SafetyGuard.
type GuardOption func(*guardOptions)

// WithAllowedHosts sets the URL hosts that the browser is allowed to navigate to.
// An empty set means all hosts are allowed (not recommended for production).
// Any port suffix on a host entry (for example "example.com:8080") is
// stripped at registration so that the comparison against url.Hostname()
// — which also strips the port — is consistent.
func WithAllowedHosts(hosts ...string) GuardOption {
	return func(o *guardOptions) {
		for _, h := range hosts {
			normalized := strings.ToLower(h)
			if i := strings.LastIndex(normalized, ":"); i >= 0 {
				// Strip ":port" suffix but not IPv6 brackets.
				if !strings.Contains(normalized[:i], ":") {
					normalized = normalized[:i]
				}
			}
			o.allowedHosts[normalized] = true
		}
	}
}

// WithMaxActionsPerMinute sets the maximum number of actions per minute.
// Defaults to 60.
func WithMaxActionsPerMinute(n int) GuardOption {
	return func(o *guardOptions) {
		if n > 0 {
			o.maxActionsPerMinute = n
		}
	}
}

// WithBlockPatterns sets URL patterns that are always blocked,
// even if the host is allowed.
func WithBlockPatterns(patterns ...string) GuardOption {
	return func(o *guardOptions) {
		o.blockPatterns = append(o.blockPatterns, patterns...)
	}
}

// SafetyGuard enforces URL allowlisting and action rate limiting for
// computer use operations. It prevents unauthorized navigation and
// excessive action rates.
type SafetyGuard struct {
	opts guardOptions

	mu          sync.Mutex
	actionTimes []time.Time
}

// NewSafetyGuard creates a new SafetyGuard with the given options.
func NewSafetyGuard(opts ...GuardOption) *SafetyGuard {
	o := guardOptions{
		allowedHosts:        make(map[string]bool),
		maxActionsPerMinute: 60,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &SafetyGuard{opts: o}
}

// CheckURL validates that a URL is allowed by the guard's policy.
// Returns an error if the URL is blocked.
func (g *SafetyGuard) CheckURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return core.NewError(checkURLOp, core.ErrInvalidInput,
			fmt.Sprintf("invalid URL: %s", rawURL), err)
	}

	// Only allow http and https schemes. Schemes such as file://,
	// javascript:, data: and others have either no hostname or an
	// irrelevant one and must never bypass the allowlist.
	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http", "https":
		// allowed
	default:
		return core.NewError(checkURLOp, core.ErrInvalidInput,
			fmt.Sprintf("URL scheme %q is not allowed", scheme), nil)
	}

	// Check block patterns.
	for _, pattern := range g.opts.blockPatterns {
		if strings.Contains(rawURL, pattern) {
			return core.NewError(checkURLOp, core.ErrGuardBlocked,
				fmt.Sprintf("URL matches block pattern %q", pattern), nil)
		}
	}

	// Check allowed hosts (empty set = all allowed).
	if len(g.opts.allowedHosts) > 0 {
		host := strings.ToLower(parsed.Hostname())
		if !g.opts.allowedHosts[host] {
			return core.NewError(checkURLOp, core.ErrGuardBlocked,
				fmt.Sprintf("host %q is not in the allowed list", host), nil)
		}
	}

	return nil
}

// CheckAction validates that the action rate limit has not been exceeded.
// Returns an error if the rate limit is exceeded.
func (g *SafetyGuard) CheckAction() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// Remove expired entries.
	valid := g.actionTimes[:0]
	for _, t := range g.actionTimes {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	g.actionTimes = valid

	if len(g.actionTimes) >= g.opts.maxActionsPerMinute {
		return core.NewError("computeruse.guard.check_action", core.ErrRateLimit,
			fmt.Sprintf("rate limit exceeded: %d actions/minute", g.opts.maxActionsPerMinute), nil)
	}

	g.actionTimes = append(g.actionTimes, now)
	return nil
}

// ActionsInWindow returns the number of actions recorded in the last minute.
func (g *SafetyGuard) ActionsInWindow() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	cutoff := time.Now().Add(-time.Minute)
	count := 0
	for _, t := range g.actionTimes {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}
