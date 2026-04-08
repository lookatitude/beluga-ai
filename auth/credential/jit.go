package credential

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// PermissionNarrower is a function that narrows a set of requested permissions
// to the minimum required set. It returns the narrowed permissions and an error
// if the requested permissions are not allowed.
type PermissionNarrower func(agentID string, requested []string) ([]string, error)

// jitOptions holds configuration for JITProvider.
type jitOptions struct {
	ttl      time.Duration
	narrower PermissionNarrower
}

// JITOption configures a JITProvider.
type JITOption func(*jitOptions)

// WithJITTTL sets the time-to-live for JIT-issued credentials.
// Defaults to 2 minutes.
func WithJITTTL(d time.Duration) JITOption {
	return func(o *jitOptions) { o.ttl = d }
}

// WithPermissionNarrowing sets a function that narrows requested permissions
// to the minimum required set. If not set, all requested permissions are
// granted as-is.
func WithPermissionNarrowing(fn PermissionNarrower) JITOption {
	return func(o *jitOptions) { o.narrower = fn }
}

func defaultJITOptions() jitOptions {
	return jitOptions{
		ttl: 2 * time.Minute,
	}
}

// JITProvider issues just-in-time credentials with minimal TTL scoped to
// only the permissions required for an immediate operation.
type JITProvider struct {
	issuer CredentialIssuer
	opts   jitOptions
}

// NewJITProvider creates a new JITProvider backed by the given issuer.
func NewJITProvider(issuer CredentialIssuer, opts ...JITOption) *JITProvider {
	o := defaultJITOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &JITProvider{
		issuer: issuer,
		opts:   o,
	}
}

// Issue creates a JIT credential for the agent with the requested permissions.
// If a PermissionNarrower is configured, the requested permissions are narrowed
// before issuance.
func (p *JITProvider) Issue(ctx context.Context, agentID string, requested []string) (*AgentCredential, error) {
	if agentID == "" {
		return nil, core.NewError("credential.jit.issue", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}
	if len(requested) == 0 {
		return nil, core.NewError("credential.jit.issue", core.ErrInvalidInput, "requested permissions must not be empty", nil)
	}

	permissions := requested
	if p.opts.narrower != nil {
		narrowed, err := p.opts.narrower(agentID, requested)
		if err != nil {
			return nil, core.NewError("credential.jit.issue", core.ErrAuth, "permission narrowing failed", err)
		}
		if len(narrowed) == 0 {
			return nil, core.NewError("credential.jit.issue", core.ErrAuth, "all requested permissions were denied", nil)
		}
		permissions = narrowed
	}

	cred, err := p.issuer.Issue(ctx, agentID, permissions, p.opts.ttl)
	if err != nil {
		return nil, fmt.Errorf("credential.jit.issue: %w", err)
	}

	if cred.Metadata == nil {
		cred.Metadata = make(map[string]string)
	}
	cred.Metadata["issued_by"] = "jit"

	return cred, nil
}
