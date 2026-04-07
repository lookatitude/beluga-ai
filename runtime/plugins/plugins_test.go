package plugins_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/audit"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/cost"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/runtime/plugins"
	"github.com/lookatitude/beluga-ai/schema"
)

// ---- helpers ----------------------------------------------------------------

func newSession() *runtime.Session {
	return runtime.NewSession("agent-1", "session-1")
}

func textMsg(s string) schema.Message {
	return schema.NewHumanMessage(s)
}

func emptyEvents() []agent.Event {
	return []agent.Event{}
}

// ---- mock audit store -------------------------------------------------------

type mockAuditStore struct {
	mu      sync.Mutex
	entries []audit.Entry
	err     error
}

func (m *mockAuditStore) Log(_ context.Context, e audit.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.entries = append(m.entries, e)
	return nil
}

func (m *mockAuditStore) Actions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.entries))
	for i, e := range m.entries {
		out[i] = e.Action
	}
	return out
}

// ---- RetryAndReflect --------------------------------------------------------

func TestRetryAndReflect_Name(t *testing.T) {
	p := plugins.NewRetryAndReflect(3)
	if got := p.Name(); got != "retry_reflect" {
		t.Errorf("Name() = %q, want %q", got, "retry_reflect")
	}
}

func TestRetryAndReflect_BeforeTurn_PassThrough(t *testing.T) {
	p := plugins.NewRetryAndReflect(3)
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hello")

	got, err := p.BeforeTurn(ctx, sess, msg)
	if err != nil {
		t.Fatalf("BeforeTurn() error = %v", err)
	}
	if got != msg {
		t.Error("BeforeTurn() should return the same message unchanged")
	}
}

func TestRetryAndReflect_AfterTurn_PassThrough(t *testing.T) {
	p := plugins.NewRetryAndReflect(3)
	ctx := context.Background()
	sess := newSession()
	evs := emptyEvents()

	got, err := p.AfterTurn(ctx, sess, evs)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}
	if len(got) != 0 {
		t.Error("AfterTurn() should return events unchanged")
	}
}

func TestRetryAndReflect_OnError(t *testing.T) {
	retryableErr := core.NewError("op", core.ErrRateLimit, "rate limited", nil)
	permanentErr := core.NewError("op", core.ErrInvalidInput, "bad input", nil)
	genericErr := errors.New("something went wrong")

	tests := []struct {
		name        string
		maxRetries  int
		errs        []error // sequence of OnError calls
		wantNilLast bool    // last call should return nil (retry allowed)
		wantSameErr bool    // last call should return the original error
	}{
		{
			name:        "retryable within budget returns nil",
			maxRetries:  2,
			errs:        []error{retryableErr},
			wantNilLast: true,
		},
		{
			name:        "retryable exhausted returns error",
			maxRetries:  1,
			errs:        []error{retryableErr, retryableErr},
			wantNilLast: false,
			wantSameErr: true,
		},
		{
			name:        "non-retryable always returns error",
			maxRetries:  5,
			errs:        []error{permanentErr},
			wantNilLast: false,
			wantSameErr: true,
		},
		{
			name:        "generic error not retried",
			maxRetries:  5,
			errs:        []error{genericErr},
			wantNilLast: false,
			wantSameErr: true,
		},
		{
			name:        "zero maxRetries never retries",
			maxRetries:  0,
			errs:        []error{retryableErr},
			wantNilLast: false,
			wantSameErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := plugins.NewRetryAndReflect(tt.maxRetries)
			ctx := context.Background()

			var lastErr error
			for _, err := range tt.errs {
				lastErr = p.OnError(ctx, err)
			}

			if tt.wantNilLast && lastErr != nil {
				t.Errorf("OnError() = %v, want nil (retry allowed)", lastErr)
			}
			if tt.wantSameErr && lastErr == nil {
				t.Error("OnError() = nil, want non-nil error")
			}
		})
	}
}

// ---- AuditPlugin ------------------------------------------------------------

func TestAuditPlugin_Name(t *testing.T) {
	store := &mockAuditStore{}
	p := plugins.NewAuditPlugin(store)
	if got := p.Name(); got != "audit" {
		t.Errorf("Name() = %q, want %q", got, "audit")
	}
}

func TestAuditPlugin_BeforeTurn_LogsStart(t *testing.T) {
	store := &mockAuditStore{}
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	got, err := p.BeforeTurn(ctx, sess, msg)
	if err != nil {
		t.Fatalf("BeforeTurn() error = %v", err)
	}
	if got != msg {
		t.Error("BeforeTurn() should return message unchanged")
	}

	actions := store.Actions()
	if len(actions) != 1 || actions[0] != "agent.turn.start" {
		t.Errorf("expected [agent.turn.start], got %v", actions)
	}
}

func TestAuditPlugin_AfterTurn_LogsEnd(t *testing.T) {
	store := &mockAuditStore{}
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	sess := newSession()
	evs := emptyEvents()

	got, err := p.AfterTurn(ctx, sess, evs)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}
	if len(got) != len(evs) {
		t.Error("AfterTurn() should return events unchanged")
	}

	actions := store.Actions()
	if len(actions) != 1 || actions[0] != "agent.turn.end" {
		t.Errorf("expected [agent.turn.end], got %v", actions)
	}
}

func TestAuditPlugin_OnError_LogsError(t *testing.T) {
	store := &mockAuditStore{}
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	origErr := errors.New("boom")

	returned := p.OnError(ctx, origErr)
	if returned != origErr {
		t.Error("OnError() should return the original error unchanged")
	}

	actions := store.Actions()
	if len(actions) != 1 || actions[0] != "agent.turn.error" {
		t.Errorf("expected [agent.turn.error], got %v", actions)
	}
}

func TestAuditPlugin_StoreError_DoesNotAbortTurn(t *testing.T) {
	store := &mockAuditStore{err: errors.New("store down")}
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	_, err := p.BeforeTurn(ctx, sess, msg)
	if err != nil {
		t.Errorf("BeforeTurn() should not propagate store errors, got %v", err)
	}
}

// ---- CostTracking -----------------------------------------------------------

func TestCostTracking_Name(t *testing.T) {
	p := plugins.NewCostTracking(cost.Budget{})
	if got := p.Name(); got != "cost_tracking" {
		t.Errorf("Name() = %q, want %q", got, "cost_tracking")
	}
}

func TestCostTracking_BeforeTurn_PassThrough(t *testing.T) {
	p := plugins.NewCostTracking(cost.Budget{})
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	got, err := p.BeforeTurn(ctx, sess, msg)
	if err != nil {
		t.Fatalf("BeforeTurn() error = %v", err)
	}
	if got != msg {
		t.Error("BeforeTurn() should return message unchanged")
	}
}

func TestCostTracking_AfterTurn_RecordsSchemaUsage(t *testing.T) {
	tracker := cost.NewInMemoryTracker(cost.Budget{})
	p := plugins.NewCostTracking(cost.Budget{}, plugins.WithTracker(tracker))
	ctx := context.Background()
	sess := newSession()

	evs := []agent.Event{
		{
			Type: agent.EventDone,
			Metadata: map[string]any{
				"usage": schema.Usage{
					InputTokens:  10,
					OutputTokens: 20,
					TotalTokens:  30,
				},
			},
		},
	}

	got, err := p.AfterTurn(ctx, sess, evs)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}
	if len(got) != len(evs) {
		t.Error("AfterTurn() should return events unchanged")
	}

	total, err := tracker.Total(ctx)
	if err != nil {
		t.Fatalf("tracker.Total() error = %v", err)
	}
	if total.TotalTokens != 30 {
		t.Errorf("tracker total tokens = %d, want 30", total.TotalTokens)
	}
}

func TestCostTracking_AfterTurn_RecordsCostUsage(t *testing.T) {
	tracker := cost.NewInMemoryTracker(cost.Budget{})
	p := plugins.NewCostTracking(cost.Budget{}, plugins.WithTracker(tracker))
	ctx := context.Background()
	sess := newSession()

	evs := []agent.Event{
		{
			Type: agent.EventDone,
			Metadata: map[string]any{
				"usage": cost.Usage{
					InputTokens:  5,
					OutputTokens: 15,
					TotalTokens:  20,
				},
			},
		},
	}

	_, err := p.AfterTurn(ctx, sess, evs)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}

	total, _ := tracker.Total(ctx)
	if total.TotalTokens != 20 {
		t.Errorf("tracker total tokens = %d, want 20", total.TotalTokens)
	}
}

func TestCostTracking_AfterTurn_BudgetExhaustedReturnsError(t *testing.T) {
	budget := cost.Budget{MaxTotalTokens: 5}
	p := plugins.NewCostTracking(budget)
	ctx := context.Background()
	sess := newSession()

	evs := []agent.Event{
		{
			Type: agent.EventDone,
			Metadata: map[string]any{
				"usage": schema.Usage{TotalTokens: 10},
			},
		},
	}

	_, err := p.AfterTurn(ctx, sess, evs)
	if err == nil {
		t.Fatal("AfterTurn() expected error when budget is exhausted")
	}
	var ce *core.Error
	if !errors.As(err, &ce) || ce.Code != core.ErrBudgetExhausted {
		t.Errorf("expected ErrBudgetExhausted, got %v", err)
	}
}

func TestCostTracking_AfterTurn_SkipsEventsWithoutUsage(t *testing.T) {
	tracker := cost.NewInMemoryTracker(cost.Budget{})
	p := plugins.NewCostTracking(cost.Budget{}, plugins.WithTracker(tracker))
	ctx := context.Background()
	sess := newSession()

	evs := []agent.Event{
		{Type: agent.EventText, Text: "hello"},
	}

	_, err := p.AfterTurn(ctx, sess, evs)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}

	total, _ := tracker.Total(ctx)
	if total.TotalTokens != 0 {
		t.Errorf("expected 0 tokens, got %d", total.TotalTokens)
	}
}

// ---- RateLimit --------------------------------------------------------------

func TestRateLimit_Name(t *testing.T) {
	p := plugins.NewRateLimit()
	if got := p.Name(); got != "rate_limit" {
		t.Errorf("Name() = %q, want %q", got, "rate_limit")
	}
}

func TestRateLimit_AllowsRequestsWithinBurst(t *testing.T) {
	p := plugins.NewRateLimit(
		plugins.WithRequestsPerMinute(60),
		plugins.WithBurstSize(5),
	)
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	for i := 0; i < 5; i++ {
		_, err := p.BeforeTurn(ctx, sess, msg)
		if err != nil {
			t.Errorf("request %d: unexpected error %v", i+1, err)
		}
	}
}

func TestRateLimit_BlocksWhenBucketEmpty(t *testing.T) {
	p := plugins.NewRateLimit(
		plugins.WithRequestsPerMinute(60),
		plugins.WithBurstSize(2),
	)
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	// Drain the bucket.
	for i := 0; i < 2; i++ {
		if _, err := p.BeforeTurn(ctx, sess, msg); err != nil {
			t.Fatalf("request %d: unexpected error %v", i+1, err)
		}
	}

	// Next request must be rejected.
	_, err := p.BeforeTurn(ctx, sess, msg)
	if err == nil {
		t.Fatal("BeforeTurn() expected rate limit error, got nil")
	}
	var ce *core.Error
	if !errors.As(err, &ce) || ce.Code != core.ErrRateLimit {
		t.Errorf("expected ErrRateLimit, got %v", err)
	}
}

func TestRateLimit_RefillsOverTime(t *testing.T) {
	// 1 request per minute = 1 token per minute refill rate
	// Burst of 1 so first request drains it.
	p := plugins.NewRateLimit(
		plugins.WithRequestsPerMinute(600),
		plugins.WithBurstSize(1),
	)
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	// Drain.
	if _, err := p.BeforeTurn(ctx, sess, msg); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	// Should be blocked immediately.
	if _, err := p.BeforeTurn(ctx, sess, msg); err == nil {
		t.Fatal("second request should be blocked")
	}

	// Wait for refill (600 req/min = 10 req/sec; 1 token takes ~100ms).
	time.Sleep(120 * time.Millisecond)

	if _, err := p.BeforeTurn(ctx, sess, msg); err != nil {
		t.Errorf("after refill, request should succeed: %v", err)
	}
}

func TestRateLimit_AfterTurn_PassThrough(t *testing.T) {
	p := plugins.NewRateLimit()
	ctx := context.Background()
	sess := newSession()
	evs := emptyEvents()

	got, err := p.AfterTurn(ctx, sess, evs)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}
	if len(got) != 0 {
		t.Error("AfterTurn() should return events unchanged")
	}
}

func TestRateLimit_OnError_PassThrough(t *testing.T) {
	p := plugins.NewRateLimit()
	ctx := context.Background()
	origErr := errors.New("boom")

	if got := p.OnError(ctx, origErr); got != origErr {
		t.Error("OnError() should return the original error unchanged")
	}
}

func TestRateLimit_DefaultOptions(t *testing.T) {
	// Default: 60 rpm, burst = 60; should allow 60 back-to-back requests.
	p := plugins.NewRateLimit()
	ctx := context.Background()
	sess := newSession()
	msg := textMsg("hi")

	for i := 0; i < 60; i++ {
		if _, err := p.BeforeTurn(ctx, sess, msg); err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
	}

	// 61st should be blocked.
	if _, err := p.BeforeTurn(ctx, sess, msg); err == nil {
		t.Error("61st request should be rate limited")
	}
}
