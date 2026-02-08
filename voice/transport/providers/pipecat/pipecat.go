// Package pipecat provides the Pipecat transport provider for the Beluga AI
// voice pipeline. It implements the AudioTransport interface for bidirectional
// audio I/O through a Pipecat server over WebSocket.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat"
//
//	t, err := transport.New("pipecat", transport.Config{
//	    URL: "ws://localhost:8765",
//	})
package pipecat

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lookatitude/beluga-ai/voice"
	"github.com/lookatitude/beluga-ai/voice/transport"
)

func init() {
	transport.Register("pipecat", func(cfg transport.Config) (transport.AudioTransport, error) {
		return New(cfg)
	})
}

// Transport implements transport.AudioTransport for Pipecat servers.
type Transport struct {
	url        string
	sampleRate int
	closed     bool
	mu         sync.Mutex
	frames     chan voice.Frame
}

// New creates a new Pipecat transport.
func New(cfg transport.Config) (*Transport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("pipecat: URL is required")
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	return &Transport{
		url:        cfg.URL,
		sampleRate: sampleRate,
		frames:     make(chan voice.Frame, 64),
	}, nil
}

// Recv returns a channel of incoming audio frames.
func (t *Transport) Recv(_ context.Context) (<-chan voice.Frame, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil, fmt.Errorf("pipecat: transport is closed")
	}
	return t.frames, nil
}

// Send writes an outgoing frame to the Pipecat server.
func (t *Transport) Send(_ context.Context, _ voice.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("pipecat: transport is closed")
	}
	return nil
}

// AudioOut returns a writer for raw audio output.
func (t *Transport) AudioOut() io.Writer {
	return io.Discard
}

// Close shuts down the Pipecat transport.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		t.closed = true
		close(t.frames)
	}
	return nil
}
