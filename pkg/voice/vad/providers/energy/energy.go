package energy

import (
	"math"
)

// calculateEnergy calculates the energy of an audio signal
func calculateEnergy(audio []byte) float64 {
	if len(audio) == 0 {
		return 0.0
	}

	var sum float64
	sampleCount := 0

	for i := 0; i < len(audio); i += 2 {
		if i+1 < len(audio) {
			// Convert 16-bit sample to float
			sample := int16(audio[i]) | int16(audio[i+1])<<8
			value := float64(sample) / 32768.0 // Normalize to [-1, 1]
			sum += value * value
			sampleCount++
		}
	}

	if sampleCount == 0 {
		return 0.0
	}

	// Return RMS energy
	return math.Sqrt(sum / float64(sampleCount))
}

// calculateWindowedEnergy calculates energy over a sliding window
func calculateWindowedEnergy(audio []byte, windowSize int) []float64 {
	if len(audio) < windowSize*2 {
		// If audio is smaller than window, return single energy value
		return []float64{calculateEnergy(audio)}
	}

	energies := make([]float64, 0)
	for i := 0; i <= len(audio)-windowSize*2; i += windowSize {
		window := audio[i : i+windowSize*2]
		energy := calculateEnergy(window)
		energies = append(energies, energy)
	}

	return energies
}

// adaptiveThreshold calculates an adaptive threshold based on background noise
func adaptiveThreshold(energies []float64, noiseFloor float64) float64 {
	if len(energies) == 0 {
		return noiseFloor
	}

	// Calculate median energy as noise floor estimate
	sorted := make([]float64, len(energies))
	copy(sorted, energies)

	// Simple sort (bubble sort for small arrays)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	median := sorted[len(sorted)/2]

	// Adaptive threshold is median + margin
	threshold := median + (median * 0.5) // 50% margin above median

	// Ensure threshold is at least noise floor
	if threshold < noiseFloor {
		threshold = noiseFloor
	}

	return threshold
}
