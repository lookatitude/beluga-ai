package runtime

import (
	"context"
	"errors"
	"iter"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// --- Mock Agent ---

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id       string
	streamFn func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error]
}

var _ agent.Agent = (*mockAgent)(nil)

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	var result string
	for evt, err := range m.Stream(ctx, input, opts...) {
		if err != nil {
			return "", err
		}
		if evt.Type == agent.EventText {
			result += evt.Text
		}
	}
	return result, nil
}
func (m *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, input, opts...)
	}
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "hello", AgentID: m.id}, nil)
	}
}

// --- Mock Plugin ---

type mockPlugin struct {
	name          string
	beforeTurnFn  func(ctx context.Context, s *Session, msg schema.Message) (schema.Message, error)
	afterTurnFn   func(ctx context.Context, s *Session, events []agent.Event) ([]agent.Event, error)
	onErrorFn     func(ctx context.Context, err error) error
	beforeCalled  atomic.Int32
	afterCalled   atomic.Int32
	onErrorCalled atomic.Int32
}

func (p *mockPlugin) Name() string { return p.name }

func (p *mockPlugin) BeforeTurn(ctx context.Context, s *Session, msg schema.Message) (schema.Message, error) {
	p.beforeCalled.Add(1)
	if p.beforeTurnFn != nil {
		return p.beforeTurnFn(ctx, s, msg)
	}
	return msg, nil
}

func (p *mockPlugin) AfterTurn(ctx context.Context, s *Session, events []agent.Event) ([]agent.Event, error) {
	p.afterCalled.Add(1)
	if p.afterTurnFn != nil {
		return p.afterTurnFn(ctx, s, events)
	}
	return events, nil
}

func (p *mockPlugin) OnError(ctx context.Context, err error) error {
	p.onErrorCalled.Add(1)
	if p.onErrorFn != nil {
		return p.onErrorFn(ctx, err)
	}
	return err
}

func TestNewRunner(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		a := &mockAgent{id: "test-agent"}
		r := NewRunner(a)

		if r.agent != a {
			t.Error("agent not set")
		}
		if r.sessionService == nil {
			t.Error("session service should default to InMemorySessionService")
		}
		if r.pluginChain == nil {
			t.Error("plugin chain should be initialized")
		}
		if r.workerPool == nil {
			t.Error("worker pool should be initialized")
		}
		if r.config.WorkerPoolSize != defaultWorkerPoolSize {
			t.Errorf("worker pool size = %d, want %d", r.config.WorkerPoolSize, defaultWorkerPoolSize)
		}
	})

	t.Run("with options", func(t *testing.T) {
		a := &mockAgent{id: "test-agent"}
		svc := NewInMemorySessionService()
		p := &mockPlugin{name: "test"}

		r := NewRunner(a,
			WithSessionService(svc),
			WithPlugins(p),
			WithWorkerPoolSize(5),
		)

		if r.sessionService != svc {
			t.Error("session service not set by option")
		}
		if r.config.WorkerPoolSize != 5 {
			t.Errorf("worker pool size = %d, want 5", r.config.WorkerPoolSize)
		}
	})

	t.Run("with runner config", func(t *testing.T) {
		a := &mockAgent{id: "test-agent"}
		cfg := RunnerConfig{
			WorkerPoolSize:          3,
			SessionTTL:              5 * time.Minute,
			GracefulShutdownTimeout: 10 * time.Second,
		}
		r := NewRunner(a, WithRunnerConfig(cfg))

		if r.config.WorkerPoolSize != 3 {
			t.Errorf("worker pool size = %d, want 3", r.config.WorkerPoolSize)
		}
		if r.config.SessionTTL != 5*time.Minute {
			t.Errorf("session TTL = %v, want 5m", r.config.SessionTTL)
		}
	})
}

func TestRunnerRun(t *testing.T) {
	tests := []struct {
		name       string
		agent      *mockAgent
		plugins    []Plugin
		input      schema.Message
		sessionID  string
		wantEvents int
		wantErr    bool
	}{
		{
			name: "happy path with single text event",
			agent: &mockAgent{
				id: "agent-1",
				streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
					return func(yield func(agent.Event, error) bool) {
						yield(agent.Event{Type: agent.EventText, Text: "hello world", AgentID: "agent-1"}, nil)
					}
				},
			},
			input:      schema.NewHumanMessage("hi"),
			sessionID:  "",
			wantEvents: 1,
		},
		{
			name: "multiple events",
			agent: &mockAgent{
				id: "agent-2",
				streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
					return func(yield func(agent.Event, error) bool) {
						if !yield(agent.Event{Type: agent.EventText, Text: "part1", AgentID: "agent-2"}, nil) {
							return
						}
						if !yield(agent.Event{Type: agent.EventText, Text: "part2", AgentID: "agent-2"}, nil) {
							return
						}
						yield(agent.Event{Type: agent.EventDone, AgentID: "agent-2"}, nil)
					}
				},
			},
			input:      schema.NewHumanMessage("hello"),
			sessionID:  "",
			wantEvents: 3,
		},
		{
			name: "agent stream error",
			agent: &mockAgent{
				id: "agent-err",
				streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
					return func(yield func(agent.Event, error) bool) {
						yield(agent.Event{}, errors.New("agent failed"))
					}
				},
			},
			input:   schema.NewHumanMessage("hi"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRunner(tt.agent, WithWorkerPoolSize(2))

			ctx := context.Background()
			var events []agent.Event
			var gotErr error

			for evt, err := range r.Run(ctx, tt.sessionID, tt.input) {
				if err != nil {
					gotErr = err
					break
				}
				events = append(events, evt)
			}

			if tt.wantErr {
				if gotErr == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if gotErr != nil {
				t.Fatalf("unexpected error: %v", gotErr)
			}

			if len(events) != tt.wantEvents {
				t.Errorf("got %d events, want %d", len(events), tt.wantEvents)
			}
		})
	}
}

func TestRunnerPluginChainFires(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: input, AgentID: "agent-1"}, nil)
			}
		},
	}

	p := &mockPlugin{name: "test-plugin"}

	r := NewRunner(a, WithPlugins(p), WithWorkerPoolSize(2))

	ctx := context.Background()
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if p.beforeCalled.Load() != 1 {
		t.Errorf("BeforeTurn called %d times, want 1", p.beforeCalled.Load())
	}
	if p.afterCalled.Load() != 1 {
		t.Errorf("AfterTurn called %d times, want 1", p.afterCalled.Load())
	}
}

func TestRunnerPluginModifiesInput(t *testing.T) {
	var capturedInput string
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			capturedInput = input
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "ok", AgentID: "agent-1"}, nil)
			}
		},
	}

	p := &mockPlugin{
		name: "modifier",
		beforeTurnFn: func(ctx context.Context, s *Session, msg schema.Message) (schema.Message, error) {
			return schema.NewHumanMessage("modified input"), nil
		},
	}

	r := NewRunner(a, WithPlugins(p), WithWorkerPoolSize(2))

	ctx := context.Background()
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("original")) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if capturedInput != "modified input" {
		t.Errorf("agent received %q, want %q", capturedInput, "modified input")
	}
}

func TestRunnerPluginModifiesOutput(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "original", AgentID: "agent-1"}, nil)
			}
		},
	}

	p := &mockPlugin{
		name: "modifier",
		afterTurnFn: func(ctx context.Context, s *Session, events []agent.Event) ([]agent.Event, error) {
			// Replace all text events.
			modified := make([]agent.Event, len(events))
			for i, evt := range events {
				if evt.Type == agent.EventText {
					evt.Text = "modified"
				}
				modified[i] = evt
			}
			return modified, nil
		},
	}

	r := NewRunner(a, WithPlugins(p), WithWorkerPoolSize(2))

	ctx := context.Background()
	var events []agent.Event
	for evt, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, evt)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Text != "modified" {
		t.Errorf("event text = %q, want %q", events[0].Text, "modified")
	}
}

func TestRunnerSessionCreatedAndLoaded(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "ok", AgentID: "agent-1"}, nil)
			}
		},
	}

	svc := NewInMemorySessionService()
	r := NewRunner(a, WithSessionService(svc), WithWorkerPoolSize(2))

	// First call creates a session (empty sessionID).
	ctx := context.Background()
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Create a known session and run with its ID.
	session, err := svc.Create(ctx, "agent-1")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	for _, err := range r.Run(ctx, session.ID, schema.NewHumanMessage("hello again")) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// The session should have a turn recorded.
	updated, err := svc.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if len(updated.Turns) != 1 {
		t.Errorf("session has %d turns, want 1", len(updated.Turns))
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	a := &mockAgent{
		id: "agent-slow",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				// Simulate a slow agent that checks context.
				select {
				case <-ctx.Done():
					yield(agent.Event{}, ctx.Err())
					return
				case <-time.After(5 * time.Second):
					yield(agent.Event{Type: agent.EventText, Text: "done", AgentID: "agent-slow"}, nil)
				}
			}
		},
	}

	r := NewRunner(a, WithWorkerPoolSize(2))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var gotErr error
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Error("expected context cancellation error, got nil")
	}
}

func TestRunnerShutdown(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "ok", AgentID: "agent-1"}, nil)
			}
		},
	}

	r := NewRunner(a, WithWorkerPoolSize(2))

	ctx := context.Background()
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// After shutdown, Run should return an error.
	var gotErr error
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Error("expected error after shutdown, got nil")
	}
}

func TestRunnerWorkerPoolLimitsConcurrency(t *testing.T) {
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				cur := concurrent.Add(1)
				// Track the maximum concurrency seen.
				for {
					max := maxConcurrent.Load()
					if cur <= max || maxConcurrent.CompareAndSwap(max, cur) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				concurrent.Add(-1)
				yield(agent.Event{Type: agent.EventText, Text: "ok", AgentID: "agent-1"}, nil)
			}
		},
	}

	poolSize := 2
	r := NewRunner(a, WithWorkerPoolSize(poolSize))

	ctx := context.Background()

	// Launch more goroutines than the pool size.
	done := make(chan struct{})
	numTasks := 6
	var completed atomic.Int32

	for i := 0; i < numTasks; i++ {
		go func() {
			for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
				if err != nil {
					break
				}
			}
			if completed.Add(1) == int32(numTasks) {
				close(done)
			}
		}()
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for all tasks to complete")
	}

	if maxConcurrent.Load() > int32(poolSize) {
		t.Errorf("max concurrent = %d, exceeds pool size %d", maxConcurrent.Load(), poolSize)
	}
}

func TestRunnerBeforeTurnError(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "should not reach", AgentID: "agent-1"}, nil)
			}
		},
	}

	p := &mockPlugin{
		name: "failing-before",
		beforeTurnFn: func(ctx context.Context, s *Session, msg schema.Message) (schema.Message, error) {
			return nil, errors.New("before turn failed")
		},
	}

	r := NewRunner(a, WithPlugins(p), WithWorkerPoolSize(2))

	ctx := context.Background()
	var gotErr error
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Error("expected error from BeforeTurn plugin, got nil")
	}

	// OnError should also have been called.
	if p.onErrorCalled.Load() != 1 {
		t.Errorf("OnError called %d times, want 1", p.onErrorCalled.Load())
	}
}

func TestRunnerAfterTurnError(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "ok", AgentID: "agent-1"}, nil)
			}
		},
	}

	p := &mockPlugin{
		name: "failing-after",
		afterTurnFn: func(ctx context.Context, s *Session, events []agent.Event) ([]agent.Event, error) {
			return nil, errors.New("after turn failed")
		},
	}

	r := NewRunner(a, WithPlugins(p), WithWorkerPoolSize(2))

	ctx := context.Background()
	var gotErr error
	for _, err := range r.Run(ctx, "", schema.NewHumanMessage("hi")) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Error("expected error from AfterTurn plugin, got nil")
	}
}

func TestRunnerNilInput(t *testing.T) {
	a := &mockAgent{
		id: "agent-1",
		streamFn: func(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "ok", AgentID: "agent-1"}, nil)
			}
		},
	}

	r := NewRunner(a, WithWorkerPoolSize(2))

	ctx := context.Background()
	// Passing nil message should not panic.
	var gotErr error
	for _, err := range r.Run(ctx, "", nil) {
		if err != nil {
			gotErr = err
			break
		}
	}

	// Should succeed — nil message is valid (empty input).
	if gotErr != nil {
		t.Fatalf("unexpected error with nil input: %v", gotErr)
	}
}
