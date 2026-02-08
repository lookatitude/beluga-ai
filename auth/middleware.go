package auth

import (
	"context"
	"log/slog"
)

// Middleware wraps a Policy to add cross-cutting behavior.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(Policy) Policy

// ApplyMiddleware wraps policy with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(p Policy, mws ...Middleware) Policy {
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return p
}

// WithHooks returns middleware that invokes the given Hooks around each
// Authorize call.
func WithHooks(hooks Hooks) Middleware {
	return func(next Policy) Policy {
		return &hookedPolicy{next: next, hooks: hooks}
	}
}

// hookedPolicy wraps a Policy with hook callbacks.
type hookedPolicy struct {
	next  Policy
	hooks Hooks
}

func (p *hookedPolicy) Name() string { return p.next.Name() }

func (p *hookedPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	// OnAuthorize
	if p.hooks.OnAuthorize != nil {
		if err := p.hooks.OnAuthorize(ctx, subject, permission, resource); err != nil {
			return false, err
		}
	}

	// Execute
	allowed, err := p.next.Authorize(ctx, subject, permission, resource)

	// OnError
	if err != nil && p.hooks.OnError != nil {
		err = p.hooks.OnError(ctx, err)
	}

	// OnAllow / OnDeny
	if err == nil {
		if allowed {
			if p.hooks.OnAllow != nil {
				p.hooks.OnAllow(ctx, subject, permission, resource)
			}
		} else {
			if p.hooks.OnDeny != nil {
				p.hooks.OnDeny(ctx, subject, permission, resource)
			}
		}
	}

	return allowed, err
}

// WithAudit returns middleware that logs all Authorize calls using the
// provided slog.Logger.
func WithAudit(logger *slog.Logger) Middleware {
	return func(next Policy) Policy {
		return &auditPolicy{next: next, logger: logger}
	}
}

// auditPolicy logs every authorization decision.
type auditPolicy struct {
	next   Policy
	logger *slog.Logger
}

func (p *auditPolicy) Name() string { return p.next.Name() }

func (p *auditPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	allowed, err := p.next.Authorize(ctx, subject, permission, resource)

	if err != nil {
		p.logger.ErrorContext(ctx, "auth.authorize.error",
			"policy", p.next.Name(),
			"subject", subject,
			"permission", string(permission),
			"resource", resource,
			"error", err,
		)
	} else if allowed {
		p.logger.InfoContext(ctx, "auth.authorize.allow",
			"policy", p.next.Name(),
			"subject", subject,
			"permission", string(permission),
			"resource", resource,
		)
	} else {
		p.logger.WarnContext(ctx, "auth.authorize.deny",
			"policy", p.next.Name(),
			"subject", subject,
			"permission", string(permission),
			"resource", resource,
		)
	}

	return allowed, err
}

// Ensure compile-time interface compliance.
var (
	_ Policy = (*hookedPolicy)(nil)
	_ Policy = (*auditPolicy)(nil)
)
