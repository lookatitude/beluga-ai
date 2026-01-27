package internal

import (
	"sync"
)

// ResponseStrategyManager manages configurable response strategies.
type ResponseStrategyManager struct {
	mu       sync.RWMutex
	strategy ResponseStrategy
}

// NewResponseStrategyManager creates a new response strategy manager.
func NewResponseStrategyManager(strategy ResponseStrategy) *ResponseStrategyManager {
	return &ResponseStrategyManager{
		strategy: strategy,
	}
}

// GetStrategy returns the current response strategy.
func (rsm *ResponseStrategyManager) GetStrategy() ResponseStrategy {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()
	return rsm.strategy
}

// SetStrategy sets the response strategy.
func (rsm *ResponseStrategyManager) SetStrategy(strategy ResponseStrategy) {
	rsm.mu.Lock()
	defer rsm.mu.Unlock()
	rsm.strategy = strategy
}

// ShouldUsePreemptive determines if a preemptive response should be used.
func (rsm *ResponseStrategyManager) ShouldUsePreemptive(interim, final string) bool {
	rsm.mu.RLock()
	strategy := rsm.strategy
	rsm.mu.RUnlock()

	switch strategy {
	case ResponseStrategyDiscard:
		return false
	case ResponseStrategyUseIfSimilar:
		// Use the calculateStringSimilarity from final_handler.go
		similarity := calculateStringSimilarity(interim, final)
		return similarity > 0.8 // 80% similarity threshold
	case ResponseStrategyAlwaysUse:
		return true
	default:
		return false
	}
}
