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
