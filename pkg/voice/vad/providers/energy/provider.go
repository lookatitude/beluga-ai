package energy

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
)

// EnergyProvider implements the VADProvider interface for Energy-based VAD.
type EnergyProvider struct {
	config            *EnergyConfig
	energyHistory     []float64
	adaptiveThreshold float64
	mu                sync.RWMutex
}

// NewEnergyProvider creates a new Energy-based VAD provider.
func NewEnergyProvider(config *vad.Config) (vadiface.VADProvider, error) {
	if config == nil {
		return nil, vad.NewVADError("NewEnergyProvider", vad.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Energy config
	energyConfig := &EnergyConfig{
		Config: config,
	}

	// Set defaults if not provided
	if energyConfig.Threshold == 0 {
		energyConfig.Threshold = 0.01
	}
	if energyConfig.EnergyWindowSize == 0 {
		energyConfig.EnergyWindowSize = 256
	}
	if energyConfig.MinSpeechDuration == 0 {
		energyConfig.MinSpeechDuration = 250 * time.Millisecond
	}
	if energyConfig.MaxSilenceDuration == 0 {
		energyConfig.MaxSilenceDuration = 500 * time.Millisecond
	}
	if energyConfig.NoiseFloor == 0 {
		energyConfig.NoiseFloor = 0.001
	}

	provider := &EnergyProvider{
		config:            energyConfig,
		energyHistory:     make([]float64, 0, 100),
		adaptiveThreshold: energyConfig.Threshold,
	}

	return provider, nil
}

// Process implements the VADProvider interface.
func (p *EnergyProvider) Process(ctx context.Context, audio []byte) (bool, error) {
	if len(audio) == 0 {
		return false, nil
	}

	// Calculate energy
	energy := calculateEnergy(audio)

	// Update energy history for adaptive threshold
	p.mu.Lock()
	p.energyHistory = append(p.energyHistory, energy)
	if len(p.energyHistory) > 100 {
		p.energyHistory = p.energyHistory[1:]
	}

	// Calculate adaptive threshold if enabled
	threshold := p.config.Threshold
	if p.config.AdaptiveThreshold && len(p.energyHistory) > 10 {
		p.adaptiveThreshold = adaptiveThreshold(p.energyHistory, p.config.NoiseFloor)
		threshold = p.adaptiveThreshold
	}
	p.mu.Unlock()

	// Compare energy to threshold
	return energy >= threshold, nil
}

// ProcessStream implements the VADProvider interface.
func (p *EnergyProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan iface.VADResult, error) {
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

				// Calculate confidence based on energy
				energy := calculateEnergy(audio)
				threshold := p.config.Threshold
				if p.config.AdaptiveThreshold {
					p.mu.RLock()
					threshold = p.adaptiveThreshold
					p.mu.RUnlock()
				}

				confidence := 0.0
				if speech {
					// Confidence is proportional to how much energy exceeds threshold
					excess := (energy - threshold) / threshold
					confidence = math.Min(0.5+excess*0.5, 1.0)
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
