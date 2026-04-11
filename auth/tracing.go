package auth

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
)

// WithTracing returns middleware that wraps a Policy with OTel spans following
// the GenAI semantic conventions. Each Authorize call produces a span named
// "auth.authorize" carrying a gen_ai.operation.name attribute. Errors are
// recorded on the span and the status is set to StatusError on failure; deny
// decisions are surfaced as an auth.decision attribute.
//
// Enable tracing by composing with other middleware:
//
//	pol = auth.ApplyMiddleware(pol, auth.WithTracing(), auth.WithHooks(h))
func WithTracing() Middleware {
	return func(next Policy) Policy {
		return &tracedPolicy{next: next}
	}
}

// tracedPolicy wraps a Policy and emits a span around each Authorize call.
type tracedPolicy struct {
	next Policy
}

func (p *tracedPolicy) Name() string { return p.next.Name() }

func (p *tracedPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	ctx, span := o11y.StartSpan(ctx, "auth.authorize", o11y.Attrs{
		o11y.AttrOperationName: "auth.authorize",
		"auth.policy":          p.next.Name(),
		"auth.subject":         subject,
		"auth.permission":      string(permission),
		"auth.resource":        resource,
	})
	defer span.End()

	allowed, err := p.next.Authorize(ctx, subject, permission, resource)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return allowed, err
	}
	span.SetAttributes(o11y.Attrs{"auth.decision": decisionString(allowed)})
	span.SetStatus(o11y.StatusOK, "")
	return allowed, nil
}

func decisionString(allowed bool) string {
	if allowed {
		return "allow"
	}
	return "deny"
}

// Ensure tracedPolicy implements Policy at compile time.
var _ Policy = (*tracedPolicy)(nil)
