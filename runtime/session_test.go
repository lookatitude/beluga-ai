package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// newTestService is a helper that returns a fresh InMemorySessionService.
func newTestService(opts ...SessionOption) *InMemorySessionService {
	return NewInMemorySessionService(opts...)
}

// assertCoreError asserts that err is a *core.Error with the expected code.
func assertCoreError(t *testing.T, err error, code core.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", code)
	}
	var ce *core.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *core.Error, got %T: %v", err, err)
	}
	if ce.Code != code {
		t.Fatalf("expected error code %q, got %q", code, ce.Code)
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_HappyPath(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	sess, err := svc.Create(ctx, "agent-1")
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	if sess.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if sess.AgentID != "agent-1" {
		t.Errorf("expected AgentID %q, got %q", "agent-1", sess.AgentID)
	}
	if sess.State == nil {
		t.Error("expected non-nil State map")
	}
	if sess.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if sess.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestCreate_UniqueIDs(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	const n = 100
	ids := make(map[string]struct{}, n)
	for range n {
		sess, err := svc.Create(ctx, "agent-1")
		if err != nil {
			t.Fatalf("Create error: %v", err)
		}
		if _, dup := ids[sess.ID]; dup {
			t.Fatalf("duplicate session ID generated: %s", sess.ID)
		}
		ids[sess.ID] = struct{}{}
	}
}

func TestCreate_WithTTL(t *testing.T) {
	svc := newTestService(WithSessionTTL(time.Hour))
	ctx := context.Background()

	sess, err := svc.Create(ctx, "agent-1")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if sess.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt when TTL is set")
	}
}

func TestCreate_WithTenantID(t *testing.T) {
	svc := newTestService(WithSessionTenantID("tenant-42"))
	ctx := context.Background()

	sess, err := svc.Create(ctx, "agent-1")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if sess.TenantID != "tenant-42" {
		t.Errorf("expected TenantID %q, got %q", "tenant-42", sess.TenantID)
	}
}

func TestCreate_CancelledContext(t *testing.T) {
	svc := newTestService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Create(ctx, "agent-1")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet_HappyPath(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, "agent-1")

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, got.ID)
	}
	if got.AgentID != created.AgentID {
		t.Errorf("expected AgentID %q, got %q", created.AgentID, got.AgentID)
	}
}

func TestGet_NonExistent(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Get(ctx, "does-not-exist")
	assertCoreError(t, err, core.ErrNotFound)
}

func TestGet_ExpiredSession(t *testing.T) {
	// Use a very short TTL and then wait for it to pass.
	svc := newTestService(WithSessionTTL(time.Millisecond))
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "agent-1")
	time.Sleep(5 * time.Millisecond)

	_, err := svc.Get(ctx, sess.ID)
	assertCoreError(t, err, core.ErrNotFound)
}

func TestGet_CancelledContext(t *testing.T) {
	svc := newTestService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Get(ctx, "any-id")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestGet_ReturnsCopy(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, "agent-1")
	got, _ := svc.Get(ctx, created.ID)

	// Mutating the returned copy must not affect stored state.
	got.State["key"] = "value"
	got2, _ := svc.Get(ctx, created.ID)
	if _, ok := got2.State["key"]; ok {
		t.Error("mutation of returned session propagated into store")
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUpdate_HappyPath(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "agent-1")
	sess.State["foo"] = "bar"

	if err := svc.Update(ctx, sess); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	got, _ := svc.Get(ctx, sess.ID)
	if got.State["foo"] != "bar" {
		t.Errorf("expected State[foo]=bar, got %v", got.State["foo"])
	}
}

func TestUpdate_UpdatesTimestamp(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "agent-1")
	originalUpdatedAt := sess.UpdatedAt

	time.Sleep(time.Millisecond) // ensure clock advances
	_ = svc.Update(ctx, sess)

	got, _ := svc.Get(ctx, sess.ID)
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to advance after Update")
	}
}

func TestUpdate_AppendsTurns(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "agent-1")
	sess.Turns = append(sess.Turns, schema.Turn{
		Timestamp: time.Now(),
	})

	if err := svc.Update(ctx, sess); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	got, _ := svc.Get(ctx, sess.ID)
	if len(got.Turns) != 1 {
		t.Errorf("expected 1 turn, got %d", len(got.Turns))
	}
}

func TestUpdate_NonExistent(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	err := svc.Update(ctx, &Session{ID: "ghost"})
	assertCoreError(t, err, core.ErrNotFound)
}

func TestUpdate_NilSession(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	err := svc.Update(ctx, nil)
	assertCoreError(t, err, core.ErrInvalidInput)
}

func TestUpdate_CancelledContext(t *testing.T) {
	svc := newTestService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.Update(ctx, &Session{ID: "any"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete_HappyPath(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "agent-1")
	if err := svc.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	_, err := svc.Get(ctx, sess.ID)
	assertCoreError(t, err, core.ErrNotFound)
}

func TestDelete_NonExistent(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	err := svc.Delete(ctx, "ghost")
	assertCoreError(t, err, core.ErrNotFound)
}

func TestDelete_CancelledContext(t *testing.T) {
	svc := newTestService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.Delete(ctx, "any")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety (run with -race)
// ---------------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// Pre-create a session to act as the update/delete target.
	base, _ := svc.Create(ctx, "agent-base")

	const workers = 50
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func(i int) {
			defer wg.Done()
			switch i % 4 {
			case 0:
				svc.Create(ctx, "agent-concurrent") //nolint:errcheck
			case 1:
				svc.Get(ctx, base.ID) //nolint:errcheck
			case 2:
				cp := *base
				cp.State = map[string]any{"i": i}
				svc.Update(ctx, &cp) //nolint:errcheck
			case 3:
				// Create then immediately delete.
				s, err := svc.Create(ctx, "agent-del")
				if err == nil {
					svc.Delete(ctx, s.ID) //nolint:errcheck
				}
			}
		}(i)
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Table-driven: CRUD round-trip
// ---------------------------------------------------------------------------

func TestCRUDTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
		stateKV map[string]any
	}{
		{name: "empty state", agentID: "agent-a", stateKV: nil},
		{name: "with state", agentID: "agent-b", stateKV: map[string]any{"x": 1, "y": "hello"}},
		{name: "empty agentID", agentID: "", stateKV: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService()
			ctx := context.Background()

			// Create
			sess, err := svc.Create(ctx, tt.agentID)
			if err != nil {
				t.Fatalf("Create error: %v", err)
			}

			// Populate state
			for k, v := range tt.stateKV {
				sess.State[k] = v
			}

			// Update
			if err := svc.Update(ctx, sess); err != nil {
				t.Fatalf("Update error: %v", err)
			}

			// Get
			got, err := svc.Get(ctx, sess.ID)
			if err != nil {
				t.Fatalf("Get error: %v", err)
			}
			for k, want := range tt.stateKV {
				if got.State[k] != want {
					t.Errorf("State[%q]: want %v, got %v", k, want, got.State[k])
				}
			}

			// Delete
			if err := svc.Delete(ctx, sess.ID); err != nil {
				t.Fatalf("Delete error: %v", err)
			}

			// Verify gone
			_, err = svc.Get(ctx, sess.ID)
			assertCoreError(t, err, core.ErrNotFound)
		})
	}
}
