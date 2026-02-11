package hitl

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestApplyMiddleware(t *testing.T) {
	var order []string

	mw1 := func(next Manager) Manager {
		return &testMiddleware{next: next, name: "mw1", order: &order}
	}
	mw2 := func(next Manager) Manager {
		return &testMiddleware{next: next, name: "mw2", order: &order}
	}

	base := NewManager(WithTimeout(50 * time.Millisecond))
	wrapped := ApplyMiddleware(base, mw1, mw2)

	// mw1 is outermost, mw2 is inner (right-to-left application)
	wrapped.ShouldApprove(context.Background(), "test", 0.5, RiskReadOnly)

	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}
	if order[0] != "mw1" {
		t.Errorf("expected mw1 first, got %s", order[0])
	}
	if order[1] != "mw2" {
		t.Errorf("expected mw2 second, got %s", order[1])
	}
}

type testMiddleware struct {
	next  Manager
	name  string
	order *[]string
}

func (m *testMiddleware) RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error) {
	*m.order = append(*m.order, m.name)
	return m.next.RequestInteraction(ctx, req)
}

func (m *testMiddleware) AddPolicy(policy ApprovalPolicy) error {
	return m.next.AddPolicy(policy)
}

func (m *testMiddleware) ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error) {
	*m.order = append(*m.order, m.name)
	return m.next.ShouldApprove(ctx, toolName, confidence, risk)
}

func (m *testMiddleware) Respond(ctx context.Context, requestID string, resp InteractionResponse) error {
	return m.next.Respond(ctx, requestID, resp)
}

func TestWithHooks_Approve(t *testing.T) {
	var approved bool

	hooks := Hooks{
		OnApprove: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
			approved = true
		},
	}

	base := NewManager(WithTimeout(5 * time.Second))
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	req := InteractionRequest{
		ID:       "hook-approve",
		Type:     TypeApproval,
		ToolName: "test",
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		wrapped.Respond(context.Background(), "hook-approve", InteractionResponse{
			Decision: DecisionApprove,
		})
	}()

	resp, err := wrapped.RequestInteraction(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestInteraction: %v", err)
	}

	if resp.Decision != DecisionApprove {
		t.Errorf("expected approve, got %s", resp.Decision)
	}
	if !approved {
		t.Error("expected OnApprove hook to be called")
	}
}

func TestWithHooks_DelegateShouldApprove(t *testing.T) {
	base := NewManager()
	base.AddPolicy(ApprovalPolicy{
		ToolPattern:    "delete_*",
		RequireExplicit: true,
	})

	wrapped := ApplyMiddleware(base, WithHooks(Hooks{}))

	need, err := wrapped.ShouldApprove(context.Background(), "delete_user", 0.5, RiskReadOnly)
	if err != nil {
		t.Fatalf("ShouldApprove: %v", err)
	}
	if need {
		t.Error("expected false (RequireExplicit = true means no auto-approve)")
	}
}

func TestWithHooks_DelegateAddPolicy(t *testing.T) {
	base := NewManager()
	wrapped := ApplyMiddleware(base, WithHooks(Hooks{}))

	if err := wrapped.AddPolicy(ApprovalPolicy{ToolPattern: "test_*", MinConfidence: 0.5, MaxRiskLevel: RiskReadOnly}); err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}

	// test_foo with enough confidence and low risk should auto-approve.
	approved, err := wrapped.ShouldApprove(context.Background(), "test_foo", 0.9, RiskReadOnly)
	if err != nil {
		t.Fatalf("ShouldApprove: %v", err)
	}
	if !approved {
		t.Error("expected auto-approve")
	}
}

func TestWithHooks_Reject(t *testing.T) {
	var rejected bool

	hooks := Hooks{
		OnReject: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
			rejected = true
		},
	}

	base := NewManager(WithTimeout(5 * time.Second))
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	req := InteractionRequest{
		ID:       "hook-reject",
		Type:     TypeApproval,
		ToolName: "test",
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		wrapped.Respond(context.Background(), "hook-reject", InteractionResponse{
			Decision: DecisionReject,
		})
	}()

	resp, err := wrapped.RequestInteraction(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestInteraction: %v", err)
	}
	if resp.Decision != DecisionReject {
		t.Errorf("expected reject, got %s", resp.Decision)
	}
	if !rejected {
		t.Error("expected OnReject hook to be called")
	}
}

func TestWithHooks_OnRequestError(t *testing.T) {
	hooks := Hooks{
		OnRequest: func(_ context.Context, _ InteractionRequest) error {
			return fmt.Errorf("blocked by middleware hook")
		},
	}

	base := NewManager(WithTimeout(5 * time.Second))
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	resp, err := wrapped.RequestInteraction(context.Background(), InteractionRequest{
		ID:       "mw-blocked",
		ToolName: "test",
	})
	if err == nil {
		t.Fatal("expected error from OnRequest hook")
	}
	if resp != nil {
		t.Error("expected nil response for error")
	}
}

func TestWithHooks_OnErrorSuppress(t *testing.T) {
	hooks := Hooks{
		OnError: func(_ context.Context, err error) error {
			return nil // suppress the error
		},
	}

	base := NewManager(WithTimeout(50 * time.Millisecond))
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	// This will timeout, and OnError will suppress the error.
	resp, err := wrapped.RequestInteraction(context.Background(), InteractionRequest{
		ID:       "mw-suppress",
		ToolName: "test",
	})

	// Error should be suppressed (OnError returns nil).
	if err != nil {
		t.Fatalf("expected no error (suppressed), got %v", err)
	}
	// resp may be nil when error was suppressed.
	_ = resp
}

func TestWithHooks_OnErrorReplace(t *testing.T) {
	hooks := Hooks{
		OnError: func(_ context.Context, err error) error {
			return fmt.Errorf("replaced: %w", err)
		},
	}

	base := NewManager(WithTimeout(50 * time.Millisecond))
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	_, err := wrapped.RequestInteraction(context.Background(), InteractionRequest{
		ID:       "mw-replace",
		ToolName: "test",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "replaced") {
		t.Errorf("expected replaced error, got %v", err)
	}
}

func TestWithHooks_DelegateRespond(t *testing.T) {
	base := NewManager(WithTimeout(5 * time.Second))
	wrapped := ApplyMiddleware(base, WithHooks(Hooks{}))

	// Respond to nonexistent request should fail.
	err := wrapped.Respond(context.Background(), "nonexistent", InteractionResponse{Decision: DecisionApprove})
	if err == nil {
		t.Fatal("expected error for nonexistent request")
	}
}

func TestWithHooks_AutoApprovePassThrough(t *testing.T) {
	var approveCount int
	hooks := Hooks{
		OnApprove: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
			approveCount++
		},
	}

	base := NewManager()
	base.AddPolicy(ApprovalPolicy{
		ToolPattern:   "*",
		MinConfidence: 0.0,
		MaxRiskLevel:  RiskIrreversible,
	})

	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	// This will auto-approve from the base manager.
	// The hookedManager should see the approve decision and call OnApprove.
	resp, err := wrapped.RequestInteraction(context.Background(), InteractionRequest{
		ToolName:   "any",
		Confidence: 0.5,
		RiskLevel:  RiskReadOnly,
	})
	if err != nil {
		t.Fatalf("RequestInteraction: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Decision != DecisionApprove {
		t.Errorf("expected approve, got %s", resp.Decision)
	}
	if approveCount != 1 {
		t.Errorf("expected OnApprove called once, got %d", approveCount)
	}
}

// contains is a helper to check if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
