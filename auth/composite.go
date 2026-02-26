package auth

import (
	"context"
)

// CompositeMode determines how multiple policies are combined.
type CompositeMode string

const (
	// AllowIfAny allows access if any policy allows (logical OR).
	AllowIfAny CompositeMode = "allow_if_any"
	// AllowIfAll allows access only if all policies allow (logical AND).
	AllowIfAll CompositeMode = "allow_if_all"
	// DenyIfAny denies access if any policy denies (conservative).
	DenyIfAny CompositeMode = "deny_if_any"
)

// CompositePolicy combines multiple policies using a configurable mode.
// It delegates authorization to each child policy and combines the results
// according to the mode.
type CompositePolicy struct {
	name     string
	policies []Policy
	mode     CompositeMode
}

// NewCompositePolicy creates a composite policy that combines the given
// policies using the specified mode.
func NewCompositePolicy(name string, mode CompositeMode, policies ...Policy) *CompositePolicy {
	return &CompositePolicy{
		name:     name,
		policies: policies,
		mode:     mode,
	}
}

// Name returns the policy name.
func (p *CompositePolicy) Name() string { return p.name }

// Authorize checks all child policies and combines results according to mode:
//   - AllowIfAny: returns true if any policy allows.
//   - AllowIfAll: returns true only if all policies allow.
//   - DenyIfAny: returns false if any policy denies.
//
// Errors from child policies are propagated immediately.
func (p *CompositePolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	if len(p.policies) == 0 {
		return false, nil
	}

	switch p.mode {
	case AllowIfAny:
		return p.authorizeAny(ctx, subject, permission, resource)
	case AllowIfAll, DenyIfAny:
		return p.authorizeAll(ctx, subject, permission, resource)
	default:
		return false, nil
	}
}

// authorizeAny returns true if any child policy allows access.
func (p *CompositePolicy) authorizeAny(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	for _, policy := range p.policies {
		allowed, err := policy.Authorize(ctx, subject, permission, resource)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

// authorizeAll returns true only if all child policies allow access.
func (p *CompositePolicy) authorizeAll(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	for _, policy := range p.policies {
		allowed, err := policy.Authorize(ctx, subject, permission, resource)
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

// Ensure CompositePolicy implements Policy at compile time.
var _ Policy = (*CompositePolicy)(nil)
