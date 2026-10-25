package runtime

import (
	"context"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// capturePlanner records the conversation it is given and echoes the input.
type capturePlanner struct {
	mu   sync.Mutex
	seen [][]schema.Message
}

func (p *capturePlanner) Plan(_ context.Context, s agent.PlannerState) ([]agent.Action, error) {
	p.mu.Lock()
	p.seen = append(p.seen, s.Messages)
	p.mu.Unlock()
	return []agent.Action{{Type: agent.ActionFinish, Message: "re:" + s.Input}}, nil
}

func (p *capturePlanner) Replan(ctx context.Context, s agent.PlannerState) ([]agent.Action, error) {
	return p.Plan(ctx, s)
}

func runTurn(t *testing.T, r *Runner, sessionID, text string) (string, []agent.Event) {
	t.Helper()
	var events []agent.Event
	sid := ""
	for evt, err := range r.Run(context.Background(), sessionID, schema.NewHumanMessage(text)) {
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		events = append(events, evt)
		if v, ok := evt.Metadata[MetadataSessionID].(string); ok {
			sid = v
		}
	}
	return sid, events
}

// TestRunner_SessionHistoryReachesAgent is a regression test for FWC-13(a,c):
// session turns were stored but never given to the agent, and the ID of a
// newly created session was never returned to the caller.
func TestRunner_SessionHistoryReachesAgent(t *testing.T) {
	p := &capturePlanner{}
	r := NewRunner(agent.New("a", agent.WithPlanner(p)))

	sid, events := runTurn(t, r, "", "first")
	if sid == "" {
		t.Fatal("no session ID in event metadata for a new session")
	}
	for _, evt := range events {
		if evt.Metadata[MetadataSessionID] != sid {
			t.Errorf("event %s missing session ID", evt.Type)
		}
	}
	sid2, _ := runTurn(t, r, sid, "second")
	if sid2 != sid {
		t.Fatalf("second turn ran in session %q, want %q", sid2, sid)
	}

	if len(p.seen) != 2 {
		t.Fatalf("planner calls = %d, want 2", len(p.seen))
	}
	second := p.seen[1]
	// human(first), ai(re:first), human(second)
	if len(second) != 3 {
		t.Fatalf("second turn messages = %d, want 3 (history + input)", len(second))
	}
	if got := extractMessageText(second[0]); got != "first" {
		t.Errorf("history[0] = %q, want first", got)
	}
	if got := extractMessageText(second[1]); got != "re:first" {
		t.Errorf("history[1] = %q, want re:first", got)
	}
}

// TestRunner_StreamsEventsLive is a regression test for FWC-13(b): Run used
// to buffer the whole turn before yielding the first event.
func TestRunner_StreamsEventsLive(t *testing.T) {
	release := make(chan struct{})
	a := &mockAgent{id: "a", streamFn: func(ctx context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
		return func(yield func(agent.Event, error) bool) {
			if !yield(agent.Event{Type: agent.EventText, Text: "one"}, nil) {
				return
			}
			select {
			case <-release:
			case <-ctx.Done():
				return
			}
			yield(agent.Event{Type: agent.EventText, Text: "two"}, nil)
		}
	}}
	r := NewRunner(a)
	got := make(chan string, 2)
	go func() {
		for evt, err := range r.Run(context.Background(), "", schema.NewHumanMessage("hi")) {
			if err == nil {
				got <- evt.Text
			}
		}
		close(got)
	}()
	select {
	case txt := <-got:
		if txt != "one" {
			t.Fatalf("first event = %q", txt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first event not delivered before the turn finished")
	}
	close(release)
	if txt := <-got; txt != "two" {
		t.Fatalf("second event = %q", txt)
	}
}

// TestRunner_EarlyBreakCancelsTurn checks a consumer break stops the agent.
func TestRunner_EarlyBreakCancelsTurn(t *testing.T) {
	stopped := make(chan struct{})
	a := &mockAgent{id: "a", streamFn: func(ctx context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
		return func(yield func(agent.Event, error) bool) {
			defer close(stopped)
			for {
				if !yield(agent.Event{Type: agent.EventText, Text: "x"}, nil) {
					return
				}
			}
		}
	}}
	r := NewRunner(a)
	for range r.Run(context.Background(), "", schema.NewHumanMessage("hi")) {
		break
	}
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("agent kept running after the consumer stopped")
	}
}

// TestRunner_ConcurrentTurnsKeepAllUpdates is a regression test for
// FWC-13(e): concurrent turns on one session overwrote each other's turns.
func TestRunner_ConcurrentTurnsKeepAllUpdates(t *testing.T) {
	svc := NewInMemorySessionService()
	r := NewRunner(&mockAgent{id: "a"}, WithSessionService(svc), WithWorkerPoolSize(8))
	sess, err := svc.Create(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			for _, err := range r.Run(context.Background(), sess.ID, schema.NewHumanMessage("hi")) {
				if err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	got, err := svc.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Turns) != n {
		t.Errorf("turns = %d, want %d", len(got.Turns), n)
	}
	r.locksMu.Lock()
	left := len(r.sessionLocks)
	r.locksMu.Unlock()
	if left != 0 {
		t.Errorf("session locks not released: %d", left)
	}
}

// TestInMemorySessionService_EvictsExpired is a regression test for
// FWC-13(d): expired sessions stayed in memory forever.
func TestInMemorySessionService_EvictsExpired(t *testing.T) {
	svc := NewInMemorySessionService(WithSessionTTL(time.Millisecond), WithMaxSessions(1))
	ctx := context.Background()
	s1, err := svc.Create(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	// The expired session must not block a new one under the limit.
	s2, err := svc.Create(ctx, "a")
	if err != nil {
		t.Fatalf("Create after expiry: %v", err)
	}
	svc.mu.RLock()
	_, stillThere := svc.sessions[s1.ID]
	svc.mu.RUnlock()
	if stillThere {
		t.Error("expired session not evicted on Create")
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := svc.Get(ctx, s2.ID); err == nil {
		t.Fatal("expected expired error")
	}
	svc.mu.RLock()
	n := len(svc.sessions)
	svc.mu.RUnlock()
	if n != 0 {
		t.Errorf("expired session not evicted on Get: %d left", n)
	}
}

// TestExtractMessageText_AllParts is a regression test for FWC-13(f).
func TestExtractMessageText_AllParts(t *testing.T) {
	msg := &schema.HumanMessage{Parts: []schema.ContentPart{
		schema.TextPart{Text: "a"}, schema.TextPart{Text: "b"},
	}}
	if got := extractMessageText(msg); got != "a\nb" {
		t.Errorf("extractMessageText = %q, want a\\nb", got)
	}
}
