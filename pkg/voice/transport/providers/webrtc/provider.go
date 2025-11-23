package webrtc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
	transportiface "github.com/lookatitude/beluga-ai/pkg/voice/transport/iface"
)

// WebRTCTransport implements the Transport interface for WebRTC
type WebRTCTransport struct {
	config        *WebRTCConfig
	peerConn      *PeerConnection
	audioCh       chan []byte
	mu            sync.RWMutex
	connected     bool
	audioCallback func([]byte)
}

// PeerConnection represents a WebRTC peer connection
type PeerConnection struct {
	connected bool
	mu        sync.RWMutex
}

// NewWebRTCTransport creates a new WebRTC Transport provider
func NewWebRTCTransport(config *transport.Config) (transportiface.Transport, error) {
	if config == nil {
		return nil, transport.NewTransportError("NewWebRTCTransport", transport.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to WebRTC config
	webrtcConfig := &WebRTCConfig{
		Config: config,
	}

	// Set defaults if not provided
	if len(webrtcConfig.STUNServers) == 0 {
		webrtcConfig.STUNServers = []string{"stun:stun.l.google.com:19302"}
	}
	if webrtcConfig.ICEConnectionTimeout == 0 {
		webrtcConfig.ICEConnectionTimeout = 30 * time.Second
	}
	if webrtcConfig.ICERestartTimeout == 0 {
		webrtcConfig.ICERestartTimeout = 5 * time.Second
	}
	if webrtcConfig.AudioCodec == "" {
		webrtcConfig.AudioCodec = "opus"
	}
	if webrtcConfig.BundlePolicy == "" {
		webrtcConfig.BundlePolicy = "balanced"
	}
	if webrtcConfig.RTCPMuxPolicy == "" {
		webrtcConfig.RTCPMuxPolicy = "require"
	}

	// Create peer connection
	peerConn := &PeerConnection{
		connected: false,
	}

	return &WebRTCTransport{
		config:    webrtcConfig,
		peerConn:  peerConn,
		audioCh:   make(chan []byte, 100),
		connected: false,
	}, nil
}

// SendAudio implements the Transport interface
func (t *WebRTCTransport) SendAudio(ctx context.Context, audio []byte) error {
	t.mu.RLock()
	connected := t.connected
	t.mu.RUnlock()

	if !connected {
		return transport.NewTransportError("SendAudio", transport.ErrCodeNotConnected,
			fmt.Errorf("transport not connected"))
	}

	// TODO: Actual WebRTC audio sending would go here
	// In a real implementation, this would:
	// 1. Encode audio using the configured codec (Opus, PCMU, PCMA)
	// 2. Packetize into RTP packets
	// 3. Send via WebRTC data channel or RTP track
	// 4. Handle errors and retries

	// Placeholder: Just validate audio data
	if len(audio) == 0 {
		return transport.NewTransportError("SendAudio", transport.ErrCodeInvalidInput,
			fmt.Errorf("audio data is empty"))
	}

	return nil
}

// ReceiveAudio implements the Transport interface
func (t *WebRTCTransport) ReceiveAudio() <-chan []byte {
	return t.audioCh
}

// OnAudioReceived implements the Transport interface
func (t *WebRTCTransport) OnAudioReceived(callback func(audio []byte)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.audioCallback = callback

	// TODO: In a real implementation, this would set up the callback
	// to be called when audio is received from the WebRTC peer connection
}

// Close implements the Transport interface
func (t *WebRTCTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	// Close peer connection
	if t.peerConn != nil {
		t.peerConn.mu.Lock()
		t.peerConn.connected = false
		t.peerConn.mu.Unlock()
	}

	// Close audio channel
	close(t.audioCh)

	t.connected = false
	return nil
}

// Connect is a helper method to establish WebRTC connection
// Note: This is not part of the Transport interface but useful for testing
func (t *WebRTCTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// TODO: Actual WebRTC connection establishment would go here
	// In a real implementation, this would:
	// 1. Create WebRTC peer connection
	// 2. Configure STUN/TURN servers
	// 3. Create audio tracks
	// 4. Exchange SDP offers/answers via signaling
	// 5. Establish ICE connection
	// 6. Start RTP/RTCP streams

	// Placeholder: Mark as connected
	t.peerConn.mu.Lock()
	t.peerConn.connected = true
	t.peerConn.mu.Unlock()

	t.connected = true
	return nil
}
