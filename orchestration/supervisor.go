package orchestration

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
)

// StrategyFunc selects an agent from the available agents for the given input.
// Returning a nil agent signals that execution should stop.
type StrategyFunc func(ctx context.Context, input any, agents []agent.Agent) (agent.Agent, error)

// Supervisor orchestrates multiple agents by delegating work using a strategy
// function. It loops up to maxRounds, passing each result back to the strategy
// for the next selection. Execution stops when the strategy returns nil or
// maxRounds is reached.
type Supervisor struct {
	agents    []agent.Agent
	strategy  StrategyFunc
	maxRounds int
}

// NewSupervisor creates a Supervisor with the given strategy and agents.
func NewSupervisor(strategy StrategyFunc, agents ...agent.Agent) *Supervisor {
	return &Supervisor{
		agents:    agents,
		strategy:  strategy,
		maxRounds: 1,
	}
}

// WithMaxRounds sets the maximum number of delegation rounds.
func (s *Supervisor) WithMaxRounds(n int) *Supervisor {
	if n > 0 {
		s.maxRounds = n
	}
	return s
}

// Invoke selects agents via the strategy and invokes them, looping until the
// strategy returns nil or maxRounds is reached.
func (s *Supervisor) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if len(s.agents) == 0 {
		return nil, fmt.Errorf("orchestration/supervisor: no agents configured")
	}

	current := input
	for round := 0; round < s.maxRounds; round++ {
		selected, err := s.strategy(ctx, current, s.agents)
		if err != nil {
			return nil, fmt.Errorf("orchestration/supervisor: strategy: %w", err)
		}
		if selected == nil {
			return current, nil
		}

		// Convert input to string for agent.Invoke.
		inputStr := fmt.Sprintf("%v", current)
		result, err := selected.Invoke(ctx, inputStr)
		if err != nil {
			return nil, fmt.Errorf("orchestration/supervisor: agent %q: %w", selected.ID(), err)
		}
		current = result
	}
	return current, nil
}

// Stream selects agents and streams the last invocation.
func (s *Supervisor) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		if len(s.agents) == 0 {
			yield(nil, fmt.Errorf("orchestration/supervisor: no agents configured"))
			return
		}

		s.streamRounds(ctx, input, yield)
	}
}

// streamRounds iterates through delegation rounds, streaming the final round.
func (s *Supervisor) streamRounds(ctx context.Context, input any, yield func(any, error) bool) {
	current := input
	for round := 0; round < s.maxRounds; round++ {
		selected, err := s.strategy(ctx, current, s.agents)
		if err != nil {
			yield(nil, fmt.Errorf("orchestration/supervisor: strategy: %w", err))
			return
		}
		if selected == nil {
			yield(current, nil)
			return
		}

		inputStr := fmt.Sprintf("%v", current)

		if round == s.maxRounds-1 {
			streamAgent(ctx, selected, inputStr, yield)
			return
		}

		result, err := selected.Invoke(ctx, inputStr)
		if err != nil {
			yield(nil, fmt.Errorf("orchestration/supervisor: agent %q: %w", selected.ID(), err))
			return
		}
		current = result
	}
}

// streamAgent streams an agent's output through the yield function.
func streamAgent(ctx context.Context, a agent.Agent, input string, yield func(any, error) bool) {
	for event, err := range a.Stream(ctx, input) {
		if err != nil {
			yield(nil, fmt.Errorf("orchestration/supervisor: agent %q: %w", a.ID(), err))
			return
		}
		if !yield(event, nil) {
			return
		}
	}
}

// DelegateBySkill returns a strategy that picks the agent whose persona goal
// best matches the input by simple keyword overlap.
func DelegateBySkill() StrategyFunc {
	return func(_ context.Context, input any, agents []agent.Agent) (agent.Agent, error) {
		inputStr := strings.ToLower(fmt.Sprintf("%v", input))
		words := strings.Fields(inputStr)

		best := bestSkillMatch(words, agents)
		if best == nil && len(agents) > 0 {
			best = agents[0]
		}
		return best, nil
	}
}

// bestSkillMatch returns the agent whose goal has the most keyword overlap with words.
func bestSkillMatch(words []string, agents []agent.Agent) agent.Agent {
	var best agent.Agent
	bestScore := 0

	for _, a := range agents {
		score := skillScore(words, strings.ToLower(a.Persona().Goal))
		if score > bestScore {
			bestScore = score
			best = a
		}
	}
	return best
}

// skillScore counts how many words (longer than 2 chars) appear in the goal string.
func skillScore(words []string, goal string) int {
	score := 0
	for _, w := range words {
		if len(w) > 2 && strings.Contains(goal, w) {
			score++
		}
	}
	return score
}

// RoundRobin returns a strategy that cycles through agents in order.
func RoundRobin() StrategyFunc {
	var counter atomic.Int64
	return func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		if len(agents) == 0 {
			return nil, nil
		}
		idx := counter.Add(1) - 1
		return agents[idx%int64(len(agents))], nil
	}
}

// LoadBalanced returns a strategy that picks the agent with the lowest
// invocation count. This distributes work evenly across agents.
// Safe for concurrent use.
func LoadBalanced() StrategyFunc {
	var mu sync.Mutex
	counts := make(map[string]*atomic.Int64)
	return func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		if len(agents) == 0 {
			return nil, nil
		}

		mu.Lock()
		defer mu.Unlock()

		var best agent.Agent
		bestCount := int64(1<<63 - 1)

		for _, a := range agents {
			c, ok := counts[a.ID()]
			if !ok {
				c = &atomic.Int64{}
				counts[a.ID()] = c
			}
			count := c.Load()
			if count < bestCount {
				bestCount = count
				best = a
			}
		}

		if best != nil {
			counts[best.ID()].Add(1)
		}
		return best, nil
	}
}
