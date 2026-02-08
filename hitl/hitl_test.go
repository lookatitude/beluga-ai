package hitl

import (
	"context"
	"testing"
	"time"
)

func TestInteractionTypes(t *testing.T) {
	tests := []struct {
		it   InteractionType
		want string
	}{
		{TypeApproval, "approval"},
		{TypeFeedback, "feedback"},
		{TypeInput, "input"},
		{TypeAnnotation, "annotation"},
	}
	for _, tt := range tests {
		if string(tt.it) != tt.want {
			t.Errorf("InteractionType %v: got %q, want %q", tt.it, string(tt.it), tt.want)
		}
	}
}

func TestDecisionTypes(t *testing.T) {
	tests := []struct {
		d    Decision
		want string
	}{
		{DecisionApprove, "approve"},
		{DecisionReject, "reject"},
		{DecisionModify, "modify"},
	}
	for _, tt := range tests {
		if string(tt.d) != tt.want {
			t.Errorf("Decision %v: got %q, want %q", tt.d, string(tt.d), tt.want)
		}
	}
}

func TestRiskLevels(t *testing.T) {
	tests := []struct {
		r    RiskLevel
		want string
	}{
		{RiskReadOnly, "read_only"},
		{RiskDataModification, "data_modification"},
		{RiskIrreversible, "irreversible"},
	}
	for _, tt := range tests {
		if string(tt.r) != tt.want {
			t.Errorf("RiskLevel %v: got %q, want %q", tt.r, string(tt.r), tt.want)
		}
	}
}

func TestRiskOrder(t *testing.T) {
	if riskOrder[RiskReadOnly] >= riskOrder[RiskDataModification] {
		t.Error("read_only should be less than data_modification")
	}
	if riskOrder[RiskDataModification] >= riskOrder[RiskIrreversible] {
		t.Error("data_modification should be less than irreversible")
	}
}

func TestRegistry_Default(t *testing.T) {
	// "default" is registered in init()
	names := List()
	found := false
	for _, n := range names {
		if n == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'default' in registry from init()")
	}

	mgr, err := New("default", Config{DefaultTimeout: time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestRegistry_Unknown(t *testing.T) {
	_, err := New("nonexistent-hitl-provider", Config{})
	if err == nil {
		t.Fatal("expected error for nonexistent manager")
	}
}

func TestManager_AddPolicy(t *testing.T) {
	mgr := NewManager()

	err := mgr.AddPolicy(ApprovalPolicy{
		ToolPattern:   "delete_*",
		MinConfidence: 0.9,
		MaxRiskLevel:  RiskDataModification,
	})
	if err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}

	// Empty pattern should fail.
	if err := mgr.AddPolicy(ApprovalPolicy{ToolPattern: ""}); err == nil {
		t.Fatal("expected error for empty pattern")
	}

	// Invalid glob pattern should fail.
	if err := mgr.AddPolicy(ApprovalPolicy{ToolPattern: "[invalid"}); err == nil {
		t.Fatal("expected error for invalid pattern")
	}
}

func TestManager_ShouldApprove(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	// Auto-approve read_only tools with confidence >= 0.5
	mgr.AddPolicy(ApprovalPolicy{
		ToolPattern:   "read_*",
		MinConfidence: 0.5,
		MaxRiskLevel:  RiskReadOnly,
	})
	// Auto-approve update tools with confidence >= 0.8 and up to data_modification risk
	mgr.AddPolicy(ApprovalPolicy{
		ToolPattern:   "update_*",
		MinConfidence: 0.8,
		MaxRiskLevel:  RiskDataModification,
	})
	// Never auto-approve delete tools
	mgr.AddPolicy(ApprovalPolicy{
		ToolPattern:    "delete_*",
		RequireExplicit: true,
	})

	tests := []struct {
		name       string
		tool       string
		confidence float64
		risk       RiskLevel
		want       bool // true = auto-approve, false = needs human
	}{
		{"read auto-approved", "read_user", 0.7, RiskReadOnly, true},
		{"read low confidence", "read_user", 0.3, RiskReadOnly, false},
		{"read too high risk", "read_user", 0.9, RiskDataModification, false},
		{"update auto-approved", "update_user", 0.9, RiskDataModification, true},
		{"update auto-approved at threshold", "update_user", 0.8, RiskReadOnly, true},
		{"update low confidence", "update_user", 0.5, RiskReadOnly, false},
		{"update too high risk", "update_user", 0.9, RiskIrreversible, false},
		{"delete always needs approval", "delete_user", 1.0, RiskReadOnly, false},
		{"no matching policy", "create_user", 0.5, RiskReadOnly, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mgr.ShouldApprove(ctx, tt.tool, tt.confidence, tt.risk)
			if err != nil {
				t.Fatalf("ShouldApprove: %v", err)
			}
			if got != tt.want {
				t.Errorf("ShouldApprove(%q, %f, %s) = %v, want %v", tt.tool, tt.confidence, tt.risk, got, tt.want)
			}
		})
	}
}

func TestManager_RequestAndRespond(t *testing.T) {
	mgr := NewManager(WithTimeout(5 * time.Second))

	// No policies, so no auto-approval → will need human response.
	req := InteractionRequest{
		ID:       "test-req-1",
		Type:     TypeApproval,
		ToolName: "delete_user",
		Input:    map[string]any{"user_id": "123"},
	}

	done := make(chan struct{})
	var resp *InteractionResponse
	var reqErr error

	go func() {
		resp, reqErr = mgr.RequestInteraction(context.Background(), req)
		close(done)
	}()

	// Wait for the request to be registered.
	time.Sleep(50 * time.Millisecond)

	err := mgr.Respond(context.Background(), "test-req-1", InteractionResponse{
		Decision: DecisionApprove,
		Feedback: "looks good",
	})
	if err != nil {
		t.Fatalf("Respond: %v", err)
	}

	<-done
	if reqErr != nil {
		t.Fatalf("RequestInteraction: %v", reqErr)
	}
	if resp.Decision != DecisionApprove {
		t.Errorf("expected approve, got %s", resp.Decision)
	}
	if resp.Feedback != "looks good" {
		t.Errorf("expected 'looks good', got %q", resp.Feedback)
	}
}

func TestManager_RequestAutoApprove(t *testing.T) {
	mgr := NewManager()
	mgr.AddPolicy(ApprovalPolicy{
		ToolPattern:   "safe_*",
		MinConfidence: 0.5,
		MaxRiskLevel:  RiskReadOnly,
	})

	req := InteractionRequest{
		Type:       TypeApproval,
		ToolName:   "safe_read",
		Confidence: 0.9,
		RiskLevel:  RiskReadOnly,
	}

	resp, err := mgr.RequestInteraction(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestInteraction: %v", err)
	}
	if resp.Decision != DecisionApprove {
		t.Errorf("expected auto-approve, got %s", resp.Decision)
	}
}

func TestManager_RequestTimeout(t *testing.T) {
	mgr := NewManager(WithTimeout(50 * time.Millisecond))

	req := InteractionRequest{
		ID:       "timeout-req",
		Type:     TypeApproval,
		ToolName: "delete_user",
	}

	_, err := mgr.RequestInteraction(context.Background(), req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestManager_RequestPerRequestTimeout(t *testing.T) {
	mgr := NewManager(WithTimeout(10 * time.Second))

	req := InteractionRequest{
		ID:       "per-req-timeout",
		Type:     TypeApproval,
		ToolName: "delete_user",
		Timeout:  50 * time.Millisecond,
	}

	_, err := mgr.RequestInteraction(context.Background(), req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestManager_RequestContextCancel(t *testing.T) {
	mgr := NewManager(WithTimeout(10 * time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := InteractionRequest{
		ID:       "cancel-req",
		Type:     TypeApproval,
		ToolName: "delete_user",
	}

	_, err := mgr.RequestInteraction(ctx, req)
	if err == nil {
		t.Fatal("expected context cancel error")
	}
}

func TestManager_RespondNotFound(t *testing.T) {
	mgr := NewManager()
	err := mgr.Respond(context.Background(), "nonexistent", InteractionResponse{Decision: DecisionApprove})
	if err == nil {
		t.Fatal("expected error for nonexistent request")
	}
}

func TestManager_WithNotifier(t *testing.T) {
	var notified bool
	notifier := &testNotifier{fn: func(_ context.Context, _ InteractionRequest) error {
		notified = true
		return nil
	}}

	mgr := NewManager(
		WithNotifier(notifier),
		WithTimeout(50*time.Millisecond),
	)

	req := InteractionRequest{
		ID:       "notifier-req",
		Type:     TypeApproval,
		ToolName: "test_tool",
	}
	// Will timeout, but the notifier should be called.
	mgr.RequestInteraction(context.Background(), req)

	if !notified {
		t.Error("expected notifier to be called")
	}
}

func TestManager_GenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestManager_RejectResponse(t *testing.T) {
	var rejected bool
	mgr := NewManager(
		WithTimeout(5*time.Second),
		WithManagerHooks(Hooks{
			OnReject: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
				rejected = true
			},
		}),
	)

	req := InteractionRequest{
		ID:       "reject-req",
		Type:     TypeApproval,
		ToolName: "delete_user",
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		mgr.Respond(context.Background(), "reject-req", InteractionResponse{
			Decision: DecisionReject,
			Feedback: "too risky",
		})
	}()

	resp, err := mgr.RequestInteraction(context.Background(), req)
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

type testNotifier struct {
	fn func(ctx context.Context, req InteractionRequest) error
}

func (n *testNotifier) Notify(ctx context.Context, req InteractionRequest) error {
	return n.fn(ctx, req)
}
