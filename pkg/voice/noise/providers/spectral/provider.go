package spectral

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/lookatitude/beluga-ai/pkg/voice/noise/iface"
)

// SpectralProvider implements noise cancellation using Spectral Subtraction.
type SpectralProvider struct {
	config         *SpectralConfig
	noiseProfile   *AdaptiveNoiseProfile
	windowFunction WindowFunction
	mu             sync.RWMutex
}

// NewSpectralProvider creates a new Spectral Subtraction noise cancellation provider.
func NewSpectralProvider(config *noise.Config) (iface.NoiseCancellation, error) {
	if config == nil {
		return nil, noise.NewNoiseCancellationError("NewSpectralProvider", noise.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Spectral config
	spectralConfig := &SpectralConfig{
		Config: config,
	}

	// Set defaults if not provided
	if spectralConfig.Alpha == 0 {
		spectralConfig.Alpha = 2.0
	}
	if spectralConfig.Beta == 0 {
		spectralConfig.Beta = 0.1
	}
	if spectralConfig.FFTSize == 0 {
		spectralConfig.FFTSize = 512
	}
	if spectralConfig.WindowType == "" {
		spectralConfig.WindowType = "hann"
	}
	if spectralConfig.Overlap == 0 {
		spectralConfig.Overlap = 0.5
	}
	if spectralConfig.NoiseProfileUpdateRate == 0 {
		spectralConfig.NoiseProfileUpdateRate = 100
	}

	// Initialize noise profile
	noiseProfile := NewAdaptiveNoiseProfile(spectralConfig.FFTSize, spectralConfig.NoiseProfileUpdateRate, 0.95)

	// Get window function
	windowFunction := GetWindowFunction(spectralConfig.WindowType)

	return &SpectralProvider{
		config:         spectralConfig,
		noiseProfile:   noiseProfile,
		windowFunction: windowFunction,
	}, nil
}

// Process implements the NoiseCancellation interface.
func (p *SpectralProvider) Process(ctx context.Context, audio []byte) ([]byte, error) {
	if len(audio) == 0 {
		return audio, nil
	}

	// Convert audio bytes to float64 samples
	samples := bytesToFloat64(audio)

	// Apply window function
	windowed := make([]float64, len(samples))
	copy(windowed, samples)
	p.windowFunction(windowed)

	// Perform FFT
	fft := FFT(windowed)
	magnitude := MagnitudeSpectrum(fft)
	phase := PhaseSpectrum(fft)

	// Get noise estimate
	noiseMagnitude := p.noiseProfile.GetNoiseMagnitude()
	if len(noiseMagnitude) == 0 || len(noiseMagnitude) != len(magnitude) {
		// Initialize noise profile if empty
		noiseMagnitude = make([]float64, len(magnitude))
		copy(noiseMagnitude, magnitude)
		p.noiseProfile.Update(noiseMagnitude)
	}

	// Perform spectral subtraction
	cleanedMagnitude := SpectralSubtraction(magnitude, noiseMagnitude, p.config.Alpha, p.config.Beta)

	// Update noise profile (adaptive)
	if p.config.EnableAdaptiveProcessing {
		noiseEstimate := EstimateNoiseMagnitude(magnitude, 0.1)
		p.noiseProfile.Update(noiseEstimate)
	}

	// Reconstruct signal
	cleanedFFT := ReconstructFromMagnitudePhase(cleanedMagnitude, phase)
	cleanedSamples := IFFT(cleanedFFT)

	// Convert back to bytes
	cleanedAudio := float64ToBytes(cleanedSamples)

	return cleanedAudio, nil
}

// ProcessStream implements the NoiseCancellation interface.
func (p *SpectralProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan []byte, error) {
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

				processed, err := p.Process(ctx, audio)
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

// bytesToFloat64 converts audio bytes to float64 samples.
func bytesToFloat64(audio []byte) []float64 {
	samples := make([]float64, len(audio))
	for i, b := range audio {
		// Convert byte (0-255) to float64 (-1.0 to 1.0)
		samples[i] = (float64(b) - 128.0) / 128.0
	}
	return samples
}

// float64ToBytes converts float64 samples to audio bytes.
func float64ToBytes(samples []float64) []byte {
	audio := make([]byte, len(samples))
	for i, s := range samples {
		// Clamp to [-1, 1] range
		s = math.Max(-1.0, math.Min(1.0, s))
		// Convert float64 (-1.0 to 1.0) to byte (0-255)
		audio[i] = byte((s + 1.0) * 127.5)
	}
	return audio
}
