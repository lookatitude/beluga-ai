package runtime

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
)

// ----------------------------------------------------------------------------
// Test helpers / stubs
// ----------------------------------------------------------------------------

// stubPlugin is a configurable Plugin implementation used in tests.
type stubPlugin struct {
	name         string
	beforeTurnFn func(ctx context.Context, s *Session, in schema.Message) (schema.Message, error)
	afterTurnFn  func(ctx context.Context, s *Session, evts []agent.Event) ([]agent.Event, error)
	onErrorFn    func(ctx context.Context, err error) error
}

var _ Plugin = (*stubPlugin)(nil)

func (p *stubPlugin) Name() string { return p.name }

func (p *stubPlugin) BeforeTurn(ctx context.Context, s *Session, in schema.Message) (schema.Message, error) {
	if p.beforeTurnFn != nil {
		return p.beforeTurnFn(ctx, s, in)
	}
	return in, nil
}

func (p *stubPlugin) AfterTurn(ctx context.Context, s *Session, evts []agent.Event) ([]agent.Event, error) {
	if p.afterTurnFn != nil {
		return p.afterTurnFn(ctx, s, evts)
	}
	return evts, nil
}

func (p *stubPlugin) OnError(ctx context.Context, err error) error {
	if p.onErrorFn != nil {
		return p.onErrorFn(ctx, err)
	}
	return err
}

// ----------------------------------------------------------------------------
// Plugin interface compliance
// ----------------------------------------------------------------------------

func TestPluginInterfaceCompliance(t *testing.T) {
	// compile-time check via variable assignment
	var _ Plugin = (*stubPlugin)(nil)
}

// ----------------------------------------------------------------------------
// NewPluginChain
// ----------------------------------------------------------------------------

func TestNewPluginChain_EmptyChain(t *testing.T) {
	c := NewPluginChain()
	if len(c.plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(c.plugins))
	}
}

func TestNewPluginChain_DoesNotAliasInput(t *testing.T) {
	p := &stubPlugin{name: "p1"}
	plugins := []Plugin{p}
	c := NewPluginChain(plugins...)
	// mutating the original slice should not affect the chain
	plugins[0] = &stubPlugin{name: "modified"}
	if c.plugins[0].Name() != "p1" {
		t.Fatal("plugin chain aliased caller's slice")
	}
}

// ----------------------------------------------------------------------------
// RunBeforeTurn
// ----------------------------------------------------------------------------

func TestRunBeforeTurn_EmptyChain_IsNoop(t *testing.T) {
	c := NewPluginChain()
	sess := NewSession("s1", "a1")
	input := schema.NewHumanMessage("hello")

	got, err := c.RunBeforeTurn(context.Background(), sess, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Fatal("empty chain should return input unchanged")
	}
}

func TestRunBeforeTurn_SinglePlugin_ModifiesInput(t *testing.T) {
	replacement := schema.NewHumanMessage("modified")
	p := &stubPlugin{
		name: "modifier",
		beforeTurnFn: func(_ context.Context, _ *Session, _ schema.Message) (schema.Message, error) {
			return replacement, nil
		},
	}
	c := NewPluginChain(p)
	sess := NewSession("s1", "a1")

	got, err := c.RunBeforeTurn(context.Background(), sess, schema.NewHumanMessage("original"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != replacement {
		t.Fatal("plugin should have replaced input message")
	}
}

func TestRunBeforeTurn_ChainPassesModifiedMessageToNextPlugin(t *testing.T) {
	// Each plugin appends its name to the message text.
	makePlugin := func(name string) *stubPlugin {
		return &stubPlugin{
			name: name,
			beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
				text := extractTextFromMessage(in)
				return schema.NewHumanMessage(text + "|" + name), nil
			},
		}
	}

	c := NewPluginChain(makePlugin("A"), makePlugin("B"), makePlugin("C"))
	sess := NewSession("s1", "a1")

	got, err := c.RunBeforeTurn(context.Background(), sess, schema.NewHumanMessage("start"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "start|A|B|C"
	if extractTextFromMessage(got) != want {
		t.Fatalf("got %q, want %q", extractTextFromMessage(got), want)
	}
}

func TestRunBeforeTurn_StopsOnFirstError(t *testing.T) {
	sentinel := errors.New("before-turn error")
	called := make([]string, 0)

	p1 := &stubPlugin{name: "p1", beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
		called = append(called, "p1")
		return in, nil
	}}
	p2 := &stubPlugin{name: "p2", beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
		called = append(called, "p2")
		return in, sentinel
	}}
	p3 := &stubPlugin{name: "p3", beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
		called = append(called, "p3")
		return in, nil
	}}

	c := NewPluginChain(p1, p2, p3)
	_, err := c.RunBeforeTurn(context.Background(), NewSession("s1", "a1"), schema.NewHumanMessage("hi"))

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if len(called) != 2 || called[0] != "p1" || called[1] != "p2" {
		t.Fatalf("expected [p1 p2] to be called, got %v", called)
	}
}

func TestRunBeforeTurn_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	p := &stubPlugin{
		name: "p",
		beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
			t.Fatal("plugin should not be called after context cancellation")
			return in, nil
		},
	}
	c := NewPluginChain(p)

	_, err := c.RunBeforeTurn(ctx, NewSession("s1", "a1"), schema.NewHumanMessage("hi"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// RunAfterTurn
// ----------------------------------------------------------------------------

func TestRunAfterTurn_EmptyChain_IsNoop(t *testing.T) {
	c := NewPluginChain()
	sess := NewSession("s1", "a1")
	events := []agent.Event{{Type: agent.EventText, Text: "hello"}}

	got, err := c.RunAfterTurn(context.Background(), sess, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Text != "hello" {
		t.Fatal("empty chain should return events unchanged")
	}
}

func TestRunAfterTurn_SinglePlugin_ModifiesEvents(t *testing.T) {
	extra := agent.Event{Type: agent.EventText, Text: "extra"}
	p := &stubPlugin{
		name: "appender",
		afterTurnFn: func(_ context.Context, _ *Session, evts []agent.Event) ([]agent.Event, error) {
			return append(evts, extra), nil
		},
	}
	c := NewPluginChain(p)
	sess := NewSession("s1", "a1")
	original := []agent.Event{{Type: agent.EventText, Text: "original"}}

	got, err := c.RunAfterTurn(context.Background(), sess, original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[1].Text != "extra" {
		t.Fatal("appended event not found")
	}
}

func TestRunAfterTurn_ChainPassesModifiedEventsToNextPlugin(t *testing.T) {
	makePlugin := func(name string) *stubPlugin {
		return &stubPlugin{
			name: name,
			afterTurnFn: func(_ context.Context, _ *Session, evts []agent.Event) ([]agent.Event, error) {
				return append(evts, agent.Event{Type: agent.EventText, Text: name}), nil
			},
		}
	}

	c := NewPluginChain(makePlugin("A"), makePlugin("B"))
	sess := NewSession("s1", "a1")

	got, err := c.RunAfterTurn(context.Background(), sess, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].Text != "A" || got[1].Text != "B" {
		t.Fatalf("unexpected events: %v", got)
	}
}

func TestRunAfterTurn_StopsOnFirstError(t *testing.T) {
	sentinel := errors.New("after-turn error")
	called := 0

	p1 := &stubPlugin{name: "p1", afterTurnFn: func(_ context.Context, _ *Session, evts []agent.Event) ([]agent.Event, error) {
		called++
		return evts, sentinel
	}}
	p2 := &stubPlugin{name: "p2", afterTurnFn: func(_ context.Context, _ *Session, evts []agent.Event) ([]agent.Event, error) {
		called++
		return evts, nil
	}}

	c := NewPluginChain(p1, p2)
	_, err := c.RunAfterTurn(context.Background(), NewSession("s1", "a1"), nil)

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if called != 1 {
		t.Fatalf("expected 1 plugin call, got %d", called)
	}
}

func TestRunAfterTurn_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &stubPlugin{
		name: "p",
		afterTurnFn: func(_ context.Context, _ *Session, evts []agent.Event) ([]agent.Event, error) {
			t.Fatal("plugin should not be called after context cancellation")
			return evts, nil
		},
	}
	c := NewPluginChain(p)

	_, err := c.RunAfterTurn(ctx, NewSession("s1", "a1"), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// RunOnError
// ----------------------------------------------------------------------------

func TestRunOnError_EmptyChain_ReturnsOriginalError(t *testing.T) {
	c := NewPluginChain()
	sentinel := errors.New("original")

	got := c.RunOnError(context.Background(), sentinel)
	if !errors.Is(got, sentinel) {
		t.Fatalf("expected original error, got %v", got)
	}
}

func TestRunOnError_SinglePlugin_WrapsError(t *testing.T) {
	p := &stubPlugin{
		name: "wrapper",
		onErrorFn: func(_ context.Context, err error) error {
			return fmt.Errorf("wrapped: %w", err)
		},
	}
	c := NewPluginChain(p)
	original := errors.New("original")

	got := c.RunOnError(context.Background(), original)
	if got == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(got, original) {
		t.Fatalf("expected wrapped error to contain original, got %v", got)
	}
}

func TestRunOnError_PluginCanSuppressError(t *testing.T) {
	called := make([]string, 0)

	p1 := &stubPlugin{name: "suppressor", onErrorFn: func(_ context.Context, _ error) error {
		called = append(called, "suppressor")
		return nil // suppresses the error
	}}
	p2 := &stubPlugin{name: "observer", onErrorFn: func(_ context.Context, err error) error {
		called = append(called, "observer")
		return err // passes nil through
	}}

	c := NewPluginChain(p1, p2)
	got := c.RunOnError(context.Background(), errors.New("boom"))

	if got != nil {
		t.Fatalf("expected suppressed error (nil), got %v", got)
	}
	if len(called) != 2 {
		t.Fatalf("expected both plugins called, got %v", called)
	}
}

func TestRunOnError_PropagatesInOrder(t *testing.T) {
	order := make([]string, 0)

	makePlugin := func(name string) *stubPlugin {
		return &stubPlugin{
			name: name,
			onErrorFn: func(_ context.Context, err error) error {
				order = append(order, name)
				return err
			},
		}
	}

	c := NewPluginChain(makePlugin("first"), makePlugin("second"), makePlugin("third"))
	_ = c.RunOnError(context.Background(), errors.New("err"))

	if len(order) != 3 || order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Fatalf("unexpected execution order: %v", order)
	}
}

func TestRunOnError_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &stubPlugin{
		name: "p",
		onErrorFn: func(_ context.Context, err error) error {
			t.Fatal("plugin should not be called after context cancellation")
			return err
		},
	}
	c := NewPluginChain(p)

	got := c.RunOnError(ctx, errors.New("original"))
	if !errors.Is(got, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", got)
	}
}

// ----------------------------------------------------------------------------
// Table-driven integration: full turn lifecycle
// ----------------------------------------------------------------------------

func TestPluginChain_TableDriven(t *testing.T) {
	type testCase struct {
		name        string
		plugins     []Plugin
		inputText   string
		wantText    string
		inputEvents []agent.Event
		wantEvents  int
		injectErr   error
		wantErr     bool
	}

	tests := []testCase{
		{
			name:        "no plugins — passthrough",
			plugins:     nil,
			inputText:   "hello",
			wantText:    "hello",
			inputEvents: []agent.Event{{Type: agent.EventText, Text: "response"}},
			wantEvents:  1,
		},
		{
			name: "single modifier plugin",
			plugins: []Plugin{
				&stubPlugin{
					name: "upper",
					beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
						return schema.NewHumanMessage("UPPER"), nil
					},
					afterTurnFn: func(_ context.Context, _ *Session, evts []agent.Event) ([]agent.Event, error) {
						return append(evts, agent.Event{Type: agent.EventDone}), nil
					},
				},
			},
			inputText:   "lower",
			wantText:    "UPPER",
			inputEvents: []agent.Event{{Type: agent.EventText}},
			wantEvents:  2,
		},
		{
			name: "error in BeforeTurn stops chain",
			plugins: []Plugin{
				&stubPlugin{
					name: "failer",
					beforeTurnFn: func(_ context.Context, _ *Session, in schema.Message) (schema.Message, error) {
						return in, errors.New("before-turn-fail")
					},
				},
			},
			inputText: "hi",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewPluginChain(tc.plugins...)
			sess := NewSession("s", "a")

			gotMsg, err := c.RunBeforeTurn(context.Background(), sess, schema.NewHumanMessage(tc.inputText))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantText != "" && extractTextFromMessage(gotMsg) != tc.wantText {
				t.Fatalf("BeforeTurn: got %q, want %q", extractTextFromMessage(gotMsg), tc.wantText)
			}

			gotEvts, err := c.RunAfterTurn(context.Background(), sess, tc.inputEvents)
			if err != nil {
				t.Fatalf("AfterTurn unexpected error: %v", err)
			}
			if len(gotEvts) != tc.wantEvents {
				t.Fatalf("AfterTurn: got %d events, want %d", len(gotEvts), tc.wantEvents)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

// extractTextFromMessage extracts the text content from a schema.Message.
func extractTextFromMessage(m schema.Message) string {
	for _, p := range m.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
