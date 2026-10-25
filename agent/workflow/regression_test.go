package workflow

import (
	"context"
	"iter"
	"runtime"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// endlessStream yields text events until the consumer stops. It does not
// look at ctx, so only a ctx-aware send in the parent can unblock it.
func endlessStream(id string) func(context.Context, string) iter.Seq2[agent.Event, error] {
	return func(_ context.Context, _ string) iter.Seq2[agent.Event, error] {
		return func(yield func(agent.Event, error) bool) {
			for {
				if !yield(agent.Event{Type: agent.EventText, Text: id, AgentID: id}, nil) {
					return
				}
			}
		}
	}
}

// TestParallelAgent_StreamEarlyBreakNoLeak is a regression test for FWC-05:
// child goroutines blocked forever on the channel send once the consumer
// stopped reading.
func TestParallelAgent_StreamEarlyBreakNoLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 10 {
		pa := NewParallelAgent("p", []agent.Agent{
			&parMockAgent{id: "a", streamFn: endlessStream("a")},
			&parMockAgent{id: "b", streamFn: endlessStream("b")},
		})
		for range pa.Stream(context.Background(), "x") {
			break
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			t.Fatalf("goroutines leaked: before=%d after=%d", before, runtime.NumGoroutine())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestWorkflowAgents_PreserveChildErrorCode is a regression test for FWC-03:
// wrapping a child's guard_blocked error as ErrProviderDown made it retryable.
func TestWorkflowAgents_PreserveChildErrorCode(t *testing.T) {
	blocked := core.Errorf(core.ErrGuardBlocked, "blocked by guard")
	failing := func() agent.Agent {
		return &parMockAgent{
			id:       "child",
			invokeFn: func(context.Context, string) (string, error) { return "", blocked },
			streamFn: func(context.Context, string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) { yield(agent.Event{}, blocked) }
			},
		}
	}
	agents := map[string]agent.Agent{
		"parallel":   NewParallelAgent("p", []agent.Agent{failing()}),
		"sequential": NewSequentialAgent("s", []agent.Agent{failing()}),
		"loop":       NewLoopAgent("l", failing(), WithLoopMaxIterations(2)),
	}
	for name, a := range agents {
		t.Run(name+"/invoke", func(t *testing.T) {
			_, err := a.Invoke(context.Background(), "x")
			assertGuardBlocked(t, err)
		})
		t.Run(name+"/stream", func(t *testing.T) {
			var got error
			for _, err := range a.Stream(context.Background(), "x") {
				if err != nil {
					got = err
					break
				}
			}
			assertGuardBlocked(t, got)
		})
	}
}

func assertGuardBlocked(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if core.IsRetryable(err) {
		t.Errorf("error became retryable: %v", err)
	}
	if code := core.CodeOf(err, ""); code != core.ErrGuardBlocked {
		t.Errorf("code = %q, want %q", code, core.ErrGuardBlocked)
	}
}
