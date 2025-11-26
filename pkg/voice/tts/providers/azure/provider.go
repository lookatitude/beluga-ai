package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	ttsiface "github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
)

// AzureProvider implements the TTSProvider interface for Azure Speech Services.
type AzureProvider struct {
	config     *AzureConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewAzureProvider creates a new Azure Speech Services provider.
func NewAzureProvider(config *tts.Config) (ttsiface.TTSProvider, error) {
	if config == nil {
		return nil, tts.NewTTSError("NewAzureProvider", tts.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Azure config
	azConfig := &AzureConfig{
		Config: config,
	}

	// Set defaults if not provided
	if azConfig.Region == "" {
		azConfig.Region = "eastus"
	}
	if azConfig.VoiceName == "" {
		azConfig.VoiceName = "en-US-AriaNeural"
	}
	if azConfig.Language == "" {
		azConfig.Language = "en-US"
	}
	if azConfig.VoiceRate == "" {
		azConfig.VoiceRate = "medium"
	}
	if azConfig.VoicePitch == "" {
		azConfig.VoicePitch = "medium"
	}
	if azConfig.AudioFormat == "" {
		azConfig.AudioFormat = "audio-24khz-48kbitrate-mono-mp3"
	}
	// Use BaseURL from embedded Config if available
	if azConfig.BaseURL == "" {
		if config.BaseURL != "" {
			azConfig.BaseURL = config.BaseURL
		}
	}
	if azConfig.Timeout == 0 {
		azConfig.Timeout = 30 * time.Second
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

// GenerateSpeech implements the TTSProvider interface using Azure Speech Services API.
func (p *AzureProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	startTime := time.Now()

	// Build SSML
	ssml := p.config.buildSSML(text)

	// Build request URL
	url := p.config.GetBaseURL() + "/cognitiveservices/v1"

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(ssml))
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeInvalidRequest, err)
	}

	// Set headers
	req.Header.Set("Ocp-Apim-Subscription-Key", p.config.APIKey)
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-OutputFormat", p.config.AudioFormat)
	req.Header.Set("User-Agent", "beluga-ai-tts")

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := p.config.MaxRetries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			delay := time.Duration(float64(p.config.RetryDelay) * float64(attempt) * p.config.RetryBackoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = p.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		// Check if error is retryable
		if err != nil && !tts.IsRetryableError(err) {
			break
		}
		if resp != nil && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	if err != nil {
		_ = time.Since(startTime) // Record duration for potential metrics
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeTimeout, err)
		}
		return nil, tts.ErrorFromHTTPStatus("GenerateSpeech", 0, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		_ = time.Since(startTime) // Record duration for potential metrics
		return nil, tts.ErrorFromHTTPStatus("GenerateSpeech", resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	// Read audio data
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeInvalidResponse, err)
	}

	if len(audio) == 0 {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeEmptyResponse,
			errors.New("no audio data in response"))
	}

	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := tts.GetMetrics()
		if metrics != nil {
			metrics.RecordGeneration(ctx, "azure", p.config.VoiceName, p.config.Language, duration)
		}
	}

	return audio, nil
}

// StreamGenerate implements the TTSProvider interface
// Azure Speech Services supports streaming, but for simplicity we return the full audio as a reader.
func (p *AzureProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	// Generate speech first
	audio, err := p.GenerateSpeech(ctx, text)
	if err != nil {
		return nil, err
	}

	// Return as reader
	return bytes.NewReader(audio), nil
}

// buildSSML builds the SSML for Azure TTS.
func (c *AzureConfig) buildSSML(text string) string {
	var ssml strings.Builder
	ssml.WriteString(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="`)
	ssml.WriteString(html.EscapeString(c.Language))
	ssml.WriteString(`">`)
	ssml.WriteString(`<voice name="`)
	ssml.WriteString(html.EscapeString(c.VoiceName))
	ssml.WriteString(`"`)

	if c.VoiceStyle != "" {
		ssml.WriteString(` style="`)
		ssml.WriteString(html.EscapeString(c.VoiceStyle))
		ssml.WriteString(`"`)
	}

	ssml.WriteString(`>`)

	if c.VoiceRate != "" && c.VoiceRate != "medium" {
		ssml.WriteString(`<prosody rate="`)
		ssml.WriteString(html.EscapeString(c.VoiceRate))
		ssml.WriteString(`"`)
		if c.VoicePitch != "" && c.VoicePitch != "medium" {
			ssml.WriteString(` pitch="`)
			ssml.WriteString(html.EscapeString(c.VoicePitch))
			ssml.WriteString(`"`)
		}
		ssml.WriteString(`>`)
		ssml.WriteString(html.EscapeString(text))
		ssml.WriteString(`</prosody>`)
	} else {
		ssml.WriteString(html.EscapeString(text))
	}

	ssml.WriteString(`</voice>`)
	ssml.WriteString(`</speak>`)
	return ssml.String()
}
