package transport

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lookatitude/beluga-ai/voice"
)

// WSOption configures a WebSocketTransport.
type WSOption func(*wsConfig)

type wsConfig struct {
	sampleRate int
	channels   int
}

// WithWSSampleRate sets the audio sample rate for the WebSocket transport.
func WithWSSampleRate(rate int) WSOption {
	return func(cfg *wsConfig) {
		cfg.sampleRate = rate
	}
}

// WithWSChannels sets the number of audio channels for the WebSocket transport.
func WithWSChannels(channels int) WSOption {
	return func(cfg *wsConfig) {
		cfg.channels = channels
	}
}

// WebSocketTransport is a stub implementation of AudioTransport that
// communicates over WebSocket. Full implementation will use gorilla/websocket
// or nhooyr.io/websocket.
type WebSocketTransport struct {
	url    string
	config wsConfig
	closed bool
	mu     sync.Mutex
}

// NewWebSocketTransport creates a new WebSocketTransport for the given URL.
func NewWebSocketTransport(url string, opts ...WSOption) *WebSocketTransport {
	cfg := wsConfig{
		sampleRate: 16000,
		channels:   1,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &WebSocketTransport{
		url:    url,
		config: cfg,
	}
}

// Recv returns a channel of incoming audio frames from the WebSocket.
// Stub implementation returns a closed channel.
func (t *WebSocketTransport) Recv(_ context.Context) (<-chan voice.Frame, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil, fmt.Errorf("transport: websocket transport is closed")
	}
	ch := make(chan voice.Frame)
	close(ch) // stub: no frames
	return ch, nil
}

// Send writes an outgoing frame to the client over WebSocket.
// Stub implementation is a no-op.
func (t *WebSocketTransport) Send(_ context.Context, _ voice.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport: websocket transport is closed")
	}
	return nil
}

// AudioOut returns a writer for raw audio output.
// Stub implementation returns a discard writer.
func (t *WebSocketTransport) AudioOut() io.Writer {
	return io.Discard
}

// Close shuts down the WebSocket connection.
func (t *WebSocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

func init() {
	Register("websocket", func(cfg Config) (AudioTransport, error) {
		var opts []WSOption
		if cfg.SampleRate > 0 {
			opts = append(opts, WithWSSampleRate(cfg.SampleRate))
		}
		if cfg.Channels > 0 {
			opts = append(opts, WithWSChannels(cfg.Channels))
		}
		return NewWebSocketTransport(cfg.URL, opts...), nil
	})
}
