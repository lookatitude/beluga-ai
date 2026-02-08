package orchestration

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"sync"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
)

// TerminationFunc decides whether the blackboard has reached a terminal state.
// It receives the current board state and returns true to stop iteration.
type TerminationFunc func(board map[string]any) bool

// Blackboard implements the blackboard architecture pattern: multiple agents
// collaborate by reading from and writing to a shared board. Each round, every
// agent sees the current board state and produces output. Agents' outputs are
// stored on the board under their ID. Execution continues until the
// termination condition is met or maxRounds is reached.
type Blackboard struct {
	agents      []agent.Agent
	board       map[string]any
	termination TerminationFunc
	maxRounds   int
	mu          sync.RWMutex
}

// NewBlackboard creates a Blackboard with the given termination condition and agents.
func NewBlackboard(termination TerminationFunc, agents ...agent.Agent) *Blackboard {
	return &Blackboard{
		agents:      agents,
		board:       make(map[string]any),
		termination: termination,
		maxRounds:   10,
	}
}

// WithMaxRounds sets the maximum number of rounds.
func (b *Blackboard) WithMaxRounds(n int) *Blackboard {
	if n > 0 {
		b.maxRounds = n
	}
	return b
}

// Set stores a value on the board.
func (b *Blackboard) Set(key string, value any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.board[key] = value
}

// Get retrieves a value from the board.
func (b *Blackboard) Get(key string) (any, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	v, ok := b.board[key]
	return v, ok
}

// snapshot returns a copy of the current board state.
func (b *Blackboard) snapshot() map[string]any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return maps.Clone(b.board)
}

// Invoke runs the blackboard loop: each round, all agents see the current
// board state and produce output. Stops when termination returns true or
// maxRounds is reached.
func (b *Blackboard) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if len(b.agents) == 0 {
		return nil, fmt.Errorf("orchestration/blackboard: no agents configured")
	}

	// Set the initial input on the board.
	b.Set("input", input)

	for round := 0; round < b.maxRounds; round++ {
		snap := b.snapshot()

		// Check termination condition.
		if b.termination(snap) {
			return snap, nil
		}

		// Each agent processes the board state.
		for _, a := range b.agents {
			inputStr := fmt.Sprintf("%v", snap)
			result, err := a.Invoke(ctx, inputStr)
			if err != nil {
				return nil, fmt.Errorf("orchestration/blackboard: agent %q round %d: %w", a.ID(), round, err)
			}
			b.Set(a.ID(), result)
		}
	}

	// Return final board state.
	return b.snapshot(), nil
}

// Stream runs the blackboard loop and yields the board state after each round.
func (b *Blackboard) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		if len(b.agents) == 0 {
			yield(nil, fmt.Errorf("orchestration/blackboard: no agents configured"))
			return
		}

		b.Set("input", input)

		for round := 0; round < b.maxRounds; round++ {
			snap := b.snapshot()

			if b.termination(snap) {
				yield(snap, nil)
				return
			}

			for _, a := range b.agents {
				inputStr := fmt.Sprintf("%v", snap)
				result, err := a.Invoke(ctx, inputStr)
				if err != nil {
					yield(nil, fmt.Errorf("orchestration/blackboard: agent %q round %d: %w", a.ID(), round, err))
					return
				}
				b.Set(a.ID(), result)
			}

			// Yield board state after each round.
			if !yield(b.snapshot(), nil) {
				return
			}
		}
	}
}
