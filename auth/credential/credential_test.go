package credential

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/auth"
	"github.com/lookatitude/beluga-ai/core"
)

// --- AgentCredential tests ---

func TestAgentCredential_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			want:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &AgentCredential{ExpiresAt: tt.expiresAt}
			if got := cred.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgentCredential_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		revoked   bool
		want      bool
	}{
		{
			name:      "valid",
			expiresAt: time.Now().Add(time.Hour),
			revoked:   false,
			want:      true,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			revoked:   false,
			want:      false,
		},
		{
			name:      "revoked",
			expiresAt: time.Now().Add(time.Hour),
			revoked:   true,
			want:      false,
		},
		{
			name:      "expired and revoked",
			expiresAt: time.Now().Add(-time.Hour),
			revoked:   true,
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &AgentCredential{ExpiresAt: tt.expiresAt, Revoked: tt.revoked}
			if got := cred.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgentCredential_HasPermission(t *testing.T) {
	cred := &AgentCredential{Permissions: []string{"tool:execute", "memory:read"}}

	if !cred.HasPermission("tool:execute") {
		t.Error("HasPermission(tool:execute) = false, want true")
	}
	if !cred.HasPermission("memory:read") {
		t.Error("HasPermission(memory:read) = false, want true")
	}
	if cred.HasPermission("memory:write") {
		t.Error("HasPermission(memory:write) = true, want false")
	}
}

// --- Context tests ---

func TestContext_RoundTrip(t *testing.T) {
	cred := &AgentCredential{ID: "test-123", AgentID: "agent-1"}
	ctx := WithCredential(context.Background(), cred)

	got := CredentialFromContext(ctx)
	if got == nil {
		t.Fatal("CredentialFromContext returned nil")
	}
	if got.ID != "test-123" {
		t.Errorf("ID = %q, want %q", got.ID, "test-123")
	}
}

func TestContext_Missing(t *testing.T) {
	got := CredentialFromContext(context.Background())
	if got != nil {
		t.Errorf("CredentialFromContext returned %v, want nil", got)
	}
}

// --- InMemoryIssuer tests ---

func TestInMemoryIssuer_Issue(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer(WithDefaultTTL(10 * time.Minute))

	cred, err := iss.Issue(ctx, "agent-1", []string{"tool:execute"}, 0)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if cred.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", cred.AgentID, "agent-1")
	}
	if len(cred.Permissions) != 1 || cred.Permissions[0] != "tool:execute" {
		t.Errorf("Permissions = %v, want [tool:execute]", cred.Permissions)
	}
	if cred.IsExpired() {
		t.Error("newly issued credential should not be expired")
	}
	if cred.Revoked {
		t.Error("newly issued credential should not be revoked")
	}
	if cred.ID == "" {
		t.Error("credential ID should not be empty")
	}

	// Verify it's retrievable.
	got, err := iss.Get(ctx, cred.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != cred.ID {
		t.Errorf("Get returned ID = %q, want %q", got.ID, cred.ID)
	}
}

func TestInMemoryIssuer_Issue_CustomTTL(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	cred, err := iss.Issue(ctx, "agent-1", []string{"tool:execute"}, 30*time.Second)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	// ExpiresAt should be roughly 30s from now.
	if time.Until(cred.ExpiresAt) > 31*time.Second || time.Until(cred.ExpiresAt) < 29*time.Second {
		t.Errorf("ExpiresAt too far from expected: %v", cred.ExpiresAt)
	}
}

func TestInMemoryIssuer_Issue_EmptyAgentID(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	_, err := iss.Issue(ctx, "", []string{"tool:execute"}, 0)
	if err == nil {
		t.Fatal("Issue() with empty agent ID should return error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("error code = %v, want %v", err, core.ErrInvalidInput)
	}
}

func TestInMemoryIssuer_Issue_EmptyPermissions(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	_, err := iss.Issue(ctx, "agent-1", nil, 0)
	if err == nil {
		t.Fatal("Issue() with nil permissions should return error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("error code = %v, want %v", err, core.ErrInvalidInput)
	}
}

func TestInMemoryIssuer_Issue_MaxCredentials(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer(WithMaxCredentials(2))

	_, err := iss.Issue(ctx, "a1", []string{"p"}, time.Hour)
	if err != nil {
		t.Fatalf("first Issue() error = %v", err)
	}
	_, err = iss.Issue(ctx, "a2", []string{"p"}, time.Hour)
	if err != nil {
		t.Fatalf("second Issue() error = %v", err)
	}
	_, err = iss.Issue(ctx, "a3", []string{"p"}, time.Hour)
	if err == nil {
		t.Fatal("third Issue() should fail when max reached")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrBudgetExhausted {
		t.Errorf("error code = %v, want %v", err, core.ErrBudgetExhausted)
	}
}

func TestInMemoryIssuer_Revoke(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	cred, err := iss.Issue(ctx, "agent-1", []string{"p"}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if err := iss.Revoke(ctx, cred.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	got, err := iss.Get(ctx, cred.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !got.Revoked {
		t.Error("credential should be revoked")
	}
}

func TestInMemoryIssuer_Revoke_NotFound(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	err := iss.Revoke(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Revoke() of nonexistent should return error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrNotFound {
		t.Errorf("error code = %v, want %v", err, core.ErrNotFound)
	}
}

func TestInMemoryIssuer_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	_, err := iss.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Get() of nonexistent should return error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrNotFound {
		t.Errorf("error code = %v, want %v", err, core.ErrNotFound)
	}
}

func TestInMemoryIssuer_PermissionsCopy(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	perms := []string{"a", "b"}
	cred, err := iss.Issue(ctx, "agent-1", perms, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	// Mutate original slice; credential should be unaffected.
	perms[0] = "mutated"
	if cred.Permissions[0] != "a" {
		t.Error("credential permissions were mutated via original slice")
	}
}

func TestInMemoryIssuer_Concurrent(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer(WithMaxCredentials(1000))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = iss.Issue(ctx, "agent", []string{"p"}, time.Hour)
		}()
	}
	wg.Wait()

	if iss.Count() != 100 {
		t.Errorf("Count() = %d, want 100", iss.Count())
	}
}

func TestInMemoryIssuer_Expired(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	// Issue one expired credential (negative TTL trick: issue then manually set)
	cred, err := iss.Issue(ctx, "agent-1", []string{"p"}, time.Millisecond)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	// Force expiry.
	iss.mu.Lock()
	iss.store[cred.ID].ExpiresAt = time.Now().Add(-time.Second)
	iss.mu.Unlock()

	// Issue one valid credential.
	_, err = iss.Issue(ctx, "agent-2", []string{"p"}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	expired := iss.Expired()
	if len(expired) != 1 {
		t.Errorf("Expired() returned %d credentials, want 1", len(expired))
	}
	if len(expired) > 0 && expired[0].ID != cred.ID {
		t.Errorf("expired credential ID = %q, want %q", expired[0].ID, cred.ID)
	}
}

// --- AutoRevoker tests ---

func TestAutoRevoker_Lifecycle(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()
	revoker := NewAutoRevoker(iss, WithScanInterval(10*time.Millisecond))

	// Health before start.
	h := revoker.Health()
	if h.Status != core.HealthUnhealthy {
		t.Errorf("Health before start = %v, want unhealthy", h.Status)
	}

	if err := revoker.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Health after start.
	h = revoker.Health()
	if h.Status != core.HealthHealthy {
		t.Errorf("Health after start = %v, want healthy", h.Status)
	}

	// Double start should fail.
	if err := revoker.Start(ctx); err == nil {
		t.Error("double Start() should return error")
	}

	if err := revoker.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Double stop should be safe.
	if err := revoker.Stop(ctx); err != nil {
		t.Fatalf("double Stop() error = %v", err)
	}
}

func TestAutoRevoker_RevokesExpired(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	// Issue a credential and force it to be expired.
	cred, err := iss.Issue(ctx, "agent-1", []string{"p"}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	iss.mu.Lock()
	iss.store[cred.ID].ExpiresAt = time.Now().Add(-time.Second)
	iss.mu.Unlock()

	var revoked int
	hooks := RevokerHooks{
		OnScanComplete: func(_ context.Context, n int) {
			revoked += n
		},
	}

	revoker := NewAutoRevoker(iss,
		WithScanInterval(10*time.Millisecond),
		WithRevokerHooks(hooks),
	)

	if err := revoker.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for at least one scan.
	time.Sleep(50 * time.Millisecond)

	if err := revoker.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	got, err := iss.Get(ctx, cred.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !got.Revoked {
		t.Error("expired credential should have been revoked by AutoRevoker")
	}
	if revoked < 1 {
		t.Errorf("OnScanComplete reported %d revocations, want >= 1", revoked)
	}
}

// --- JITProvider tests ---

func TestJITProvider_Issue(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()
	jit := NewJITProvider(iss, WithJITTTL(1*time.Minute))

	cred, err := jit.Issue(ctx, "agent-1", []string{"tool:execute"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if cred.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", cred.AgentID, "agent-1")
	}
	if !cred.HasPermission("tool:execute") {
		t.Error("credential should have tool:execute permission")
	}
	if cred.Metadata["issued_by"] != "jit" {
		t.Errorf("issued_by metadata = %q, want %q", cred.Metadata["issued_by"], "jit")
	}
	// TTL should be ~1 minute.
	if time.Until(cred.ExpiresAt) > 61*time.Second {
		t.Errorf("ExpiresAt too far in the future: %v", cred.ExpiresAt)
	}
}

func TestJITProvider_Issue_EmptyAgent(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()
	jit := NewJITProvider(iss)

	_, err := jit.Issue(ctx, "", []string{"p"})
	if err == nil {
		t.Fatal("Issue() with empty agent should error")
	}
}

func TestJITProvider_Issue_EmptyPermissions(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()
	jit := NewJITProvider(iss)

	_, err := jit.Issue(ctx, "agent-1", nil)
	if err == nil {
		t.Fatal("Issue() with nil permissions should error")
	}
}

func TestJITProvider_PermissionNarrowing(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	narrower := func(_ string, requested []string) ([]string, error) {
		// Only allow tool:execute.
		var allowed []string
		for _, p := range requested {
			if p == "tool:execute" {
				allowed = append(allowed, p)
			}
		}
		return allowed, nil
	}

	jit := NewJITProvider(iss, WithPermissionNarrowing(narrower))

	cred, err := jit.Issue(ctx, "agent-1", []string{"tool:execute", "memory:write"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if len(cred.Permissions) != 1 || cred.Permissions[0] != "tool:execute" {
		t.Errorf("Permissions = %v, want [tool:execute]", cred.Permissions)
	}
}

func TestJITProvider_PermissionNarrowing_AllDenied(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	narrower := func(_ string, _ []string) ([]string, error) {
		return nil, nil // deny all
	}

	jit := NewJITProvider(iss, WithPermissionNarrowing(narrower))

	_, err := jit.Issue(ctx, "agent-1", []string{"tool:execute"})
	if err == nil {
		t.Fatal("Issue() should fail when all permissions denied")
	}
}

func TestJITProvider_PermissionNarrowing_Error(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()

	narrower := func(_ string, _ []string) ([]string, error) {
		return nil, errors.New("policy unavailable")
	}

	jit := NewJITProvider(iss, WithPermissionNarrowing(narrower))

	_, err := jit.Issue(ctx, "agent-1", []string{"tool:execute"})
	if err == nil {
		t.Fatal("Issue() should propagate narrower error")
	}
}

// --- CredentialMiddleware tests ---

func TestCredentialMiddleware_NoCredential(t *testing.T) {
	ctx := context.Background()
	policy := newAllowAllPolicy()
	wrapped := CredentialMiddleware()(policy)

	allowed, err := wrapped.Authorize(ctx, "alice", auth.PermToolExec, "calc")
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !allowed {
		t.Error("should delegate to underlying policy when no credential")
	}
}

func TestCredentialMiddleware_ValidCredential(t *testing.T) {
	cred := &AgentCredential{
		ID:          "cred-1",
		AgentID:     "agent-1",
		Permissions: []string{string(auth.PermToolExec)},
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	ctx := WithCredential(context.Background(), cred)
	policy := newAllowAllPolicy()
	wrapped := CredentialMiddleware()(policy)

	allowed, err := wrapped.Authorize(ctx, "alice", auth.PermToolExec, "calc")
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !allowed {
		t.Error("should allow with valid credential and matching permission")
	}
}

func TestCredentialMiddleware_ExpiredCredential(t *testing.T) {
	cred := &AgentCredential{
		ID:          "cred-1",
		AgentID:     "agent-1",
		Permissions: []string{string(auth.PermToolExec)},
		ExpiresAt:   time.Now().Add(-time.Hour),
	}
	ctx := WithCredential(context.Background(), cred)
	policy := newAllowAllPolicy()
	wrapped := CredentialMiddleware()(policy)

	_, err := wrapped.Authorize(ctx, "alice", auth.PermToolExec, "calc")
	if err == nil {
		t.Fatal("should return error for expired credential")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrAuth {
		t.Errorf("error = %v, want auth error", err)
	}
}

func TestCredentialMiddleware_RevokedCredential(t *testing.T) {
	cred := &AgentCredential{
		ID:          "cred-1",
		AgentID:     "agent-1",
		Permissions: []string{string(auth.PermToolExec)},
		ExpiresAt:   time.Now().Add(time.Hour),
		Revoked:     true,
	}
	ctx := WithCredential(context.Background(), cred)
	policy := newAllowAllPolicy()
	wrapped := CredentialMiddleware()(policy)

	_, err := wrapped.Authorize(ctx, "alice", auth.PermToolExec, "calc")
	if err == nil {
		t.Fatal("should return error for revoked credential")
	}
}

func TestCredentialMiddleware_MissingPermission(t *testing.T) {
	cred := &AgentCredential{
		ID:          "cred-1",
		AgentID:     "agent-1",
		Permissions: []string{"memory:read"},
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	ctx := WithCredential(context.Background(), cred)
	policy := newAllowAllPolicy()
	wrapped := CredentialMiddleware()(policy)

	allowed, err := wrapped.Authorize(ctx, "alice", auth.PermToolExec, "calc")
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if allowed {
		t.Error("should deny when credential lacks required permission")
	}
}

// --- Full lifecycle integration test ---

func TestFullLifecycle(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer(WithDefaultTTL(5*time.Minute), WithMaxCredentials(100))
	jit := NewJITProvider(iss, WithJITTTL(1*time.Minute))

	// Issue via JIT.
	cred, err := jit.Issue(ctx, "agent-alpha", []string{string(auth.PermToolExec)})
	if err != nil {
		t.Fatalf("JIT Issue() error = %v", err)
	}
	if !cred.IsValid() {
		t.Fatal("fresh JIT credential should be valid")
	}

	// Use in middleware.
	credCtx := WithCredential(ctx, cred)
	policy := auth.ApplyMiddleware(newAllowAllPolicy(), CredentialMiddleware())

	allowed, err := policy.Authorize(credCtx, "agent-alpha", auth.PermToolExec, "calculator")
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !allowed {
		t.Error("should allow valid JIT credential")
	}

	// Revoke.
	if err := iss.Revoke(ctx, cred.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	// Should now fail.
	_, err = policy.Authorize(credCtx, "agent-alpha", auth.PermToolExec, "calculator")
	if err == nil {
		t.Fatal("should return error after revocation")
	}
}

// --- test helpers ---

// allowAllPolicy is a test helper that always allows authorization.
type allowAllPolicy struct{}

func newAllowAllPolicy() auth.Policy { return &allowAllPolicy{} }

func (p *allowAllPolicy) Name() string { return "allow-all" }

func (p *allowAllPolicy) Authorize(_ context.Context, _ string, _ auth.Permission, _ string) (bool, error) {
	return true, nil
}

var _ auth.Policy = (*allowAllPolicy)(nil)
