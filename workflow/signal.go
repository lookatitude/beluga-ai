package workflow

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// SignalChannel is the interface for sending and receiving signals in workflows.
// It enables human-in-the-loop interactions by allowing external systems to
// communicate with running workflows.
type SignalChannel interface {
	// Send transmits a signal to a specific workflow.
	// It returns an error if the signal cannot be delivered.
	Send(ctx context.Context, workflowID string, signal Signal) error

	// Receive blocks until a signal with the specified name is available for the workflow,
	// or the context is canceled/deadline exceeded.
	// It returns nil if no signal is received within the deadline.
	Receive(ctx context.Context, workflowID string, signalName string) (*Signal, error)
}

// InMemorySignalChannel is an in-memory implementation of SignalChannel
// that stores signals in maps protected by a mutex. It's suitable for
// testing and single-instance deployments.
type InMemorySignalChannel struct {
	mu sync.RWMutex
	// signals maps workflowID -> signal name -> slice of signals
	signals map[string]map[string][]*Signal
}

// Compile-time check: InMemorySignalChannel implements SignalChannel.
var _ SignalChannel = (*InMemorySignalChannel)(nil)

// NewInMemorySignalChannel creates a new in-memory signal channel.
func NewInMemorySignalChannel() *InMemorySignalChannel {
	return &InMemorySignalChannel{
		signals: make(map[string]map[string][]*Signal),
	}
}

// Send transmits a signal to a specific workflow.
// It adds the signal to the queue for that workflow ID and signal name.
func (sc *InMemorySignalChannel) Send(ctx context.Context, workflowID string, signal Signal) error {
	if workflowID == "" {
		return core.Errorf(core.ErrInvalidInput, "workflow signal: workflowID cannot be empty")
	}
	if signal.Name == "" {
		return core.Errorf(core.ErrInvalidInput, "workflow signal: signal name cannot be empty")
	}

	// Capture the current time if not set
	if signal.SentAt.IsZero() {
		signal.SentAt = time.Now()
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Ensure the workflow ID entry exists
	if _, ok := sc.signals[workflowID]; !ok {
		sc.signals[workflowID] = make(map[string][]*Signal)
	}

	// Append the signal to the queue for this signal name
	sc.signals[workflowID][signal.Name] = append(sc.signals[workflowID][signal.Name], &signal)

	return nil
}

// Receive blocks until a signal with the specified name is available for the workflow,
// or until the context is canceled/deadline exceeded.
// It removes and returns the first signal from the queue.
func (sc *InMemorySignalChannel) Receive(ctx context.Context, workflowID string, signalName string) (*Signal, error) {
	if workflowID == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "workflow signal: workflowID cannot be empty")
	}
	if signalName == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "workflow signal: signalName cannot be empty")
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			sc.mu.Lock()
			if workflow, ok := sc.signals[workflowID]; ok {
				if signals, ok := workflow[signalName]; ok && len(signals) > 0 {
					// Pop the first signal from the queue
					signal := signals[0]
					workflow[signalName] = signals[1:]
					sc.mu.Unlock()
					return signal, nil
				}
			}
			sc.mu.Unlock()
		}
	}
}
