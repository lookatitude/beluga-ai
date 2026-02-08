package hitl

import (
	"context"
	"fmt"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

// seq is a package-level counter for generating unique request IDs.
var seq atomic.Int64

// generateID produces a unique request ID using an atomic counter.
func generateID() string {
	return fmt.Sprintf("hitl-%d", seq.Add(1))
}

// DefaultManager is the default Manager implementation. It stores pending
// interaction requests in memory and supports policy-based auto-approval,
// notifier integration, and request timeouts.
type DefaultManager struct {
	policies       []ApprovalPolicy
	pending        map[string]chan InteractionResponse
	notifier       Notifier
	hooks          Hooks
	defaultTimeout time.Duration
	mu             sync.RWMutex
}

// ManagerOption is a functional option for configuring a DefaultManager.
type ManagerOption func(*DefaultManager)

// WithNotifier sets the notifier for the manager.
func WithNotifier(n Notifier) ManagerOption {
	return func(m *DefaultManager) {
		m.notifier = n
	}
}

// WithTimeout sets the default timeout for interaction requests.
func WithTimeout(d time.Duration) ManagerOption {
	return func(m *DefaultManager) {
		m.defaultTimeout = d
	}
}

// WithManagerHooks sets the lifecycle hooks for the manager.
func WithManagerHooks(h Hooks) ManagerOption {
	return func(m *DefaultManager) {
		m.hooks = h
	}
}

// NewManager creates a new DefaultManager with the given options.
func NewManager(opts ...ManagerOption) *DefaultManager {
	m := &DefaultManager{
		pending:        make(map[string]chan InteractionResponse),
		defaultTimeout: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// AddPolicy registers an approval policy. Returns an error if the tool
// pattern is empty or invalid.
func (m *DefaultManager) AddPolicy(policy ApprovalPolicy) error {
	if policy.ToolPattern == "" {
		return fmt.Errorf("hitl/add_policy: tool pattern is required")
	}
	// Validate the glob pattern using path.Match.
	if _, err := path.Match(policy.ToolPattern, ""); err != nil {
		return fmt.Errorf("hitl/add_policy: invalid pattern %q: %w", policy.ToolPattern, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies = append(m.policies, policy)
	return nil
}

// ShouldApprove checks whether an action can be auto-approved based on
// registered policies, the tool name, confidence level, and risk level.
// Returns true if auto-approval is granted, false if human approval is needed.
//
// Policy evaluation:
//  1. Find the first policy whose ToolPattern matches the toolName.
//  2. If RequireExplicit is set, return false (always needs human).
//  3. If confidence >= MinConfidence AND the action's risk level does not
//     exceed the policy's MaxRiskLevel, return true (auto-approve).
//  4. If no policy matches, default to false (require human approval).
func (m *DefaultManager) ShouldApprove(_ context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.policies {
		matched, err := path.Match(p.ToolPattern, toolName)
		if err != nil {
			return false, fmt.Errorf("hitl/should_approve: %w", err)
		}
		if !matched {
			continue
		}

		// First matching policy wins.
		if p.RequireExplicit {
			return false, nil
		}

		// Check confidence threshold.
		if confidence < p.MinConfidence {
			return false, nil
		}

		// Check risk level: the action's risk must not exceed the policy max.
		actionRisk, ok := riskOrder[risk]
		if !ok {
			// Unknown risk level — require approval.
			return false, nil
		}
		policyMaxRisk, ok := riskOrder[p.MaxRiskLevel]
		if !ok {
			// Unknown policy risk — require approval.
			return false, nil
		}
		if actionRisk > policyMaxRisk {
			return false, nil
		}

		return true, nil
	}

	// No matching policy: require human approval by default.
	return false, nil
}

// RequestInteraction sends an interaction request, optionally notifies via the
// configured Notifier, and blocks until a response is received or the
// timeout/context expires.
//
// If policies allow auto-approval (ShouldApprove returns true), the request
// is immediately approved without waiting for a human.
func (m *DefaultManager) RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error) {
	// Assign an ID if not provided.
	if req.ID == "" {
		req.ID = generateID()
	}

	// Fire OnRequest hook.
	if m.hooks.OnRequest != nil {
		if err := m.hooks.OnRequest(ctx, req); err != nil {
			return nil, fmt.Errorf("hitl/request: on_request hook: %w", err)
		}
	}

	// Check if auto-approval applies.
	autoApprove, err := m.ShouldApprove(ctx, req.ToolName, req.Confidence, req.RiskLevel)
	if err != nil {
		if m.hooks.OnError != nil {
			if e := m.hooks.OnError(ctx, err); e != nil {
				return nil, fmt.Errorf("hitl/request: %w", e)
			}
		}
		return nil, fmt.Errorf("hitl/request: %w", err)
	}

	if autoApprove {
		resp := &InteractionResponse{
			RequestID: req.ID,
			Decision:  DecisionApprove,
		}
		if m.hooks.OnApprove != nil {
			m.hooks.OnApprove(ctx, req, *resp)
		}
		return resp, nil
	}

	// Create a pending channel for this request.
	respCh := make(chan InteractionResponse, 1)
	m.mu.Lock()
	m.pending[req.ID] = respCh
	m.mu.Unlock()

	// Notify via the configured notifier.
	if m.notifier != nil {
		if err := m.notifier.Notify(ctx, req); err != nil {
			if m.hooks.OnError != nil {
				m.hooks.OnError(ctx, err)
			}
		}
	}

	// Determine timeout: prefer per-request, fall back to default.
	timeout := m.defaultTimeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}

	timeoutCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		timeoutCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	select {
	case resp := <-respCh:
		m.mu.Lock()
		delete(m.pending, req.ID)
		m.mu.Unlock()

		resp.RequestID = req.ID
		switch resp.Decision {
		case DecisionApprove:
			if m.hooks.OnApprove != nil {
				m.hooks.OnApprove(ctx, req, resp)
			}
		case DecisionReject:
			if m.hooks.OnReject != nil {
				m.hooks.OnReject(ctx, req, resp)
			}
		}
		return &resp, nil

	case <-timeoutCtx.Done():
		m.mu.Lock()
		delete(m.pending, req.ID)
		m.mu.Unlock()

		if m.hooks.OnTimeout != nil {
			m.hooks.OnTimeout(ctx, req)
		}
		return nil, fmt.Errorf("hitl/request: %w", timeoutCtx.Err())
	}
}

// Respond delivers a human response to a pending interaction request.
func (m *DefaultManager) Respond(_ context.Context, requestID string, resp InteractionResponse) error {
	m.mu.RLock()
	ch, ok := m.pending[requestID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("hitl/respond: request %q not found", requestID)
	}

	resp.RequestID = requestID
	ch <- resp
	return nil
}

// Ensure DefaultManager implements Manager at compile time.
var _ Manager = (*DefaultManager)(nil)

func init() {
	Register("default", func(cfg Config) (Manager, error) {
		opts := []ManagerOption{
			WithTimeout(cfg.DefaultTimeout),
		}
		if cfg.Notifier != nil {
			opts = append(opts, WithNotifier(cfg.Notifier))
		}
		return NewManager(opts...), nil
	})
}
