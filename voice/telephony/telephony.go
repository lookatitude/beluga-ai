package telephony

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// SIPEndpoint represents a SIP connection endpoint for making and receiving calls.
type SIPEndpoint interface {
	// Connect establishes a connection to the SIP server.
	Connect(ctx context.Context) error
	// Disconnect closes the SIP connection.
	Disconnect(ctx context.Context) error
	// Status returns the current connection status.
	Status(ctx context.Context) (EndpointStatus, error)
}

// CallRouter routes incoming calls to appropriate handlers.
type CallRouter interface {
	// Route determines how to handle an incoming call based on its metadata.
	Route(ctx context.Context, call IncomingCall) (CallAction, error)
}

// TelephonyProvider is the main interface for telephony service providers.
type TelephonyProvider interface {
	// PlaceCall initiates an outbound call.
	PlaceCall(ctx context.Context, req CallRequest) (*Call, error)
	// HangUp terminates an active call.
	HangUp(ctx context.Context, callID string) error
}

// EndpointStatus describes the current state of a telephony endpoint.
type EndpointStatus struct {
	// Connected indicates whether the endpoint is connected.
	Connected bool `json:"connected"`
	// ActiveCalls is the number of currently active calls.
	ActiveCalls int `json:"active_calls"`
	// LastError is the most recent error, if any.
	LastError string `json:"last_error,omitempty"`
}

// IncomingCall represents an incoming phone call.
type IncomingCall struct {
	// ID uniquely identifies this call.
	ID string `json:"id"`
	// From is the caller's phone number.
	From string `json:"from"`
	// To is the called phone number.
	To string `json:"to"`
	// Timestamp is when the call arrived.
	Timestamp time.Time `json:"timestamp"`
	// Metadata holds provider-specific attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CallAction describes how to handle a call.
type CallAction struct {
	// Type is the action: "accept", "reject", "forward".
	Type string `json:"type"`
	// AgentID is the agent to route the call to (for "accept").
	AgentID string `json:"agent_id,omitempty"`
	// ForwardTo is the number to forward to (for "forward").
	ForwardTo string `json:"forward_to,omitempty"`
	// Reason explains the action (for "reject").
	Reason string `json:"reason,omitempty"`
}

// CallRequest describes an outbound call to place.
type CallRequest struct {
	// To is the destination phone number.
	To string `json:"to"`
	// From is the caller ID to display.
	From string `json:"from"`
	// AgentID is the agent handling this call.
	AgentID string `json:"agent_id"`
	// Timeout is the maximum time to wait for answer.
	Timeout time.Duration `json:"timeout"`
	// Metadata holds provider-specific attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Call represents an active or completed phone call.
type Call struct {
	// ID uniquely identifies this call.
	ID string `json:"id"`
	// Status is the call status: "ringing", "in_progress", "completed", "failed".
	Status string `json:"status"`
	// From is the caller's number.
	From string `json:"from"`
	// To is the called number.
	To string `json:"to"`
	// StartTime is when the call was initiated.
	StartTime time.Time `json:"start_time"`
	// Duration is the call duration (zero if still active).
	Duration time.Duration `json:"duration"`
}

// Option configures a telephony provider.
type Option func(*providerOptions)

type providerOptions struct {
	maxConcurrentCalls int
	defaultTimeout     time.Duration
}

// WithMaxConcurrentCalls sets the maximum number of simultaneous calls.
func WithMaxConcurrentCalls(n int) Option {
	return func(o *providerOptions) { o.maxConcurrentCalls = n }
}

// WithDefaultTimeout sets the default call timeout.
func WithDefaultTimeout(d time.Duration) Option {
	return func(o *providerOptions) { o.defaultTimeout = d }
}

// Factory creates a TelephonyProvider.
type Factory func(cfg Config) (TelephonyProvider, error)

// Config holds configuration for a telephony provider.
type Config struct {
	// AccountSID is the provider account identifier.
	AccountSID string
	// Region is the provider region.
	Region string
	// MaxConcurrentCalls limits simultaneous calls.
	MaxConcurrentCalls int
	// DefaultTimeout is the default call timeout.
	DefaultTimeout time.Duration
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a telephony provider factory to the global registry.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a TelephonyProvider by name from the registry.
func New(name string, cfg Config) (TelephonyProvider, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("telephony: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered providers, sorted alphabetically.
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

// PrefixRouter routes calls based on phone number prefixes.
type PrefixRouter struct {
	mu     sync.RWMutex
	rules  []prefixRule
	defAct CallAction
}

type prefixRule struct {
	prefix string
	action CallAction
}

// NewPrefixRouter creates a router that matches call destinations by prefix.
func NewPrefixRouter(defaultAction CallAction) *PrefixRouter {
	return &PrefixRouter{
		defAct: defaultAction,
	}
}

var _ CallRouter = (*PrefixRouter)(nil)

// AddRule adds a routing rule for a phone number prefix.
func (r *PrefixRouter) AddRule(prefix string, action CallAction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, prefixRule{prefix: prefix, action: action})
}

// Route returns the action for the given call based on prefix matching.
func (r *PrefixRouter) Route(_ context.Context, call IncomingCall) (CallAction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestMatch prefixRule
	for _, rule := range r.rules {
		if len(call.To) >= len(rule.prefix) && call.To[:len(rule.prefix)] == rule.prefix {
			if len(rule.prefix) > len(bestMatch.prefix) {
				bestMatch = rule
			}
		}
	}

	if bestMatch.prefix != "" {
		return bestMatch.action, nil
	}
	return r.defAct, nil
}

// InMemoryEndpoint is a test implementation of SIPEndpoint.
type InMemoryEndpoint struct {
	mu        sync.Mutex
	connected bool
	calls     int
}

var _ SIPEndpoint = (*InMemoryEndpoint)(nil)

// NewInMemoryEndpoint creates an in-memory SIP endpoint for testing.
func NewInMemoryEndpoint() *InMemoryEndpoint {
	return &InMemoryEndpoint{}
}

// Connect marks the endpoint as connected.
func (e *InMemoryEndpoint) Connect(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connected = true
	return nil
}

// Disconnect marks the endpoint as disconnected.
func (e *InMemoryEndpoint) Disconnect(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connected = false
	return nil
}

// Status returns the endpoint status.
func (e *InMemoryEndpoint) Status(_ context.Context) (EndpointStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return EndpointStatus{
		Connected:   e.connected,
		ActiveCalls: e.calls,
	}, nil
}
