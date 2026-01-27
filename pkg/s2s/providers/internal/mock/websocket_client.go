package mock

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConn is an interface for WebSocket connection operations.
// This allows us to mock WebSocket connections in tests.
type WebSocketConn interface {
	ReadJSON(v any) error
	WriteJSON(v any) error
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// WebSocketDialer is an interface for WebSocket dialing operations.
type WebSocketDialer interface {
	Dial(url string, headers map[string][]string) (WebSocketConn, error)
}

// MockWebSocketConn is a mock implementation of WebSocketConn for testing.
type MockWebSocketConn struct {
	readDeadline  time.Time
	writeDeadline time.Time
	readError     error
	writeError    error
	messages      []any
	writeMessages []any
	readIndex     int
	mu            sync.RWMutex
	closed        bool
}

// NewMockWebSocketConn creates a new mock WebSocket connection.
func NewMockWebSocketConn() *MockWebSocketConn {
	return &MockWebSocketConn{
		messages:      make([]any, 0),
		writeMessages: make([]any, 0),
	}
}

// SetReadMessages sets the messages to be returned by ReadJSON.
func (m *MockWebSocketConn) SetReadMessages(messages []any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = messages
	m.readIndex = 0
}

// AddReadMessage adds a message to be returned by ReadJSON.
func (m *MockWebSocketConn) AddReadMessage(msg any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

// GetWriteMessages returns all messages written via WriteJSON.
func (m *MockWebSocketConn) GetWriteMessages() []any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]any, len(m.writeMessages))
	copy(result, m.writeMessages)
	return result
}

// SetReadError sets an error to return from ReadJSON.
func (m *MockWebSocketConn) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readError = err
}

// SetWriteError sets an error to return from WriteJSON.
func (m *MockWebSocketConn) SetWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeError = err
}

// ReadJSON implements WebSocketConn interface.
func (m *MockWebSocketConn) ReadJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return io.EOF
	}

	if m.readError != nil {
		return m.readError
	}

	if m.readIndex >= len(m.messages) {
		return io.EOF
	}

	msg := m.messages[m.readIndex]
	m.readIndex++

	// Marshal and unmarshal to simulate JSON encoding/decoding
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// WriteJSON implements WebSocketConn interface.
func (m *MockWebSocketConn) WriteJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return websocket.ErrCloseSent
	}

	if m.writeError != nil {
		return m.writeError
	}

	m.writeMessages = append(m.writeMessages, v)
	return nil
}

// Close implements WebSocketConn interface.
func (m *MockWebSocketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// SetReadDeadline implements WebSocketConn interface.
func (m *MockWebSocketConn) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDeadline = t
	return nil
}

// SetWriteDeadline implements WebSocketConn interface.
func (m *MockWebSocketConn) SetWriteDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeDeadline = t
	return nil
}

// IsClosed returns whether the connection is closed.
func (m *MockWebSocketConn) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// MockWebSocketDialer is a mock implementation of WebSocketDialer for testing.
type MockWebSocketDialer struct {
	defaultDialError error
	connections      map[string]*MockWebSocketConn
	defaultConn      *MockWebSocketConn
	dialError        map[string]error
	mu               sync.RWMutex
}

// NewMockWebSocketDialer creates a new mock WebSocket dialer.
func NewMockWebSocketDialer() *MockWebSocketDialer {
	return &MockWebSocketDialer{
		connections: make(map[string]*MockWebSocketConn),
		dialError:   make(map[string]error),
	}
}

// SetConnection sets a mock connection for a specific URL.
func (m *MockWebSocketDialer) SetConnection(url string, conn *MockWebSocketConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[url] = conn
}

// SetDefaultConnection sets the default connection for unmatched URLs.
func (m *MockWebSocketDialer) SetDefaultConnection(conn *MockWebSocketConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultConn = conn
}

// SetDialError sets an error to return for a specific URL.
func (m *MockWebSocketDialer) SetDialError(url string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dialError[url] = err
}

// SetDefaultDialError sets the default error to return for unmatched URLs.
func (m *MockWebSocketDialer) SetDefaultDialError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDialError = err
}

// Dial implements WebSocketDialer interface.
func (m *MockWebSocketDialer) Dial(url string, headers map[string][]string) (WebSocketConn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for explicit error
	if err, ok := m.dialError[url]; ok {
		return nil, err
	}

	if m.defaultDialError != nil {
		return nil, m.defaultDialError
	}

	// Find matching connection
	var conn *MockWebSocketConn
	if c, ok := m.connections[url]; ok {
		conn = c
	} else {
		conn = m.defaultConn
	}

	if conn == nil {
		// Create a new connection if none exists
		conn = NewMockWebSocketConn()
	}

	return conn, nil
}
