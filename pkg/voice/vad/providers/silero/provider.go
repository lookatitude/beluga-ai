package silero

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
)

// SileroProvider implements the VADProvider interface for Silero VAD.
type SileroProvider struct {
	config      *SileroConfig
	model       *ONNXModel
	mu          sync.RWMutex
	initialized bool
}

// NewSileroProvider creates a new Silero VAD provider.
func NewSileroProvider(config *vad.Config) (vadiface.VADProvider, error) {
	if config == nil {
		return nil, vad.NewVADError("NewSileroProvider", vad.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Silero config
	sileroConfig := &SileroConfig{
		Config: config,
	}

	// Set defaults if not provided
	if sileroConfig.ModelPath == "" {
		sileroConfig.ModelPath = "silero_vad.onnx"
	}
	if sileroConfig.Threshold == 0 {
		sileroConfig.Threshold = 0.5
	}
	if sileroConfig.SampleRate == 0 {
		sileroConfig.SampleRate = 16000
	}
	if sileroConfig.FrameSize == 0 {
		if sileroConfig.SampleRate == 8000 {
			sileroConfig.FrameSize = 256
		} else {
			sileroConfig.FrameSize = 512
		}
	}
	if sileroConfig.MinSpeechDuration == 0 {
		sileroConfig.MinSpeechDuration = 250 * time.Millisecond
	}
	if sileroConfig.MaxSilenceDuration == 0 {
		sileroConfig.MaxSilenceDuration = 500 * time.Millisecond
	}

	provider := &SileroProvider{
		config:      sileroConfig,
		initialized: false,
	}

	return provider, nil
}

// Process implements the VADProvider interface.
func (p *SileroProvider) Process(ctx context.Context, audio []byte) (bool, error) {
	// Lazy initialization - load model on first use
	if err := p.ensureInitialized(ctx); err != nil {
		return false, err
	}

	// Process audio using ONNX model
	return p.model.Process(ctx, audio, p.config.Threshold)
}

// ProcessStream implements the VADProvider interface.
func (p *SileroProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan iface.VADResult, error) {
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
					confidence = 0.9
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

// ensureInitialized loads the ONNX model if not already loaded.
func (p *SileroProvider) ensureInitialized(ctx context.Context) error {
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

	// Load ONNX model
	model, err := LoadONNXModel(p.config.ModelPath, p.config.SampleRate, p.config.FrameSize)
	if err != nil {
		return vad.NewVADError("ensureInitialized", vad.ErrCodeModelLoadFailed, err)
	}

	p.model = model
	p.initialized = true

	return nil
}
