package azure

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

// AzureProvider implements the STTProvider interface for Azure Speech Services
type AzureProvider struct {
	config     *AzureConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewAzureProvider creates a new Azure Speech Services provider
func NewAzureProvider(config *stt.Config) (sttiface.STTProvider, error) {
	if config == nil {
		return nil, stt.NewSTTError("NewAzureProvider", stt.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to Azure config
	azConfig := &AzureConfig{
		Config: config,
	}

	// Set defaults if not provided
	if azConfig.Region == "" {
		azConfig.Region = "eastus"
	}
	if azConfig.Language == "" {
		azConfig.Language = "en-US"
	}
	// Use BaseURL from embedded Config if available
	if azConfig.BaseURL == "" && config.BaseURL != "" {
		azConfig.BaseURL = config.BaseURL
	}
	if azConfig.Timeout == 0 {
		if config.Timeout != 0 {
			azConfig.Timeout = config.Timeout
		} else {
			azConfig.Timeout = 30 * time.Second
		}
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: azConfig.Timeout,
	}

	return &AzureProvider{
		config:     azConfig,
		httpClient: httpClient,
	}, nil
}

// Transcribe implements the STTProvider interface using Azure Speech Services REST API
func (p *AzureProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return p.TranscribeREST(ctx, audio)
}

// StartStreaming implements the STTProvider interface using Azure Speech Services WebSocket API
func (p *AzureProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	// Use WebSocket implementation
	return NewAzureStreamingSession(ctx, p.config)
}
