package evolving

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// MetaOptimizer analyzes agent performance and suggests improvements.
type MetaOptimizer interface {
	// Optimize analyzes experiences and returns optimization suggestions.
	Optimize(ctx context.Context, experiences []Experience) ([]Suggestion, error)
}

// ExperienceDistiller extracts reusable knowledge from raw interactions.
type ExperienceDistiller interface {
	// Distill converts raw interactions into structured experiences.
	Distill(ctx context.Context, interactions []Interaction) ([]Experience, error)
}

// Interaction represents a single agent interaction for learning.
type Interaction struct {
	// Input is what the agent received.
	Input string
	// Output is what the agent produced.
	Output string
	// Duration is how long the interaction took.
	Duration time.Duration
	// Success indicates whether the interaction was successful.
	Success bool
	// Feedback is optional human or automated feedback.
	Feedback string
	// Metadata holds extra attributes.
	Metadata map[string]any
	// Timestamp is when this interaction occurred.
	Timestamp time.Time
}

// Experience represents distilled knowledge from interactions.
type Experience struct {
	// Category classifies the experience (e.g., "error_handling", "tool_usage").
	Category string
	// Pattern describes the learned pattern.
	Pattern string
	// Confidence is the confidence level in [0.0, 1.0].
	Confidence float64
	// Examples are representative interaction IDs.
	Examples []string
	// LearnedAt is when this experience was distilled.
	LearnedAt time.Time
}

// Suggestion is an optimization recommendation from the MetaOptimizer.
type Suggestion struct {
	// Type classifies the suggestion (e.g., "prompt_update", "tool_addition").
	Type string
	// Description explains the suggestion.
	Description string
	// Priority indicates importance (higher = more important).
	Priority int
	// Confidence is confidence in the suggestion [0.0, 1.0].
	Confidence float64
}

// Option configures an EvolvingAgent.
type Option func(*evolvingOptions)

type evolvingOptions struct {
	optimizer   MetaOptimizer
	distiller   ExperienceDistiller
	logger      *slog.Logger
	maxMemory   int
	learnEveryN int
}

// WithOptimizer sets the meta-optimizer.
func WithOptimizer(o MetaOptimizer) Option {
	return func(opts *evolvingOptions) { opts.optimizer = o }
}

// WithDistiller sets the experience distiller.
func WithDistiller(d ExperienceDistiller) Option {
	return func(opts *evolvingOptions) { opts.distiller = d }
}

// WithMaxMemory sets the maximum number of interactions to retain.
//
// n must be greater than 0. Values <= 0 are ignored, and the default
// (1000) is preserved.
func WithMaxMemory(n int) Option {
	return func(opts *evolvingOptions) {
		if n > 0 {
			opts.maxMemory = n
		}
	}
}

// WithLearnEveryN triggers learning every N interactions.
//
// n must be greater than 0. Values <= 0 are ignored, and the default
// (10) is preserved.
func WithLearnEveryN(n int) Option {
	return func(opts *evolvingOptions) {
		if n > 0 {
			opts.learnEveryN = n
		}
	}
}

// WithLogger sets the logger used to report background learning errors.
// If nil or unset, slog.Default() is used.
func WithLogger(l *slog.Logger) Option {
	return func(opts *evolvingOptions) { opts.logger = l }
}

// EvolvingAgent wraps a BaseAgent and adds self-improvement capabilities.
type EvolvingAgent struct {
	inner        agent.Agent
	opts         evolvingOptions
	mu           sync.Mutex
	interactions []Interaction
	experiences  []Experience
	suggestions  []Suggestion
	counter      int
}

var _ agent.Agent = (*EvolvingAgent)(nil)

// New creates an EvolvingAgent wrapping the given agent.
//
// Defaults: learnEveryN=10, maxMemory=1000. Invalid option values (<=0)
// are ignored and replaced with defaults, so the agent is always safe
// to use after construction.
func New(inner agent.Agent, opts ...Option) *EvolvingAgent {
	o := evolvingOptions{
		optimizer:   &FrequencyOptimizer{},
		distiller:   &SimpleDistiller{},
		maxMemory:   1000,
		learnEveryN: 10,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.logger == nil {
		o.logger = slog.Default()
	}
	return &EvolvingAgent{inner: inner, opts: o}
}

// ID returns the wrapped agent's ID.
func (e *EvolvingAgent) ID() string { return e.inner.ID() }

// Persona returns the wrapped agent's persona.
func (e *EvolvingAgent) Persona() agent.Persona { return e.inner.Persona() }

// Tools returns the wrapped agent's tools.
func (e *EvolvingAgent) Tools() []tool.Tool { return e.inner.Tools() }

// Children returns the wrapped agent's children.
func (e *EvolvingAgent) Children() []agent.Agent { return e.inner.Children() }

// Invoke calls the inner agent and records the interaction.
func (e *EvolvingAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	start := time.Now()
	result, err := e.inner.Invoke(ctx, input, opts...)
	duration := time.Since(start)

	e.recordInteraction(ctx, Interaction{
		Input:     input,
		Output:    result,
		Duration:  duration,
		Success:   err == nil,
		Timestamp: start,
	})

	return result, err
}

// Stream calls the inner agent's stream and records the interaction.
func (e *EvolvingAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		start := time.Now()
		var output string
		var hadError bool

		for evt, err := range e.inner.Stream(ctx, input, opts...) {
			if err != nil {
				hadError = true
				if !yield(evt, err) {
					return
				}
				continue
			}
			if evt.Type == agent.EventText {
				output += evt.Text
			}
			if !yield(evt, nil) {
				return
			}
		}

		e.recordInteraction(ctx, Interaction{
			Input:     input,
			Output:    output,
			Duration:  time.Since(start),
			Success:   !hadError,
			Timestamp: start,
		})
	}
}

func (e *EvolvingAgent) recordInteraction(ctx context.Context, interaction Interaction) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.interactions = append(e.interactions, interaction)
	e.counter++

	// Trim if over max memory.
	if len(e.interactions) > e.opts.maxMemory {
		e.interactions = e.interactions[len(e.interactions)-e.opts.maxMemory:]
	}

	// Trigger learning periodically. Decouple the background context
	// from the request context so the goroutine outlives the caller
	// while preserving trace spans, tenant ID, and other values.
	if e.counter%e.opts.learnEveryN == 0 {
		go e.learn(context.WithoutCancel(ctx))
	}
}

func (e *EvolvingAgent) learn(ctx context.Context) {
	e.mu.Lock()
	interactions := make([]Interaction, len(e.interactions))
	copy(interactions, e.interactions)
	e.mu.Unlock()

	experiences, err := e.opts.distiller.Distill(ctx, interactions)
	if err != nil {
		e.opts.logger.LogAttrs(ctx, slog.LevelWarn,
			"evolving: distiller failed",
			slog.String("agent_id", e.inner.ID()),
			slog.String("error", err.Error()),
		)
		return
	}

	suggestions, err := e.opts.optimizer.Optimize(ctx, experiences)
	if err != nil {
		e.opts.logger.LogAttrs(ctx, slog.LevelWarn,
			"evolving: optimizer failed",
			slog.String("agent_id", e.inner.ID()),
			slog.String("error", err.Error()),
		)
		return
	}

	e.mu.Lock()
	e.experiences = experiences
	e.suggestions = suggestions
	e.mu.Unlock()
}

// Experiences returns the current distilled experiences.
func (e *EvolvingAgent) Experiences() []Experience {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]Experience, len(e.experiences))
	copy(result, e.experiences)
	return result
}

// Suggestions returns the current optimization suggestions.
func (e *EvolvingAgent) Suggestions() []Suggestion {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]Suggestion, len(e.suggestions))
	copy(result, e.suggestions)
	return result
}

// InteractionCount returns the total number of recorded interactions.
func (e *EvolvingAgent) InteractionCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.counter
}

// SimpleDistiller distills experiences by analyzing success rates and patterns.
type SimpleDistiller struct{}

var _ ExperienceDistiller = (*SimpleDistiller)(nil)

// Distill extracts experiences from interactions.
func (d *SimpleDistiller) Distill(_ context.Context, interactions []Interaction) ([]Experience, error) {
	if len(interactions) == 0 {
		return nil, nil
	}

	var experiences []Experience

	// Analyze success rate.
	var successes, failures int
	for _, i := range interactions {
		if i.Success {
			successes++
		} else {
			failures++
		}
	}

	successRate := float64(successes) / float64(len(interactions))
	// Confidence reflects the observed success rate so downstream
	// optimizers can act on poor reliability. A low success rate
	// produces a low-confidence reliability experience, which
	// FrequencyOptimizer will flag as needing improvement.
	experiences = append(experiences, Experience{
		Category:   "reliability",
		Pattern:    fmt.Sprintf("Success rate: %.1f%% (%d/%d)", successRate*100, successes, len(interactions)),
		Confidence: successRate,
		LearnedAt:  time.Now(),
	})

	// Analyze response time distribution.
	var totalDuration time.Duration
	for _, i := range interactions {
		totalDuration += i.Duration
	}
	avgDuration := totalDuration / time.Duration(len(interactions))

	experiences = append(experiences, Experience{
		Category:   "performance",
		Pattern:    fmt.Sprintf("Average response time: %v", avgDuration.Round(time.Millisecond)),
		Confidence: 0.8,
		LearnedAt:  time.Now(),
	})

	return experiences, nil
}

// FrequencyOptimizer suggests improvements based on frequency analysis.
type FrequencyOptimizer struct{}

var _ MetaOptimizer = (*FrequencyOptimizer)(nil)

// Optimize generates suggestions from experiences.
func (o *FrequencyOptimizer) Optimize(_ context.Context, experiences []Experience) ([]Suggestion, error) {
	if len(experiences) == 0 {
		return nil, nil
	}

	var suggestions []Suggestion

	for _, exp := range experiences {
		switch exp.Category {
		case "reliability":
			if exp.Confidence < 0.8 {
				suggestions = append(suggestions, Suggestion{
					Type:        "prompt_update",
					Description: "Consider refining system prompt to improve reliability",
					Priority:    3,
					Confidence:  0.6,
				})
			}
		case "performance":
			suggestions = append(suggestions, Suggestion{
				Type:        "optimization",
				Description: "Monitor response time trends for degradation",
				Priority:    1,
				Confidence:  0.5,
			})
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority > suggestions[j].Priority
	})

	return suggestions, nil
}
