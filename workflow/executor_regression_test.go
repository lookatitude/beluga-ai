package workflow

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// mapStore is a minimal in-package WorkflowStore for tests.
type mapStore struct {
	mu     sync.Mutex
	states map[string]WorkflowState
}

func newMapStore() *mapStore { return &mapStore{states: map[string]WorkflowState{}} }

func (m *mapStore) Save(_ context.Context, st WorkflowState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[st.WorkflowID] = st
	return nil
}

func (m *mapStore) Load(_ context.Context, id string) (*WorkflowState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.states[id]
	if !ok {
		return nil, nil
	}
	return &st, nil
}

func (m *mapStore) List(context.Context, WorkflowFilter) ([]WorkflowState, error) { return nil, nil }

func (m *mapStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, id)
	return nil
}

func blockUntilCanceled(ctx WorkflowContext, _ any) (any, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestDefaultExecutor_CancelEndsCanceled is a regression test for FWC-10:
// Cancel's StatusCanceled was overwritten with StatusFailed and the fail hook
// fired.
func TestDefaultExecutor_CancelEndsCanceled(t *testing.T) {
	var failHooks atomic.Int32
	e := NewExecutor(WithExecutorHooks(Hooks{
		OnWorkflowFail: func(context.Context, string, error) { failHooks.Add(1) },
	}))
	for range 50 {
		h, err := e.Execute(context.Background(), blockUntilCanceled, WorkflowOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Cancel(context.Background(), h.ID()); err != nil {
			t.Fatal(err)
		}
		_, resErr := h.Result(context.Background())
		if h.Status() != StatusCanceled {
			t.Fatalf("status after Cancel = %s, want canceled", h.Status())
		}
		if !errors.Is(resErr, context.Canceled) || core.IsRetryable(resErr) {
			t.Fatalf("Result error = %v, want non-retryable cancellation", resErr)
		}
	}
	if n := failHooks.Load(); n != 0 {
		t.Errorf("OnWorkflowFail fired %d times for user cancels", n)
	}
}

// TestDefaultExecutor_SignalRespectsContext is a regression test for FWC-11:
// the 11th unconsumed signal blocked forever, ignoring ctx.
func TestDefaultExecutor_SignalRespectsContext(t *testing.T) {
	e := NewExecutor()
	h, err := e.Execute(context.Background(), blockUntilCanceled, WorkflowOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = e.Cancel(context.Background(), h.ID()) }()

	for i := range 10 {
		if err := e.Signal(context.Background(), h.ID(), Signal{Name: "s", Payload: i}); err != nil {
			t.Fatalf("signal %d: %v", i, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- e.Signal(ctx, h.ID(), Signal{Name: "s", Payload: 10}) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("11th Signal error = %v, want deadline exceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Signal blocked past its ctx deadline")
	}
}

// TestDefaultExecutor_SignalAfterFinish checks a blocked Signal returns once
// the workflow finishes.
func TestDefaultExecutor_SignalAfterFinish(t *testing.T) {
	e := NewExecutor()
	release := make(chan struct{})
	h, err := e.Execute(context.Background(), func(WorkflowContext, any) (any, error) {
		<-release
		return "ok", nil
	}, WorkflowOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 10 {
		_ = e.Signal(context.Background(), h.ID(), Signal{Name: "s", Payload: i})
	}
	done := make(chan error, 1)
	go func() { done <- e.Signal(context.Background(), h.ID(), Signal{Name: "s", Payload: 10}) }()
	time.Sleep(20 * time.Millisecond)
	close(release)
	select {
	case err := <-done:
		if core.CodeOf(err, "") != core.ErrNotFound {
			t.Errorf("Signal error = %v, want not_found", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Signal stayed blocked after the workflow finished")
	}
}

// TestDefaultExecutor_PersistsHistory is a regression test for FWC-12: the
// final save dropped History and CreatedAt.
func TestDefaultExecutor_PersistsHistory(t *testing.T) {
	store := newMapStore()
	e := NewExecutor(WithStore(store))
	h, err := e.Execute(context.Background(), func(ctx WorkflowContext, in any) (any, error) {
		v, err := ctx.ExecuteActivity(func(_ context.Context, in any) (any, error) { return in, nil }, in)
		if err != nil {
			return nil, err
		}
		_, _ = ctx.ExecuteActivity(func(context.Context, any) (any, error) { return nil, errors.New("nope") }, "bad")
		return v, nil
	}, WorkflowOptions{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Result(context.Background()); err != nil {
		t.Fatal(err)
	}
	st, _ := store.Load(context.Background(), h.ID())
	if st == nil {
		t.Fatal("no final state saved")
	}
	if st.CreatedAt.IsZero() {
		t.Error("final state lost CreatedAt")
	}
	var types []EventType
	for _, ev := range st.History {
		types = append(types, ev.Type)
	}
	want := []EventType{EventWorkflowStarted, EventActivityStarted, EventActivityCompleted, EventActivityStarted, EventActivityFailed, EventWorkflowCompleted}
	if len(types) != len(want) {
		t.Fatalf("history = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Errorf("history[%d] = %s, want %s", i, types[i], want[i])
		}
		if st.History[i].ID != i+1 {
			t.Errorf("history[%d].ID = %d", i, st.History[i].ID)
		}
	}
}
