package replay

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// ResponseProvider supplies pre-recorded LLM responses for replay. Given a
// turn index, it returns the AI message that was originally produced at that
// turn. This allows deterministic re-execution without calling a live LLM.
type ResponseProvider func(turnIndex int) (*schema.AIMessage, error)

// TurnProcessor is a function that processes a single turn during replay,
// receiving the checkpoint state, the turn input, and the recorded AI response.
// It returns the events produced by that turn.
type TurnProcessor func(ctx context.Context, state map[string]any, input schema.Message, response *schema.AIMessage) ([]schema.AgentEvent, error)

// replayerOptions holds configuration for a Replayer.
type replayerOptions struct {
	store        CheckpointStore
	processor    TurnProcessor
	maxTurns     int
	turnTimeout  time.Duration
	onTurnReplay func(turnIndex int, events []schema.AgentEvent)
}

// ReplayerOption configures a Replayer.
type ReplayerOption func(*replayerOptions)

// WithStore sets the checkpoint store for the replayer.
func WithStore(store CheckpointStore) ReplayerOption {
	return func(o *replayerOptions) {
		o.store = store
	}
}

// WithProcessor sets the turn processor for replay execution.
func WithProcessor(p TurnProcessor) ReplayerOption {
	return func(o *replayerOptions) {
		o.processor = p
	}
}

// WithMaxTurns sets the maximum number of turns to replay.
// Zero means replay all turns from the checkpoint.
func WithMaxTurns(n int) ReplayerOption {
	return func(o *replayerOptions) {
		if n >= 0 {
			o.maxTurns = n
		}
	}
}

// WithTurnTimeout sets the timeout for each individual turn during replay.
func WithTurnTimeout(d time.Duration) ReplayerOption {
	return func(o *replayerOptions) {
		o.turnTimeout = d
	}
}

// WithOnTurnReplay sets a callback invoked after each turn is replayed.
func WithOnTurnReplay(fn func(turnIndex int, events []schema.AgentEvent)) ReplayerOption {
	return func(o *replayerOptions) {
		o.onTurnReplay = fn
	}
}

// ReplayResult contains the outcome of a replay operation.
type ReplayResult struct {
	// Events contains all events produced during replay, in order.
	Events []schema.AgentEvent

	// TurnsReplayed is the number of turns that were replayed.
	TurnsReplayed int

	// Duration is the wall-clock time of the replay.
	Duration time.Duration
}

// Replayer re-executes agent turns from a checkpoint using pre-recorded LLM
// responses. This enables deterministic replay for debugging, testing, and
// analysis.
type Replayer struct {
	opts replayerOptions
}

// NewReplayer creates a new Replayer with the given options.
func NewReplayer(opts ...ReplayerOption) *Replayer {
	o := replayerOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return &Replayer{opts: o}
}

// Replay re-executes turns from the given checkpoint using the provided
// response provider for LLM responses. It returns the events produced during
// replay and the number of turns replayed.
func (r *Replayer) Replay(ctx context.Context, cp *Checkpoint, responses ResponseProvider) (*ReplayResult, error) {
	if cp == nil {
		return nil, core.NewError("replay.replay", core.ErrInvalidInput, "checkpoint must not be nil", nil)
	}
	if responses == nil {
		return nil, core.NewError("replay.replay", core.ErrInvalidInput, "response provider must not be nil", nil)
	}
	if r.opts.processor == nil {
		return nil, core.NewError("replay.replay", core.ErrInvalidInput, "turn processor must be set via WithProcessor", nil)
	}

	start := time.Now()

	// Determine how many turns to replay.
	startTurn := cp.TurnIndex + 1
	totalTurns := len(cp.Turns) - startTurn
	if totalTurns < 0 {
		totalTurns = 0
	}
	if r.opts.maxTurns > 0 && totalTurns > r.opts.maxTurns {
		totalTurns = r.opts.maxTurns
	}

	// Copy checkpoint state for replay.
	state := make(map[string]any, len(cp.State))
	for k, v := range cp.State {
		state[k] = v
	}

	var allEvents []schema.AgentEvent
	replayed := 0

	for i := 0; i < totalTurns; i++ {
		turnIdx := startTurn + i

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("replay cancelled at turn %d: %w", turnIdx, err)
		}

		response, err := responses(turnIdx)
		if err != nil {
			return nil, core.NewError("replay.replay", core.ErrInvalidInput,
				fmt.Sprintf("no response for turn %d", turnIdx), err)
		}

		// Run the processor in a closure so any per-turn context is cancelled
		// eagerly at the end of each iteration (avoiding resource accumulation
		// that would occur if defer cancel() were used inside the loop) while
		// still guaranteeing cancellation on panic via defer.
		events, err := func() ([]schema.AgentEvent, error) {
			turnCtx := ctx
			if r.opts.turnTimeout > 0 {
				var cancel context.CancelFunc
				turnCtx, cancel = context.WithTimeout(ctx, r.opts.turnTimeout)
				defer cancel()
			}
			return r.opts.processor(turnCtx, state, cp.Turns[turnIdx].Input, response)
		}()
		if err != nil {
			return nil, core.NewError("replay.replay", core.ErrToolFailed,
				fmt.Sprintf("turn processor failed at turn %d", turnIdx), err)
		}

		allEvents = append(allEvents, events...)
		replayed++

		if r.opts.onTurnReplay != nil {
			r.opts.onTurnReplay(turnIdx, events)
		}
	}

	return &ReplayResult{
		Events:        allEvents,
		TurnsReplayed: replayed,
		Duration:      time.Since(start),
	}, nil
}
