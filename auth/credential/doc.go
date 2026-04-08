// Package credential provides temporal least-privilege credential management
// for the Beluga AI framework. Credentials are time-bounded, scoped to
// specific permissions, and automatically revoked when expired.
//
// # AgentCredential
//
// An AgentCredential ties a set of permissions to an agent identity with
// explicit issuance and expiry timestamps. Credentials can be revoked
// manually or automatically by the AutoRevoker.
//
//	cred := credential.AgentCredential{
//	    ID:          "cred-123",
//	    AgentID:     "agent-alpha",
//	    Permissions: []string{"tool:execute", "memory:read"},
//	    IssuedAt:    time.Now(),
//	    ExpiresAt:   time.Now().Add(5 * time.Minute),
//	}
//
// # CredentialIssuer
//
// The CredentialIssuer interface provides Issue, Revoke, and Get operations.
// InMemoryIssuer is the built-in thread-safe implementation with bounded
// storage and configurable default TTL.
//
//	issuer := credential.NewInMemoryIssuer(
//	    credential.WithDefaultTTL(10 * time.Minute),
//	    credential.WithMaxCredentials(1000),
//	)
//
// # AutoRevoker
//
// AutoRevoker implements core.Lifecycle and runs a background scan that
// revokes expired credentials on a configurable interval.
//
//	revoker := credential.NewAutoRevoker(issuer,
//	    credential.WithScanInterval(30 * time.Second),
//	)
//	revoker.Start(ctx)
//	defer revoker.Stop(ctx)
//
// # JITProvider
//
// JITProvider issues just-in-time credentials with minimal TTL scoped to
// only the permissions required for an immediate operation.
//
//	jit := credential.NewJITProvider(issuer,
//	    credential.WithJITTTL(2 * time.Minute),
//	)
//	cred, err := jit.Issue(ctx, "agent-alpha", []string{"tool:execute"})
//
// # Context Helpers
//
// WithCredential and CredentialFromContext store and retrieve credentials
// from context.Context, enabling middleware-based enforcement.
//
// # Middleware
//
// CredentialMiddleware wraps an auth.Policy to enforce credential-based
// authorization: it extracts the credential from context, validates
// expiry, and checks that the required permission is present.
package credential
