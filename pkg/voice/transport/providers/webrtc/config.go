package webrtc

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

// WebRTCConfig extends the base Transport config with WebRTC-specific settings
type WebRTCConfig struct {
	*transport.Config

	// STUNServers specifies STUN server URLs
	STUNServers []string `mapstructure:"stun_servers" yaml:"stun_servers" default:"stun:stun.l.google.com:19302"`

	// TURNServers specifies TURN server URLs
	TURNServers []string `mapstructure:"turn_servers" yaml:"turn_servers"`

	// ICEConnectionTimeout specifies the ICE connection timeout
	ICEConnectionTimeout time.Duration `mapstructure:"ice_connection_timeout" yaml:"ice_connection_timeout" default:"30s"`

	// ICERestartTimeout specifies the ICE restart timeout
	ICERestartTimeout time.Duration `mapstructure:"ice_restart_timeout" yaml:"ice_restart_timeout" default:"5s"`

	// SignalingURL specifies the WebRTC signaling server URL
	SignalingURL string `mapstructure:"signaling_url" yaml:"signaling_url"`

	// EnableDTLS enables DTLS for secure connections
	EnableDTLS bool `mapstructure:"enable_dtls" yaml:"enable_dtls" default:"true"`

	// EnableSRTP enables SRTP for secure RTP
	EnableSRTP bool `mapstructure:"enable_srtp" yaml:"enable_srtp" default:"true"`

	// AudioCodec specifies the preferred audio codec ("opus", "pcmu", "pcma")
	AudioCodec string `mapstructure:"audio_codec" yaml:"audio_codec" default:"opus" validate:"oneof=opus pcmu pcma"`

	// BundlePolicy specifies the bundle policy ("balanced", "max-compat", "max-bundle")
	BundlePolicy string `mapstructure:"bundle_policy" yaml:"bundle_policy" default:"balanced" validate:"oneof=balanced max-compat max-bundle"`

	// RTCPMuxPolicy specifies the RTCP mux policy ("negotiate", "require")
	RTCPMuxPolicy string `mapstructure:"rtcp_mux_policy" yaml:"rtcp_mux_policy" default:"require" validate:"oneof=negotiate require"`
}

// DefaultWebRTCConfig returns a default WebRTC Transport configuration
func DefaultWebRTCConfig() *WebRTCConfig {
	return &WebRTCConfig{
		Config:               transport.DefaultConfig(),
		STUNServers:          []string{"stun:stun.l.google.com:19302"},
		TURNServers:          []string{},
		ICEConnectionTimeout: 30 * time.Second,
		ICERestartTimeout:    5 * time.Second,
		EnableDTLS:           true,
		EnableSRTP:           true,
		AudioCodec:           "opus",
		BundlePolicy:         "balanced",
		RTCPMuxPolicy:        "require",
	}
}
