package credential

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/auth"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// CredentialMiddleware returns an auth.Middleware that enforces credential-based
// authorization. It extracts the credential from the context, validates that it
// is not expired or revoked, and checks that it contains the requested
// permission. If any check fails, authorization is denied.
func CredentialMiddleware() auth.Middleware {
	return func(next auth.Policy) auth.Policy {
		return &credentialPolicy{next: next}
	}
}

// credentialPolicy wraps an auth.Policy with credential enforcement.
type credentialPolicy struct {
	next auth.Policy
}

// Compile-time check that credentialPolicy implements auth.Policy.
var _ auth.Policy = (*credentialPolicy)(nil)

// Name returns the name of the wrapped policy.
func (p *credentialPolicy) Name() string { return p.next.Name() }

// Authorize checks the credential from context before delegating to the
// wrapped policy. If no credential is present, it delegates directly to the
// wrapped policy (allowing non-credential-based authorization to pass through).
// If a credential is present but invalid or missing the required permission,
// authorization is denied.
func (p *credentialPolicy) Authorize(ctx context.Context, subject string, permission auth.Permission, resource string) (bool, error) {
	cred := CredentialFromContext(ctx)
	if cred == nil {
		// No credential in context; delegate to underlying policy.
		return p.next.Authorize(ctx, subject, permission, resource)
	}

	// Validate the credential.
	if cred.Revoked {
		return false, core.NewError("credential.middleware", core.ErrAuth, "credential has been revoked", nil)
	}
	if cred.IsExpired() {
		return false, core.NewError("credential.middleware", core.ErrAuth, "credential has expired", nil)
	}

	// Check that the credential has the required permission.
	if !cred.HasPermission(string(permission)) {
		return false, nil // clean deny: credential lacks permission
	}

	// Credential is valid and has permission; delegate to underlying policy.
	return p.next.Authorize(ctx, subject, permission, resource)
}
