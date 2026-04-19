package hitl

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/o11y"
)

// WithTracing returns middleware that wraps a Manager with OTel spans following
// the GenAI semantic conventions. Each operation produces a span named
// "hitl.<op>" carrying a gen_ai.operation.name attribute. Errors are
// recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	mgr = hitl.ApplyMiddleware(mgr, hitl.WithTracing(), hitl.WithHooks(h))
func WithTracing() Middleware {
	return func(next Manager) Manager {
		return &tracedManager{next: next}
	}
}

// tracedManager wraps a Manager and emits a span around each operation.
type tracedManager struct {
	next Manager
}

func (m *tracedManager) RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error) {
	ctx, span := o11y.StartSpan(ctx, "hitl.request_interaction", o11y.Attrs{
		o11y.AttrOperationName:    "hitl.request_interaction",
		"hitl.request.type":       string(req.Type),
		"hitl.request.tool":       req.ToolName,
		"hitl.request.risk_level": string(req.RiskLevel),
		"hitl.request.confidence": req.Confidence,
	})
	defer span.End()

	resp, err := m.next.RequestInteraction(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	if resp != nil {
		span.SetAttributes(o11y.Attrs{"hitl.response.decision": string(resp.Decision)})
	}
	span.SetStatus(o11y.StatusOK, "")
	return resp, nil
}

func (m *tracedManager) AddPolicy(policy ApprovalPolicy) error {
	_, span := o11y.StartSpan(context.Background(), "hitl.add_policy", o11y.Attrs{
		o11y.AttrOperationName: "hitl.add_policy",
		"hitl.policy.name":     policy.Name,
		"hitl.policy.pattern":  policy.ToolPattern,
	})
	defer span.End()

	err := m.next.AddPolicy(policy)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (m *tracedManager) ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error) {
	ctx, span := o11y.StartSpan(ctx, "hitl.should_approve", o11y.Attrs{
		o11y.AttrOperationName: "hitl.should_approve",
		"hitl.tool":            toolName,
		"hitl.confidence":      confidence,
		"hitl.risk_level":      string(risk),
	})
	defer span.End()

	approved, err := m.next.ShouldApprove(ctx, toolName, confidence, risk)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return false, err
	}
	span.SetAttributes(o11y.Attrs{"hitl.auto_approved": approved})
	span.SetStatus(o11y.StatusOK, "")
	return approved, nil
}

func (m *tracedManager) Respond(ctx context.Context, requestID string, resp InteractionResponse) error {
	ctx, span := o11y.StartSpan(ctx, "hitl.respond", o11y.Attrs{
		o11y.AttrOperationName:   "hitl.respond",
		"hitl.request_id":        requestID,
		"hitl.response.decision": string(resp.Decision),
	})
	defer span.End()

	err := m.next.Respond(ctx, requestID, resp)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Ensure tracedManager implements Manager at compile time.
var _ Manager = (*tracedManager)(nil)
