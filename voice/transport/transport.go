package transport

import (
	"context"
	"io"
	"iter"

	"github.com/lookatitude/beluga-ai/v2/voice"
)

// AudioTransport is the interface for bidirectional audio I/O between clients
// and the voice pipeline. Implementations handle the underlying transport
// protocol (WebSocket, WebRTC via LiveKit/Daily, etc.).
type AudioTransport interface {
	// Recv returns an iterator of incoming audio frames from the remote client.
	// Transport-level errors are delivered via the iterator's second element;
	// a non-nil error terminates the stream. The iterator ends when the
	// connection closes or ctx is cancelled.
	Recv(ctx context.Context) iter.Seq2[voice.Frame, error]

	// Send writes an outgoing frame to the remote client.
	Send(ctx context.Context, frame voice.Frame) error

	// AudioOut returns a writer for raw audio output. This is useful for
	// piping synthesized audio directly without frame wrapping.
	AudioOut() io.Writer

	// Close shuts down the transport and releases resources.
	Close() error
}

// Config holds base configuration for transports.
type Config struct {
	// URL is the transport endpoint URL.
	URL string

	// Token is the authentication token (e.g., LiveKit token, API key).
	Token string

	// SampleRate is the audio sample rate in Hz.
	SampleRate int

	// Channels is the number of audio channels (1 = mono, 2 = stereo).
	Channels int

	// Extra holds transport-specific configuration.
	Extra map[string]any
}

// Option configures a transport.
type Option func(*Config)

// WithURL sets the transport endpoint URL.
func WithURL(url string) Option {
	return func(cfg *Config) {
		cfg.URL = url
	}
}

// WithToken sets the authentication token.
func WithToken(token string) Option {
	return func(cfg *Config) {
		cfg.Token = token
	}
}

// WithSampleRate sets the audio sample rate.
func WithSampleRate(rate int) Option {
	return func(cfg *Config) {
		cfg.SampleRate = rate
	}
}

// WithChannels sets the number of audio channels.
func WithChannels(channels int) Option {
	return func(cfg *Config) {
		cfg.Channels = channels
	}
}

// AsVoiceTransport wraps an AudioTransport to satisfy the voice.Transport
// interface defined in the pipeline package.
type AsVoiceTransport struct {
	T AudioTransport
}

// Recv delegates to the underlying AudioTransport.
func (a *AsVoiceTransport) Recv(ctx context.Context) iter.Seq2[voice.Frame, error] {
	return a.T.Recv(ctx)
}

// Send delegates to the underlying AudioTransport.
func (a *AsVoiceTransport) Send(ctx context.Context, frame voice.Frame) error {
	return a.T.Send(ctx, frame)
}

// Close delegates to the underlying AudioTransport.
func (a *AsVoiceTransport) Close() error {
	return a.T.Close()
}
