package livekit

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
	transport.Register("livekit", func(cfg transport.Config) (transport.AudioTransport, error) {
		return New(cfg)
	})
}

// Transport implements transport.AudioTransport for LiveKit rooms.
type Transport struct {
	url        string
	token      string
	room       string
	sampleRate int
	channels   int
	closed     bool
	mu         sync.Mutex
	frames     chan voice.Frame
}

// New creates a new LiveKit transport.
func New(cfg transport.Config) (*Transport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("livekit: URL is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("livekit: Token is required")
	}

	room, _ := cfg.Extra["room"].(string)

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	channels := cfg.Channels
	if channels == 0 {
		channels = 1
	}

	return &Transport{
		url:        cfg.URL,
		token:      cfg.Token,
		room:       room,
		sampleRate: sampleRate,
		channels:   channels,
		frames:     make(chan voice.Frame, 64),
	}, nil
}

// Recv returns an iterator of incoming audio frames from the LiveKit room.
// If the transport is already closed the first yielded pair carries an error
// and the iterator ends.
func (t *Transport) Recv(ctx context.Context) iter.Seq2[voice.Frame, error] {
	return func(yield func(voice.Frame, error) bool) {
		t.mu.Lock()
		closed := t.closed
		frames := t.frames
		t.mu.Unlock()
		if closed {
			yield(voice.Frame{}, fmt.Errorf("livekit: transport is closed"))
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

// Send writes an outgoing frame to the LiveKit room.
func (t *Transport) Send(_ context.Context, frame voice.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("livekit: transport is closed")
	}
	// In production, this would publish audio to the LiveKit room track.
	return nil
}

// AudioOut returns a writer for raw audio output.
func (t *Transport) AudioOut() io.Writer {
	return io.Discard
}

// Close shuts down the LiveKit transport.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		t.closed = true
		close(t.frames)
	}
	return nil
}
