package iface

import (
	"context"
)

// Transport defines the interface for audio transport providers.
// Implementations of this interface will provide access to different
// transport mechanisms (e.g., WebRTC, WebSocket).
//
// Transport follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to audio transport operations.
type Transport interface {
	// SendAudio sends audio data to the remote endpoint.
	// It takes a context and audio data and returns an error if the send fails.
	SendAudio(ctx context.Context, audio []byte) error

	// ReceiveAudio receives audio data from the remote endpoint.
	// It returns a channel that receives audio data chunks.
	ReceiveAudio() <-chan []byte

	// OnAudioReceived sets a callback function that is called when audio is received.
	OnAudioReceived(callback func(audio []byte))

	// Close closes the transport connection.
	Close() error
}
