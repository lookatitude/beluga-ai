package agentic

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/guard"
)

// Compile-time check.
var _ guard.Guard = (*CascadeGuard)(nil)

// Default limits for cascade protection.
const (
	defaultMaxRecursionDepth = 10
	defaultMaxIterations     = 100
	defaultMaxTokenBudget    = 1_000_000
)

// CascadeGuard prevents cascading failures in multi-agent chains by tracking
// recursive call depth, iteration count, and cumulative token budget. When
// any limit is exceeded, further execution is blocked. It addresses OWASP
// AG08 (Cascading Failure) and AG10 (Excessive Autonomy).
type CascadeGuard struct {
	maxDepth      int
	maxIterations int
	maxTokens     int64

	mu         sync.Mutex
	depths     map[string]int   // chain_id -> current depth
	iterations map[string]int   // chain_id -> iteration count
	tokens     map[string]int64 // chain_id -> accumulated tokens

	// circuitOpen tracks whether the circuit breaker has tripped for a chain.
	circuitOpen map[string]bool
	// failureThreshold is the number of consecutive limit violations before
	// the circuit opens permanently (until reset).
	failureThreshold int
	failures         map[string]int
}

// CascadeOption configures a CascadeGuard.
type CascadeOption func(*CascadeGuard)

// WithMaxRecursionDepth sets the maximum allowed recursive call depth per
// chain.
func WithMaxRecursionDepth(depth int) CascadeOption {
	return func(g *CascadeGuard) {
		if depth > 0 {
			g.maxDepth = depth
		}
	}
}

// WithMaxIterations sets the maximum number of iterations allowed per chain.
func WithMaxIterations(n int) CascadeOption {
	return func(g *CascadeGuard) {
		if n > 0 {
			g.maxIterations = n
		}
	}
}

// WithMaxTokenBudget sets the cumulative token budget across an agent chain.
func WithMaxTokenBudget(tokens int64) CascadeOption {
	return func(g *CascadeGuard) {
		if tokens > 0 {
			g.maxTokens = tokens
		}
	}
}

// WithFailureThreshold sets the number of consecutive limit violations
// before the circuit breaker opens for a chain. Default is 3.
func WithFailureThreshold(n int) CascadeOption {
	return func(g *CascadeGuard) {
		if n > 0 {
			g.failureThreshold = n
		}
	}
}

// NewCascadeGuard creates a CascadeGuard with the given options.
func NewCascadeGuard(opts ...CascadeOption) *CascadeGuard {
	g := &CascadeGuard{
		maxDepth:         defaultMaxRecursionDepth,
		maxIterations:    defaultMaxIterations,
		maxTokens:        defaultMaxTokenBudget,
		failureThreshold: 3,
		depths:           make(map[string]int),
		iterations:       make(map[string]int),
		tokens:           make(map[string]int64),
		circuitOpen:      make(map[string]bool),
		failures:         make(map[string]int),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Name returns "cascade_guard".
func (g *CascadeGuard) Name() string {
	return "cascade_guard"
}

// Validate checks cascade limits for the chain identified in input.Metadata.
// Expected metadata keys:
//
//   - "chain_id": string    -- unique identifier for the agent chain
//   - "depth": int          -- current recursion depth
//   - "iteration": int      -- current iteration count
//   - "tokens_used": int64  -- tokens consumed so far in this chain
func (g *CascadeGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	select {
	case <-ctx.Done():
		return guard.GuardResult{}, ctx.Err()
	default:
	}

	chainID, _ := input.Metadata["chain_id"].(string)
	if chainID == "" {
		// No chain context -- nothing to validate.
		return guard.GuardResult{Allowed: true}, nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Circuit breaker check.
	if g.circuitOpen[chainID] {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("circuit breaker open for chain %q due to repeated limit violations", chainID),
			GuardName: g.Name(),
		}, nil
	}

	// Extract and track depth.
	depth := toInt(input.Metadata["depth"])
	if depth > 0 {
		g.depths[chainID] = depth
	}
	currentDepth := g.depths[chainID]
	if currentDepth > g.maxDepth {
		g.recordFailure(chainID)
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("recursion depth %d exceeds maximum of %d for chain %q", currentDepth, g.maxDepth, chainID),
			GuardName: g.Name(),
		}, nil
	}

	// Track iterations.
	iteration := toInt(input.Metadata["iteration"])
	if iteration > 0 {
		g.iterations[chainID] = iteration
	} else {
		g.iterations[chainID]++
	}
	currentIter := g.iterations[chainID]
	if currentIter > g.maxIterations {
		g.recordFailure(chainID)
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("iteration count %d exceeds maximum of %d for chain %q", currentIter, g.maxIterations, chainID),
			GuardName: g.Name(),
		}, nil
	}

	// Track token budget.
	tokensUsed := toInt64(input.Metadata["tokens_used"])
	if tokensUsed > 0 {
		g.tokens[chainID] = tokensUsed
	}
	currentTokens := g.tokens[chainID]
	if currentTokens > g.maxTokens {
		g.recordFailure(chainID)
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("token usage %d exceeds budget of %d for chain %q", currentTokens, g.maxTokens, chainID),
			GuardName: g.Name(),
		}, nil
	}

	// Success -- reset failure counter.
	g.failures[chainID] = 0
	return guard.GuardResult{Allowed: true}, nil
}

// ResetChain clears all tracked state for the given chain, including the
// circuit breaker.
func (g *CascadeGuard) ResetChain(chainID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.depths, chainID)
	delete(g.iterations, chainID)
	delete(g.tokens, chainID)
	delete(g.circuitOpen, chainID)
	delete(g.failures, chainID)
}

// recordFailure increments the failure counter and trips the circuit breaker
// when the threshold is reached. Caller must hold g.mu.
func (g *CascadeGuard) recordFailure(chainID string) {
	g.failures[chainID]++
	if g.failures[chainID] >= g.failureThreshold {
		g.circuitOpen[chainID] = true
	}
}

// toInt converts an any value to int, handling common numeric types from JSON
// unmarshalling.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// toInt64 converts an any value to int64.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func init() {
	guard.Register("cascade_guard", func(cfg map[string]any) (guard.Guard, error) {
		return NewCascadeGuard(), nil
	})
}
