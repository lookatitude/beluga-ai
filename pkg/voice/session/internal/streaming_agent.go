package internal

import (
	"context"
	"errors"
	"sync"
)

// StreamingAgent manages streaming agent response integration.
type StreamingAgent struct {
	agentCallback func(ctx context.Context, transcript string) (string, error)
	mu            sync.RWMutex
	streaming     bool
}

// NewStreamingAgent creates a new streaming agent manager.
func NewStreamingAgent(agentCallback func(ctx context.Context, transcript string) (string, error)) *StreamingAgent {
	return &StreamingAgent{
		agentCallback: agentCallback,
		streaming:     false,
	}
}

// StartStreaming starts streaming agent responses.
func (sa *StreamingAgent) StartStreaming(ctx context.Context, transcript string) (<-chan string, error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.streaming {
		return nil, errors.New("streaming already active")
	}

	if sa.agentCallback == nil {
		return nil, errors.New("agent callback not set")
	}

	responseCh := make(chan string, 10)
	sa.streaming = true

	// Start async response generation
	go func() {
		defer close(responseCh)
		defer func() {
			sa.mu.Lock()
			sa.streaming = false
			sa.mu.Unlock()
		}()

		response, err := sa.agentCallback(ctx, transcript)
		if err != nil {
			return
		}

		// In a real implementation, this would stream tokens/chunks
		// For now, send the complete response
		select {
		case <-ctx.Done():
			return
		case responseCh <- response:
		}
	}()

	return responseCh, nil
}

// StopStreaming stops streaming agent responses.
func (sa *StreamingAgent) StopStreaming() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.streaming = false
}

// IsStreaming returns whether streaming is active.
func (sa *StreamingAgent) IsStreaming() bool {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.streaming
}

// TODO: In a real implementation, this would integrate with the actual agent package
// to provide token-by-token streaming, tool calling, function execution, etc.
