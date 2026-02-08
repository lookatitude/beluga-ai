// Package hitl provides human-in-the-loop interaction management for the
// Beluga AI framework. It supports approval workflows, feedback collection,
// and confidence-based auto-approval policies.
//
// The package implements a Manager interface that routes interaction requests
// through configurable ApprovalPolicy rules. Policies can auto-approve
// low-risk, high-confidence actions while escalating uncertain or dangerous
// operations to human reviewers.
//
// Usage:
//
//	mgr := hitl.NewManager(
//	    hitl.WithTimeout(30 * time.Second),
//	    hitl.WithNotifier(hitl.NewLogNotifier(slog.Default())),
//	)
//	mgr.AddPolicy(hitl.ApprovalPolicy{
//	    Name:          "read-only-auto",
//	    ToolPattern:   "get_*",
//	    MinConfidence: 0.5,
//	    MaxRiskLevel:  hitl.RiskReadOnly,
//	})
//	resp, err := mgr.RequestInteraction(ctx, req)
package hitl

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// InteractionType identifies the kind of human interaction required.
type InteractionType string

const (
	// TypeApproval requests approval to proceed with an action.
	TypeApproval InteractionType = "approval"

	// TypeFeedback requests human feedback on a result.
	TypeFeedback InteractionType = "feedback"

	// TypeInput requests human input or additional information.
	TypeInput InteractionType = "input"

	// TypeAnnotation requests human annotation of data.
	TypeAnnotation InteractionType = "annotation"
)

// Decision represents the human's decision on an interaction request.
type Decision string

const (
	// DecisionApprove indicates the action is approved.
	DecisionApprove Decision = "approve"

	// DecisionReject indicates the action is rejected.
	DecisionReject Decision = "reject"

	// DecisionModify indicates the action should be modified before proceeding.
	DecisionModify Decision = "modify"
)

// RiskLevel categorizes the risk associated with an action.
type RiskLevel string

const (
	// RiskReadOnly indicates a read-only operation with minimal risk.
	RiskReadOnly RiskLevel = "read_only"

	// RiskDataModification indicates a data modification operation.
	RiskDataModification RiskLevel = "data_modification"

	// RiskIrreversible indicates an irreversible operation.
	RiskIrreversible RiskLevel = "irreversible"
)

// riskOrder maps risk levels to numeric order for comparison.
// Lower values represent lower risk.
var riskOrder = map[RiskLevel]int{
	RiskReadOnly:         0,
	RiskDataModification: 1,
	RiskIrreversible:     2,
}

// InteractionRequest describes a pending human interaction.
type InteractionRequest struct {
	// ID uniquely identifies this interaction request.
	ID string

	// Type is the kind of interaction required.
	Type InteractionType

	// ToolName is the name of the tool requesting interaction.
	ToolName string

	// Description provides a human-readable explanation of what is being requested.
	Description string

	// Input holds the tool input data for context.
	Input map[string]any

	// RiskLevel categorizes the risk of this action.
	RiskLevel RiskLevel

	// Confidence is the model's confidence level (0.0–1.0).
	Confidence float64

	// Timeout specifies how long to wait for a response. Zero means use
	// the manager's default timeout.
	Timeout time.Duration

	// Metadata holds arbitrary metadata.
	Metadata map[string]any
}

// InteractionResponse holds the human's response to an interaction request.
type InteractionResponse struct {
	// RequestID links this response to its request.
	RequestID string

	// Decision is the human's decision.
	Decision Decision

	// Feedback is optional text feedback from the human.
	Feedback string

	// Modified holds updated input values when Decision is DecisionModify.
	Modified map[string]any

	// Metadata holds arbitrary metadata.
	Metadata map[string]any
}

// ApprovalPolicy defines when human approval is required.
type ApprovalPolicy struct {
	// Name uniquely identifies this policy.
	Name string

	// ToolPattern is a glob pattern matched against tool names. Use "*" to
	// match all tools.
	ToolPattern string

	// MinConfidence is the minimum confidence score required for auto-approval.
	// Actions with confidence >= MinConfidence may be auto-approved if other
	// criteria are met.
	MinConfidence float64

	// MaxRiskLevel is the maximum risk level that can be auto-approved.
	// Actions with higher risk are always escalated.
	MaxRiskLevel RiskLevel

	// RequireExplicit forces human approval regardless of confidence or risk.
	RequireExplicit bool
}

// Manager is the interface for managing human-in-the-loop interactions.
// Implementations must be safe for concurrent use.
type Manager interface {
	// RequestInteraction sends an interaction request and waits for a response.
	// It blocks until a response is received, the request times out, or the
	// context is canceled.
	RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error)

	// AddPolicy registers an approval policy. Policies are evaluated in the
	// order they are added; the first matching policy wins.
	AddPolicy(policy ApprovalPolicy) error

	// ShouldApprove checks whether an action can be auto-approved based on
	// registered policies, the tool name, confidence level, and risk level.
	// Returns true if auto-approval is granted, false if human approval is needed.
	ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error)

	// Respond provides a response to a pending interaction request.
	Respond(ctx context.Context, requestID string, resp InteractionResponse) error
}

// Factory creates a Manager from the given configuration.
type Factory func(cfg Config) (Manager, error)

// Config holds configuration for creating a Manager via the registry.
type Config struct {
	// DefaultTimeout is the default timeout for interaction requests that
	// do not specify their own timeout.
	DefaultTimeout time.Duration

	// Notifier is used to alert humans about pending interaction requests.
	Notifier Notifier

	// Extra holds provider-specific configuration values.
	Extra map[string]any
}

// Package-level registry for Manager factories.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named Manager factory to the global registry. It is safe to
// call from init functions. Register panics if name is empty or already
// registered.
func Register(name string, f Factory) {
	if name == "" {
		panic("hitl: Register called with empty name")
	}
	if f == nil {
		panic("hitl: Register called with nil factory for " + name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic("hitl: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates a Manager by looking up the named factory in the registry and
// invoking it with cfg. Returns an error if the name is not registered.
func New(name string, cfg Config) (Manager, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("hitl: unknown manager %q", name)
	}
	return f(cfg)
}

// List returns the sorted names of all registered Manager factories.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
