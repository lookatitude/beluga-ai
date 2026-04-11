package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/voice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAudioTransport is a test implementation of the AudioTransport interface.
type mockAudioTransport struct {
	recvFunc     func(context.Context) iter.Seq2[voice.Frame, error]
	sendFunc     func(context.Context, voice.Frame) error
	audioOutFunc func() io.Writer
	closeFunc    func() error
	closed       bool
}

// Compile-time interface check.
var _ AudioTransport = (*mockAudioTransport)(nil)

func (m *mockAudioTransport) Recv(ctx context.Context) iter.Seq2[voice.Frame, error] {
	if m.recvFunc != nil {
		return m.recvFunc(ctx)
	}
	return func(yield func(voice.Frame, error) bool) {
		// Intentionally empty: default mock produces no frames.
		// Tests that need frames override recvFunc.
	}
}

// drainFrames collects all frames from an iter.Seq2, stopping at the first error.
func drainFrames(stream iter.Seq2[voice.Frame, error]) ([]voice.Frame, error) {
	var frames []voice.Frame
	for f, err := range stream {
		if err != nil {
			return frames, err
		}
		frames = append(frames, f)
	}
	return frames, nil
}

func (m *mockAudioTransport) Send(ctx context.Context, frame voice.Frame) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, frame)
	}
	return nil
}

func (m *mockAudioTransport) AudioOut() io.Writer {
	if m.audioOutFunc != nil {
		return m.audioOutFunc()
	}
	return &bytes.Buffer{}
}

func (m *mockAudioTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	m.closed = true
	return nil
}

func TestRegistry_RegisterAndNew(t *testing.T) {
	// Register a mock provider.
	Register("mock-transport", func(cfg Config) (AudioTransport, error) {
		return &mockAudioTransport{}, nil
	})

	// Create a transport using the registered provider.
	transport, err := New("mock-transport", Config{URL: "ws://localhost:8080"})
	require.NoError(t, err)
	require.NotNil(t, transport)
	defer transport.Close()
}

func TestRegistry_UnknownProvider(t *testing.T) {
	_, err := New("nonexistent-transport-provider", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestList(t *testing.T) {
	// Register a test provider.
	Register("test-transport-list", func(cfg Config) (AudioTransport, error) {
		return &mockAudioTransport{}, nil
	})

	names := List()
	require.NotEmpty(t, names)

	// Verify the list is sorted and contains our provider.
	assert.Contains(t, names, "test-transport-list")
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "list should be sorted")
	}
}

func TestConfig_WithOptions(t *testing.T) {
	cfg := Config{}

	WithURL("ws://example.com")(&cfg)
	assert.Equal(t, "ws://example.com", cfg.URL)

	WithToken("secret-token")(&cfg)
	assert.Equal(t, "secret-token", cfg.Token)

	WithSampleRate(16000)(&cfg)
	assert.Equal(t, 16000, cfg.SampleRate)

	WithChannels(2)(&cfg)
	assert.Equal(t, 2, cfg.Channels)
}

func TestMockAudioTransport_Recv(t *testing.T) {
	expectedFrames := []voice.Frame{
		voice.NewAudioFrame([]byte{0x01, 0x02}, 16000),
		voice.NewTextFrame("test"),
	}

	transport := &mockAudioTransport{
		recvFunc: func(ctx context.Context) iter.Seq2[voice.Frame, error] {
			return func(yield func(voice.Frame, error) bool) {
				for _, f := range expectedFrames {
					if !yield(f, nil) {
						return
					}
				}
			}
		},
	}

	receivedFrames, err := drainFrames(transport.Recv(context.Background()))
	require.NoError(t, err)
	require.Len(t, receivedFrames, 2)
	assert.Equal(t, voice.FrameAudio, receivedFrames[0].Type)
	assert.Equal(t, voice.FrameText, receivedFrames[1].Type)
}

func TestMockAudioTransport_Send(t *testing.T) {
	var sentFrames []voice.Frame

	transport := &mockAudioTransport{
		sendFunc: func(ctx context.Context, frame voice.Frame) error {
			sentFrames = append(sentFrames, frame)
			return nil
		},
	}

	err := transport.Send(context.Background(), voice.NewAudioFrame([]byte{0x01}, 16000))
	require.NoError(t, err)

	err = transport.Send(context.Background(), voice.NewTextFrame("hello"))
	require.NoError(t, err)

	require.Len(t, sentFrames, 2)
	assert.Equal(t, voice.FrameAudio, sentFrames[0].Type)
	assert.Equal(t, voice.FrameText, sentFrames[1].Type)
}

func TestMockAudioTransport_AudioOut(t *testing.T) {
	buf := &bytes.Buffer{}

	transport := &mockAudioTransport{
		audioOutFunc: func() io.Writer {
			return buf
		},
	}

	writer := transport.AudioOut()
	n, err := writer.Write([]byte{0xAA, 0xBB, 0xCC})
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte{0xAA, 0xBB, 0xCC}, buf.Bytes())
}

func TestMockAudioTransport_Close(t *testing.T) {
	transport := &mockAudioTransport{}

	assert.False(t, transport.closed)

	err := transport.Close()
	require.NoError(t, err)
	assert.True(t, transport.closed)
}

func TestAsVoiceTransport_Recv(t *testing.T) {
	mockTransport := &mockAudioTransport{
		recvFunc: func(ctx context.Context) iter.Seq2[voice.Frame, error] {
			return func(yield func(voice.Frame, error) bool) {
				yield(voice.NewAudioFrame([]byte{0x01}, 16000), nil)
			}
		},
	}

	wrapped := &AsVoiceTransport{T: mockTransport}

	frames, err := drainFrames(wrapped.Recv(context.Background()))
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Equal(t, voice.FrameAudio, frames[0].Type)
}

func TestAsVoiceTransport_Send(t *testing.T) {
	var sentFrame voice.Frame

	mockTransport := &mockAudioTransport{
		sendFunc: func(ctx context.Context, frame voice.Frame) error {
			sentFrame = frame
			return nil
		},
	}

	wrapped := &AsVoiceTransport{T: mockTransport}

	testFrame := voice.NewTextFrame("test")
	err := wrapped.Send(context.Background(), testFrame)
	require.NoError(t, err)
	assert.Equal(t, voice.FrameText, sentFrame.Type)
	assert.Equal(t, "test", sentFrame.Text())
}

func TestAsVoiceTransport_Close(t *testing.T) {
	mockTransport := &mockAudioTransport{}

	wrapped := &AsVoiceTransport{T: mockTransport}

	err := wrapped.Close()
	require.NoError(t, err)
	assert.True(t, mockTransport.closed)
}

func TestConfig_Extra(t *testing.T) {
	cfg := Config{
		URL:        "ws://localhost:8080",
		Token:      "token123",
		SampleRate: 16000,
		Channels:   1,
		Extra: map[string]any{
			"custom_field": "value",
			"numeric":      42,
		},
	}

	assert.Equal(t, "ws://localhost:8080", cfg.URL)
	assert.Equal(t, "token123", cfg.Token)
	assert.Equal(t, 16000, cfg.SampleRate)
	assert.Equal(t, 1, cfg.Channels)
	require.NotNil(t, cfg.Extra)
	assert.Equal(t, "value", cfg.Extra["custom_field"])
	assert.Equal(t, 42, cfg.Extra["numeric"])
}

func TestMockAudioTransport_RecvError(t *testing.T) {
	transport := &mockAudioTransport{
		recvFunc: func(ctx context.Context) iter.Seq2[voice.Frame, error] {
			return func(yield func(voice.Frame, error) bool) {
				yield(voice.Frame{}, assert.AnError)
			}
		},
	}

	_, err := drainFrames(transport.Recv(context.Background()))
	require.Error(t, err)
}

func TestMockAudioTransport_SendError(t *testing.T) {
	transport := &mockAudioTransport{
		sendFunc: func(ctx context.Context, frame voice.Frame) error {
			return assert.AnError
		},
	}

	err := transport.Send(context.Background(), voice.NewAudioFrame([]byte{0x01}, 16000))
	require.Error(t, err)
}

func TestMockAudioTransport_CloseError(t *testing.T) {
	transport := &mockAudioTransport{
		closeFunc: func() error {
			return assert.AnError
		},
	}

	err := transport.Close()
	require.Error(t, err)
}

// --- WebSocketTransport test helpers ---

// newWSTestServer creates an httptest server that upgrades to WebSocket and
// runs handler on the accepted connection.
func newWSTestServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Logf("websocket accept error: %v", err)
			return
		}
		handler(conn)
	}))
}

// wsURL converts an httptest server URL to a ws:// URL.
func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// --- WebSocketTransport tests ---

func TestWebSocketTransport_ConnectSuccess(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		// Keep connection alive until client closes.
		conn.Read(context.Background())
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)
	require.NotNil(t, ws)
	defer ws.Close()

	assert.Equal(t, wsURL(srv), ws.url)
	assert.Equal(t, 16000, ws.config.sampleRate)
	assert.Equal(t, 1, ws.config.channels)
}

func TestWebSocketTransport_ConnectWithOptions(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Read(context.Background())
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv),
		WithWSSampleRate(44100),
		WithWSChannels(2),
		WithWSBufferSize(128),
		WithWSReadLimit(2<<20),
		WithWSWriteTimeout(10*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, ws)
	defer ws.Close()

	assert.Equal(t, 44100, ws.config.sampleRate)
	assert.Equal(t, 2, ws.config.channels)
	assert.Equal(t, 128, ws.config.bufferSize)
	assert.Equal(t, int64(2<<20), ws.config.readLimit)
	assert.Equal(t, 10*time.Second, ws.config.writeTimeout)
}

func TestWebSocketTransport_URLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"empty URL", "", "must not be empty"},
		{"http scheme", "http://localhost:8080", "must use ws:// or wss://"},
		{"https scheme", "https://localhost:8080", "must use ws:// or wss://"},
		{"no scheme", "localhost:8080", "must use ws:// or wss://"},
		{"ftp scheme", "ftp://localhost:8080", "must use ws:// or wss://"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWebSocketTransport(context.Background(), tt.url)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestWebSocketTransport_BufferSizeClamping(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Read(context.Background())
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv),
		WithWSBufferSize(-1),
		WithWSReadLimit(-100),
	)
	require.NoError(t, err)
	defer ws.Close()

	assert.Equal(t, 64, ws.config.bufferSize, "negative bufferSize should be clamped to default")
	assert.Equal(t, int64(1<<20), ws.config.readLimit, "negative readLimit should be clamped to default")
}

func TestWebSocketTransport_ConnectDialError(t *testing.T) {
	_, err := NewWebSocketTransport(context.Background(), "ws://127.0.0.1:1/bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "websocket dial")
}

func TestWebSocketTransport_SendRecvAudioRoundTrip(t *testing.T) {
	// Echo server: reads one binary message and sends it back.
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		mt, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		conn.Write(ctx, mt, data)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	ctx := context.Background()
	ws, err := NewWebSocketTransport(ctx, wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	// Send an audio frame.
	audioData := []byte{0x01, 0x02, 0x03, 0x04}
	err = ws.Send(ctx, voice.NewAudioFrame(audioData, 16000))
	require.NoError(t, err)

	// Receive the echoed frame via iterator, delivered on a channel so we
	// can enforce a test timeout.
	frameCh := pumpFirstFrame(ctx, ws)
	select {
	case frame := <-frameCh:
		assert.Equal(t, voice.FrameAudio, frame.Type)
		assert.Equal(t, audioData, frame.Data)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for echoed audio frame")
	}
}

// pumpFirstFrame consumes the first frame from ws.Recv(ctx) on a goroutine and
// delivers it via a channel. Used so tests can enforce their own timeout.
func pumpFirstFrame(ctx context.Context, ws *WebSocketTransport) <-chan voice.Frame {
	out := make(chan voice.Frame, 1)
	go func() {
		for f, err := range ws.Recv(ctx) {
			if err != nil {
				return
			}
			out <- f
			return
		}
	}()
	return out
}

func TestWebSocketTransport_SendRecvTextFrame(t *testing.T) {
	// Echo server: reads one text message and sends it back.
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		mt, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		conn.Write(ctx, mt, data)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	ctx := context.Background()
	ws, err := NewWebSocketTransport(ctx, wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	// Send a text frame.
	err = ws.Send(ctx, voice.NewTextFrame("hello world"))
	require.NoError(t, err)

	frameCh := pumpFirstFrame(ctx, ws)
	select {
	case frame := <-frameCh:
		assert.Equal(t, voice.FrameText, frame.Type)
		assert.Equal(t, "hello world", frame.Text())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for echoed text frame")
	}
}

func TestWebSocketTransport_SendControlFrame(t *testing.T) {
	// Server reads one text message, verifies it's a control frame, echoes it back.
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		mt, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		// Verify it was sent as text (JSON envelope).
		if mt != websocket.MessageText {
			return
		}
		conn.Write(ctx, mt, data)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	ctx := context.Background()
	ws, err := NewWebSocketTransport(ctx, wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	err = ws.Send(ctx, voice.NewControlFrame(voice.SignalInterrupt))
	require.NoError(t, err)

	frameCh := pumpFirstFrame(ctx, ws)
	select {
	case frame := <-frameCh:
		assert.Equal(t, voice.FrameControl, frame.Type)
		assert.Equal(t, voice.SignalInterrupt, frame.Signal())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for echoed control frame")
	}
}

func TestWebSocketTransport_RecvClosed(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Read(context.Background())
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)
	require.NoError(t, ws.Close())

	_, err = drainFrames(ws.Recv(context.Background()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestWebSocketTransport_SendClosed(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Read(context.Background())
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)
	require.NoError(t, ws.Close())

	err = ws.Send(context.Background(), voice.NewAudioFrame([]byte{0x01}, 16000))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestWebSocketTransport_CloseIdempotent(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Read(context.Background())
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)

	// First close should succeed.
	err = ws.Close()
	require.NoError(t, err)

	// Second close should not panic and should return nil (sync.Once).
	err = ws.Close()
	require.NoError(t, err)
}

func TestWebSocketTransport_ContextCancellation(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		// Keep connection alive.
		conn.Read(context.Background())
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ws, err := NewWebSocketTransport(ctx, wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	// Run the iterator on a goroutine; it should exit after cancel.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ws.Recv(ctx) {
			// discard
		}
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for readLoop to stop after context cancel")
	}
}

func TestWebSocketTransport_AudioOutWriter(t *testing.T) {
	// Server collects binary messages.
	received := make(chan []byte, 4)
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for {
			mt, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if mt == websocket.MessageBinary {
				received <- data
			}
		}
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	writer := ws.AudioOut()
	require.NotNil(t, writer)

	// AudioOut should return the same writer each time.
	assert.Equal(t, writer, ws.AudioOut())

	// Write audio data.
	data := []byte{0xAA, 0xBB, 0xCC}
	n, err := writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	// Verify the server received the binary message.
	select {
	case got := <-received:
		assert.Equal(t, data, got)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to receive audio data")
	}
}

func TestWebSocketTransport_WireFrameJSON(t *testing.T) {
	// Verify the wireFrame JSON format by having the server decode it.
	decoded := make(chan wireFrame, 1)
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var wf wireFrame
		if err := json.Unmarshal(data, &wf); err != nil {
			return
		}
		decoded <- wf
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	err = ws.Send(context.Background(), voice.NewTextFrame("wire format test"))
	require.NoError(t, err)

	select {
	case wf := <-decoded:
		assert.Equal(t, voice.FrameText, wf.Type)
		assert.Equal(t, "wire format test", wf.Text)
		assert.Nil(t, wf.Data, "text frames should not duplicate data")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to decode wireFrame")
	}
}

func TestWebSocketTransport_WithWSHeaders(t *testing.T) {
	var receivedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom-Auth")
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn.Read(context.Background())
	}))
	defer srv.Close()

	headers := http.Header{}
	headers.Set("X-Custom-Auth", "bearer-token-123")

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv),
		WithWSHeaders(headers),
	)
	require.NoError(t, err)
	defer ws.Close()

	assert.Equal(t, "bearer-token-123", receivedHeader)
}

// --- Registry panic tests ---

func TestRegistry_PanicEmptyName(t *testing.T) {
	assert.Panics(t, func() {
		Register("", func(cfg Config) (AudioTransport, error) {
			return nil, nil
		})
	})
}

func TestRegistry_PanicNilFactory(t *testing.T) {
	assert.Panics(t, func() {
		Register("nil-factory-test", nil)
	})
}

func TestRegistry_PanicDuplicate(t *testing.T) {
	// "websocket" is already registered via init() in websocket.go.
	assert.Panics(t, func() {
		Register("websocket", func(cfg Config) (AudioTransport, error) {
			return nil, nil
		})
	})
}

// --- WebSocket registry integration test ---

func TestWebSocketTransport_RegistryIntegration(t *testing.T) {
	// "websocket" is registered in websocket.go init().
	names := List()
	assert.Contains(t, names, "websocket")

	// Start a test server for the registry to connect to.
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Echo one message then close.
		mt, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		conn.Write(ctx, mt, data)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	// Create a WebSocket transport via the registry with config options.
	tr, err := New("websocket", Config{
		URL:        wsURL(srv),
		SampleRate: 48000,
		Channels:   2,
	})
	require.NoError(t, err)
	require.NotNil(t, tr)
	defer tr.Close()

	// Verify it's a *WebSocketTransport with the expected config.
	ws, ok := tr.(*WebSocketTransport)
	require.True(t, ok, "expected *WebSocketTransport from registry")
	assert.Equal(t, wsURL(srv), ws.url)
	assert.Equal(t, 48000, ws.config.sampleRate)
	assert.Equal(t, 2, ws.config.channels)

	// Verify the interface methods work: send and receive.
	// Start iterator in goroutine so we can enforce a timeout.
	frameCh := make(chan voice.Frame, 1)
	go func() {
		for f, err := range tr.Recv(context.Background()) {
			if err != nil {
				return
			}
			frameCh <- f
			return
		}
	}()

	err = tr.Send(context.Background(), voice.NewTextFrame("test"))
	require.NoError(t, err)

	select {
	case frame := <-frameCh:
		assert.Equal(t, voice.FrameText, frame.Type)
		assert.Equal(t, "test", frame.Text())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for echoed frame in registry integration test")
	}

	// AudioOut should return a writer (not io.Discard anymore).
	writer := tr.AudioOut()
	assert.NotNil(t, writer)

	err = tr.Close()
	require.NoError(t, err)

	// After close, Send and Recv should fail.
	_, rerr := drainFrames(tr.Recv(context.Background()))
	require.Error(t, rerr)

	err = tr.Send(context.Background(), voice.NewTextFrame("fail"))
	require.Error(t, err)
}

func TestWebSocketTransport_MalformedJSONResilience(t *testing.T) {
	// Server sends: malformed text, then valid binary audio, then closes.
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Send a malformed JSON text message.
		conn.Write(ctx, websocket.MessageText, []byte("not valid json {{{"))

		// Follow with a valid binary audio frame.
		conn.Write(ctx, websocket.MessageBinary, []byte{0xDE, 0xAD})

		// Give client time to receive, then close.
		time.Sleep(200 * time.Millisecond)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	ws, err := NewWebSocketTransport(context.Background(), wsURL(srv))
	require.NoError(t, err)
	defer ws.Close()

	frameCh := pumpFirstFrame(context.Background(), ws)
	// The transport should skip the malformed text and deliver the audio frame.
	select {
	case frame := <-frameCh:
		assert.Equal(t, voice.FrameAudio, frame.Type)
		assert.Equal(t, []byte{0xDE, 0xAD}, frame.Data)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for audio frame after malformed JSON")
	}
}
