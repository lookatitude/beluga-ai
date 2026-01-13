package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// AzureStreamingSession implements the StreamingSession interface for Azure Speech Services WebSocket.
type AzureStreamingSession struct {
	config   *AzureConfig
	conn     *websocket.Conn
	resultCh chan iface.TranscriptResult
	ctx      context.Context
	cancel   context.CancelFunc
	closed   bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewAzureStreamingSession creates a new Azure Speech Services WebSocket streaming session.
func NewAzureStreamingSession(ctx context.Context, config *AzureConfig) (iface.StreamingSession, error) {
	// Build WebSocket URL with query parameters
	url := fmt.Sprintf("%s?language=%s&format=detailed",
		config.GetWebSocketURL(),
		config.Language,
	)

	// Add optional parameters
	if config.EnablePunctuation {
		url += "&punctuation=true"
	}
	if config.EnableWordLevelTimestamps {
		url += "&wordLevelTimestamps=true"
	}
	if config.EnableSpeakerDiarization {
		url += "&diarization=true"
	}
	if config.EndpointID != "" {
		url += "&endpointId=" + config.EndpointID
	}

	// Create request headers
	headers := http.Header{}
	headers.Set("Ocp-Apim-Subscription-Key", config.APIKey)
	headers.Set("X-ConnectionId", generateConnectionID())

	// Dial WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: config.Timeout,
	}

	conn, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				// Error closing response body during error handling - non-critical
			}
			return nil, stt.ErrorFromHTTPStatus("StartStreaming", resp.StatusCode, err)
		}
		return nil, stt.NewSTTError("StartStreaming", stt.ErrCodeNetworkError, err)
	}
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - non-critical in cleanup context
		}
	}

	// Create context with cancel
	sessionCtx, cancel := context.WithCancel(ctx)

	session := &AzureStreamingSession{
		config:   config,
		conn:     conn,
		resultCh: make(chan iface.TranscriptResult, 10),
		ctx:      sessionCtx,
		cancel:   cancel,
	}

	// Start receiving messages
	session.wg.Add(1)
	go session.receiveMessages()

	// Record metrics if enabled
	if config.EnableMetrics {
		metrics := stt.GetMetrics()
		if metrics != nil {
			metrics.IncrementActiveStreams(ctx, "azure", config.Language)
		}
	}

	return session, nil
}

// SendAudio sends audio data to the streaming session.
func (s *AzureStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	closed := s.closed
	conn := s.conn
	s.mu.RUnlock()

	if closed {
		return stt.NewSTTError("SendAudio", stt.ErrCodeStreamClosed, errors.New("session closed"))
	}

	if conn == nil {
		return stt.NewSTTError("SendAudio", stt.ErrCodeStreamClosed, errors.New("connection not established"))
	}

	// Write audio data as binary message
	err := conn.WriteMessage(websocket.BinaryMessage, audio)
	if err != nil {
		return stt.NewSTTError("SendAudio", stt.ErrCodeStreamError, err)
	}

	return nil
}

// ReceiveTranscript returns the channel for receiving transcript results.
func (s *AzureStreamingSession) ReceiveTranscript() <-chan iface.TranscriptResult {
	return s.resultCh
}

// Close closes the streaming session.
func (s *AzureStreamingSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.cancel()

	// Close WebSocket connection
	if s.conn != nil {
		// Send close message
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		_ = s.conn.WriteMessage(websocket.CloseMessage, closeMsg)
		_ = s.conn.Close()
	}

	// Wait for goroutines to finish
	s.wg.Wait()

	// Close result channel
	close(s.resultCh)

	// Record metrics if enabled
	if s.config.EnableMetrics {
		metrics := stt.GetMetrics()
		if metrics != nil {
			metrics.DecrementActiveStreams(context.Background(), "azure", s.config.Language)
		}
	}

	return nil
}

// receiveMessages receives messages from the WebSocket connection.
func (s *AzureStreamingSession) receiveMessages() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Set read deadline
			_ = s.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			// Read message
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				// Check if connection is closed
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					result := iface.TranscriptResult{
						Error: stt.NewSTTError("ReceiveTranscript", stt.ErrCodeStreamError, err),
					}
					select {
					case s.resultCh <- result:
					case <-s.ctx.Done():
					}
				}
				return
			}

			// Parse Azure response
			var response struct {
				RecognitionStatus string `json:"RecognitionStatus"`
				DisplayText       string `json:"DisplayText"`
				Offset            int64  `json:"Offset"`
				Duration          int64  `json:"Duration"`
			}

			if err := json.Unmarshal(message, &response); err != nil {
				result := iface.TranscriptResult{
					Error: stt.NewSTTError("ReceiveTranscript", stt.ErrCodeMalformedResponse, err),
				}
				select {
				case s.resultCh <- result:
				case <-s.ctx.Done():
					return
				}
				continue
			}

			// Extract transcript
			if response.RecognitionStatus == "Success" && response.DisplayText != "" {
				result := iface.TranscriptResult{
					Text:       response.DisplayText,
					IsFinal:    true,
					Language:   s.config.Language,
					Confidence: 1.0, // Azure doesn't provide confidence in this format
				}

				select {
				case s.resultCh <- result:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

// generateConnectionID generates a unique connection ID.
func generateConnectionID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
