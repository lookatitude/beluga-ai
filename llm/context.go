package llm

import (
	"context"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// ContextManager fits a sequence of messages within a token budget.
// Different strategies (truncation, sliding window, etc.) implement this
// interface to manage context window limits.
type ContextManager interface {
	// Fit trims or transforms msgs so the total token count does not exceed budget.
	// The returned messages preserve conversation coherence as much as possible.
	Fit(ctx context.Context, msgs []schema.Message, budget int) ([]schema.Message, error)
}

// ContextOption configures a ContextManager created by NewContextManager.
type ContextOption func(*contextConfig)

type contextConfig struct {
	strategy   string
	tokenizer  Tokenizer
	keepSystem bool
}

// WithContextStrategy sets the strategy name: "truncate" or "sliding".
// Defaults to "truncate".
func WithContextStrategy(name string) ContextOption {
	return func(cfg *contextConfig) {
		cfg.strategy = name
	}
}

// WithTokenizer sets the tokenizer used for counting tokens. Defaults to
// SimpleTokenizer if unset.
func WithTokenizer(t Tokenizer) ContextOption {
	return func(cfg *contextConfig) {
		cfg.tokenizer = t
	}
}

// WithKeepSystemMessages ensures system messages are never removed by
// truncation. Defaults to true.
func WithKeepSystemMessages(keep bool) ContextOption {
	return func(cfg *contextConfig) {
		cfg.keepSystem = keep
	}
}

// NewContextManager creates a ContextManager with the given options.
func NewContextManager(opts ...ContextOption) ContextManager {
	cfg := &contextConfig{
		strategy:   "truncate",
		tokenizer:  &SimpleTokenizer{},
		keepSystem: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	switch cfg.strategy {
	case "sliding":
		return &SlidingStrategy{
			tokenizer:  cfg.tokenizer,
			keepSystem: cfg.keepSystem,
		}
	default:
		return &TruncateStrategy{
			tokenizer:  cfg.tokenizer,
			keepSystem: cfg.keepSystem,
		}
	}
}

// TruncateStrategy drops the oldest non-system messages until the total
// token count fits within the budget.
type TruncateStrategy struct {
	tokenizer  Tokenizer
	keepSystem bool
}

// Fit removes the oldest non-system messages until the token count is within budget.
func (s *TruncateStrategy) Fit(_ context.Context, msgs []schema.Message, budget int) ([]schema.Message, error) {
	if budget <= 0 {
		return nil, core.NewError("llm.context", core.ErrInvalidInput, "budget must be positive", nil)
	}

	total := s.tokenizer.CountMessages(msgs)
	if total <= budget {
		return msgs, nil
	}

	// Separate system messages from the rest.
	var system []schema.Message
	var rest []schema.Message
	for _, m := range msgs {
		if s.keepSystem && m.GetRole() == schema.RoleSystem {
			system = append(system, m)
		} else {
			rest = append(rest, m)
		}
	}

	// Calculate token budget remaining after system messages.
	systemTokens := s.tokenizer.CountMessages(system)
	remaining := budget - systemTokens
	if remaining <= 0 {
		// Even system messages exceed budget; return system only.
		return system, nil
	}

	// Drop oldest non-system messages from the front.
	for len(rest) > 0 {
		tokens := s.tokenizer.CountMessages(rest)
		if tokens <= remaining {
			break
		}
		rest = rest[1:]
	}

	result := make([]schema.Message, 0, len(system)+len(rest))
	result = append(result, system...)
	result = append(result, rest...)
	return result, nil
}

// SlidingStrategy keeps the most recent N messages that fit within the budget,
// always preserving system messages.
type SlidingStrategy struct {
	tokenizer  Tokenizer
	keepSystem bool
}

// Fit keeps the last messages that fit within the budget.
func (s *SlidingStrategy) Fit(_ context.Context, msgs []schema.Message, budget int) ([]schema.Message, error) {
	if budget <= 0 {
		return nil, core.NewError("llm.context", core.ErrInvalidInput, "budget must be positive", nil)
	}

	total := s.tokenizer.CountMessages(msgs)
	if total <= budget {
		return msgs, nil
	}

	// Separate system messages.
	var system []schema.Message
	var rest []schema.Message
	for _, m := range msgs {
		if s.keepSystem && m.GetRole() == schema.RoleSystem {
			system = append(system, m)
		} else {
			rest = append(rest, m)
		}
	}

	systemTokens := s.tokenizer.CountMessages(system)
	remaining := budget - systemTokens
	if remaining <= 0 {
		return system, nil
	}

	// Build from the end: add messages from the back until budget is exceeded.
	var window []schema.Message
	used := 0
	for i := len(rest) - 1; i >= 0; i-- {
		msgTokens := s.tokenizer.CountMessages([]schema.Message{rest[i]})
		if used+msgTokens > remaining {
			break
		}
		used += msgTokens
		window = append([]schema.Message{rest[i]}, window...)
	}

	result := make([]schema.Message, 0, len(system)+len(window))
	result = append(result, system...)
	result = append(result, window...)
	return result, nil
}
