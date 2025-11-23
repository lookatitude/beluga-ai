// Package transport provides advanced test utilities and comprehensive mocks for testing Transport implementations.
package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockTransport provides a comprehensive mock implementation for testing
type AdvancedMockTransport struct {
	mock.Mock

	// Configuration
	transportName string
	callCount     int
	mu            sync.RWMutex

	// Configurable behavior
	shouldError          bool
	errorToReturn        error
	audioData            [][]byte
	dataIndex            int
	processingDelay      time.Duration
	simulateNetworkDelay bool
	connected            bool
	audioCallback        func([]byte)
}

// NewAdvancedMockTransport creates a new advanced mock with configurable behavior
func NewAdvancedMockTransport(transportName string, opts ...MockOption) *AdvancedMockTransport {
	m := &AdvancedMockTransport{
		transportName:   transportName,
		audioData:       [][]byte{},
		processingDelay: 10 * time.Millisecond,
		connected:       false,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockTransport
type MockOption func(*AdvancedMockTransport)

// WithTransportName sets the transport name
func WithTransportName(name string) MockOption {
	return func(m *AdvancedMockTransport) {
		m.transportName = name
	}
}

// WithAudioData sets the audio data to return
func WithAudioData(data ...[]byte) MockOption {
	return func(m *AdvancedMockTransport) {
		m.audioData = data
	}
}

// WithError configures the mock to return an error
func WithError(err error) MockOption {
	return func(m *AdvancedMockTransport) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithProcessingDelay sets the delay for processing
func WithProcessingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockTransport) {
		m.processingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockTransport) {
		m.simulateNetworkDelay = enabled
	}
}

// WithConnected sets the connection state
func WithConnected(connected bool) MockOption {
	return func(m *AdvancedMockTransport) {
		m.connected = connected
	}
}

// Connect is a helper method for testing (not part of the interface)
func (m *AdvancedMockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return &TransportError{
			Op:   "Connect",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	// Simulate network delay if enabled
	if m.simulateNetworkDelay {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	m.connected = true
	return nil
}

// Disconnect is a helper method for testing (not part of the interface)
func (m *AdvancedMockTransport) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return &TransportError{
			Op:   "Disconnect",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	m.connected = false
	return nil
}

// SendAudio implements the Transport interface
func (m *AdvancedMockTransport) SendAudio(ctx context.Context, audio []byte) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if !m.connected {
		return &TransportError{
			Op:   "SendAudio",
			Code: ErrCodeNotConnected,
			Err:  nil,
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return &TransportError{
			Op:   "SendAudio",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	// Simulate processing delay
	if m.processingDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.processingDelay):
		}
	}

	return nil
}

// ReceiveAudio implements the Transport interface
func (m *AdvancedMockTransport) ReceiveAudio() <-chan []byte {
	audioCh := make(chan []byte, 10)

	if !m.connected {
		close(audioCh)
		return audioCh
	}

	if m.shouldError {
		close(audioCh)
		return audioCh
	}

	// Send audio data in a goroutine
	go func() {
		defer close(audioCh)
		for _, data := range m.audioData {
			audioCh <- data
		}
	}()

	return audioCh
}

// OnAudioReceived implements the Transport interface
func (m *AdvancedMockTransport) OnAudioReceived(callback func(audio []byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audioCallback = callback
}

// Close implements the Transport interface
func (m *AdvancedMockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return &TransportError{
			Op:   "Close",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	m.connected = false
	return nil
}

// IsConnected is a helper method for testing
func (m *AdvancedMockTransport) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetCallCount returns the number of times methods have been called
func (m *AdvancedMockTransport) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AssertTransportInterface ensures that a type implements the Transport interface
func AssertTransportInterface(t *testing.T, transport iface.Transport) {
	assert.NotNil(t, transport, "Transport should not be nil")

	// Test SendAudio method
	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	_ = transport.SendAudio(ctx, audio)

	// Test ReceiveAudio method
	audioCh := transport.ReceiveAudio()
	_ = audioCh

	// Test OnAudioReceived method
	transport.OnAudioReceived(func(audio []byte) {
		_ = audio
	})

	// Test Close method
	_ = transport.Close()
}
