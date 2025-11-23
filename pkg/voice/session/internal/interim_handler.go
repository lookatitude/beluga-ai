package internal

import (
	"context"
	"sync"
)

// InterimHandler manages handling of interim transcripts
type InterimHandler struct {
	mu           sync.RWMutex
	handler      func(transcript string)
	lastInterim  string
	interimCount int
}

// NewInterimHandler creates a new interim handler
func NewInterimHandler(handler func(transcript string)) *InterimHandler {
	return &InterimHandler{
		handler:      handler,
		interimCount: 0,
	}
}

// Handle processes an interim transcript
func (ih *InterimHandler) Handle(ctx context.Context, transcript string) {
	ih.mu.Lock()
	ih.lastInterim = transcript
	ih.interimCount++
	handler := ih.handler
	ih.mu.Unlock()

	if handler != nil {
		handler(transcript)
	}
}

// GetLastInterim returns the last interim transcript
func (ih *InterimHandler) GetLastInterim() string {
	ih.mu.RLock()
	defer ih.mu.RUnlock()
	return ih.lastInterim
}

// GetInterimCount returns the number of interim transcripts received
func (ih *InterimHandler) GetInterimCount() int {
	ih.mu.RLock()
	defer ih.mu.RUnlock()
	return ih.interimCount
}

// Reset resets the interim handler state
func (ih *InterimHandler) Reset() {
	ih.mu.Lock()
	defer ih.mu.Unlock()
	ih.lastInterim = ""
	ih.interimCount = 0
}
