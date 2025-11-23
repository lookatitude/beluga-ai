package rnnoise

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
)

// RNNoiseProvider implements the VADProvider interface for RNNoise VAD
type RNNoiseProvider struct {
	config      *RNNoiseConfig
	model       *RNNoiseModel
	mu          sync.RWMutex
	initialized bool
}

// RNNoiseModel represents a loaded RNNoise model
type RNNoiseModel struct {
	modelPath string
	loaded    bool
}

// NewRNNoiseProvider creates a new RNNoise VAD provider
func NewRNNoiseProvider(config *vad.Config) (vadiface.VADProvider, error) {
	if config == nil {
		return nil, vad.NewVADError("NewRNNoiseProvider", vad.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to RNNoise config
	rnnoiseConfig := &RNNoiseConfig{
		Config: config,
	}

	// Set defaults if not provided
	if rnnoiseConfig.ModelPath == "" {
		rnnoiseConfig.ModelPath = "rnnoise_model.rnn"
	}
	if rnnoiseConfig.Threshold == 0 {
		rnnoiseConfig.Threshold = 0.5
	}
	if rnnoiseConfig.SampleRate == 0 {
		rnnoiseConfig.SampleRate = 48000
	}
	if rnnoiseConfig.FrameSize == 0 {
		rnnoiseConfig.FrameSize = 480
	}
	if rnnoiseConfig.MinSpeechDuration == 0 {
		rnnoiseConfig.MinSpeechDuration = 250 * time.Millisecond
	}
	if rnnoiseConfig.MaxSilenceDuration == 0 {
		rnnoiseConfig.MaxSilenceDuration = 500 * time.Millisecond
	}

	provider := &RNNoiseProvider{
		config:      rnnoiseConfig,
		initialized: false,
	}

	return provider, nil
}

// Process implements the VADProvider interface
func (p *RNNoiseProvider) Process(ctx context.Context, audio []byte) (bool, error) {
	// Lazy initialization - load model on first use
	if err := p.ensureInitialized(ctx); err != nil {
		return false, err
	}

	// Validate frame size
	expectedFrameSize := p.config.FrameSize * 2 // 16-bit samples
	if len(audio) < expectedFrameSize {
		return false, vad.NewVADError("Process", vad.ErrCodeFrameSizeError,
			fmt.Errorf("audio length %d is less than expected %d", len(audio), expectedFrameSize))
	}

	// Process audio using RNNoise model
	// Note: This is a simplified implementation. A full implementation would use
	// the actual RNNoise library (github.com/xiph/rnnoise)
	// For now, we'll use a placeholder that simulates VAD behavior

	// Placeholder: Simple energy-based detection as fallback
	// Real implementation would call: p.model.Process(audio)
	energy := calculateEnergy(audio)
	speechProbability := energy / 1000.0 // Normalize (placeholder)
	if speechProbability > 1.0 {
		speechProbability = 1.0
	}

	return speechProbability >= p.config.Threshold, nil
}

// ProcessStream implements the VADProvider interface
func (p *RNNoiseProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan iface.VADResult, error) {
	// Lazy initialization - load model on first use
	if err := p.ensureInitialized(ctx); err != nil {
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
					confidence = 0.9 // RNNoise typically has high confidence
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

// ensureInitialized loads the RNNoise model if not already loaded
func (p *RNNoiseProvider) ensureInitialized(ctx context.Context) error {
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

	// Load RNNoise model
	model, err := LoadRNNoiseModel(p.config.ModelPath)
	if err != nil {
		return vad.NewVADError("ensureInitialized", vad.ErrCodeModelLoadFailed, err)
	}

	p.model = model
	p.initialized = true

	return nil
}

// LoadRNNoiseModel loads an RNNoise model from the specified path
func LoadRNNoiseModel(modelPath string) (*RNNoiseModel, error) {
	// TODO: Actual RNNoise model loading would go here
	// In a real implementation, this would:
	// 1. Load the RNNoise model file
	// 2. Initialize the RNNoise state
	// 3. Prepare for inference

	model := &RNNoiseModel{
		modelPath: modelPath,
		loaded:    true,
	}

	return model, nil
}

// calculateEnergy calculates the energy of an audio signal
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
