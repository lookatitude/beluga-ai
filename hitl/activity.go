package hitl

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Activity returns a workflow activity function that requests human
// interaction through mgr. Its input must be an InteractionRequest; it blocks
// until a human responds, the request times out, or ctx is done, and returns
// the *InteractionResponse.
//
// The returned func has the signature of workflow.ActivityFunc and can be
// passed straight to WorkflowContext.ExecuteActivity. It lives here (rather
// than in package workflow) so that workflow does not depend on hitl.
func Activity(mgr Manager) func(ctx context.Context, input any) (any, error) {
	return func(ctx context.Context, input any) (any, error) {
		req, ok := input.(InteractionRequest)
		if !ok {
			return nil, core.Errorf(core.ErrInvalidInput, "hitl/activity: expected hitl.InteractionRequest, got %T", input)
		}
		resp, err := mgr.RequestInteraction(ctx, req)
		if err != nil {
			return nil, core.Wrapf(core.ErrInternal, "hitl/activity: %w", err)
		}
		return resp, nil
	}
}
