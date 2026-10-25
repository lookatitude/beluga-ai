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
//
// Deprecated: CredentialMiddleware trusts the credential snapshot stored in
// the context, so a credential revoked at the issuer after it was placed in
// the context is still accepted, and it does not bind the credential to the
// subject being authorized. Use MiddlewareWithIssuer.
func CredentialMiddleware() auth.Middleware { //nolint:revive // stuttering name is part of the published v2 API; renaming would break callers
	return func(next auth.Policy) auth.Policy {
		return &credentialPolicy{next: next}
	}
}

// MiddlewareWithIssuer returns an auth.Middleware that enforces
// credential-based authorization against the live state held by iss. For a
// credential found in the context it:
//
//   - re-fetches the credential from iss by ID, so revocation and expiry are
//     evaluated on the issuer's current copy, not the context snapshot;
//   - requires the credential's AgentID to equal the subject being
//     authorized, so agent A's credential cannot authorize subject B;
//   - requires the requested permission on the issuer's copy.
//
// Requests without a credential in the context are delegated unchanged.
func MiddlewareWithIssuer(iss CredentialIssuer) auth.Middleware {
	return func(next auth.Policy) auth.Policy {
		return &credentialPolicy{next: next, issuer: iss}
	}
}

// credentialPolicy wraps an auth.Policy with credential enforcement.
type credentialPolicy struct {
	next auth.Policy
	// issuer, when set, is the source of truth for credential state and
	// enables subject binding.
	issuer CredentialIssuer
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

	if p.issuer != nil {
		live, err := p.issuer.Get(ctx, cred.ID)
		if err != nil {
			return false, core.NewError("credential.middleware", core.ErrAuth, "credential is not known to the issuer", err)
		}
		if live.AgentID != subject {
			return false, core.NewError("credential.middleware", core.ErrAuth, "credential was not issued to this subject", nil)
		}
		cred = live
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
