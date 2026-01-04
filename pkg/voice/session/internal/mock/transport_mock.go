// Package mock provides mocks for voice session internal testing.
// This file provides Transport mocks specifically for agent integration testing.
package mock

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockTransport provides a mock implementation of Transport for agent integration testing.
type MockTransport struct {
	errorToReturn   error
	audioCh         chan []byte
	audioCallback   func([]byte)
	sendCount       int
	receiveCount    int
	processingDelay time.Duration
	mu              sync.RWMutex
	shouldError     bool
	connected       bool
	simulateDelay   bool
}

// NewMockTransport creates a new mock transport.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		audioCh:         make(chan []byte, 100),
		connected:       false,
		processingDelay: 10 * time.Millisecond,
	}
}

// WithError configures the mock to return an error.
func (m *MockTransport) WithError(err error) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorToReturn = err
	return m
}

// WithConnected sets the connected state.
func (m *MockTransport) WithConnected(connected bool) *MockTransport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
	return m
}

// SendAudio implements the Transport interface.
func (m *MockTransport) SendAudio(ctx context.Context, audio []byte) error {
	m.mu.Lock()
	m.sendCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	connected := m.connected
	processingDelay := m.processingDelay
	simulateDelay := m.simulateDelay
	m.mu.Unlock()

	if !connected {
		return errors.New("transport not connected")
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return errors.New("mock transport error")
	}

	if simulateDelay && processingDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(processingDelay):
		}
	}

	return nil
}

// ReceiveAudio implements the Transport interface.
func (m *MockTransport) ReceiveAudio() <-chan []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.audioCh
}

// OnAudioReceived implements the Transport interface.
func (m *MockTransport) OnAudioReceived(callback func(audio []byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audioCallback = callback
}

// Close implements the Transport interface.
func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil
	}

	close(m.audioCh)
	m.connected = false
	return nil
}

// GetSendCount returns the number of SendAudio calls.
func (m *MockTransport) GetSendCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sendCount
}

// GetReceiveCount returns the number of received audio chunks.
func (m *MockTransport) GetReceiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.receiveCount
}

// SimulateReceivedAudio simulates receiving audio for testing.
func (m *MockTransport) SimulateReceivedAudio(audio []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.audioCallback != nil {
		go m.audioCallback(audio)
	}

	select {
	case m.audioCh <- audio:
		m.receiveCount++
	default:
		// Channel full, drop
	}
}
