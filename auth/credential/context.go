package credential

import "context"

// contextKey is an unexported type used for context keys in this package to
// prevent collisions with keys defined in other packages.
type contextKey int

const credentialKey contextKey = iota

// WithCredential returns a copy of ctx carrying the given credential.
func WithCredential(ctx context.Context, cred *AgentCredential) context.Context {
	return context.WithValue(ctx, credentialKey, cred)
}

// CredentialFromContext extracts the AgentCredential from ctx. It returns nil
// if no credential is present.
func CredentialFromContext(ctx context.Context) *AgentCredential {
	cred, _ := ctx.Value(credentialKey).(*AgentCredential)
	return cred
}
