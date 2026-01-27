package twilio

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	twiliov2010 "github.com/twilio/twilio-go/rest/api/v2010"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// MediaStreamMessage represents a message in Twilio Media Streams protocol.
type MediaStreamMessage struct {
	Event  string       `json:"event"`
	Stream StreamInfo   `json:"streamSid,omitempty"`
	Media  MediaPayload `json:"media,omitempty"`
	Start  StartPayload `json:"start,omitempty"`
	Stop   StopPayload  `json:"stop,omitempty"`
}

// StreamInfo contains stream identifier information.
type StreamInfo struct {
	SID string `json:"streamSid"`
}

// MediaPayload contains audio data in base64.
type MediaPayload struct {
	Payload string `json:"payload"` // Base64-encoded mu-law audio
}

// StartPayload contains stream start information.
type StartPayload struct {
	AccountSID string `json:"accountSid"`
	CallSID    string `json:"callSid"`
	From       string `json:"from"`
	To         string `json:"to"`
}

// StopPayload contains stream stop information.
type StopPayload struct {
	AccountSID string `json:"accountSid"`
	CallSID    string `json:"callSid"`
}

// AudioStream manages bidirectional WebSocket audio streaming for a Twilio call.
type AudioStream struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	conn                 *websocket.Conn
	audioIn              chan []byte
	audioOut             chan []byte
	metrics              *Metrics
	streamSID            string
	callSID              string
	reconnectAttempts    int
	maxReconnectAttempts int
	mu                   sync.RWMutex
	closed               bool
}

// NewAudioStream creates a new audio stream for a Twilio call.
func NewAudioStream(ctx context.Context, streamURL, streamSID, callSID string, metrics *Metrics) (*AudioStream, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Connect to Twilio Media Stream WebSocket
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, streamURL, nil)
	if err != nil {
		cancel()
		return nil, NewTwilioError("NewAudioStream", ErrCodeTwilioStreamFailed, err)
	}

	stream := &AudioStream{
		ctx:                  ctx,
		cancel:               cancel,
		conn:                 conn,
		streamSID:            streamSID,
		callSID:              callSID,
		audioIn:              make(chan []byte, 100),
		audioOut:             make(chan []byte, 100),
		closed:               false,
		metrics:              metrics,
		maxReconnectAttempts: 3,
	}

	// Start receiver goroutine
	go stream.receiveMessages()

	// Start sender goroutine
	go stream.sendMessages()

	return stream, nil
}

// StreamAudio creates and manages a WebSocket audio stream for a session.
func (b *TwilioBackend) StreamAudio(ctx context.Context, sessionID string) (*AudioStream, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.StreamAudio")
	defer span.End()

	span.SetAttributes(attribute.String("session_id", sessionID))

	// Get session to verify it exists
	b.mu.RLock()
	_, exists := b.sessions[sessionID]
	b.mu.RUnlock()

	if !exists {
		err := NewTwilioError("StreamAudio", "session_not_found", fmt.Errorf("session %s not found", sessionID))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Get call SID from session
	// For Twilio, the session ID is the call SID (they're the same)
	callSID := sessionID

	// Build WebSocket URL for Media Stream
	// The URL should point to our WebSocket server endpoint
	streamURL := b.config.WebhookURL
	if streamURL == "" {
		err := NewTwilioError("StreamAudio", ErrCodeTwilioInvalidConfig, errors.New("webhook URL not configured"))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	// Ensure it's a WebSocket URL (wss://)
	// Note: Twilio Media Streams require wss:// URLs
	if len(streamURL) < 4 {
		err := NewTwilioError("StreamAudio", ErrCodeTwilioInvalidConfig, errors.New("invalid webhook URL"))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if streamURL[:4] != "wss:" && streamURL[:4] != "ws:" {
		// Convert http/https to ws/wss
		if len(streamURL) >= 5 && streamURL[:5] == "https" {
			streamURL = "wss" + streamURL[5:]
		} else if len(streamURL) >= 4 && streamURL[:4] == "http" {
			streamURL = "ws" + streamURL[4:]
		} else {
			// Default to wss:// if no protocol specified
			streamURL = "wss://" + streamURL
		}
	}
	// Append stream path if needed (Twilio expects the full WebSocket endpoint URL)
	if len(streamURL) > 0 && streamURL[len(streamURL)-1] != '/' {
		streamURL += "/stream"
	}

	// Create Twilio Media Stream via API
	streamParams := &twiliov2010.CreateStreamParams{}
	streamParams.SetUrl(streamURL)
	streamParams.SetName("BelugaVoiceStream")

	streamResp, err := b.client.Api.CreateStream(callSID, streamParams)
	if err != nil {
		err = MapTwilioError("StreamAudio", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	streamSID := getStringValue(streamResp.Sid)
	span.SetAttributes(
		attribute.String("stream_sid", streamSID),
		attribute.String("stream_url", streamURL),
	)

	// The actual WebSocket URL comes from Twilio when they connect to our server
	// For now, we use the configured URL - Twilio will connect to it
	stream, err := NewAudioStream(ctx, streamURL, streamSID, callSID, b.metrics)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if b.metrics != nil {
		b.metrics.IncrementActiveStreams(ctx)
		b.metrics.RecordStream(ctx, streamSID, 0, true)
	}

	span.SetStatus(codes.Ok, "stream created")
	return stream, nil
}

// receiveMessages receives messages from the Twilio Media Stream WebSocket.
func (s *AudioStream) receiveMessages() {
	defer close(s.audioIn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				s.mu.Lock()
				closed := s.closed
				s.mu.Unlock()

				if closed {
					return
				}

				// Handle network failure - attempt reconnection
				if s.reconnectAttempts < s.maxReconnectAttempts {
					s.reconnectAttempts++
					time.Sleep(time.Duration(s.reconnectAttempts) * time.Second)
					// Attempt to reconnect (simplified - full implementation would recreate connection)
					continue
				}

				// Send error to channel
				s.audioIn <- nil // Signal error
				return
			}

			// Parse Media Stream message
			var msg MediaStreamMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			switch msg.Event {
			case "media":
				// Decode base64 audio payload (mu-law)
				audioData, err := decodeMuLawAudio(msg.Media.Payload)
				if err != nil {
					continue
				}
				select {
				case s.audioIn <- audioData:
				case <-s.ctx.Done():
					return
				}
			case "start":
				// Stream started
				if s.metrics != nil {
					s.metrics.RecordStream(s.ctx, msg.Stream.SID, 0, true)
				}
			case "stop":
				// Stream stopped
				return
			}
		}
	}
}

// sendMessages sends audio messages to the Twilio Media Stream WebSocket.
func (s *AudioStream) sendMessages() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case audio, ok := <-s.audioOut:
			if !ok {
				return
			}

			s.mu.RLock()
			closed := s.closed
			s.mu.RUnlock()

			if closed {
				return
			}

			// Encode audio as base64 mu-law
			payload := encodeMuLawAudio(audio)

			msg := MediaStreamMessage{
				Event: "media",
				Media: MediaPayload{
					Payload: payload,
				},
			}

			if err := s.conn.WriteJSON(msg); err != nil {
				// Handle network failure - attempt reconnection
				if s.reconnectAttempts < s.maxReconnectAttempts {
					s.reconnectAttempts++
					time.Sleep(time.Duration(s.reconnectAttempts) * time.Second)
					continue
				}
				return
			}
		}
	}
}

// SendAudio sends audio data to Twilio (mu-law encoded).
func (s *AudioStream) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	closed := s.closed
	s.mu.RUnlock()

	if closed {
		return NewTwilioError("SendAudio", ErrCodeTwilioStreamFailed, errors.New("stream is closed"))
	}

	select {
	case s.audioOut <- audio:
		return nil
	case <-ctx.Done():
		return NewTwilioError("SendAudio", ErrCodeTwilioTimeout, ctx.Err())
	case <-s.ctx.Done():
		return NewTwilioError("SendAudio", ErrCodeTwilioStreamFailed, errors.New("stream context canceled"))
	}
}

// ReceiveAudio returns a channel for receiving audio from Twilio (mu-law encoded).
func (s *AudioStream) ReceiveAudio() <-chan []byte {
	return s.audioIn
}

// Close closes the audio stream.
func (s *AudioStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.cancel()

	// Send stop message
	if s.conn != nil {
		msg := MediaStreamMessage{
			Event: "stop",
		}
		_ = s.conn.WriteJSON(msg)
		_ = s.conn.Close()
	}

	close(s.audioOut)

	if s.metrics != nil {
		s.metrics.DecrementActiveStreams(s.ctx)
	}

	return nil
}

// decodeMuLawAudio decodes base64-encoded mu-law audio.
func decodeMuLawAudio(base64Data string) ([]byte, error) {
	// Decode base64 to get mu-law audio bytes
	audioData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 audio: %w", err)
	}
	// Audio is already in mu-law format from Twilio
	return audioData, nil
}

// encodeMuLawAudio encodes audio data as base64 mu-law.
func encodeMuLawAudio(audio []byte) string {
	// Audio should already be in mu-law format
	// Encode as base64 for WebSocket transmission
	return base64.StdEncoding.EncodeToString(audio)
}

// Mu-law encoding constants.
const (
	ulawBias = 0x84  // Bias for linear PCM (132 decimal)
	ulawClip = 32635 // Maximum linear input magnitude for μ-law
)

// convertLinearToMuLaw converts linear PCM (16-bit signed) to mu-law (PCMU).
// Input: []byte representing 16-bit signed PCM samples (little-endian)
// Output: []byte representing 8-bit mu-law encoded samples.
func convertLinearToMuLaw(linear []byte) []byte {
	if len(linear)%2 != 0 {
		// If odd length, pad with zero
		linear = append(linear, 0)
	}

	result := make([]byte, len(linear)/2)
	for i := 0; i < len(linear); i += 2 {
		// Read 16-bit signed PCM sample (little-endian)
		// Note: Converting uint16 bytes to int16 is safe here as we're interpreting
		// the raw bytes as a signed 16-bit integer (standard PCM format)
		// #nosec G115 -- Safe conversion: interpreting bytes as signed PCM sample
		sample := int16(binary.LittleEndian.Uint16(linear[i : i+2]))
		result[i/2] = encodeMuLawSample(sample)
	}
	return result
}

// convertMuLawToLinear converts mu-law (PCMU) to linear PCM (16-bit signed).
// Input: []byte representing 8-bit mu-law encoded samples
// Output: []byte representing 16-bit signed PCM samples (little-endian).
func convertMuLawToLinear(mulaw []byte) []byte {
	result := make([]byte, len(mulaw)*2)
	for i, u := range mulaw {
		sample := parseMuLawSample(u)
		// Write 16-bit signed PCM sample (little-endian)
		// Note: Converting int16 to uint16 for binary encoding is safe here as we're
		// writing the raw bytes of a signed integer (standard PCM format)
		// #nosec G115 -- Safe conversion: writing signed PCM sample as raw bytes
		binary.LittleEndian.PutUint16(result[i*2:i*2+2], uint16(sample))
	}
	return result
}

// readMuLawFrame reads a mu-law audio frame from a reader.
func readMuLawFrame(r io.Reader, frameSize int) ([]byte, error) {
	frame := make([]byte, frameSize)
	n, err := r.Read(frame)
	if err != nil {
		return nil, err
	}
	if n != frameSize {
		return nil, fmt.Errorf("incomplete frame: read %d bytes, expected %d", n, frameSize)
	}
	return frame, nil
}

// writeMuLawFrame writes a mu-law audio frame to a writer.
func writeMuLawFrame(w io.Writer, frame []byte) error {
	_, err := w.Write(frame)
	return err
}

// parseMuLawSample parses a single mu-law sample (8-bit) to 16-bit linear PCM.
// Implements ITU-T G.711 μ-law decoding with proper bit inversion.
func parseMuLawSample(sample byte) int16 {
	// Invert all bits (μ-law uses inverted representation)
	sample = ^sample

	sign := sample & 0x80
	exponent := (sample & 0x70) >> 4
	mantissa := sample & 0x0F

	// magnitude = ((mantissa << 3) + bias) << exponent − bias
	magnitude := ((uint16(mantissa) << 3) + ulawBias) << exponent
	magnitude -= ulawBias

	// Clamp magnitude to int16 range to prevent overflow
	// Mu-law encoding limits output to ulawClip (32635), which fits in int16
	const maxInt16 = 32767
	if magnitude > maxInt16 {
		magnitude = maxInt16
	}

	var linear int16
	if sign != 0 {
		// #nosec G115 -- Safe conversion: magnitude clamped to int16 range
		linear = -int16(magnitude)
	} else {
		// #nosec G115 -- Safe conversion: magnitude clamped to int16 range
		linear = int16(magnitude)
	}
	return linear
}

// encodeMuLawSample encodes a single 16-bit linear PCM sample to 8-bit μ-law.
// Implements ITU-T G.711 μ-law encoding with proper bias and bit inversion.
func encodeMuLawSample(sample int16) byte {
	var sign byte
	var magnitude uint16

	// Get the sign and magnitude
	if sample < 0 {
		sign = 0x80
		magnitude = uint16(-sample)
	} else {
		sign = 0x00
		magnitude = uint16(sample)
	}

	// Add bias for better resolution of small signals
	magnitude += ulawBias
	if magnitude > ulawClip {
		magnitude = ulawClip
	}

	// Determine exponent (segment) - find highest bit position
	var exponent byte
	switch {
	case magnitude >= 0x4000: // 16384
		exponent = 7
	case magnitude >= 0x2000: // 8192
		exponent = 6
	case magnitude >= 0x1000: // 4096
		exponent = 5
	case magnitude >= 0x800: // 2048
		exponent = 4
	case magnitude >= 0x400: // 1024
		exponent = 3
	case magnitude >= 0x200: // 512
		exponent = 2
	case magnitude >= 0x100: // 256
		exponent = 1
	default:
		exponent = 0
	}

	// Mantissa within the segment: 4 bits
	mantissa := byte((magnitude >> (exponent + 3)) & 0x0F)

	ulawByte := sign | (exponent << 4) | mantissa
	// Invert all bits (μ-law uses inverted representation)
	ulawByte = ^ulawByte
	return ulawByte
}

// convertPCMToMuLaw converts 16-bit PCM samples to mu-law.
// This is an alias for convertLinearToMuLaw for backward compatibility.
func convertPCMToMuLaw(pcm []byte) []byte {
	return convertLinearToMuLaw(pcm)
}

// convertMuLawToPCM converts mu-law samples to 16-bit PCM.
// This is an alias for convertMuLawToLinear for backward compatibility.
func convertMuLawToPCM(mulaw []byte) []byte {
	return convertMuLawToLinear(mulaw)
}
