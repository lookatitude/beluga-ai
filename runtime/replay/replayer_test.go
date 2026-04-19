package replay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleProcessor creates events from the response text.
func simpleProcessor(_ context.Context, _ map[string]any, _ schema.Message, resp *schema.AIMessage) ([]schema.AgentEvent, error) {
	return []schema.AgentEvent{
		{Type: "response", AgentID: "test-agent", Payload: resp.Text(), Timestamp: time.Now()},
	}, nil
}

func TestReplayer_Replay(t *testing.T) {
	turns := []schema.Turn{
		{Input: schema.NewHumanMessage("turn-0"), Output: schema.NewAIMessage("resp-0"), Timestamp: time.Now()},
		{Input: schema.NewHumanMessage("turn-1"), Output: schema.NewAIMessage("resp-1"), Timestamp: time.Now()},
		{Input: schema.NewHumanMessage("turn-2"), Output: schema.NewAIMessage("resp-2"), Timestamp: time.Now()},
	}

	responses := func(turnIndex int) (*schema.AIMessage, error) {
		if turnIndex < 0 || turnIndex >= len(turns) {
			return nil, errors.New("no response")
		}
		return schema.NewAIMessage("replayed-" + intToStr(turnIndex)), nil
	}

	tests := []struct {
		name          string
		checkpoint    *Checkpoint
		responses     ResponseProvider
		processor     TurnProcessor
		maxTurns      int
		wantTurns     int
		wantErr       bool
		wantEventType string
	}{
		{
			name: "replay from beginning",
			checkpoint: &Checkpoint{
				ID:        "cp-1",
				SessionID: "sess-1",
				TurnIndex: -1,
				Turns:     turns,
				State:     map[string]any{"key": "value"},
			},
			responses:     responses,
			processor:     simpleProcessor,
			wantTurns:     3,
			wantEventType: "response",
		},
		{
			name: "replay from middle",
			checkpoint: &Checkpoint{
				ID:        "cp-2",
				SessionID: "sess-1",
				TurnIndex: 1,
				Turns:     turns,
				State:     map[string]any{},
			},
			responses: responses,
			processor: simpleProcessor,
			wantTurns: 1,
		},
		{
			name: "replay with max turns",
			checkpoint: &Checkpoint{
				ID:        "cp-3",
				SessionID: "sess-1",
				TurnIndex: -1,
				Turns:     turns,
				State:     map[string]any{},
			},
			responses: responses,
			processor: simpleProcessor,
			maxTurns:  2,
			wantTurns: 2,
		},
		{
			name:      "nil checkpoint",
			responses: responses,
			processor: simpleProcessor,
			wantErr:   true,
		},
		{
			name: "nil responses",
			checkpoint: &Checkpoint{
				ID:        "cp-4",
				SessionID: "sess-1",
				TurnIndex: -1,
				Turns:     turns,
			},
			responses: nil,
			processor: simpleProcessor,
			wantErr:   true,
		},
		{
			name: "no processor",
			checkpoint: &Checkpoint{
				ID:        "cp-5",
				SessionID: "sess-1",
				TurnIndex: -1,
				Turns:     turns,
			},
			responses: responses,
			processor: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []ReplayerOption{}
			if tt.processor != nil {
				opts = append(opts, WithProcessor(tt.processor))
			}
			if tt.maxTurns > 0 {
				opts = append(opts, WithMaxTurns(tt.maxTurns))
			}

			r := NewReplayer(opts...)
			result, err := r.Replay(context.Background(), tt.checkpoint, tt.responses)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTurns, result.TurnsReplayed)
			assert.Len(t, result.Events, tt.wantTurns)
			assert.True(t, result.Duration >= 0)
		})
	}
}

func TestReplayer_ContextCancellation(t *testing.T) {
	turns := make([]schema.Turn, 10)
	for i := range turns {
		turns[i] = schema.Turn{
			Input:     schema.NewHumanMessage("turn"),
			Output:    schema.NewAIMessage("resp"),
			Timestamp: time.Now(),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	processor := func(ctx context.Context, _ map[string]any, _ schema.Message, resp *schema.AIMessage) ([]schema.AgentEvent, error) {
		callCount++
		if callCount >= 2 {
			cancel()
		}
		return []schema.AgentEvent{{Type: "test"}}, nil
	}

	cp := &Checkpoint{
		ID:        "cp-cancel",
		SessionID: "sess-1",
		TurnIndex: -1,
		Turns:     turns,
		State:     map[string]any{},
	}

	responses := func(turnIndex int) (*schema.AIMessage, error) {
		return schema.NewAIMessage("resp"), nil
	}

	r := NewReplayer(WithProcessor(processor))
	_, err := r.Replay(ctx, cp, responses)
	require.Error(t, err)
	assert.True(t, callCount <= 3, "should stop early on cancellation")
}

func TestReplayer_OnTurnReplayCallback(t *testing.T) {
	turns := []schema.Turn{
		{Input: schema.NewHumanMessage("t0"), Output: schema.NewAIMessage("r0"), Timestamp: time.Now()},
		{Input: schema.NewHumanMessage("t1"), Output: schema.NewAIMessage("r1"), Timestamp: time.Now()},
	}

	var callbackTurns []int
	callback := func(turnIndex int, events []schema.AgentEvent) {
		callbackTurns = append(callbackTurns, turnIndex)
	}

	cp := &Checkpoint{
		ID: "cp-cb", SessionID: "sess-1", TurnIndex: -1,
		Turns: turns, State: map[string]any{},
	}
	responses := func(turnIndex int) (*schema.AIMessage, error) {
		return schema.NewAIMessage("resp"), nil
	}

	r := NewReplayer(
		WithProcessor(simpleProcessor),
		WithOnTurnReplay(callback),
	)
	result, err := r.Replay(context.Background(), cp, responses)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TurnsReplayed)
	assert.Equal(t, []int{0, 1}, callbackTurns)
}

func TestReplayer_ProcessorError(t *testing.T) {
	turns := []schema.Turn{
		{Input: schema.NewHumanMessage("t0"), Output: schema.NewAIMessage("r0"), Timestamp: time.Now()},
	}

	failProcessor := func(_ context.Context, _ map[string]any, _ schema.Message, _ *schema.AIMessage) ([]schema.AgentEvent, error) {
		return nil, errors.New("processor failed")
	}

	cp := &Checkpoint{
		ID: "cp-err", SessionID: "sess-1", TurnIndex: -1,
		Turns: turns, State: map[string]any{},
	}
	responses := func(turnIndex int) (*schema.AIMessage, error) {
		return schema.NewAIMessage("resp"), nil
	}

	r := NewReplayer(WithProcessor(failProcessor))
	_, err := r.Replay(context.Background(), cp, responses)
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrToolFailed, coreErr.Code)
}
