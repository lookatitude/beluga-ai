package spectral

import (
	"sync"
)

// AdaptiveNoiseProfile maintains an adaptive noise profile for spectral subtraction
type AdaptiveNoiseProfile struct {
	mu             sync.RWMutex
	noiseMagnitude []float64
	updateCount    int
	updateRate     int
	alpha          float64 // Smoothing factor
}

// NewAdaptiveNoiseProfile creates a new adaptive noise profile
func NewAdaptiveNoiseProfile(fftSize int, updateRate int, alpha float64) *AdaptiveNoiseProfile {
	return &AdaptiveNoiseProfile{
		noiseMagnitude: make([]float64, fftSize),
		updateRate:     updateRate,
		alpha:          alpha,
	}
}

// Update updates the noise profile with a new noise estimate
func (anp *AdaptiveNoiseProfile) Update(noiseMagnitude []float64) {
	anp.mu.Lock()
	defer anp.mu.Unlock()

	anp.updateCount++
	if anp.updateCount%anp.updateRate != 0 {
		return
	}

	// Exponential moving average update
	for i := range anp.noiseMagnitude {
		if i < len(noiseMagnitude) {
			anp.noiseMagnitude[i] = anp.alpha*anp.noiseMagnitude[i] + (1-anp.alpha)*noiseMagnitude[i]
		}
	}
}

// GetNoiseMagnitude returns the current noise magnitude estimate
func (anp *AdaptiveNoiseProfile) GetNoiseMagnitude() []float64 {
	anp.mu.RLock()
	defer anp.mu.RUnlock()

	// Return a copy
	result := make([]float64, len(anp.noiseMagnitude))
	copy(result, anp.noiseMagnitude)
	return result
}

// SpectralSubtraction performs spectral subtraction noise reduction
func SpectralSubtraction(signalMagnitude, noiseMagnitude []float64, alpha, beta float64) []float64 {
	result := make([]float64, len(signalMagnitude))
	for i := range signalMagnitude {
		// Over-subtraction with spectral floor
		subtracted := signalMagnitude[i] - alpha*noiseMagnitude[i]

		// Apply spectral floor
		floor := beta * signalMagnitude[i]
		if subtracted < floor {
			subtracted = floor
		}

		// Ensure non-negative
		if subtracted < 0 {
			subtracted = 0
		}

		result[i] = subtracted
	}
	return result
}

// EstimateNoiseMagnitude estimates noise magnitude from signal
// This is a simple implementation - in production, use VAD or other methods
func EstimateNoiseMagnitude(signalMagnitude []float64, percentile float64) []float64 {
	// Simple percentile-based estimation
	// In production, use more sophisticated methods
	noiseEstimate := make([]float64, len(signalMagnitude))

	// Use a simple percentile approach
	values := make([]float64, len(signalMagnitude))
	copy(values, signalMagnitude)

	// Sort and take percentile (simplified)
	threshold := percentile * maxFloat64(values)

	for i := range noiseEstimate {
		if signalMagnitude[i] < threshold {
			noiseEstimate[i] = signalMagnitude[i]
		} else {
			noiseEstimate[i] = threshold
		}
	}

	return noiseEstimate
}

// maxFloat64 returns the maximum value in a slice
func maxFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
