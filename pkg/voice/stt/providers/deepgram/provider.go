package deepgram

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	sttiface "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

// DeepgramProvider implements the STTProvider interface for Deepgram
type DeepgramProvider struct {
	config     *DeepgramConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewDeepgramProvider creates a new Deepgram STT provider
func NewDeepgramProvider(config *stt.Config) (sttiface.STTProvider, error) {
	if config == nil {
		return nil, stt.NewSTTError("NewDeepgramProvider", stt.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to Deepgram config
	dgConfig := &DeepgramConfig{
		Config: config,
	}

	// Set defaults if not provided
	if dgConfig.Model == "" {
		dgConfig.Model = "nova-2"
	}
	// Use BaseURL from embedded Config if available
	if dgConfig.BaseURL == "" {
		if config.BaseURL != "" {
			dgConfig.BaseURL = config.BaseURL
		} else {
			dgConfig.BaseURL = "https://api.deepgram.com"
		}
	}
	if dgConfig.WebSocketURL == "" {
		dgConfig.WebSocketURL = "wss://api.deepgram.com/v1/listen"
	}
	if dgConfig.Timeout == 0 {
		if config.Timeout != 0 {
			dgConfig.Timeout = config.Timeout
		} else {
			dgConfig.Timeout = 30 * time.Second
		}
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: dgConfig.Timeout,
	}

	return &DeepgramProvider{
		config:     dgConfig,
		httpClient: httpClient,
	}, nil
}

// Transcribe implements the STTProvider interface using Deepgram REST API
func (p *DeepgramProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return p.TranscribeREST(ctx, audio)
}

// StartStreaming implements the STTProvider interface using Deepgram WebSocket API
func (p *DeepgramProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	// Use WebSocket implementation
	return NewDeepgramStreamingSession(ctx, p.config)
}
