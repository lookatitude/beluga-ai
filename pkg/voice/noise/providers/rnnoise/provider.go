package rnnoise

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/lookatitude/beluga-ai/pkg/voice/noise/iface"
)

// RNNoiseProvider implements noise cancellation using RNNoise
type RNNoiseProvider struct {
	config *RNNoiseConfig
	model  *RNNoiseModel
	mu     sync.RWMutex
}

// NewRNNoiseProvider creates a new RNNoise noise cancellation provider
func NewRNNoiseProvider(config *noise.Config) (iface.NoiseCancellation, error) {
	if config == nil {
		return nil, noise.NewNoiseCancellationError("NewRNNoiseProvider", noise.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to RNNoise config
	rnnoiseConfig := &RNNoiseConfig{
		Config: config,
	}

	// Set defaults if not provided
	if rnnoiseConfig.ModelPath == "" {
		rnnoiseConfig.ModelPath = "rnnoise.rnn"
	}

	// Get frame size and sample rate from config (use defaults if zero)
	frameSize := config.FrameSize
	if frameSize == 0 {
		frameSize = 480
	}
	sampleRate := config.SampleRate
	if sampleRate == 0 {
		sampleRate = 48000
	}

	// Validate RNNoise-specific requirements
	if frameSize != 480 {
		return nil, noise.NewNoiseCancellationError("NewRNNoiseProvider", noise.ErrCodeFrameSizeError,
			fmt.Errorf("RNNoise requires frame size of 480 samples, got %d", config.FrameSize))
	}
	if sampleRate != 48000 {
		return nil, noise.NewNoiseCancellationError("NewRNNoiseProvider", noise.ErrCodeSampleRateError,
			fmt.Errorf("RNNoise requires sample rate of 48000 Hz, got %d", config.SampleRate))
	}

	// Set validated values
	rnnoiseConfig.FrameSize = frameSize
	rnnoiseConfig.SampleRate = sampleRate

	// Create and load model
	model := NewRNNoiseModel(rnnoiseConfig.ModelPath)
	if err := model.Load(); err != nil {
		return nil, noise.NewNoiseCancellationError("NewRNNoiseProvider", noise.ErrCodeModelLoadFailed, err)
	}

	return &RNNoiseProvider{
		config: rnnoiseConfig,
		model:  model,
	}, nil
}

// Process implements the NoiseCancellation interface
func (p *RNNoiseProvider) Process(ctx context.Context, audio []byte) ([]byte, error) {
	if len(audio) == 0 {
		return audio, nil
	}

	// RNNoise processes frames of exactly 480 samples
	// If audio is not exactly 480 samples, we pad or truncate
	expectedSize := p.config.FrameSize * 2 // 2 bytes per sample (16-bit)

	var frame []byte
	if len(audio) < expectedSize {
		// Pad with zeros
		frame = make([]byte, expectedSize)
		copy(frame, audio)
	} else if len(audio) > expectedSize {
		// Truncate
		frame = audio[:expectedSize]
	} else {
		frame = audio
	}

	// Process frame using RNNoise model
	processed, err := p.model.Process(frame)
	if err != nil {
		return nil, noise.NewNoiseCancellationError("Process", noise.ErrCodeProcessingError, err)
	}

	// Return processed audio (trim to original size if padded)
	if len(audio) < expectedSize {
		return processed[:len(audio)], nil
	}
	return processed, nil
}

// ProcessStream implements the NoiseCancellation interface
func (p *RNNoiseProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan []byte, error) {
	processedCh := make(chan []byte, 10)

	go func() {
		defer close(processedCh)

		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioCh:
				if !ok {
					return
				}

				// Process frame
				processed, err := p.model.Process(audio)
				if err != nil {
					return
				}

				select {
				case <-ctx.Done():
					return
				case processedCh <- processed:
				}
			}
		}
	}()

	return processedCh, nil
}
