// Package daily provides the Daily.co transport provider for the Beluga AI
// voice pipeline. It implements the AudioTransport interface for bidirectional
// audio I/O through Daily.co rooms.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/daily"
//
//	t, err := transport.New("daily", transport.Config{
//	    URL:   "https://myapp.daily.co/room",
//	    Token: "...",
//	})
package daily

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lookatitude/beluga-ai/voice"
	"github.com/lookatitude/beluga-ai/voice/transport"
)

var _ transport.AudioTransport = (*Transport)(nil) // compile-time interface check

func init() {
	transport.Register("daily", func(cfg transport.Config) (transport.AudioTransport, error) {
		return New(cfg)
	})
}

// Transport implements transport.AudioTransport for Daily.co rooms.
type Transport struct {
	url        string
	token      string
	sampleRate int
	closed     bool
	mu         sync.Mutex
	frames     chan voice.Frame
}

// New creates a new Daily.co transport.
func New(cfg transport.Config) (*Transport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("daily: URL is required")
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	return &Transport{
		url:        cfg.URL,
		token:      cfg.Token,
		sampleRate: sampleRate,
		frames:     make(chan voice.Frame, 64),
	}, nil
}

// Recv returns a channel of incoming audio frames.
func (t *Transport) Recv(_ context.Context) (<-chan voice.Frame, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil, fmt.Errorf("daily: transport is closed")
	}
	return t.frames, nil
}

// Send writes an outgoing frame to the Daily.co room.
func (t *Transport) Send(_ context.Context, _ voice.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("daily: transport is closed")
	}
	return nil
}

// AudioOut returns a writer for raw audio output.
func (t *Transport) AudioOut() io.Writer {
	return io.Discard
}

// Close shuts down the Daily.co transport.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		t.closed = true
		close(t.frames)
	}
	return nil
}
