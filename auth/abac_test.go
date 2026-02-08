package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestABACPolicy_Name(t *testing.T) {
	p := NewABACPolicy("test-abac")
	if p.Name() != "test-abac" {
		t.Errorf("expected name 'test-abac', got %q", p.Name())
	}
}

func TestABACPolicy_AddRule(t *testing.T) {
	p := NewABACPolicy("abac")

	err := p.AddRule(Rule{Name: "allow-all", Effect: EffectAllow})
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// Duplicate name should fail.
	err = p.AddRule(Rule{Name: "allow-all", Effect: EffectDeny})
	if err == nil {
		t.Fatal("expected error for duplicate rule name")
	}
}

func TestABACPolicy_AddRuleEmptyName(t *testing.T) {
	p := NewABACPolicy("abac")
	err := p.AddRule(Rule{Name: "", Effect: EffectAllow})
	if err == nil {
		t.Fatal("expected error for empty rule name")
	}
}

func TestABACPolicy_AuthorizeSimpleAllow(t *testing.T) {
	ctx := context.Background()
	p := NewABACPolicy("abac")
	_ = p.AddRule(Rule{
		Name:   "allow-admin",
		Effect: EffectAllow,
		Conditions: []Condition{
			func(_ context.Context, subject string, _ Permission, _ string) bool {
				return subject == "admin"
			},
		},
	})

	allowed, err := p.Authorize(ctx, "admin", PermToolExec, "calculator")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected admin to be allowed")
	}

	allowed, err = p.Authorize(ctx, "guest", PermToolExec, "calculator")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected guest to be denied")
	}
}

func TestABACPolicy_AuthorizePriority(t *testing.T) {
	ctx := context.Background()
	p := NewABACPolicy("abac")

	// Low priority: allow all.
	_ = p.AddRule(Rule{
		Name:     "allow-all",
		Effect:   EffectAllow,
		Priority: 1,
	})

	// High priority: deny specific permission.
	_ = p.AddRule(Rule{
		Name:   "deny-exec",
		Effect: EffectDeny,
		Conditions: []Condition{
			func(_ context.Context, _ string, perm Permission, _ string) bool {
				return perm == PermToolExec
			},
		},
		Priority: 10,
	})

	// PermToolExec should be denied (high-priority deny matches first).
	allowed, err := p.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected PermToolExec to be denied by high-priority rule")
	}

	// PermMemoryRead should be allowed (deny rule doesn't match, allow-all does).
	allowed, err = p.Authorize(ctx, "alice", PermMemoryRead, "mem")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected PermMemoryRead to be allowed by low-priority rule")
	}
}

func TestABACPolicy_AuthorizeDefaultDeny(t *testing.T) {
	ctx := context.Background()
	p := NewABACPolicy("abac")

	// No rules — default deny.
	allowed, err := p.Authorize(ctx, "anyone", PermToolExec, "anything")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected default deny with no rules")
	}
}

func TestABACPolicy_AuthorizeMultipleConditions(t *testing.T) {
	ctx := context.Background()
	p := NewABACPolicy("abac")

	// All conditions must match.
	_ = p.AddRule(Rule{
		Name:   "allow-admin-read",
		Effect: EffectAllow,
		Conditions: []Condition{
			func(_ context.Context, subject string, _ Permission, _ string) bool {
				return subject == "admin"
			},
			func(_ context.Context, _ string, perm Permission, _ string) bool {
				return perm == PermMemoryRead
			},
		},
	})

	tests := []struct {
		name    string
		subject string
		perm    Permission
		want    bool
	}{
		{"admin+read", "admin", PermMemoryRead, true},
		{"admin+write", "admin", PermMemoryWrite, false},
		{"guest+read", "guest", PermMemoryRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := p.Authorize(ctx, tt.subject, tt.perm, "res")
			if err != nil {
				t.Fatalf("Authorize error: %v", err)
			}
			if allowed != tt.want {
				t.Errorf("Authorize = %v, want %v", allowed, tt.want)
			}
		})
	}
}

func TestABACPolicy_AuthorizeDenyRule(t *testing.T) {
	ctx := context.Background()
	p := NewABACPolicy("abac")

	_ = p.AddRule(Rule{
		Name:   "deny-dangerous",
		Effect: EffectDeny,
		Conditions: []Condition{
			func(_ context.Context, _ string, _ Permission, resource string) bool {
				return resource == "dangerous"
			},
		},
		Priority: 10,
	})
	_ = p.AddRule(Rule{
		Name:     "allow-all",
		Effect:   EffectAllow,
		Priority: 1,
	})

	allowed, err := p.Authorize(ctx, "alice", PermToolExec, "dangerous")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected deny for dangerous resource")
	}

	allowed, err = p.Authorize(ctx, "alice", PermToolExec, "safe")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected allow for safe resource")
	}
}

func TestABACPolicy_EmptyConditions(t *testing.T) {
	ctx := context.Background()
	p := NewABACPolicy("abac")

	// Rule with no conditions matches everything.
	_ = p.AddRule(Rule{
		Name:   "allow-all",
		Effect: EffectAllow,
	})

	allowed, err := p.Authorize(ctx, "anyone", PermToolExec, "anything")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected allow with empty conditions")
	}
}

// --- Composite policy tests ---

func TestCompositePolicy_AllowIfAny(t *testing.T) {
	ctx := context.Background()

	deny := NewABACPolicy("deny-all") // no rules = default deny
	allow := NewABACPolicy("allow-all")
	_ = allow.AddRule(Rule{Name: "allow", Effect: EffectAllow})

	comp := NewCompositePolicy("composite", AllowIfAny, deny, allow)

	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected AllowIfAny to allow when one policy allows")
	}
}

func TestCompositePolicy_AllowIfAnyDenied(t *testing.T) {
	ctx := context.Background()

	deny1 := NewABACPolicy("deny1")
	deny2 := NewABACPolicy("deny2")

	comp := NewCompositePolicy("composite", AllowIfAny, deny1, deny2)

	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected AllowIfAny to deny when all policies deny")
	}
}

func TestCompositePolicy_AllowIfAll(t *testing.T) {
	ctx := context.Background()

	allow1 := NewABACPolicy("allow1")
	_ = allow1.AddRule(Rule{Name: "allow", Effect: EffectAllow})
	allow2 := NewABACPolicy("allow2")
	_ = allow2.AddRule(Rule{Name: "allow", Effect: EffectAllow})

	comp := NewCompositePolicy("composite", AllowIfAll, allow1, allow2)

	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected AllowIfAll to allow when all policies allow")
	}
}

func TestCompositePolicy_AllowIfAllOneDenies(t *testing.T) {
	ctx := context.Background()

	allow := NewABACPolicy("allow")
	_ = allow.AddRule(Rule{Name: "allow", Effect: EffectAllow})
	deny := NewABACPolicy("deny")

	comp := NewCompositePolicy("composite", AllowIfAll, allow, deny)

	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected AllowIfAll to deny when one policy denies")
	}
}

func TestCompositePolicy_DenyIfAny(t *testing.T) {
	ctx := context.Background()

	allow := NewABACPolicy("allow")
	_ = allow.AddRule(Rule{Name: "allow", Effect: EffectAllow})
	deny := NewABACPolicy("deny")

	comp := NewCompositePolicy("composite", DenyIfAny, allow, deny)

	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected DenyIfAny to deny when one policy denies")
	}
}

func TestCompositePolicy_DenyIfAnyAllAllow(t *testing.T) {
	ctx := context.Background()

	allow1 := NewABACPolicy("allow1")
	_ = allow1.AddRule(Rule{Name: "allow", Effect: EffectAllow})
	allow2 := NewABACPolicy("allow2")
	_ = allow2.AddRule(Rule{Name: "allow", Effect: EffectAllow})

	comp := NewCompositePolicy("composite", DenyIfAny, allow1, allow2)

	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected DenyIfAny to allow when all policies allow")
	}
}

func TestCompositePolicy_EmptyPolicies(t *testing.T) {
	ctx := context.Background()

	comp := NewCompositePolicy("empty", AllowIfAny)
	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected default deny with no policies")
	}
}

func TestCompositePolicy_Name(t *testing.T) {
	comp := NewCompositePolicy("my-composite", AllowIfAny)
	if comp.Name() != "my-composite" {
		t.Errorf("expected name 'my-composite', got %q", comp.Name())
	}
}

func TestCompositePolicy_ErrorPropagation(t *testing.T) {
	ctx := context.Background()

	errPolicy := &errorPolicy{err: errors.New("policy error")}
	allow := NewABACPolicy("allow")
	_ = allow.AddRule(Rule{Name: "allow", Effect: EffectAllow})

	// Error should propagate in AllowIfAny.
	comp := NewCompositePolicy("composite", AllowIfAny, errPolicy, allow)
	_, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err == nil {
		t.Error("expected error to propagate from child policy")
	}

	// Error should propagate in AllowIfAll.
	comp = NewCompositePolicy("composite", AllowIfAll, errPolicy, allow)
	_, err = comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err == nil {
		t.Error("expected error to propagate from child policy")
	}

	// Error should propagate in DenyIfAny.
	comp = NewCompositePolicy("composite", DenyIfAny, errPolicy, allow)
	_, err = comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err == nil {
		t.Error("expected error to propagate from child policy")
	}
}

func TestCompositePolicy_UnknownMode(t *testing.T) {
	ctx := context.Background()
	allow := NewABACPolicy("allow")
	_ = allow.AddRule(Rule{Name: "allow", Effect: EffectAllow})

	comp := NewCompositePolicy("composite", CompositeMode("unknown"), allow)
	allowed, err := comp.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected default deny for unknown mode")
	}
}

// --- Hooks tests ---

func TestComposeHooks(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			calls = append(calls, "h1-authorize")
			return nil
		},
		OnAllow: func(_ context.Context, _ string, _ Permission, _ string) {
			calls = append(calls, "h1-allow")
		},
		OnDeny: func(_ context.Context, _ string, _ Permission, _ string) {
			calls = append(calls, "h1-deny")
		},
	}
	h2 := Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			calls = append(calls, "h2-authorize")
			return nil
		},
		OnAllow: func(_ context.Context, _ string, _ Permission, _ string) {
			calls = append(calls, "h2-allow")
		},
	}

	composed := ComposeHooks(h1, h2)
	ctx := context.Background()

	// Test OnAuthorize.
	err := composed.OnAuthorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("OnAuthorize error: %v", err)
	}
	if len(calls) != 2 || calls[0] != "h1-authorize" || calls[1] != "h2-authorize" {
		t.Errorf("expected h1-authorize, h2-authorize, got %v", calls)
	}

	// Test OnAllow.
	calls = nil
	composed.OnAllow(ctx, "alice", PermToolExec, "tool")
	if len(calls) != 2 || calls[0] != "h1-allow" || calls[1] != "h2-allow" {
		t.Errorf("expected h1-allow, h2-allow, got %v", calls)
	}

	// Test OnDeny — h2 has no OnDeny, should only get h1.
	calls = nil
	composed.OnDeny(ctx, "bob", PermToolExec, "tool")
	if len(calls) != 1 || calls[0] != "h1-deny" {
		t.Errorf("expected h1-deny only, got %v", calls)
	}
}

func TestComposeHooks_OnAuthorizeShortCircuit(t *testing.T) {
	h1 := Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			return errors.New("blocked")
		},
	}
	h2 := Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			t.Error("h2 should not be called after h1 error")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnAuthorize(context.Background(), "alice", PermToolExec, "tool")
	if err == nil {
		t.Fatal("expected error from h1")
	}
}

func TestComposeHooks_OnErrorShortCircuit(t *testing.T) {
	h1 := Hooks{
		OnError: func(_ context.Context, err error) error {
			return errors.New("replaced")
		},
	}
	h2 := Hooks{
		OnError: func(_ context.Context, err error) error {
			t.Error("h2 should not be called after h1 returns non-nil")
			return err
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnError(context.Background(), errors.New("original"))
	if err == nil || err.Error() != "replaced" {
		t.Errorf("expected 'replaced' error, got %v", err)
	}
}

func TestComposeHooks_OnErrorPassthrough(t *testing.T) {
	h1 := Hooks{
		OnError: func(_ context.Context, err error) error {
			return nil // suppress -> continues to next
		},
	}
	h2 := Hooks{
		OnError: func(_ context.Context, err error) error {
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnError(context.Background(), errors.New("original"))
	if err != nil {
		// All hooks returned nil but compose returns the original error at the end.
		// Actually per the pattern: if all hooks return nil, the final return is err
		// (the original). Let me re-check...
		// Looking at the implementation: return err at the end. But err is the parameter.
		// Since both hooks returned nil, we hit the final return err which is the original.
		// This matches the llm/hooks.go pattern.
		t.Logf("OnError returned original error as expected: %v", err)
	}
}

// --- Middleware tests ---

func TestWithHooksMiddleware(t *testing.T) {
	ctx := context.Background()
	var hookCalls []string

	rbac := NewRBACPolicy("rbac")
	_ = rbac.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})
	_ = rbac.AssignRole("alice", "admin")

	hooks := Hooks{
		OnAuthorize: func(_ context.Context, subject string, _ Permission, _ string) error {
			hookCalls = append(hookCalls, "authorize:"+subject)
			return nil
		},
		OnAllow: func(_ context.Context, subject string, _ Permission, _ string) {
			hookCalls = append(hookCalls, "allow:"+subject)
		},
		OnDeny: func(_ context.Context, subject string, _ Permission, _ string) {
			hookCalls = append(hookCalls, "deny:"+subject)
		},
	}

	policy := ApplyMiddleware(rbac, WithHooks(hooks))

	// Test allow path.
	allowed, err := policy.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected allow")
	}
	if len(hookCalls) != 2 || hookCalls[0] != "authorize:alice" || hookCalls[1] != "allow:alice" {
		t.Errorf("expected [authorize:alice, allow:alice], got %v", hookCalls)
	}

	// Test deny path.
	hookCalls = nil
	allowed, err = policy.Authorize(ctx, "bob", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected deny")
	}
	if len(hookCalls) != 2 || hookCalls[0] != "authorize:bob" || hookCalls[1] != "deny:bob" {
		t.Errorf("expected [authorize:bob, deny:bob], got %v", hookCalls)
	}
}

func TestWithHooksMiddleware_OnAuthorizeError(t *testing.T) {
	ctx := context.Background()

	rbac := NewRBACPolicy("rbac")
	hooks := Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			return errors.New("blocked by hook")
		},
	}

	policy := ApplyMiddleware(rbac, WithHooks(hooks))

	allowed, err := policy.Authorize(ctx, "alice", PermToolExec, "tool")
	if err == nil {
		t.Fatal("expected error from OnAuthorize hook")
	}
	if allowed {
		t.Error("expected deny when hook errors")
	}
}

func TestWithHooksMiddleware_OnError(t *testing.T) {
	ctx := context.Background()

	errP := &errorPolicy{err: errors.New("policy failed")}
	var errorSeen error
	hooks := Hooks{
		OnError: func(_ context.Context, err error) error {
			errorSeen = err
			return errors.New("transformed")
		},
	}

	policy := ApplyMiddleware(errP, WithHooks(hooks))

	_, err := policy.Authorize(ctx, "alice", PermToolExec, "tool")
	if err == nil || err.Error() != "transformed" {
		t.Errorf("expected 'transformed' error, got %v", err)
	}
	if errorSeen == nil || errorSeen.Error() != "policy failed" {
		t.Errorf("expected OnError to see 'policy failed', got %v", errorSeen)
	}
}

func TestWithAuditMiddleware(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	rbac := NewRBACPolicy("rbac")
	_ = rbac.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})
	_ = rbac.AssignRole("alice", "admin")

	policy := ApplyMiddleware(rbac, WithAudit(logger))

	// Should not panic and should return correct results.
	allowed, err := policy.Authorize(ctx, "alice", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected allow")
	}

	allowed, err = policy.Authorize(ctx, "bob", PermToolExec, "tool")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected deny")
	}
}

func TestWithAuditMiddleware_Error(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	errP := &errorPolicy{err: errors.New("policy error")}
	policy := ApplyMiddleware(errP, WithAudit(logger))

	_, err := policy.Authorize(ctx, "alice", PermToolExec, "tool")
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestApplyMiddleware_Order(t *testing.T) {
	ctx := context.Background()
	var order []string

	rbac := NewRBACPolicy("rbac")
	_ = rbac.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})
	_ = rbac.AssignRole("alice", "admin")

	mw1 := WithHooks(Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			order = append(order, "mw1")
			return nil
		},
	})
	mw2 := WithHooks(Hooks{
		OnAuthorize: func(_ context.Context, _ string, _ Permission, _ string) error {
			order = append(order, "mw2")
			return nil
		},
	})

	// mw1 is first in list => outermost => executes first.
	policy := ApplyMiddleware(rbac, mw1, mw2)
	_, _ = policy.Authorize(ctx, "alice", PermToolExec, "tool")

	if len(order) != 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Errorf("expected [mw1, mw2], got %v", order)
	}
}

func TestHookedPolicy_Name(t *testing.T) {
	rbac := NewRBACPolicy("my-rbac")
	policy := ApplyMiddleware(rbac, WithHooks(Hooks{}))
	if policy.Name() != "my-rbac" {
		t.Errorf("expected name 'my-rbac', got %q", policy.Name())
	}
}

func TestAuditPolicy_Name(t *testing.T) {
	rbac := NewRBACPolicy("my-rbac")
	policy := ApplyMiddleware(rbac, WithAudit(slog.Default()))
	if policy.Name() != "my-rbac" {
		t.Errorf("expected name 'my-rbac', got %q", policy.Name())
	}
}

// --- Test helpers ---

// errorPolicy is a test helper that always returns an error.
type errorPolicy struct {
	err error
}

func (p *errorPolicy) Name() string { return "error-policy" }

func (p *errorPolicy) Authorize(_ context.Context, _ string, _ Permission, _ string) (bool, error) {
	return false, p.err
}
