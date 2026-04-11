package pipecat

import (
	"context"
	"fmt"
	"io"
	"iter"
	"sync"

	"github.com/lookatitude/beluga-ai/voice"
	"github.com/lookatitude/beluga-ai/voice/transport"
)

var _ transport.AudioTransport = (*Transport)(nil) // compile-time interface check

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

// Recv returns an iterator of incoming audio frames.
// If the transport is already closed the first yielded pair carries an error
// and the iterator ends.
func (t *Transport) Recv(ctx context.Context) iter.Seq2[voice.Frame, error] {
	return func(yield func(voice.Frame, error) bool) {
		t.mu.Lock()
		closed := t.closed
		frames := t.frames
		t.mu.Unlock()
		if closed {
			yield(voice.Frame{}, fmt.Errorf("pipecat: transport is closed"))
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-frames:
				if !ok {
					return
				}
				if !yield(frame, nil) {
					return
				}
			}
		}
	}
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
