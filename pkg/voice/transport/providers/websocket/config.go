package websocket

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

// WebSocketConfig extends the base Transport config with WebSocket-specific settings
type WebSocketConfig struct {
	*transport.Config

	// ReadBufferSize specifies the read buffer size in bytes
	ReadBufferSize int `mapstructure:"read_buffer_size" yaml:"read_buffer_size" default:"4096" validate:"min=1024,max=65536"`

	// WriteBufferSize specifies the write buffer size in bytes
	WriteBufferSize int `mapstructure:"write_buffer_size" yaml:"write_buffer_size" default:"4096" validate:"min=1024,max=65536"`

	// HandshakeTimeout specifies the WebSocket handshake timeout
	HandshakeTimeout time.Duration `mapstructure:"handshake_timeout" yaml:"handshake_timeout" default:"10s" validate:"min=1s,max=60s"`

	// EnableCompression enables WebSocket compression
	EnableCompression bool `mapstructure:"enable_compression" yaml:"enable_compression" default:"false"`

	// Subprotocols specifies WebSocket subprotocols
	Subprotocols []string `mapstructure:"subprotocols" yaml:"subprotocols"`

	// PingInterval specifies the ping interval for keepalive
	PingInterval time.Duration `mapstructure:"ping_interval" yaml:"ping_interval" default:"30s" validate:"min=5s,max=300s"`

	// PongWait specifies the pong wait timeout
	PongWait time.Duration `mapstructure:"pong_wait" yaml:"pong_wait" default:"60s" validate:"min=10s,max=600s"`

	// MaxMessageSize specifies the maximum message size in bytes
	MaxMessageSize int64 `mapstructure:"max_message_size" yaml:"max_message_size" default:"1048576" validate:"min=1024,max=10485760"`
}

// DefaultWebSocketConfig returns a default WebSocket Transport configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Config:            transport.DefaultConfig(),
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
		Subprotocols:      []string{},
		PingInterval:      30 * time.Second,
		PongWait:          60 * time.Second,
		MaxMessageSize:    1048576, // 1MB
	}
}
