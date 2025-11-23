package testutils

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in tests
	},
}

// MockWebSocketServer is a test WebSocket server
type MockWebSocketServer struct {
	server     *httptest.Server
	conn       *websocket.Conn
	mu         sync.Mutex
	messages   [][]byte
	onMessage  func([]byte) []byte // Handler that can return a response
	closeAfter time.Duration        // Close connection after this duration
}

// NewMockWebSocketServer creates a new mock WebSocket server
func NewMockWebSocketServer() *MockWebSocketServer {
	mock := &MockWebSocketServer{
		messages: make([][]byte, 0),
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		mock.mu.Lock()
		mock.conn = conn
		mock.mu.Unlock()

		// If closeAfter is set, close after duration
		if mock.closeAfter > 0 {
			time.AfterFunc(mock.closeAfter, func() {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			})
		}

		// Read messages in a loop
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			mock.mu.Lock()
			mock.messages = append(mock.messages, message)
			handler := mock.onMessage
			mock.mu.Unlock()

			// If handler is set, send response
			if handler != nil {
				response := handler(message)
				if response != nil {
					// Send as text message if response is JSON, otherwise use same type
					responseType := websocket.TextMessage
					if messageType == websocket.BinaryMessage {
						// For binary messages, check if response looks like JSON
						if len(response) > 0 && response[0] == '{' {
							responseType = websocket.TextMessage
						} else {
							responseType = websocket.BinaryMessage
						}
					}
					conn.WriteMessage(responseType, response)
				}
			}
		}
	}))

	return mock
}

// URL returns the WebSocket URL (converts http to ws)
func (m *MockWebSocketServer) URL() string {
	url := m.server.URL
	// Convert http to ws
	if len(url) >= 4 && url[:4] == "http" {
		if url[:5] == "https" {
			url = "wss" + url[5:]
		} else {
			url = "ws" + url[4:]
		}
	}
	return url
}

// SetOnMessage sets a handler for incoming messages
func (m *MockWebSocketServer) SetOnMessage(handler func([]byte) []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onMessage = handler
}

// SendMessage sends a message to the connected client
func (m *MockWebSocketServer) SendMessage(message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil // No connection yet
	}
	return m.conn.WriteMessage(websocket.TextMessage, message)
}

// SendBinary sends a binary message to the connected client
func (m *MockWebSocketServer) SendBinary(message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil // No connection yet
	}
	return m.conn.WriteMessage(websocket.BinaryMessage, message)
}

// GetMessages returns all received messages
func (m *MockWebSocketServer) GetMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages
}

// SetCloseAfter sets the duration after which the connection should close
func (m *MockWebSocketServer) SetCloseAfter(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeAfter = duration
}

// Close closes the mock server
func (m *MockWebSocketServer) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn != nil {
		m.conn.Close()
	}
	m.server.Close()
}

// WaitForConnection waits for a connection to be established
func (m *MockWebSocketServer) WaitForConnection(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		connected := m.conn != nil
		m.mu.Unlock()
		if connected {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

