// Package mocktelephony provides an in-memory SIPEndpoint implementation for
// testing code that depends on the voice/telephony package.
package mocktelephony

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/voice/telephony"
)

// InMemoryEndpoint is a test implementation of telephony.SIPEndpoint. It
// tracks the connected state and an ActiveCalls counter that callers can
// manipulate via BeginCall/EndCall.
type InMemoryEndpoint struct {
	mu        sync.Mutex
	connected bool
	calls     int
}

var _ telephony.SIPEndpoint = (*InMemoryEndpoint)(nil)

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

// Status returns the current endpoint status, including the number of calls
// currently being tracked via BeginCall/EndCall.
func (e *InMemoryEndpoint) Status(_ context.Context) (telephony.EndpointStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return telephony.EndpointStatus{
		Connected:   e.connected,
		ActiveCalls: e.calls,
	}, nil
}

// BeginCall increments the ActiveCalls counter. Tests use this to simulate a
// new in-flight call.
func (e *InMemoryEndpoint) BeginCall() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
}

// EndCall decrements the ActiveCalls counter, clamping at zero.
func (e *InMemoryEndpoint) EndCall() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.calls > 0 {
		e.calls--
	}
}
