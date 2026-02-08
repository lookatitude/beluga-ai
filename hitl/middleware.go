package hitl

import "context"

// Middleware wraps a Manager to add cross-cutting behavior.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(Manager) Manager

// ApplyMiddleware wraps m with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(m Manager, mws ...Middleware) Manager {
	for i := len(mws) - 1; i >= 0; i-- {
		m = mws[i](m)
	}
	return m
}

// WithHooks returns a Middleware that wraps a Manager with lifecycle hooks.
func WithHooks(h Hooks) Middleware {
	return func(next Manager) Manager {
		return &hookedManager{next: next, hooks: h}
	}
}

type hookedManager struct {
	next  Manager
	hooks Hooks
}

func (m *hookedManager) RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error) {
	if m.hooks.OnRequest != nil {
		if err := m.hooks.OnRequest(ctx, req); err != nil {
			return nil, err
		}
	}

	resp, err := m.next.RequestInteraction(ctx, req)

	if err != nil {
		if m.hooks.OnError != nil {
			err = m.hooks.OnError(ctx, err)
		}
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	if resp != nil {
		switch resp.Decision {
		case DecisionApprove:
			if m.hooks.OnApprove != nil {
				m.hooks.OnApprove(ctx, req, *resp)
			}
		case DecisionReject:
			if m.hooks.OnReject != nil {
				m.hooks.OnReject(ctx, req, *resp)
			}
		}
	}

	return resp, nil
}

func (m *hookedManager) AddPolicy(policy ApprovalPolicy) error {
	return m.next.AddPolicy(policy)
}

func (m *hookedManager) ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error) {
	return m.next.ShouldApprove(ctx, toolName, confidence, risk)
}

func (m *hookedManager) Respond(ctx context.Context, requestID string, resp InteractionResponse) error {
	return m.next.Respond(ctx, requestID, resp)
}

// Ensure hookedManager implements Manager at compile time.
var _ Manager = (*hookedManager)(nil)
