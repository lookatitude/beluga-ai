package webrtc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
)

// WebRTCProvider implements the VADProvider interface for WebRTC VAD.
type WebRTCProvider struct {
	config      *WebRTCConfig
	mu          sync.RWMutex
	initialized bool
}

// NewWebRTCProvider creates a new WebRTC VAD provider.
func NewWebRTCProvider(config *vad.Config) (vadiface.VADProvider, error) {
	if config == nil {
		return nil, vad.NewVADError("NewWebRTCProvider", vad.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to WebRTC config
	webrtcConfig := &WebRTCConfig{
		Config: config,
	}

	// Set defaults if not provided
	if webrtcConfig.Mode < 0 || webrtcConfig.Mode > 3 {
		webrtcConfig.Mode = 0
	}
	if webrtcConfig.SampleRate == 0 {
		webrtcConfig.SampleRate = 16000
	}
	if webrtcConfig.FrameSize == 0 {
		// Default to 20ms frames
		webrtcConfig.FrameSize = webrtcConfig.SampleRate / 50
	}
	if webrtcConfig.MinSpeechDuration == 0 {
		webrtcConfig.MinSpeechDuration = 250 * time.Millisecond
	}
	if webrtcConfig.MaxSilenceDuration == 0 {
		webrtcConfig.MaxSilenceDuration = 500 * time.Millisecond
	}

	provider := &WebRTCProvider{
		config:      webrtcConfig,
		initialized: false,
	}

	return provider, nil
}

// Process implements the VADProvider interface.
func (p *WebRTCProvider) Process(ctx context.Context, audio []byte) (bool, error) {
	// Lazy initialization
	if err := p.ensureInitialized(); err != nil {
		return false, err
	}

	// WebRTC VAD requires specific frame sizes
	expectedFrameSize := p.config.FrameSize * 2 // 16-bit samples
	if len(audio) < expectedFrameSize {
		return false, vad.NewVADError("Process", vad.ErrCodeFrameSizeError,
			fmt.Errorf("audio length %d is less than expected %d", len(audio), expectedFrameSize))
	}

	// Process audio using WebRTC VAD
	// Note: This is a simplified implementation. A full implementation would use
	// the actual WebRTC VAD library (github.com/pion/webrtc/v3/pkg/media/rtpdump)
	// For now, we'll use a placeholder that simulates VAD behavior

	// Placeholder: Simple energy-based detection as fallback
	// Real implementation would call: p.vad.Process(audio, p.config.SampleRate)
	energy := calculateEnergy(audio)
	threshold := getThresholdForMode(p.config.Mode)

	return energy >= threshold, nil
}

// ProcessStream implements the VADProvider interface.
func (p *WebRTCProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan iface.VADResult, error) {
	// Lazy initialization
	if err := p.ensureInitialized(); err != nil {
		return nil, err
	}

	resultCh := make(chan iface.VADResult, 10)

	go func() {
		defer close(resultCh)

		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioCh:
				if !ok {
					return
				}

				// Process audio chunk
				speech, err := p.Process(ctx, audio)
				if err != nil {
					resultCh <- iface.VADResult{
						HasVoice:   false,
						Confidence: 0.0,
						Error:      err,
					}
					return
				}

				confidence := 0.0
				if speech {
					confidence = 0.85 // WebRTC VAD typically has high confidence
				}

				resultCh <- iface.VADResult{
					HasVoice:   speech,
					Confidence: confidence,
					Error:      nil,
				}
			}
		}
	}()

	return resultCh, nil
}

// ensureInitialized initializes the WebRTC VAD instance.
func (p *WebRTCProvider) ensureInitialized() error {
	p.mu.RLock()
	initialized := p.initialized
	p.mu.RUnlock()

	if initialized {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if p.initialized {
		return nil
	}

	// TODO: Initialize actual WebRTC VAD
	// In a real implementation, this would:
	// 1. Create a WebRTC VAD instance with the specified mode
	// 2. Configure it with the sample rate and frame size
	// For now, we'll just mark as initialized
	p.initialized = true

	return nil
}

// calculateEnergy calculates the energy of an audio signal.
func calculateEnergy(audio []byte) float64 {
	if len(audio) == 0 {
		return 0.0
	}

	var sum float64
	sampleCount := 0

	for i := 0; i < len(audio); i += 2 {
		if i+1 < len(audio) {
			sample := int16(audio[i]) | int16(audio[i+1])<<8
			value := float64(sample) / 32768.0
			sum += value * value
			sampleCount++
		}
	}

	if sampleCount == 0 {
		return 0.0
	}

	return sum / float64(sampleCount)
}

// getThresholdForMode returns the energy threshold for a given VAD mode.
func getThresholdForMode(mode int) float64 {
	// WebRTC VAD modes have different sensitivity levels
	thresholds := []float64{0.01, 0.015, 0.02, 0.025} // Increasing sensitivity
	if mode >= 0 && mode < len(thresholds) {
		return thresholds[mode]
	}
	return thresholds[0]
}
