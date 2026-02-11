package transport

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/lookatitude/beluga-ai/voice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAudioTransport is a test implementation of the AudioTransport interface.
type mockAudioTransport struct {
	recvFunc     func(context.Context) (<-chan voice.Frame, error)
	sendFunc     func(context.Context, voice.Frame) error
	audioOutFunc func() io.Writer
	closeFunc    func() error
	closed       bool
}

// Compile-time interface check.
var _ AudioTransport = (*mockAudioTransport)(nil)

func (m *mockAudioTransport) Recv(ctx context.Context) (<-chan voice.Frame, error) {
	if m.recvFunc != nil {
		return m.recvFunc(ctx)
	}
	ch := make(chan voice.Frame)
	close(ch)
	return ch, nil
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
		recvFunc: func(ctx context.Context) (<-chan voice.Frame, error) {
			ch := make(chan voice.Frame, len(expectedFrames))
			for _, f := range expectedFrames {
				ch <- f
			}
			close(ch)
			return ch, nil
		},
	}

	ch, err := transport.Recv(context.Background())
	require.NoError(t, err)

	var receivedFrames []voice.Frame
	for frame := range ch {
		receivedFrames = append(receivedFrames, frame)
	}

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
		recvFunc: func(ctx context.Context) (<-chan voice.Frame, error) {
			ch := make(chan voice.Frame, 1)
			ch <- voice.NewAudioFrame([]byte{0x01}, 16000)
			close(ch)
			return ch, nil
		},
	}

	wrapped := &AsVoiceTransport{T: mockTransport}

	ch, err := wrapped.Recv(context.Background())
	require.NoError(t, err)

	frame := <-ch
	assert.Equal(t, voice.FrameAudio, frame.Type)
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
		recvFunc: func(ctx context.Context) (<-chan voice.Frame, error) {
			return nil, assert.AnError
		},
	}

	ch, err := transport.Recv(context.Background())
	require.Error(t, err)
	assert.Nil(t, ch)
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

// --- WebSocketTransport tests ---

func TestWebSocketTransport_NewDefault(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")

	assert.Equal(t, "ws://localhost:9000", ws.url)
	assert.Equal(t, 16000, ws.config.sampleRate)
	assert.Equal(t, 1, ws.config.channels)
	assert.False(t, ws.closed)
}

func TestWebSocketTransport_NewWithOptions(t *testing.T) {
	ws := NewWebSocketTransport("ws://example.com/audio",
		WithWSSampleRate(44100),
		WithWSChannels(2),
	)

	assert.Equal(t, "ws://example.com/audio", ws.url)
	assert.Equal(t, 44100, ws.config.sampleRate)
	assert.Equal(t, 2, ws.config.channels)
}

func TestWebSocketTransport_Recv(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")

	ch, err := ws.Recv(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ch)

	// The stub implementation closes the channel immediately, so range should exit.
	var frames []voice.Frame
	for f := range ch {
		frames = append(frames, f)
	}
	assert.Empty(t, frames)
}

func TestWebSocketTransport_RecvClosed(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")
	require.NoError(t, ws.Close())

	ch, err := ws.Recv(context.Background())
	require.Error(t, err)
	assert.Nil(t, ch)
	assert.Contains(t, err.Error(), "closed")
}

func TestWebSocketTransport_Send(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")

	err := ws.Send(context.Background(), voice.NewAudioFrame([]byte{0x01, 0x02}, 16000))
	require.NoError(t, err)

	err = ws.Send(context.Background(), voice.NewTextFrame("hello"))
	require.NoError(t, err)
}

func TestWebSocketTransport_SendClosed(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")
	require.NoError(t, ws.Close())

	err := ws.Send(context.Background(), voice.NewAudioFrame([]byte{0x01}, 16000))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestWebSocketTransport_AudioOut(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")

	writer := ws.AudioOut()
	assert.Equal(t, io.Discard, writer)
}

func TestWebSocketTransport_Close(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")
	assert.False(t, ws.closed)

	err := ws.Close()
	require.NoError(t, err)
	assert.True(t, ws.closed)
}

func TestWebSocketTransport_CloseIdempotent(t *testing.T) {
	ws := NewWebSocketTransport("ws://localhost:9000")

	err := ws.Close()
	require.NoError(t, err)
	assert.True(t, ws.closed)

	// Closing again should not error.
	err = ws.Close()
	require.NoError(t, err)
	assert.True(t, ws.closed)
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

	// Create a WebSocket transport via the registry with config options.
	transport, err := New("websocket", Config{
		URL:        "ws://integration-test:8080",
		SampleRate: 48000,
		Channels:   2,
	})
	require.NoError(t, err)
	require.NotNil(t, transport)
	defer transport.Close()

	// Verify it's a *WebSocketTransport with the expected config.
	ws, ok := transport.(*WebSocketTransport)
	require.True(t, ok, "expected *WebSocketTransport from registry")
	assert.Equal(t, "ws://integration-test:8080", ws.url)
	assert.Equal(t, 48000, ws.config.sampleRate)
	assert.Equal(t, 2, ws.config.channels)

	// Verify the interface methods work.
	ch, err := transport.Recv(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ch)

	err = transport.Send(context.Background(), voice.NewTextFrame("test"))
	require.NoError(t, err)

	writer := transport.AudioOut()
	assert.Equal(t, io.Discard, writer)

	err = transport.Close()
	require.NoError(t, err)

	// After close, Send and Recv should fail.
	_, err = transport.Recv(context.Background())
	require.Error(t, err)

	err = transport.Send(context.Background(), voice.NewTextFrame("fail"))
	require.Error(t, err)
}
