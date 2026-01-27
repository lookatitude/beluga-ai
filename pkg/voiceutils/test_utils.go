// Package voiceutils provides test utilities for voice processing packages.
package voiceutils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MockHTTPRoundTripper is a mock implementation of http.RoundTripper for testing.
type MockHTTPRoundTripper struct {
	// Response is the response to return
	Response *http.Response
	// Error is the error to return
	Error error
	// Handler is a function that can handle requests dynamically
	Handler func(*http.Request) (*http.Response, error)
}

// RoundTrip implements http.RoundTripper.
func (m *MockHTTPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	if m.Handler != nil {
		return m.Handler(req)
	}

	if m.Response != nil {
		return m.Response, nil
	}

	// Default: return 200 OK with empty body
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}

// NewMockHTTPClient creates a new HTTP client with a mock transport.
func NewMockHTTPClient(roundTripper http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: roundTripper,
	}
}

// NewSuccessResponse creates a mock HTTP response with the given status code and body.
func NewSuccessResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// NewErrorResponse creates a mock HTTP error response.
func NewErrorResponse(statusCode int, errorMessage string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(errorMessage)),
		Header:     make(http.Header),
	}
}

// NewJSONResponse creates a mock HTTP response with JSON content type.
func NewJSONResponse(statusCode int, jsonBody string) *http.Response {
	resp := NewSuccessResponse(statusCode, jsonBody)
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

// WebSocket mock utilities

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in tests
	},
}

// MockWebSocketServer is a test WebSocket server.
type MockWebSocketServer struct {
	server     *httptest.Server
	conn       *websocket.Conn
	onMessage  func([]byte) []byte
	messages   [][]byte
	closeAfter time.Duration
	mu         sync.Mutex
}

// NewMockWebSocketServer creates a new mock WebSocket server.
func NewMockWebSocketServer() *MockWebSocketServer {
	mock := &MockWebSocketServer{
		messages: make([][]byte, 0),
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		mock.mu.Lock()
		mock.conn = conn
		mock.mu.Unlock()

		// If closeAfter is set, close after duration
		if mock.closeAfter > 0 {
			time.AfterFunc(mock.closeAfter, func() {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
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
					_ = conn.WriteMessage(responseType, response)
				}
			}
		}
	}))

	return mock
}

// URL returns the WebSocket URL (converts http to ws).
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

// SetOnMessage sets a handler for incoming messages.
func (m *MockWebSocketServer) SetOnMessage(handler func([]byte) []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onMessage = handler
}

// SendMessage sends a message to the connected client.
func (m *MockWebSocketServer) SendMessage(message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil // No connection yet
	}
	return m.conn.WriteMessage(websocket.TextMessage, message)
}

// SendBinary sends a binary message to the connected client.
func (m *MockWebSocketServer) SendBinary(message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil // No connection yet
	}
	return m.conn.WriteMessage(websocket.BinaryMessage, message)
}

// GetMessages returns all received messages.
func (m *MockWebSocketServer) GetMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages
}

// SetCloseAfter sets the duration after which the connection should close.
func (m *MockWebSocketServer) SetCloseAfter(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeAfter = duration
}

// Close closes the mock server.
func (m *MockWebSocketServer) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn != nil {
		_ = m.conn.Close()
	}
	m.server.Close()
}

// WaitForConnection waits for a connection to be established.
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

// Audio test utilities

// GenerateTestAudioData generates test audio data for testing.
// It creates a simple sine wave pattern.
func GenerateTestAudioData(samples, sampleRate int) []byte {
	// Generate PCM 16-bit mono audio
	data := make([]byte, samples*2) // 2 bytes per sample for 16-bit
	for i := 0; i < samples; i++ {
		// Simple triangle wave pattern (range: -500 to 499, safe for int16)
		sample := int16((i % 1000) - 500) // #nosec G115 -- values are bounded [-500, 499]
		data[i*2] = byte(sample & 0xFF)
		data[i*2+1] = byte((sample >> 8) & 0xFF)
	}
	return data
}

// GenerateSilentAudioData generates silent audio data for testing.
func GenerateSilentAudioData(samples int) []byte {
	return make([]byte, samples*2) // All zeros = silence
}

// MockSTTProvider is a mock STT provider for testing.
type MockSTTProvider struct {
	TranscribeFunc      func(audio []byte) (string, error)
	StartStreamingFunc  func() (MockStreamingSession, error)
	TranscribeCallCount int
	StreamingCallCount  int
	mu                  sync.Mutex
}

// Transcribe implements the STT interface.
func (m *MockSTTProvider) Transcribe(audio []byte) (string, error) {
	m.mu.Lock()
	m.TranscribeCallCount++
	m.mu.Unlock()

	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(audio)
	}
	return "mock transcription", nil
}

// StartStreaming implements the STT interface.
func (m *MockSTTProvider) StartStreaming() (MockStreamingSession, error) {
	m.mu.Lock()
	m.StreamingCallCount++
	m.mu.Unlock()

	if m.StartStreamingFunc != nil {
		return m.StartStreamingFunc()
	}
	return &MockStreamingSessionImpl{}, nil
}

// MockStreamingSession is a mock streaming session for testing.
type MockStreamingSession interface {
	SendAudio(audio []byte) error
	ReceiveTranscript() <-chan string
	Close() error
}

// MockStreamingSessionImpl is a mock implementation of StreamingSession.
type MockStreamingSessionImpl struct {
	SendAudioFunc  func(audio []byte) error
	transcriptChan chan string
	closed         bool
	mu             sync.Mutex
}

// SendAudio implements StreamingSession.
func (m *MockStreamingSessionImpl) SendAudio(audio []byte) error {
	if m.SendAudioFunc != nil {
		return m.SendAudioFunc(audio)
	}
	return nil
}

// ReceiveTranscript implements StreamingSession.
func (m *MockStreamingSessionImpl) ReceiveTranscript() <-chan string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.transcriptChan == nil {
		m.transcriptChan = make(chan string, 10)
	}
	return m.transcriptChan
}

// Close implements StreamingSession.
func (m *MockStreamingSessionImpl) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed && m.transcriptChan != nil {
		close(m.transcriptChan)
		m.closed = true
	}
	return nil
}

// SendMockTranscript sends a mock transcript to the session.
func (m *MockStreamingSessionImpl) SendMockTranscript(text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.transcriptChan == nil {
		m.transcriptChan = make(chan string, 10)
	}
	if !m.closed {
		m.transcriptChan <- text
	}
}

// MockTTSProvider is a mock TTS provider for testing.
type MockTTSProvider struct {
	SynthesizeFunc      func(text string) ([]byte, error)
	SynthesizeCallCount int
	mu                  sync.Mutex
}

// Synthesize implements the TTS interface.
func (m *MockTTSProvider) Synthesize(text string) ([]byte, error) {
	m.mu.Lock()
	m.SynthesizeCallCount++
	m.mu.Unlock()

	if m.SynthesizeFunc != nil {
		return m.SynthesizeFunc(text)
	}
	// Return some mock audio data
	return GenerateTestAudioData(1000, 16000), nil
}

// MockVADProvider is a mock VAD provider for testing.
type MockVADProvider struct {
	DetectFunc      func(audio []byte) (bool, error)
	DetectCallCount int
	mu              sync.Mutex
}

// Detect implements the VAD interface.
func (m *MockVADProvider) Detect(audio []byte) (bool, error) {
	m.mu.Lock()
	m.DetectCallCount++
	m.mu.Unlock()

	if m.DetectFunc != nil {
		return m.DetectFunc(audio)
	}
	return true, nil // Default to speech detected
}
