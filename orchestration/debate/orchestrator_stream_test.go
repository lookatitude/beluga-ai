package debate

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errDetector is a ConvergenceDetector whose Check always fails.
type errDetector struct{ err error }

func (d errDetector) Check(context.Context, DebateState) (ConvergenceResult, error) {
	return ConvergenceResult{}, d.err
}

// TestDebateOrchestrator_Stream_EarlyBreak asserts the stream stops cleanly
// when the consumer breaks at each event type, without yielding again.
func TestDebateOrchestrator_Stream_EarlyBreak(t *testing.T) {
	// Full single-round sequence with two agents:
	// RoundStart, Contribution, Contribution, RoundEnd, Convergence, Complete.
	for stopAfter := 1; stopAfter <= 5; stopAfter++ {
		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents, WithMaxRounds(1))

		seen := 0
		for _, err := range d.Stream(context.Background(), "topic") {
			require.NoError(t, err)
			seen++
			if seen == stopAfter {
				break
			}
		}
		assert.Equal(t, stopAfter, seen, "stopAfter=%d", stopAfter)
	}
}

// TestDebateOrchestrator_Stream_HookAndDetectorErrors asserts every hook and
// convergence failure inside a streamed round surfaces as a stream error.
func TestDebateOrchestrator_Stream_HookAndDetectorErrors(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name string
		opts []Option
	}{
		{"before round hook", []Option{WithHooks(Hooks{BeforeRound: func(context.Context, DebateState) error { return boom }})}},
		{"after round hook", []Option{WithHooks(Hooks{AfterRound: func(context.Context, DebateState) error { return boom }})}},
		{"convergence check", []Option{WithConvergenceDetector(errDetector{err: boom})}},
		{"on convergence hook", []Option{WithHooks(Hooks{OnConvergence: func(context.Context, ConvergenceResult) error { return boom }})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
			d := NewDebateOrchestrator(agents, append([]Option{WithMaxRounds(2)}, tt.opts...)...)

			var gotErr error
			for _, err := range d.Stream(context.Background(), "topic") {
				if err != nil {
					gotErr = err
					break
				}
			}
			require.Error(t, gotErr)
			assert.ErrorIs(t, gotErr, boom)
		})
	}
}

// TestDebateOrchestrator_Stream_AgentError asserts a failing agent ends the
// stream with an error.
func TestDebateOrchestrator_Stream_AgentError(t *testing.T) {
	boom := errors.New("agent down")
	agents := []agent.Agent{
		newMockAgentWithFn("a1", func(context.Context, string) (string, error) { return "", boom }),
		newMockAgent("a2"),
	}
	d := NewDebateOrchestrator(agents, WithMaxRounds(1))

	var gotErr error
	for _, err := range d.Stream(context.Background(), "topic") {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
}
