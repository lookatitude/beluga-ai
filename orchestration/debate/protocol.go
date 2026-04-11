package debate

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// DebateProtocol determines how agents participate in each round of a debate.
// Implementations return a map of agent ID to prompt string for each round.
type DebateProtocol interface {
	// NextRound returns the prompts to send to each agent for the next round.
	// The keys are agent IDs, the values are the formatted prompts.
	NextRound(ctx context.Context, state DebateState) (map[string]string, error)
}

// TwoPassProtocol is an optional interface implemented by protocols that
// require a second evaluation pass after participant contributions are
// collected for the current round. Protocols that do not need to see the
// current round's contributions should implement only DebateProtocol.
//
// The orchestrator runs NextRound first, collects all responses into the
// partial Round, then calls FollowUp with the in-progress round. Returned
// prompts are dispatched to the listed agents and their responses are
// appended to the same Round's Contributions slice.
type TwoPassProtocol interface {
	DebateProtocol
	// FollowUp is called after the first-pass responses for the current
	// round have been collected. The implementation may inspect
	// currentRound.Contributions to build prompts for a second pass
	// (for example, a judge that evaluates the current round's arguments).
	// Returning an empty map signals no follow-up prompts for this round.
	FollowUp(ctx context.Context, state DebateState, currentRound Round) (map[string]string, error)
}

// ProtocolFactory creates a DebateProtocol from a configuration map.
type ProtocolFactory func(cfg map[string]any) (DebateProtocol, error)

var (
	protocolMu       sync.RWMutex
	protocolRegistry = make(map[string]ProtocolFactory)
)

// RegisterProtocol registers a named DebateProtocol factory.
// This should be called from init().
func RegisterProtocol(name string, f ProtocolFactory) {
	protocolMu.Lock()
	defer protocolMu.Unlock()
	protocolRegistry[name] = f
}

// NewProtocol creates a DebateProtocol by name from the registry.
func NewProtocol(name string, cfg map[string]any) (DebateProtocol, error) {
	protocolMu.RLock()
	f, ok := protocolRegistry[name]
	protocolMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("debate: unknown protocol %q (registered: %v)", name, ListProtocols())
	}
	return f(cfg)
}

// ListProtocols returns the names of all registered protocols in sorted order.
func ListProtocols() []string {
	protocolMu.RLock()
	defer protocolMu.RUnlock()
	names := make([]string, 0, len(protocolRegistry))
	for name := range protocolRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
