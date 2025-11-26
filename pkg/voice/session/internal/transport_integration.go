package internal

import (
	"context"
	"errors"
	"sync"

	transportiface "github.com/lookatitude/beluga-ai/pkg/voice/transport/iface"
)

// TransportIntegration manages transport provider integration.
type TransportIntegration struct {
	transport transportiface.Transport
	mu        sync.RWMutex
}

// NewTransportIntegration creates a new transport integration.
func NewTransportIntegration(transport transportiface.Transport) *TransportIntegration {
	return &TransportIntegration{
		transport: transport,
	}
}

// SendAudio sends audio through the transport.
func (ti *TransportIntegration) SendAudio(ctx context.Context, audio []byte) error {
	ti.mu.RLock()
	transport := ti.transport
	ti.mu.RUnlock()

	if transport == nil {
		return errors.New("transport not set")
	}

	return transport.SendAudio(ctx, audio)
}

// ReceiveAudio receives audio from the transport.
func (ti *TransportIntegration) ReceiveAudio() <-chan []byte {
	ti.mu.RLock()
	transport := ti.transport
	ti.mu.RUnlock()

	if transport == nil {
		// Return closed channel
		ch := make(chan []byte)
		close(ch)
		return ch
	}

	return transport.ReceiveAudio()
}

// SetAudioCallback sets a callback for received audio.
func (ti *TransportIntegration) SetAudioCallback(callback func(audio []byte)) {
	ti.mu.RLock()
	transport := ti.transport
	ti.mu.RUnlock()

	if transport != nil {
		transport.OnAudioReceived(callback)
	}
}

// Close closes the transport connection.
func (ti *TransportIntegration) Close() error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if ti.transport == nil {
		return nil
	}

	return ti.transport.Close()
}
