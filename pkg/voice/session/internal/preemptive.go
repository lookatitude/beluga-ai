package internal

import (
	"context"
	"sync"
)

// PreemptiveGeneration manages preemptive generation logic
type PreemptiveGeneration struct {
	mu               sync.RWMutex
	enabled          bool
	interimHandler   func(transcript string)
	finalHandler     func(transcript string)
	responseStrategy ResponseStrategy
}

// ResponseStrategy defines how to handle preemptive responses
type ResponseStrategy int

const (
	ResponseStrategyDiscard ResponseStrategy = iota
	ResponseStrategyUseIfSimilar
	ResponseStrategyAlwaysUse
)

// NewPreemptiveGeneration creates a new preemptive generation manager
func NewPreemptiveGeneration(enabled bool, strategy ResponseStrategy) *PreemptiveGeneration {
	return &PreemptiveGeneration{
		enabled:          enabled,
		responseStrategy: strategy,
	}
}

// SetInterimHandler sets the handler for interim transcripts
func (pg *PreemptiveGeneration) SetInterimHandler(handler func(transcript string)) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.interimHandler = handler
}

// SetFinalHandler sets the handler for final transcripts
func (pg *PreemptiveGeneration) SetFinalHandler(handler func(transcript string)) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.finalHandler = handler
}

// HandleInterim handles an interim transcript
func (pg *PreemptiveGeneration) HandleInterim(ctx context.Context, transcript string) {
	if !pg.enabled {
		return
	}

	pg.mu.RLock()
	handler := pg.interimHandler
	pg.mu.RUnlock()

	if handler != nil {
		handler(transcript)
	}
}

// HandleFinal handles a final transcript
func (pg *PreemptiveGeneration) HandleFinal(ctx context.Context, transcript string) {
	pg.mu.RLock()
	handler := pg.finalHandler
	strategy := pg.responseStrategy
	pg.mu.RUnlock()

	if handler != nil {
		handler(transcript)
	}

	// Apply response strategy
	switch strategy {
	case ResponseStrategyDiscard:
		// Discard preemptive response if final differs
		// Implementation would compare and discard if needed
	case ResponseStrategyUseIfSimilar:
		// Use preemptive response if similar to final
		// Implementation would compare similarity
	case ResponseStrategyAlwaysUse:
		// Always use preemptive response
		// No additional action needed
	}
}

// GetResponseStrategy returns the current response strategy
func (pg *PreemptiveGeneration) GetResponseStrategy() ResponseStrategy {
	pg.mu.RLock()
	defer pg.mu.RUnlock()
	return pg.responseStrategy
}
