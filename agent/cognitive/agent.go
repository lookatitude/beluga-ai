package cognitive

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// DualProcessAgent wraps a fast System 1 agent and a deliberative System 2
// agent, routing requests based on input complexity scoring.
//
// For Invoke (synchronous), it uses a cascading strategy: S1 runs first,
// and if the scorer rates the output as needing escalation, it re-runs
// with S2. For Stream, it pre-classifies the input and routes directly.
type DualProcessAgent struct {
	id        string
	s1        agent.Agent
	s2        agent.Agent
	scorer    ComplexityScorer
	threshold float64
	hooks     Hooks
	metrics   *RoutingMetrics
}

// Compile-time interface check.
var _ agent.Agent = (*DualProcessAgent)(nil)

// Option configures a DualProcessAgent.
type Option func(*DualProcessAgent)

// WithScorer sets the complexity scorer used for routing decisions.
// If not set, a HeuristicScorer with default settings is used.
func WithScorer(s ComplexityScorer) Option {
	return func(a *DualProcessAgent) {
		if s != nil {
			a.scorer = s
		}
	}
}

// WithThreshold sets the confidence threshold below which S1 output is
// escalated to S2 during cascading Invoke calls. Must be between 0.0
// and 1.0. Defaults to 0.7.
func WithThreshold(t float64) Option {
	return func(a *DualProcessAgent) {
		if t >= 0.0 && t <= 1.0 {
			a.threshold = t
		}
	}
}

// WithCognitiveHooks sets lifecycle hooks for the dual-process agent.
func WithCognitiveHooks(h Hooks) Option {
	return func(a *DualProcessAgent) {
		a.hooks = h
	}
}

// WithMetrics sets a RoutingMetrics instance for tracking routing statistics.
// If not set, metrics are tracked in a private instance accessible via Metrics().
func WithMetrics(m *RoutingMetrics) Option {
	return func(a *DualProcessAgent) {
		if m != nil {
			a.metrics = m
		}
	}
}

// New creates a new DualProcessAgent with the given System 1 (fast) and
// System 2 (deliberative) agents. Both agents are required.
func New(id string, s1, s2 agent.Agent, opts ...Option) (*DualProcessAgent, error) {
	if s1 == nil {
		return nil, fmt.Errorf("cognitive: system 1 agent is required")
	}
	if s2 == nil {
		return nil, fmt.Errorf("cognitive: system 2 agent is required")
	}

	a := &DualProcessAgent{
		id:        id,
		s1:        s1,
		s2:        s2,
		scorer:    NewHeuristicScorer(),
		threshold: 0.7,
		metrics:   &RoutingMetrics{},
	}

	for _, opt := range opts {
		opt(a)
	}

	return a, nil
}

// ID returns the agent's unique identifier.
func (a *DualProcessAgent) ID() string { return a.id }

// Persona returns a persona describing the dual-process agent.
func (a *DualProcessAgent) Persona() agent.Persona {
	return agent.Persona{
		Role: "dual-process cognitive agent",
		Goal: "route requests optimally between fast (S1) and deliberative (S2) processing",
	}
}

// Tools returns nil; the dual-process agent delegates tools to its children.
func (a *DualProcessAgent) Tools() []tool.Tool { return nil }

// Children returns the S1 and S2 agents.
func (a *DualProcessAgent) Children() []agent.Agent {
	return []agent.Agent{a.s1, a.s2}
}

// Metrics returns the routing metrics tracker.
func (a *DualProcessAgent) Metrics() *RoutingMetrics { return a.metrics }

// Invoke executes the agent using cascading: run S1 first, score the input
// complexity, and escalate to S2 if the complexity score indicates it.
func (a *DualProcessAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	start := time.Now()

	// Score complexity
	score, err := a.scorer.Score(ctx, input)
	if err != nil {
		return "", fmt.Errorf("cognitive: scoring failed: %w", err)
	}

	// If clearly complex, go directly to S2
	if score.Level == Complex && score.Confidence >= a.threshold {
		a.fireOnRouted(ctx, input, score.Level, "s2")
		result, err := a.s2.Invoke(ctx, input, opts...)
		latency := time.Since(start)
		a.metrics.RecordS2(latency, 0)
		a.fireOnCompleted(ctx, "s2", latency)
		return result, err
	}

	// Cascading: try S1 first
	a.fireOnRouted(ctx, input, score.Level, "s1")
	s1Result, s1Err := a.s1.Invoke(ctx, input, opts...)
	s1Latency := time.Since(start)

	if s1Err != nil {
		// S1 failed - escalate to S2
		a.metrics.RecordEscalation()
		a.fireOnEscalated(ctx, input, "", fmt.Sprintf("s1 error: %v", s1Err))

		s2Start := time.Now()
		result, err := a.s2.Invoke(ctx, input, opts...)
		s2Latency := time.Since(s2Start)
		totalLatency := time.Since(start)
		a.metrics.RecordS2(s2Latency, 0)
		a.fireOnCompleted(ctx, "s2", totalLatency)
		return result, err
	}

	// Evaluate whether S1 output is sufficient
	// For moderate inputs with lower confidence, escalate
	if score.Level >= Moderate && score.Confidence < a.threshold {
		a.metrics.RecordEscalation()
		a.fireOnEscalated(ctx, input, s1Result, fmt.Sprintf(
			"confidence %.2f below threshold %.2f for %s input",
			score.Confidence, a.threshold, score.Level,
		))

		s2Start := time.Now()
		result, err := a.s2.Invoke(ctx, input, opts...)
		s2Latency := time.Since(s2Start)
		totalLatency := time.Since(start)
		a.metrics.RecordS2(s2Latency, 0)
		a.fireOnCompleted(ctx, "s2", totalLatency)
		return result, err
	}

	// S1 result is sufficient
	a.metrics.RecordS1(s1Latency, 0)
	a.fireOnCompleted(ctx, "s1", s1Latency)
	return s1Result, nil
}

// Stream pre-classifies the input and routes directly to S1 or S2 to avoid
// buffering the entire response for evaluation.
func (a *DualProcessAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		start := time.Now()

		// Pre-classify
		score, err := a.scorer.Score(ctx, input)
		if err != nil {
			yield(agent.Event{
				Type:    agent.EventError,
				AgentID: a.id,
			}, fmt.Errorf("cognitive: scoring failed: %w", err))
			return
		}

		// Route based on score
		var target agent.Agent
		var tier string
		if score.Level == Complex || (score.Level >= Moderate && score.Confidence < a.threshold) {
			target = a.s2
			tier = "s2"
		} else {
			target = a.s1
			tier = "s1"
		}

		a.fireOnRouted(ctx, input, score.Level, tier)

		// Emit routing metadata event
		if !yield(agent.Event{
			Type:    agent.EventText,
			AgentID: a.id,
			Metadata: map[string]any{
				"cognitive.tier":       tier,
				"cognitive.level":      score.Level.String(),
				"cognitive.confidence": score.Confidence,
			},
		}, nil) {
			return
		}

		// Stream from selected agent
		var result strings.Builder
		for event, err := range target.Stream(ctx, input, opts...) {
			if err != nil {
				yield(event, err)
				return
			}
			if event.Type == agent.EventText {
				result.WriteString(event.Text)
			}
			if !yield(event, nil) {
				return
			}
		}

		latency := time.Since(start)
		switch tier {
		case "s1":
			a.metrics.RecordS1(latency, 0)
		case "s2":
			a.metrics.RecordS2(latency, 0)
		}
		a.fireOnCompleted(ctx, tier, latency)
	}
}

// fireOnRouted calls the OnRouted hook if set.
func (a *DualProcessAgent) fireOnRouted(ctx context.Context, input string, level ComplexityLevel, target string) {
	if a.hooks.OnRouted != nil {
		a.hooks.OnRouted(ctx, input, level, target)
	}
}

// fireOnEscalated calls the OnEscalated hook if set.
func (a *DualProcessAgent) fireOnEscalated(ctx context.Context, input, s1Output, reason string) {
	if a.hooks.OnEscalated != nil {
		a.hooks.OnEscalated(ctx, input, s1Output, reason)
	}
}

// fireOnCompleted calls the OnCompleted hook if set.
func (a *DualProcessAgent) fireOnCompleted(ctx context.Context, tier string, latency time.Duration) {
	if a.hooks.OnCompleted != nil {
		a.hooks.OnCompleted(ctx, tier, latency)
	}
}
