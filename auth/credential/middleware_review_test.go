package credential

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/auth"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// FI-27: revocation at the issuer must take effect for credentials already
// placed in a context, and a credential must only authorize its own agent.
func TestMiddlewareWithIssuer(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer()
	cred, err := iss.Issue(ctx, "agent-a", []string{string(auth.PermToolExec)}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	policy := auth.ApplyMiddleware(newAllowAllPolicy(), MiddlewareWithIssuer(iss))
	credCtx := WithCredential(ctx, cred)

	ok, err := policy.Authorize(credCtx, "agent-a", auth.PermToolExec, "calc")
	if err != nil || !ok {
		t.Fatalf("valid credential: %v, %v", ok, err)
	}

	// Subject binding.
	if ok, err := policy.Authorize(credCtx, "agent-b", auth.PermToolExec, "calc"); ok || !errors.Is(err, core.NewError("", core.ErrAuth, "", nil)) {
		t.Fatalf("cross-subject: %v, %v; want auth error", ok, err)
	}

	// Missing permission on the live copy.
	if ok, err := policy.Authorize(credCtx, "agent-a", auth.PermMemoryWrite, "m"); ok || err != nil {
		t.Fatalf("missing permission: %v, %v; want clean deny", ok, err)
	}

	// Revoke at the issuer: the context copy still says not revoked.
	if err := iss.Revoke(ctx, cred.ID); err != nil {
		t.Fatal(err)
	}
	if cred.Revoked {
		t.Fatal("test premise: context snapshot must not observe revocation")
	}
	if ok, err := policy.Authorize(credCtx, "agent-a", auth.PermToolExec, "calc"); ok || err == nil {
		t.Fatalf("revoked: %v, %v; want error", ok, err)
	}

	// Unknown credential.
	forged := &AgentCredential{ID: "nope", AgentID: "agent-a", Permissions: []string{string(auth.PermToolExec)}, ExpiresAt: time.Now().Add(time.Hour)}
	if ok, err := policy.Authorize(WithCredential(ctx, forged), "agent-a", auth.PermToolExec, "calc"); ok || err == nil {
		t.Fatalf("forged: %v, %v; want error", ok, err)
	}

	// No credential delegates.
	if ok, err := policy.Authorize(ctx, "anyone", auth.PermToolExec, "calc"); !ok || err != nil {
		t.Fatalf("no credential: %v, %v", ok, err)
	}
}

// FI-42: expired credentials must not fill the store permanently.
func TestInMemoryIssuer_EvictsExpiredWhenFull(t *testing.T) {
	ctx := context.Background()
	iss := NewInMemoryIssuer(WithMaxCredentials(2))
	for i := 0; i < 2; i++ {
		if _, err := iss.Issue(ctx, "a", []string{"x"}, time.Nanosecond); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(time.Millisecond)
	if _, err := iss.Issue(ctx, "a", []string{"x"}, time.Hour); err != nil {
		t.Fatalf("Issue after expiry: %v", err)
	}
	if _, err := iss.Issue(ctx, "a", []string{"x"}, time.Hour); err != nil {
		t.Fatalf("second Issue: %v", err)
	}
	if _, err := iss.Issue(ctx, "a", []string{"x"}, time.Hour); err == nil {
		t.Fatal("store of live credentials must still be bounded")
	}
}
