package livekit

import (
	"context"
	"io"
	"iter"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/voice"
	"github.com/lookatitude/beluga-ai/v2/voice/transport"
)

var _ transport.AudioTransport = (*Transport)(nil) // compile-time interface check

func init() {
	transport.Register("livekit", func(cfg transport.Config) (transport.AudioTransport, error) {
		return New(cfg)
	})
}

// Transport is the placeholder transport.AudioTransport for LiveKit rooms.
//
// NOT IMPLEMENTED: no LiveKit connection is made. New always returns an error
// wrapping transport.ErrNotImplemented, and every method of a Transport
// value reports the same error instead of silently discarding audio. The
// type is kept so existing code compiles; use the built-in "websocket"
// transport until a real LiveKit integration ships.
type Transport struct{}

// New reports that the LiveKit transport is not implemented. It always
// returns a nil Transport and an error wrapping transport.ErrNotImplemented.
func New(_ transport.Config) (*Transport, error) {
	return nil, notImplemented()
}

func notImplemented() error {
	return core.Errorf(core.ErrInvalidInput, "livekit: %w", transport.ErrNotImplemented)
}

// Recv yields a single transport.ErrNotImplemented error.
func (t *Transport) Recv(_ context.Context) iter.Seq2[voice.Frame, error] {
	return func(yield func(voice.Frame, error) bool) {
		yield(voice.Frame{}, notImplemented())
	}
}

// Send returns transport.ErrNotImplemented.
func (t *Transport) Send(_ context.Context, _ voice.Frame) error {
	return notImplemented()
}

// AudioOut returns a writer whose writes fail with
// transport.ErrNotImplemented.
func (t *Transport) AudioOut() io.Writer {
	return errWriter{}
}

// Close is a no-op.
func (t *Transport) Close() error {
	return nil
}

// errWriter fails every write.
type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, notImplemented()
}
