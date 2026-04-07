package plugins_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/audit"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/cost"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/runtime/plugins"
	"github.com/lookatitude/beluga-ai/schema"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newSession() *runtime.Session {
	return runtime.NewSession("sess-1", "agent-1")
}

func humanMsg(text string) schema.Message {
	return schema.NewHumanMessage(text)
}

func someEvents() []agent.Event {
	return []agent.Event{{Type: agent.EventText, Text: "hello"}}
}

// retryableErr returns a retryable core.Error.
func retryableErr() error {
	return core.NewError("test.op", core.ErrRateLimit, "rate limited", nil)
}

// nonRetryableErr returns a non-retryable core.Error.
func nonRetryableErr() error {
	return core.NewError("test.op", core.ErrInvalidInput, "bad input", nil)
}

// ---------------------------------------------------------------------------
// RetryAndReflect
// ---------------------------------------------------------------------------

func TestRetryAndReflect_Name(t *testing.T) {
	p := plugins.NewRetryAndReflect(3)
	assert.Equal(t, "retry_reflect", p.Name())
}

func TestRetryAndReflect_BeforeAfterTurnNoOp(t *testing.T) {
	p := plugins.NewRetryAndReflect(3)
	ctx := context.Background()
	sess := newSession()

	msg, err := p.BeforeTurn(ctx, sess, humanMsg("hi"))
	require.NoError(t, err)
	assert.Equal(t, "human", string(msg.GetRole()))

	evts, err := p.AfterTurn(ctx, sess, someEvents())
	require.NoError(t, err)
	assert.Len(t, evts, 1)
}

func TestRetryAndReflect_OnError(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		errors     []error // sequence of errors fed to OnError
		wantNil    []bool  // whether each call should return nil
	}{
		{
			name:       "suppresses retryable errors within budget",
			maxRetries: 2,
			errors:     []error{retryableErr(), retryableErr(), retryableErr()},
			wantNil:    []bool{true, true, false},
		},
		{
			name:       "never suppresses non-retryable errors",
			maxRetries: 5,
			errors:     []error{nonRetryableErr(), nonRetryableErr()},
			wantNil:    []bool{false, false},
		},
		{
			name:       "nil error passthrough",
			maxRetries: 1,
			errors:     []error{nil},
			wantNil:    []bool{true},
		},
		{
			name:       "zero max retries never suppresses",
			maxRetries: 0,
			errors:     []error{retryableErr()},
			wantNil:    []bool{false},
		},
		{
			name:       "non-retryable not suppressed even below budget",
			maxRetries: 10,
			errors:     []error{nonRetryableErr()},
			wantNil:    []bool{false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := plugins.NewRetryAndReflect(tt.maxRetries)
			ctx := context.Background()
			for i, err := range tt.errors {
				got := p.OnError(ctx, err)
				if tt.wantNil[i] {
					assert.NoError(t, got, "call %d expected nil", i)
				} else {
					assert.Error(t, got, "call %d expected non-nil", i)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AuditPlugin
// ---------------------------------------------------------------------------

func TestAuditPlugin_Name(t *testing.T) {
	store := audit.NewInMemoryStore()
	p := plugins.NewAuditPlugin(store)
	assert.Equal(t, "audit", p.Name())
}

func TestAuditPlugin_BeforeTurn_LogsStart(t *testing.T) {
	store := audit.NewInMemoryStore()
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	sess := newSession()
	sess.TenantID = "tenant-42"

	msg, err := p.BeforeTurn(ctx, sess, humanMsg("hello"))
	require.NoError(t, err)
	assert.Equal(t, "human", string(msg.GetRole()))

	entries, err := store.Query(ctx, audit.Filter{Action: "agent.turn.start"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "agent.turn.start", e.Action)
	assert.Equal(t, "sess-1", e.SessionID)
	assert.Equal(t, "agent-1", e.AgentID)
	assert.Equal(t, "tenant-42", e.TenantID)
	assert.False(t, e.Timestamp.IsZero())
}

func TestAuditPlugin_AfterTurn_LogsEnd(t *testing.T) {
	store := audit.NewInMemoryStore()
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	sess := newSession()

	evts, err := p.AfterTurn(ctx, sess, someEvents())
	require.NoError(t, err)
	assert.Len(t, evts, 1)

	entries, err := store.Query(ctx, audit.Filter{Action: "agent.turn.end"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "agent.turn.end", entries[0].Action)
}

func TestAuditPlugin_OnError_LogsError(t *testing.T) {
	store := audit.NewInMemoryStore()
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()
	origErr := errors.New("something broke")

	returned := p.OnError(ctx, origErr)
	assert.Equal(t, origErr, returned, "error must be returned unchanged")

	entries, err := store.Query(ctx, audit.Filter{Action: "agent.turn.error"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "agent.turn.error", entries[0].Action)
	assert.Equal(t, "something broke", entries[0].Error)
}

func TestAuditPlugin_OnError_NilIsPassthrough(t *testing.T) {
	store := audit.NewInMemoryStore()
	p := plugins.NewAuditPlugin(store)
	ctx := context.Background()

	returned := p.OnError(ctx, nil)
	assert.NoError(t, returned)

	entries, err := store.Query(ctx, audit.Filter{Action: "agent.turn.error"})
	require.NoError(t, err)
	assert.Empty(t, entries, "nil error must not produce an audit entry")
}

// ---------------------------------------------------------------------------
// CostTracking
// ---------------------------------------------------------------------------

func TestCostTracking_Name(t *testing.T) {
	tracker := cost.NewInMemoryTracker()
	p := plugins.NewCostTracking(tracker, cost.Budget{})
	assert.Equal(t, "cost_tracking", p.Name())
}

func TestCostTracking_BeforeTurnNoOp(t *testing.T) {
	tracker := cost.NewInMemoryTracker()
	p := plugins.NewCostTracking(tracker, cost.Budget{})
	ctx := context.Background()

	msg, err := p.BeforeTurn(ctx, newSession(), humanMsg("hi"))
	require.NoError(t, err)
	assert.Equal(t, "human", string(msg.GetRole()))
}

func TestCostTracking_AfterTurn_RecordsUsage(t *testing.T) {
	tracker := cost.NewInMemoryTracker()
	p := plugins.NewCostTracking(tracker, cost.Budget{})
	ctx := context.Background()
	sess := newSession()
	sess.TenantID = "tenant-99"

	evts, err := p.AfterTurn(ctx, sess, someEvents())
	require.NoError(t, err)
	assert.Len(t, evts, 1)

	summary, err := tracker.Query(ctx, cost.Filter{TenantID: "tenant-99"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.EntryCount)
}

func TestCostTracking_AfterTurn_MultipleTurns(t *testing.T) {
	tracker := cost.NewInMemoryTracker()
	p := plugins.NewCostTracking(tracker, cost.Budget{})
	ctx := context.Background()
	sess := newSession()

	for i := 0; i < 5; i++ {
		_, err := p.AfterTurn(ctx, sess, someEvents())
		require.NoError(t, err)
	}

	summary, err := tracker.Query(ctx, cost.Filter{})
	require.NoError(t, err)
	assert.EqualValues(t, 5, summary.EntryCount)
}

func TestCostTracking_OnErrorNoOp(t *testing.T) {
	tracker := cost.NewInMemoryTracker()
	p := plugins.NewCostTracking(tracker, cost.Budget{})
	ctx := context.Background()
	orig := errors.New("oops")
	assert.Equal(t, orig, p.OnError(ctx, orig))
	assert.NoError(t, p.OnError(ctx, nil))
}

// ---------------------------------------------------------------------------
// RateLimit
// ---------------------------------------------------------------------------

func TestRateLimit_Name(t *testing.T) {
	p := plugins.NewRateLimit(60)
	assert.Equal(t, "rate_limit", p.Name())
}

func TestRateLimit_AfterTurnAndOnErrorNoOp(t *testing.T) {
	p := plugins.NewRateLimit(60)
	ctx := context.Background()
	sess := newSession()

	evts, err := p.AfterTurn(ctx, sess, someEvents())
	require.NoError(t, err)
	assert.Len(t, evts, 1)

	orig := errors.New("some err")
	assert.Equal(t, orig, p.OnError(ctx, orig))
}

func TestRateLimit_BeforeTurn_AllowsWithinBurst(t *testing.T) {
	// 60 req/min → burst of 60. The first 60 calls must succeed.
	p := plugins.NewRateLimit(60)
	ctx := context.Background()
	sess := newSession()

	for i := 0; i < 60; i++ {
		_, err := p.BeforeTurn(ctx, sess, humanMsg("hi"))
		require.NoError(t, err, "call %d should be allowed", i)
	}
}

func TestRateLimit_BeforeTurn_BlocksWhenExceeded(t *testing.T) {
	// 1 req/min → burst of 1. The second call must be rejected.
	p := plugins.NewRateLimit(1)
	ctx := context.Background()
	sess := newSession()

	_, err := p.BeforeTurn(ctx, sess, humanMsg("first"))
	require.NoError(t, err)

	_, err = p.BeforeTurn(ctx, sess, humanMsg("second"))
	require.Error(t, err)

	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr), "error must be a *core.Error")
	assert.Equal(t, core.ErrRateLimit, coreErr.Code)
}

func TestRateLimit_BeforeTurn_RateLimitErrorIsRetryable(t *testing.T) {
	p := plugins.NewRateLimit(1)
	ctx := context.Background()
	sess := newSession()

	// Exhaust the burst.
	_, _ = p.BeforeTurn(ctx, sess, humanMsg("first"))
	_, err := p.BeforeTurn(ctx, sess, humanMsg("second"))
	require.Error(t, err)
	assert.True(t, core.IsRetryable(err), "rate limit error must be retryable")
}

func TestRateLimit_BeforeTurn_RefillsOverTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping time-dependent test in short mode")
	}
	// 120 req/min = 2 req/s, burst 120. Exhaust burst then wait 1 s.
	p := plugins.NewRateLimit(120)
	ctx := context.Background()
	sess := newSession()

	for i := 0; i < 120; i++ {
		_, _ = p.BeforeTurn(ctx, sess, humanMsg("burst"))
	}
	// After ~1 second, at least 2 tokens should have refilled.
	time.Sleep(1100 * time.Millisecond)
	for i := 0; i < 2; i++ {
		_, err := p.BeforeTurn(ctx, sess, humanMsg("after refill"))
		assert.NoError(t, err, "call %d after refill should be allowed", i)
	}
}
