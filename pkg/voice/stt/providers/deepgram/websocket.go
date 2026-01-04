package deepgram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// DeepgramStreamingSession implements the StreamingSession interface for Deepgram WebSocket.
type DeepgramStreamingSession struct {
	config   *DeepgramConfig
	conn     *websocket.Conn
	resultCh chan iface.TranscriptResult
	ctx      context.Context //nolint:containedctx // Context is necessary for long-lived websocket connection
	cancel   context.CancelFunc
	closed   bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewDeepgramStreamingSession creates a new Deepgram WebSocket streaming session.
func NewDeepgramStreamingSession(ctx context.Context, config *DeepgramConfig) (iface.StreamingSession, error) {
	// Build WebSocket URL with query parameters
	url := fmt.Sprintf("%s?model=%s&language=%s&punctuate=%t&smart_format=%t&interim_results=%t&endpointing=%d",
		config.WebSocketURL,
		config.Model,
		config.Language,
		config.Punctuate,
		config.SmartFormat,
		config.InterimResults,
		config.Endpointing,
	)

	// Create request headers
	headers := http.Header{}
	headers.Set("Authorization", "Token "+config.APIKey)

	// Dial WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: config.Timeout,
	}

	conn, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close() //nolint:errcheck // Best effort to close response body on error
			return nil, stt.ErrorFromHTTPStatus("StartStreaming", resp.StatusCode, err)
		}
		return nil, stt.NewSTTError("StartStreaming", stt.ErrCodeNetworkError, err)
	}
	if resp != nil {
		_ = resp.Body.Close() //nolint:errcheck // Close response body after successful WebSocket handshake
	}

	// Create context with cancel
	sessionCtx, cancel := context.WithCancel(ctx)

	session := &DeepgramStreamingSession{
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
			metrics.IncrementActiveStreams(ctx, "deepgram", config.Model)
		}
	}

	return session, nil
}

// SendAudio sends audio data to the streaming session.
func (s *DeepgramStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
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
func (s *DeepgramStreamingSession) ReceiveTranscript() <-chan iface.TranscriptResult {
	return s.resultCh
}

// Close closes the streaming session.
func (s *DeepgramStreamingSession) Close() error {
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
			metrics.DecrementActiveStreams(context.Background(), "deepgram", s.config.Model)
		}
	}

	return nil
}

// receiveMessages receives messages from the WebSocket connection.
func (s *DeepgramStreamingSession) receiveMessages() {
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

			// Parse Deepgram response
			var response struct {
				Type    string `json:"type"`
				Channel struct {
					Alternatives []struct {
						Transcript string  `json:"transcript"`
						Confidence float64 `json:"confidence"`
					} `json:"alternatives"`
				} `json:"channel"`
				IsFinal bool `json:"is_final"`
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
			if len(response.Channel.Alternatives) > 0 {
				transcript := response.Channel.Alternatives[0].Transcript
				result := iface.TranscriptResult{
					Text:       transcript,
					IsFinal:    response.IsFinal,
					Language:   s.config.Language,
					Confidence: response.Channel.Alternatives[0].Confidence,
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
