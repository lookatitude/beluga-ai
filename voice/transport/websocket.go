package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/voice"
)

// Compile-time interface check.
var _ AudioTransport = (*WebSocketTransport)(nil)

// WSOption configures a WebSocketTransport.
type WSOption func(*wsConfig)

type wsConfig struct {
	sampleRate   int
	channels     int
	headers      http.Header
	pingInterval time.Duration
	readLimit    int64
	bufferSize   int
	writeTimeout time.Duration
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

// WithWSHeaders sets custom HTTP headers for the WebSocket handshake.
func WithWSHeaders(headers http.Header) WSOption {
	return func(cfg *wsConfig) {
		cfg.headers = headers
	}
}

// WithWSPingInterval sets the interval for WebSocket ping frames.
// A zero value disables pings.
func WithWSPingInterval(d time.Duration) WSOption {
	return func(cfg *wsConfig) {
		cfg.pingInterval = d
	}
}

// WithWSReadLimit sets the maximum size in bytes for incoming WebSocket messages.
// Default is 1MB.
func WithWSReadLimit(limit int64) WSOption {
	return func(cfg *wsConfig) {
		cfg.readLimit = limit
	}
}

// WithWSBufferSize sets the size of the internal receive channel buffer.
// Default is 64.
func WithWSBufferSize(size int) WSOption {
	return func(cfg *wsConfig) {
		cfg.bufferSize = size
	}
}

// WithWSWriteTimeout sets the timeout for write operations.
// Default is 5 seconds.
func WithWSWriteTimeout(d time.Duration) WSOption {
	return func(cfg *wsConfig) {
		cfg.writeTimeout = d
	}
}

// wireFrame is the JSON envelope for non-audio WebSocket messages.
type wireFrame struct {
	Type     voice.FrameType `json:"type"`
	Data     []byte          `json:"data,omitempty"`
	Text     string          `json:"text,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// WebSocketTransport implements AudioTransport over a WebSocket connection.
// Binary messages carry raw audio bytes (hot path, no JSON overhead).
// Text messages carry JSON-encoded wireFrame envelopes for non-audio frames.
type WebSocketTransport struct {
	url       string
	config    wsConfig
	conn      *websocket.Conn
	frames    chan voice.Frame
	done      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex // guards writes to conn
	audioOut  io.Writer  // cached writer from AudioOut()
	err       error      // first error encountered
}

// NewWebSocketTransport dials a WebSocket at the given URL and returns a
// connected transport. The readLoop goroutine is started automatically.
//
// The provided context governs the entire connection lifetime: when ctx is
// cancelled the read loop exits and the transport becomes unusable. Callers
// should pass a context that lives as long as the desired connection.
//
// The URL must use the ws:// or wss:// scheme. Do not embed credentials
// (userinfo) in the URL; use WithWSHeaders to pass authentication tokens.
func NewWebSocketTransport(ctx context.Context, url string, opts ...WSOption) (*WebSocketTransport, error) {
	if url == "" {
		return nil, fmt.Errorf("transport: websocket URL must not be empty")
	}
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		return nil, fmt.Errorf("transport: websocket URL must use ws:// or wss:// scheme")
	}

	cfg := wsConfig{
		sampleRate:   16000,
		channels:     1,
		readLimit:    1 << 20, // 1MB
		bufferSize:   64,
		writeTimeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Clamp invalid values to safe defaults.
	if cfg.bufferSize <= 0 {
		cfg.bufferSize = 64
	}
	if cfg.readLimit <= 0 {
		cfg.readLimit = 1 << 20
	}

	dialOpts := &websocket.DialOptions{}
	if cfg.headers != nil {
		dialOpts.HTTPHeader = cfg.headers
	}

	conn, _, err := websocket.Dial(ctx, url, dialOpts)
	if err != nil {
		return nil, fmt.Errorf("transport: websocket dial %q: %w", url, err)
	}

	conn.SetReadLimit(cfg.readLimit)

	t := &WebSocketTransport{
		url:    url,
		config: cfg,
		conn:   conn,
		frames: make(chan voice.Frame, cfg.bufferSize),
		done:   make(chan struct{}),
	}

	go t.readLoop(ctx)

	return t, nil
}

// readLoop reads messages from the WebSocket connection and dispatches them
// to the frames channel. It exits on error, context cancellation, or when
// the done channel is closed.
func (t *WebSocketTransport) readLoop(ctx context.Context) {
	defer close(t.frames)

	for {
		select {
		case <-t.done:
			return
		default:
		}

		msgType, data, err := t.conn.Read(ctx)
		if err != nil {
			// Store first error for diagnostics.
			t.mu.Lock()
			if t.err == nil {
				t.err = err
			}
			t.mu.Unlock()
			return
		}

		var frame voice.Frame

		switch msgType {
		case websocket.MessageBinary:
			frame = voice.NewAudioFrame(data, t.config.sampleRate)
		case websocket.MessageText:
			var wf wireFrame
			if err := json.Unmarshal(data, &wf); err != nil {
				// Skip malformed text messages.
				continue
			}
			frame = voice.Frame{
				Type:     wf.Type,
				Data:     wf.Data,
				Metadata: wf.Metadata,
			}
			// For text frames, prefer the Text field if Data is empty.
			if wf.Type == voice.FrameText && len(wf.Data) == 0 && wf.Text != "" {
				frame.Data = []byte(wf.Text)
			}
		default:
			continue
		}

		select {
		case t.frames <- frame:
		case <-t.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Recv returns a channel of incoming frames from the WebSocket connection.
func (t *WebSocketTransport) Recv(_ context.Context) (<-chan voice.Frame, error) {
	select {
	case <-t.done:
		return nil, fmt.Errorf("transport: websocket transport is closed")
	default:
		return t.frames, nil
	}
}

// Send writes an outgoing frame to the WebSocket connection.
// Audio frames are sent as binary messages for efficiency.
// All other frame types are JSON-encoded as text messages.
func (t *WebSocketTransport) Send(ctx context.Context, frame voice.Frame) error {
	select {
	case <-t.done:
		return fmt.Errorf("transport: websocket transport is closed")
	default:
	}

	if t.config.writeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.writeTimeout)
		defer cancel()
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if frame.Type == voice.FrameAudio {
		return t.conn.Write(ctx, websocket.MessageBinary, frame.Data)
	}

	wf := wireFrame{
		Type:     frame.Type,
		Data:     frame.Data,
		Metadata: frame.Metadata,
	}
	if frame.Type == voice.FrameText {
		wf.Text = string(frame.Data)
		wf.Data = nil // avoid sending the same data twice
	}

	data, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("transport: websocket marshal frame: %w", err)
	}
	return t.conn.Write(ctx, websocket.MessageText, data)
}

// wsAudioWriter implements io.Writer by sending binary WebSocket messages.
type wsAudioWriter struct {
	t *WebSocketTransport
}

func (w *wsAudioWriter) Write(p []byte) (int, error) {
	select {
	case <-w.t.done:
		return 0, fmt.Errorf("transport: websocket transport is closed")
	default:
	}

	ctx := context.Background()
	if w.t.config.writeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.t.config.writeTimeout)
		defer cancel()
	}

	w.t.mu.Lock()
	defer w.t.mu.Unlock()

	if err := w.t.conn.Write(ctx, websocket.MessageBinary, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// AudioOut returns an io.Writer that sends raw audio bytes as binary
// WebSocket messages. Because io.Writer does not accept a context, writes
// use context.Background() with the configured writeTimeout as a safety
// bound to prevent unbounded blocking.
func (t *WebSocketTransport) AudioOut() io.Writer {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.audioOut == nil {
		t.audioOut = &wsAudioWriter{t: t}
	}
	return t.audioOut
}

// Close gracefully shuts down the WebSocket transport. It is safe to call
// multiple times.
func (t *WebSocketTransport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.done)
		err = t.conn.Close(websocket.StatusNormalClosure, "")
	})
	return err
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

		// Extract context from Extra if provided, otherwise use background.
		ctx := context.Background()
		if cfg.Extra != nil {
			if c, ok := cfg.Extra["ctx"].(context.Context); ok {
				ctx = c
			}
		}

		// Extract headers from Extra if provided.
		if cfg.Extra != nil {
			if h, ok := cfg.Extra["headers"].(http.Header); ok {
				opts = append(opts, WithWSHeaders(h))
			}
		}

		return NewWebSocketTransport(ctx, cfg.URL, opts...)
	})
}
