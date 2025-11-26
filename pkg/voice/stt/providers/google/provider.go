package google

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	sttiface "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

// GoogleProvider implements the STTProvider interface for Google Cloud Speech-to-Text.
type GoogleProvider struct {
	config     *GoogleConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewGoogleProvider creates a new Google Cloud Speech-to-Text provider.
func NewGoogleProvider(config *stt.Config) (sttiface.STTProvider, error) {
	if config == nil {
		return nil, stt.NewSTTError("NewGoogleProvider", stt.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Google config
	gConfig := &GoogleConfig{
		Config: config,
	}

	// Set defaults if not provided
	if gConfig.Model == "" {
		gConfig.Model = "default"
	}
	if gConfig.LanguageCode == "" {
		gConfig.LanguageCode = "en-US"
	}
	// Use BaseURL from embedded Config if available
	if gConfig.BaseURL == "" {
		if config.BaseURL != "" {
			gConfig.BaseURL = config.BaseURL
		} else {
			gConfig.BaseURL = "https://speech.googleapis.com"
		}
	}
	if gConfig.Timeout == 0 {
		gConfig.Timeout = 30 * time.Second
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: gConfig.Timeout,
	}

	return &GoogleProvider{
		config:     gConfig,
		httpClient: httpClient,
	}, nil
}

// Transcribe implements the STTProvider interface using Google Cloud Speech-to-Text API.
func (p *GoogleProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return p.TranscribeREST(ctx, audio)
}

// StartStreaming implements the STTProvider interface using Google Cloud Speech-to-Text streaming API
// Note: Full streaming implementation requires gRPC client setup with proper authentication
// For now, this returns an error indicating streaming is not yet fully implemented.
func (p *GoogleProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	// TODO: Implement full gRPC streaming using google.golang.org/api/speech/v1
	// This requires proper service account credentials or OAuth setup
	return nil, stt.NewSTTError("StartStreaming", stt.ErrCodeStreamError,
		errors.New("Google Cloud Speech-to-Text streaming requires gRPC implementation (not yet fully implemented)"))
}
